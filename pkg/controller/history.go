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

func (this *Controller) UpdateHistoricProcessInstance(networkId string, historicProcessInstance camundamodel.HistoricProcessInstance) {
	err := this.db.SaveHistoricProcessInstance(model.HistoricProcessInstance{
		HistoricProcessInstance: historicProcessInstance,
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

func (this *Controller) DeleteHistoricProcessInstance(networkId string, historicInstanceId string) {
	err := this.db.RemoveHistoricProcessInstance(networkId, historicInstanceId)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
}

func (this *Controller) DeleteUnknownHistoricProcessInstances(networkId string, knownIds []string) {
	err := this.db.RemoveUnknownHistoricProcessInstances(networkId, knownIds)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
}

func (this *Controller) ApiReadHistoricProcessInstance(networkId string, id string) (result model.HistoricProcessInstance, err error, errCode int) {
	result, err = this.db.ReadHistoricProcessInstance(networkId, id)
	errCode = this.SetErrCode(err)
	return
}

func (this *Controller) ApiDeleteHistoricProcessInstance(networkId string, id string) (err error, errCode int) {
	defer func() {
		errCode = this.SetErrCode(err)
	}()
	var current model.HistoricProcessInstance
	current, err = this.db.ReadHistoricProcessInstance(networkId, id)
	if err != nil {
		return
	}
	err = this.mgw.SendProcessHistoryDeleteCommand(networkId, id)
	if err != nil {
		return
	}
	current.MarkedForDelete = true
	err = this.db.SaveHistoricProcessInstance(current)
	return
}

func (this *Controller) ApiListHistoricProcessInstance(networkIds []string, limit int64, offset int64, sort string) (result []model.HistoricProcessInstance, err error, errCode int) {
	result, err = this.db.ListHistoricProcessInstances(networkIds, limit, offset, sort)
	errCode = this.SetErrCode(err)
	return
}
