/*
 * Copyright 2024 InfAI (CC SES)
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
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
)

func (this *Controller) logAndNotify(networkid string, incident camundamodel.Incident) {
	if this.devNotifications != nil {
		go func() {
			this.logger.Info("process-incident", "snrgy-log-type", "process-incident", "network-id", networkid, "error", incident.ErrorMessage, "user", incident.TenantId, "deployment-name", incident.DeploymentName, "process-definition-id", incident.ProcessDefinitionId)
			err := this.devNotifications.SendMessage(developerNotifications.Message{
				Sender: "github.com/SENERGY-Platform/process-sync",
				Title:  "Fog-Process-Incident-User-Notification",
				Tags:   []string{"fog", "process-incident", "user-notification", incident.TenantId, networkid},
				Body: fmt.Sprintf("Notification For %v in network %v\nTitle: %v\nMessage: %v\n",
					incident.TenantId,
					networkid,
					"Fog-Process-Incident in "+incident.DeploymentName,
					incident.ErrorMessage,
				),
			})
			if err != nil {
				this.logger.Error("unable to send developer-notification", "snrgy-log-type", "error", "error", err.Error(), "user", incident.TenantId, "process-definition-id", incident.ProcessDefinitionId, "incident-msg", incident.ErrorMessage)
			}
		}()
	}
}
