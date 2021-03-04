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
	"log"
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
