/*
 * Copyright 2025 InfAI (CC SES)
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
	"errors"
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/database"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
	"net/http"
)

func (this *Controller) ApiSyncDeployments(token string, networkId string) (error, int) {
	err := this.db.RemovePlaceholderDeployments(networkId)
	if err != nil {
		return err, http.StatusInternalServerError
	}

	var limit int64 = 1000
	var offset int64 = 0
	errorList := []error{}
	for {
		deployments, err := this.db.ListDeployments([]string{networkId}, limit, offset, "id.asc")
		if err != nil {
			return err, http.StatusInternalServerError
		}
		for _, deployment := range deployments {
			if deployment.SyncInfo.MarkedForDelete {
				err = this.mgw.SendDeploymentDeleteCommand(networkId, deployment.Id)
				if err != nil {
					errorList = append(errorList, err)
					continue
				}
			}
			if deployment.SyncInfo.MarkedAsMissing {
				metadata, err := this.db.ReadDeploymentMetadata(networkId, deployment.Id)
				if err != nil && !errors.Is(err, database.ErrNotFound) {
					errorList = append(errorList, err)
					continue
				}
				if err != nil {
					continue
				}
				now := configuration.TimeNow()
				err = this.db.SaveDeployment(model.Deployment{
					Deployment: camundamodel.Deployment{
						Id:             "placeholder-" + configuration.Id(),
						Name:           deployment.Name,
						Source:         "senergy",
						DeploymentTime: now,
						TenantId:       "senergy",
					},
					SyncInfo: model.SyncInfo{
						NetworkId:       networkId,
						IsPlaceholder:   true,
						MarkedForDelete: false,
						SyncDate:        now,
					},
				})
				if err != nil {
					errorList = append(errorList, err)
					continue
				}
				err = this.mgw.SendDeploymentCommand(networkId, metadata.DeploymentModel)
				if err != nil {
					errorList = append(errorList, err)
					continue
				}
				this.deleteDeployment(networkId, deployment.Id)
			}
		}
		if int64(len(deployments)) <= limit {
			err = errors.Join(errorList...)
			if err != nil {
				return err, http.StatusInternalServerError
			}
			return nil, http.StatusOK
		}
		offset += limit
	}
}
