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
	err = this.db.RemoveDeploymentMetadata(networkId, deploymentId)
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
	err = this.db.RemoveUnknownDeploymentMetadata(networkId, knownIds)
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

func (this *Controller) ApiReadDeploymentMetadata(networkId string, deploymentId string) (result model.DeploymentMetadata, err error, errCode int) {
	result, err = this.db.ReadDeploymentMetadata(networkId, deploymentId)
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

func (this *Controller) ApiSearchDeployments(networkIds []string, search string, limit int64, offset int64, sort string) (result []model.Deployment, err error, errCode int) {
	result, err = this.db.SearchDeployments(networkIds, search, limit, offset, sort)
	errCode = this.SetErrCode(err)
	if result == nil {
		result = []model.Deployment{}
	}
	return
}

func (this *Controller) ApiCreateDeployment(networkId string, deployment deploymentmodel.Deployment) (err error, errCode int) {
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

func (this *Controller) ApiStartDeployment(networkId string, deploymentId string, parameter map[string]interface{}) (err error, errCode int) {
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

	err = this.mgw.SendDeploymentStartCommand(networkId, deploymentId, parameter)
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

func (this *Controller) ExtendDeployments(deployments []model.Deployment) (result []model.ExtendedDeployment) {
	for _, deployment := range deployments {
		if deployment.IsPlaceholder {
			result = append(result, model.ExtendedDeployment{Deployment: deployment, Diagram: constructionSvg})
			break
		}
		definition, err := this.db.GetDefinitionByDeploymentId(deployment.NetworkId, deployment.Id)
		if err != nil {
			result = append(result, model.ExtendedDeployment{Deployment: deployment, Error: err.Error()})
		} else {
			result = append(result, model.ExtendedDeployment{Deployment: deployment, Diagram: definition.Diagram, DefinitionId: definition.Id})
		}
	}
	return
}

//Image taken from the de:Straßenverkehrsordnung (German Road Regulations)
const constructionSvg = `<svg
   xmlns:ns="http://ns.adobe.com/SaveForWeb/1.0/"
   xmlns:dc="http://purl.org/dc/elements/1.1/"
   xmlns:cc="http://creativecommons.org/ns#"
   xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
   xmlns:svg="http://www.w3.org/2000/svg"
   xmlns="http://www.w3.org/2000/svg"
   version="1.1"
   width="370"
   height="290"
   viewBox="0 0 370 290"
   id="svg2"
   xml:space="preserve"><defs
   id="defs9" />
<metadata
   id="metadata3">
	<ns:sfw>
		<ns:slices />
		<ns:sliceSourceBounds
   bottomLeftOrigin="true"
   x="118.354"
   y="287.456"
   width="359.007"
   height="262.832" />
	</ns:sfw>
<rdf:RDF><cc:Work
     rdf:about=""><dc:format>image/svg+xml</dc:format><dc:type
       rdf:resource="http://purl.org/dc/dcmitype/StillImage" /><dc:title></dc:title></cc:Work></rdf:RDF></metadata>
<path
   d="m 156.818,39.098 c 0,-13.19 10.694,-23.885 23.885,-23.885 13.191,0 23.884,10.695 23.884,23.885 0,13.192 -10.692,23.884 -23.884,23.884 C 167.512,62.981 156.818,52.29 156.818,39.098 z M 9.206,241.977 c -6.9,11.952 -2.804,27.234 9.146,34.133 l 64.099,-111.034 27.85,40.411 c 0,0 -0.001,44.49 -0.001,45.637 0,13.032 9.979,23.722 22.71,24.872 V 198.328 L 107.001,158.81 67.996,140.145 9.206,241.977 z m 285.671,-53.159 c -5.894,-6.946 -14.688,-11.354 -24.512,-11.354 -9.783,0 -18.541,4.38 -24.435,11.277 0,0 -22.613,24.438 -27.077,29.044 -5.861,6.052 -22.118,6.26 -27.122,11.938 -5.004,5.677 -37.124,39.861 -37.124,39.861 -0.755,0.896 -1.227,2.038 -1.227,3.302 0,2.849 2.308,5.157 5.156,5.157 h 201.168 c 2.849,0 5.157,-2.309 5.157,-5.157 0,-1.262 -0.453,-2.419 -1.207,-3.314 L 294.877,188.818 z M 152.93,166.803 159.202,76.985 c 1.13,-12.914 -7.775,-24.389 -20.256,-26.731 -0.753,-0.159 -1.572,-0.267 -2.441,-0.339 -0.006,0 -0.013,-10e-4 -0.019,-0.002 l 0.002,0.002 c -0.692,-0.057 -1.409,-0.096 -2.178,-0.096 L 107.277,49.82 78.26,49.818 c -6.437,0 -12.008,3.665 -14.772,9.015 L 44.71,91.308 c -1.655,2.586 -2.626,5.649 -2.626,8.945 0,7.528 5.003,13.879 11.863,15.929 l 6.106,-10.618 16.07,12.39 4.141,-7.172 L 64.183,98.383 80.412,70.167 h 27.984 l -37.296,64.601 39.005,18.666 25.328,-43.87 -2.903,41.511 -11.398,-8.788 -4.142,7.172 14.852,11.451 -0.149,2.125 c -1.111,12.699 7.491,23.988 19.644,26.587 l 0.908,-12.983 53.069,40.915 c 3.437,-0.989 7.062,-2.164 8.811,-3.576 L 152.93,166.803 z"
   id="path5" />
</svg>`
