/*
 * Copyright 2021 InfAI (CC SES)
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
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
	"github.com/SENERGY-Platform/process-sync/pkg/model/deploymentmodel"
	"log"
	"net/http"
	"runtime/debug"
)

func (this *Controller) UpdateDeployment(networkId string, deployment camundamodel.Deployment) {
	err := this.db.RemovePlaceholderDeployments(networkId)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
	err = this.db.SaveDeployment(model.Deployment{
		Deployment: deployment,
		SyncInfo: model.SyncInfo{
			NetworkId:       networkId,
			IsPlaceholder:   false,
			MarkedForDelete: false,
			SyncDate:        configuration.TimeNow(),
		},
	})
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
}

func (this *Controller) DeleteDeployment(networkId string, deploymentId string) {
	err := this.db.RemoveDeployment(networkId, deploymentId)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
}

func (this *Controller) DeleteUnknownDeployments(networkId string, knownIds []string) {
	err := this.db.RemoveUnknownDeployments(networkId, knownIds)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
}

func (this *Controller) ApiReadDeployment(networkId string, deploymentId string) (result model.Deployment, err error, errCode int) {
	result, err = this.db.ReadDeployment(networkId, deploymentId)
	errCode = this.SetErrCode(err)
	return
}

func (this *Controller) ApiDeleteDeployment(networkId string, deploymentId string) (err error, errCode int) {
	defer func() {
		errCode = this.SetErrCode(err)
	}()
	var current model.Deployment
	current, err = this.db.ReadDeployment(networkId, deploymentId)
	if err != nil {
		return
	}
	if current.IsPlaceholder {
		err = this.db.RemoveDeployment(networkId, deploymentId)
	} else {
		err = this.mgw.SendDeploymentDeleteCommand(networkId, deploymentId)
		if err != nil {
			return
		}
		current.MarkedForDelete = true
		err = this.db.SaveDeployment(current)
	}
	return
}

func (this *Controller) ApiListDeployments(networkIds []string, limit int64, offset int64, sort string) (result []model.Deployment, err error, errCode int) {
	result, err = this.db.ListDeployments(networkIds, limit, offset, sort)
	errCode = this.SetErrCode(err)
	if result == nil {
		result = []model.Deployment{}
	}
	return
}

func (this *Controller) ApiCreateDeployment(networkId string, deployment deploymentmodel.Deployment) (err error, errCode int) {
	this.removeUnusedElementsFromDeployment(&deployment)
	err = deployment.Validate()
	if err != nil {
		return err, http.StatusBadRequest
	}
	defer func() {
		errCode = this.SetErrCode(err)
	}()
	err = this.mgw.SendDeploymentCommand(networkId, deployment)
	if err != nil {
		return
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
	return
}

func (this *Controller) ApiStartDeployment(networkId string, deploymentId string) (err error, errCode int) {
	defer func() {
		errCode = this.SetErrCode(err)
	}()
	var current model.Deployment
	current, err = this.db.ReadDeployment(networkId, deploymentId)
	if err != nil {
		return
	}
	if current.IsPlaceholder {
		err = IsPlaceholderProcessErr
		return
	}
	if current.MarkedForDelete {
		err = IsMarkedForDeleteErr
		return
	}

	definition, err := this.db.GetDefinitionByDeploymentId(networkId, deploymentId)
	if err != nil {
		return
	}

	err = this.mgw.SendDeploymentStartCommand(networkId, deploymentId)
	if err != nil {
		return
	}

	now := configuration.TimeNow()
	instanceId := "placeholder-" + configuration.Id()
	err = this.db.SaveProcessInstance(model.ProcessInstance{
		ProcessInstance: camundamodel.ProcessInstance{
			Id:           instanceId,
			DefinitionId: definition.Id,
			Ended:        false,
			Suspended:    false,
			TenantId:     "senergy",
		},
		SyncInfo: model.SyncInfo{
			NetworkId:       networkId,
			IsPlaceholder:   true,
			MarkedForDelete: false,
			SyncDate:        now,
		},
	})
	if err != nil {
		return
	}
	err = this.db.SaveHistoricProcessInstance(model.HistoricProcessInstance{
		HistoricProcessInstance: camundamodel.HistoricProcessInstance{
			Id:                       "placeholder-" + configuration.Id(),
			SuperProcessInstanceId:   instanceId,
			ProcessDefinitionName:    definition.Name,
			ProcessDefinitionKey:     definition.Key,
			ProcessDefinitionVersion: float64(definition.Version),
			ProcessDefinitionId:      definition.Id,
			StartTime:                now.Format(camundamodel.CamundaTimeFormat),
			DurationInMillis:         0,
			StartUserId:              "senergy",
			TenantId:                 "senergy",
			State:                    "PLACEHOLDER",
		},
		SyncInfo: model.SyncInfo{
			NetworkId:       networkId,
			IsPlaceholder:   true,
			MarkedForDelete: false,
			SyncDate:        now,
		},
	})
	return
}

func (this *Controller) removeUnusedElementsFromDeployment(deployment *deploymentmodel.Deployment) {
	deployment.Id = ""
	deployment.Elements = nil
	deployment.Diagram.XmlRaw = ""
}
