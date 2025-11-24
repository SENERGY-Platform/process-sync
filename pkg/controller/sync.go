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
	"net/http"
	"slices"

	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/database"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
	"github.com/SENERGY-Platform/process-sync/pkg/warden"
	"github.com/SENERGY-Platform/service-commons/pkg/util"
)

func (this *Controller) MigrateToWarden() (err error) {
	networkIds, err := this.db.ListKnownNetworkIds()
	if err != nil {
		return err
	}
	for _, networkId := range networkIds {
		tempErr := this.migrateToWarden(networkId)
		if tempErr != nil {
			this.config.GetLogger().Error("unable to migrate network to warden", "networkId", networkId, "error", err)
			err = errors.Join(err, tempErr)
		}
	}
	return err
}

func (this *Controller) migrateToWarden(networkId string) error {
	deployments := []model.Deployment{}
	deplIter := util.IterBatch(100, func(limit int64, offset int64) ([]model.Deployment, error) {
		return this.db.ListDeployments([]string{networkId}, limit, offset, "id.asc")
	})
	for depl, err := range deplIter {
		if err != nil {
			return err
		}
		_, exists, err := this.db.GetDeploymentWardenInfoByDeploymentId(networkId, depl.Id)
		if err != nil {
			return err
		}
		if !exists {
			deployments = append(deployments, depl)
		}
	}
	if len(deployments) == 0 {
		return nil //no deployments to migrate
	}

	metadataByDeploymentId := map[string]model.DeploymentMetadata{}
	metadataIter := util.IterBatch(100, func(limit int64, offset int64) ([]model.DeploymentMetadata, error) {
		return this.db.ListDeploymentMetadata(model.MetadataQuery{NetworkId: &networkId})
	})
	for metadata, err := range metadataIter {
		if err != nil {
			return err
		}
		index := slices.IndexFunc(deployments, func(deployment model.Deployment) bool {
			return deployment.Id == metadata.CamundaDeploymentId
		})
		if index >= 0 {
			metadataByDeploymentId[metadata.CamundaDeploymentId] = metadata
		}
	}

	definitionById := map[string]model.ProcessDefinition{}
	definitionIter := util.IterBatch(100, func(limit int64, offset int64) ([]model.ProcessDefinition, error) {
		return this.db.ListProcessDefinitions([]string{networkId}, limit, offset, "id.asc")
	})
	for def, err := range definitionIter {
		if err != nil {
			return err
		}
		index := slices.IndexFunc(deployments, func(deployment model.Deployment) bool {
			return deployment.Id == def.DeploymentId
		})
		if index >= 0 {
			definitionById[def.Id] = def
		}
	}

	instancesByDeploymentId := map[string][]model.ProcessInstance{}
	instanceIter := util.IterBatch(100, func(limit int64, offset int64) ([]model.ProcessInstance, error) {
		return this.db.FindProcessInstances(model.InstanceQuery{NetworkIds: []string{networkId}, Limit: limit, Offset: offset})
	})
	for instance, err := range instanceIter {
		if err != nil {
			return err
		}
		if this.warden.InstanceIsCreatedWithWardenHandlingIntended(instance) {
			continue
		}
		if def, ok := definitionById[instance.DefinitionId]; ok {
			instancesByDeploymentId[def.DeploymentId] = append(instancesByDeploymentId[def.DeploymentId], instance)
		}
	}

	for _, deployment := range deployments {
		err := this.warden.AddDeploymentWarden(warden.DeploymentWardenInfo{
			DeploymentId: deployment.Id,
			NetworkId:    networkId,
			Deployment:   metadataByDeploymentId[deployment.Id].DeploymentModel,
		})
		if err != nil {
			return err
		}
		metadata := metadataByDeploymentId[deployment.Id]
		if len(metadata.ProcessParameter) == 0 { //only restart process instances where no parameters are needed
			for _, instance := range instancesByDeploymentId[deployment.Id] {
				businessKey := instance.BusinessKey
				if businessKey == "" {
					businessKey = "migration_of_" + instance.Id
				}
				err, _ = this.ApiStartDeployment(networkId, deployment.Id, businessKey, map[string]interface{}{})
				if err != nil {
					return err
				}
				err, _ = this.ApiDeleteProcessInstance(networkId, instance.Id)
				if err != nil {
					this.config.GetLogger().Error("MigrateToWarden(): unable to delete old instance", "error", err, "instanceId", instance.Id)
					wardenErr := this.warden.RemoveInstanceWardenByBusinessKey(networkId, businessKey) //instance.Id is not stable --> use businessKey --> warden will remove the unneeded instance later
					if wardenErr != nil {
						this.config.GetLogger().Error("MigrateToWarden(): unable to remove instance from warden", "error", wardenErr, "businessKey", businessKey)
					}
					return err
				}
			}
		}
	}
	return nil
}

func (this *Controller) ApiSyncDeployments(networkId string) (error, int) {
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
				_, exists, err := this.db.GetDeploymentWardenInfoByDeploymentId(networkId, deployment.Id)
				if err != nil {
					errorList = append(errorList, err)
					continue
				}
				if exists {
					continue //is handled by warden --> no manual sync
				}
				metadata, err := this.db.ReadDeploymentMetadata(networkId, deployment.Id)
				if err != nil && !errors.Is(err, database.ErrNotFound) {
					errorList = append(errorList, err)
					continue
				}
				if err != nil {
					continue
				}
				now := configuration.TimeNow()
				this.deleteDeployment(networkId, deployment.Id)
				err = this.db.SaveDeployment(model.Deployment{
					Deployment: camundamodel.Deployment{
						Id:             deployment.Id,
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
