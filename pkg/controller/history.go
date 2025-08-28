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
	"runtime/debug"

	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
)

func (this *Controller) UpdateHistoricProcessInstance(networkId string, historicProcessInstance camundamodel.HistoricProcessInstance) {
	err := this.db.RemovePlaceholderHistoricProcessInstances(networkId)
	if err != nil {
		this.config.GetLogger().Error("error", "error", err, "stack", debug.Stack())
	}
	err = this.db.SaveHistoricProcessInstance(model.HistoricProcessInstance{
		HistoricProcessInstance: historicProcessInstance,
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
}

func (this *Controller) DeleteHistoricProcessInstance(networkId string, historicInstanceId string) {
	err := this.db.RemoveHistoricProcessInstance(networkId, historicInstanceId)
	if err != nil {
		this.config.GetLogger().Error("error", "error", err, "stack", debug.Stack())
	}
}

func (this *Controller) DeleteUnknownHistoricProcessInstances(networkId string, knownIds []string) {
	err := this.db.RemoveUnknownHistoricProcessInstances(networkId, knownIds)
	if err != nil {
		this.config.GetLogger().Error("error", "error", err, "stack", debug.Stack())
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
	if current.IsPlaceholder {
		err = this.db.RemoveHistoricProcessInstance(networkId, id)
	} else {
		if current.EndTime == "" {
			err = HistoryMayOnlyDeletedIfFinishedOrPlaceholderErr
			return
		}
		err = this.mgw.SendProcessHistoryDeleteCommand(networkId, id)
		if err != nil {
			return
		}
		current.MarkedForDelete = true
		err = this.db.SaveHistoricProcessInstance(current)
	}
	return
}

func (this *Controller) ApiListHistoricProcessInstance(networkIds []string, query model.HistoryQuery, limit int64, offset int64, sort string) (result []model.HistoricProcessInstance, total int64, err error, errCode int) {
	result, total, err = this.db.ListHistoricProcessInstances(networkIds, query, limit, offset, sort)
	errCode = this.SetErrCode(err)
	if result == nil {
		result = []model.HistoricProcessInstance{}
	}
	return
}
