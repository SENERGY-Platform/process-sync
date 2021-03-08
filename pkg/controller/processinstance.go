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

func (this *Controller) UpdateProcessInstance(networkId string, instance camundamodel.ProcessInstance) {
	err := this.db.RemovePlaceholderProcessInstances(networkId)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
	err = this.db.SaveProcessInstance(model.ProcessInstance{
		ProcessInstance: instance,
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

func (this *Controller) DeleteProcessInstance(networkId string, instanceId string) {
	err := this.db.RemoveProcessInstance(networkId, instanceId)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
}

func (this *Controller) DeleteUnknownProcessInstances(networkId string, knownIds []string) {
	err := this.db.RemoveUnknownProcessInstances(networkId, knownIds)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
}

func (this *Controller) ApiReadProcessInstance(networkId string, id string) (result model.ProcessInstance, err error, errCode int) {
	result, err = this.db.ReadProcessInstance(networkId, id)
	errCode = this.SetErrCode(err)
	return
}

func (this *Controller) ApiDeleteProcessInstance(networkId string, id string) (err error, errCode int) {
	defer func() {
		errCode = this.SetErrCode(err)
	}()
	var current model.ProcessInstance
	current, err = this.db.ReadProcessInstance(networkId, id)
	if err != nil {
		return
	}
	if current.IsPlaceholder {
		err = this.db.RemoveProcessInstance(networkId, id)
	} else {
		err = this.mgw.SendProcessStopCommand(networkId, id)
		if err != nil {
			return
		}
		current.MarkedForDelete = true
		err = this.db.SaveProcessInstance(current)
	}
	return
}

func (this *Controller) ApiListProcessInstances(networkIds []string, limit int64, offset int64, sort string) (result []model.ProcessInstance, err error, errCode int) {
	result, err = this.db.ListProcessInstances(networkIds, limit, offset, sort)
	errCode = this.SetErrCode(err)
	return
}
