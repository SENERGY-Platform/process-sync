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

package warden

import (
	"iter"
	"time"

	"github.com/SENERGY-Platform/process-sync/pkg/model"
)

type Processes struct{}

func (this *Processes) AllInstances() (iter.Seq[model.ProcessInstance], error) {
	//TODO implement me
	panic("implement me")
}

func (this *Processes) GetInstances(info WardenInfo) ([]model.ProcessInstance, error) {
	//TODO implement me
	panic("implement me")
}

func (this *Processes) GetYoungestProcessInstance(instances []model.ProcessInstance) (model.ProcessInstance, error) {
	//TODO implement me
	panic("implement me")
}

func (this *Processes) InstanceIsOlderThen(instance model.ProcessInstance, duration time.Duration) bool {
	//TODO implement me
	panic("implement me")
}

func (this *Processes) InstanceIsCreatedWithWardenHandlingIntended(instance model.ProcessInstance) bool {
	//TODO implement me
	panic("implement me")
}

func (this *Processes) GetInstanceHistories(info WardenInfo) ([]model.HistoricProcessInstance, error) {
	//TODO implement me
	panic("implement me")
}

func (this *Processes) GetYoungestHistory(histories []model.HistoricProcessInstance) (model.HistoricProcessInstance, error) {
	//TODO implement me
	panic("implement me")
}

func (this *Processes) HistoryIsOlderThen(history model.HistoricProcessInstance, duration time.Duration) bool {
	//TODO implement me
	panic("implement me")
}

func (this *Processes) GetIncidents(history model.HistoricProcessInstance) ([]model.Incident, error) {
	//TODO implement me
	panic("implement me")
}

func (this *Processes) GetYoungestIncident(incidents []model.Incident) (model.Incident, error) {
	//TODO implement me
	panic("implement me")
}

func (this *Processes) IncidentIsOlderThen(incident model.Incident, duration time.Duration) bool {
	//TODO implement me
	panic("implement me")
}

func (this *Processes) Start(info WardenInfo) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (this *Processes) Stop(instance model.ProcessInstance) error {
	//TODO implement me
	panic("implement me")
}

func (this *Processes) DeploymentExists(info WardenInfo) (exist bool, err error) {
	//TODO implement me
	panic("implement me")
}

func (this *Processes) Deploy(deployment model.Deployment) error {
	//TODO implement me
	panic("implement me")
}
