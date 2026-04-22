/*
 * Copyright 2026 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package controller

import (
	"fmt"

	developerNotifications "github.com/SENERGY-Platform/developer-notifications/pkg/client"
	notifyModel "github.com/SENERGY-Platform/notifier/pkg/model"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
)

func (this *Controller) HandleProcessSyncNetworkError(networkId string, msg string) {
	userId, err := this.devicerepo.GetNetworkOwner(networkId)
	if err != nil {
		this.logger.Error("unable to get network owner", "error", err.Error(), "network-id", networkId)
		return
	}
	this.logger.Warn("process-sync-client error", "network-id", networkId, "error", msg, "user", userId)

	if this.userNotifications != nil {
		_, err = this.userNotifications.CreateNotification(nil, notifyModel.Notification{
			UserId:  userId,
			Title:   "Fog-Process-Error",
			Message: "got an error from the process-sync-client: " + msg,
			Topic:   notifyModel.TopicMGW,
		}, new(int64(86400)))
	}

	if this.devNotifications != nil {
		err = this.devNotifications.SendMessage(developerNotifications.Message{
			Sender: "github.com/SENERGY-Platform/process-sync",
			Title:  "Fog-Process-Error",
			Tags:   []string{"fog", userId, networkId},
			Body: fmt.Sprintf("Notification For %v in network %v\nTitle: %v\nMessage: %v\n",
				userId,
				networkId,
				"Fog-Process-Error in Hub "+networkId,
				msg,
			),
		})
		if err != nil {
			this.logger.Error("unable to send developer-notification", "snrgy-log-type", "error", "error", err.Error(), "user", userId)
		}
	}

}

func (this *Controller) HandleProcessSyncError(networkId string, msg model.ErrorMessage) {
	if msg.NetworkId != "" && msg.NetworkId != networkId {
		this.logger.Error("unexpected networkId in mgw error message", "networkId", msg.NetworkId, "expected", networkId)
		return
	}
	msg.NetworkId = networkId
	if msg.BusinessKey != "" {
		//handle error on process-start
		this.HandleProcessSyncStartError(msg)
		return
	}
	if msg.DeploymentId != "" {
		//handle error on process-deployment
		this.HandleProcessSyncDeploymentError(msg)
		return
	}
	this.logger.Error("unknown process-sync error", "error", fmt.Sprintf("%#v", msg))
	this.HandleProcessSyncNetworkError(networkId, msg.Error)
}

func (this *Controller) HandleProcessSyncDeploymentError(msg model.ErrorMessage) {
	//msg.CamundaDeploymentId is not relevant
	//after a successful deployment the camunda deployment id replaces the deployment id in the warden
	//but this error is received because the deployment is not successful so the old id is still active
	webhooks, err := this.db.MarkErrorOnDeploymentWarden(msg.NetworkId, msg.DeploymentId, msg.Error)
	if err != nil {
		this.logger.Error("unable to add deployment error", "error", err.Error(), "network-id", msg.NetworkId, "deployment-id", msg.DeploymentId, "msg", msg.Error)
		return
	}
	this.TriggerWebhooks(webhooks, model.OnError, "deployments", msg.NetworkId, msg.DeploymentId, msg.Error)
	this.HandleProcessSyncNetworkError(msg.NetworkId, fmt.Sprintf("process deployment %v in network %v failed: %v", msg.DeploymentId, msg.NetworkId, msg.Error))
}

func (this *Controller) HandleProcessSyncStartError(msg model.ErrorMessage) {
	webhooks, err := this.db.MarkErrorOnInstanceWarden(msg.NetworkId, msg.BusinessKey, msg.Error)
	if err != nil {
		this.logger.Error("unable to add deployment error", "error", err.Error(), "network-id", msg.NetworkId, "deployment-id", msg.DeploymentId, "msg", msg.Error)
		return
	}
	this.TriggerWebhooks(webhooks, model.OnError, "process-instances-by-business-key", msg.NetworkId, msg.BusinessKey, msg.Error)
	this.HandleProcessSyncNetworkError(msg.NetworkId, fmt.Sprintf("process start (businessKey='%v') for deployment %v in network %v failed: %v", msg.BusinessKey, msg.DeploymentId, msg.NetworkId, msg.Error))
}
