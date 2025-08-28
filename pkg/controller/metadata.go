/*
 * Copyright 2023 InfAI (CC SES)
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
	"encoding/json"
	"runtime/debug"

	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
)

func (this *Controller) ListDeploymentMetadata(query model.MetadataQuery) (result []model.DeploymentMetadata, err error) {
	return this.db.ListDeploymentMetadata(query)
}

func (this *Controller) UpdateDeploymentMetadata(networkId string, metadata model.Metadata) {
	err := this.db.SaveDeploymentMetadata(model.DeploymentMetadata{
		Metadata: metadata,
		SyncInfo: model.SyncInfo{
			NetworkId:       networkId,
			IsPlaceholder:   false,
			MarkedForDelete: false,
			SyncDate:        configuration.TimeNow(),
		},
	})
	if err != nil {
		this.config.GetLogger().Error("error", "error", err, "stack", debug.Stack())
	}
	this.notifyProcessDeploymentDone(metadata.DeploymentModel.Id)
}

type DoneNotification struct {
	Command string `json:"command"`
	Id      string `json:"id"`
	Handler string `json:"handler"`
}

func (this *Controller) notifyProcessDeploymentDone(id string) {
	if this.deploymentDoneNotifier != nil {
		msg, err := json.Marshal(DoneNotification{
			Command: "PUT",
			Id:      id,
			Handler: "github.com/SENERGY-Platform/process-sync",
		})
		if err != nil {
			this.config.GetLogger().Error("error", "error", err, "stack", debug.Stack())
			return
		}
		err = this.deploymentDoneNotifier.Produce(id, msg)
		if err != nil {
			this.config.GetLogger().Error("error", "error", err, "stack", debug.Stack())
			return
		}
	}
}
