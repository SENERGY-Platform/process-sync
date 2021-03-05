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

func (this *Controller) UpdateProcessDefinition(networkId string, processDefinition camundamodel.ProcessDefinition) {
	err := this.db.SaveProcessDefinition(model.ProcessDefinition{
		ProcessDefinition: processDefinition,
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

func (this *Controller) DeleteProcessDefinition(networkId string, definitionId string) {
	err := this.db.RemoveProcessDefinition(networkId, definitionId)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
}

func (this *Controller) DeleteUnknownProcessDefinitions(networkId string, knownIds []string) {
	err := this.db.RemoveUnknownProcessDefinitions(networkId, knownIds)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
}

func (this *Controller) ApiReadProcessDefinition(networkId string, id string) (result model.ProcessDefinition, err error, errCode int) {
	result, err = this.db.ReadProcessDefinition(networkId, id)
	errCode = this.SetErrCode(err)
	return
}

func (this *Controller) ApiListProcessDefinitions(networkIds []string, limit int64, offset int64, sort string) (result []model.ProcessDefinition, err error, errCode int) {
	result, err = this.db.ListProcessDefinitions(networkIds, limit, offset, sort)
	errCode = this.SetErrCode(err)
	return
}
