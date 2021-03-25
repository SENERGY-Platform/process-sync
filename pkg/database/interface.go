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

package database

import "github.com/SENERGY-Platform/process-sync/pkg/model"

type Database interface {
	SaveDeployment(deployment model.Deployment) error
	RemoveDeployment(networkId string, deploymentId string) error
	RemovePlaceholderDeployments(networkId string) error
	RemoveUnknownDeployments(networkId string, knownIds []string) error
	ReadDeployment(networkId string, deploymentId string) (deployment model.Deployment, err error)
	ListDeployments(networkIds []string, limit int64, offset int64, sort string) (deployment []model.Deployment, err error)
	SearchDeployments(networkIds []string, search string, limit int64, offset int64, sort string) ([]model.Deployment, error)

	SaveHistoricProcessInstance(historicProcessInstance model.HistoricProcessInstance) error
	RemoveHistoricProcessInstance(networkId string, historicProcessInstanceId string) error
	RemovePlaceholderHistoricProcessInstances(id string) error
	RemoveUnknownHistoricProcessInstances(networkId string, knownIds []string) error
	ReadHistoricProcessInstance(networkId string, historicProcessInstanceId string) (historicProcessInstance model.HistoricProcessInstance, err error)
	ListHistoricProcessInstances(networkIds []string, query model.HistoryQuery, limit int64, offset int64, sort string) (historicProcessInstance []model.HistoricProcessInstance, total int64, err error)

	SaveProcessInstance(processInstance model.ProcessInstance) error
	RemoveProcessInstance(networkId string, processInstanceId string) error
	RemovePlaceholderProcessInstances(networkId string) error
	RemoveUnknownProcessInstances(networkId string, knownIds []string) error
	ReadProcessInstance(networkId string, processInstanceId string) (processInstance model.ProcessInstance, err error)
	ListProcessInstances(networkIds []string, limit int64, offset int64, sort string) (processInstance []model.ProcessInstance, err error)

	SaveProcessDefinition(processDefinition model.ProcessDefinition) error
	RemoveProcessDefinition(networkId string, processDefinitionId string) error
	RemoveUnknownProcessDefinitions(networkId string, knownIds []string) error
	ReadProcessDefinition(networkId string, processDefinitionId string) (processDefinition model.ProcessDefinition, err error)
	ListProcessDefinitions(networkIds []string, limit int64, offset int64, sort string) (processDefinition []model.ProcessDefinition, err error)
	GetDefinitionByDeploymentId(networkId string, deploymentId string) (processDefinition model.ProcessDefinition, err error)

	SaveIncident(incident model.Incident) error
	RemoveIncident(networkId string, incidentId string) error
	RemoveUnknownIncidents(networkId string, knownIds []string) error
	ReadIncident(networkId string, incidentId string) (incident model.Incident, err error)
	ListIncidents(networkIds []string, processInstanceId string, limit int64, offset int64, sort string) (incident []model.Incident, err error)
	RemoveIncidentOfInstance(networkId string, instanceId string) error
	RemoveIncidentOfDefinition(networkId string, definitionId string) error
	RemoveIncidentOfNotInstances(networkId string, notInstanceIds []string) error
	RemoveIncidentOfNotDefinitions(networkId string, notDefinitionIds []string) error

	SaveDeploymentMetadata(metadata model.DeploymentMetadata) error
	RemoveUnknownDeploymentMetadata(networkId string, knownIds []string) error
	RemoveDeploymentMetadata(networkId string, deploymentId string) error
	ReadDeploymentMetadata(networkId string, deploymentId string) (metadata model.DeploymentMetadata, err error)
}
