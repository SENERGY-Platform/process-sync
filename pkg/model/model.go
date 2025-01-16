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

package model

import (
	"github.com/SENERGY-Platform/event-worker/pkg/model"
	"github.com/SENERGY-Platform/process-deployment/lib/model/deploymentmodel"
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
	"time"
)

type SyncInfo struct {
	NetworkId       string    `json:"network_id"`
	IsPlaceholder   bool      `json:"is_placeholder"`
	MarkedForDelete bool      `json:"marked_for_delete"`
	MarkedAsMissing bool      `json:"marked_as_missing"`
	SyncDate        time.Time `json:"sync_date"`
}

type LastNetworkContact struct {
	NetworkId string    `json:"network_id"`
	Time      time.Time `json:"time"`
}

type Metadata struct {
	CamundaDeploymentId string                           `json:"camunda_deployment_id"`
	ProcessParameter    map[string]camundamodel.Variable `json:"process_parameter"`
	DeploymentModel     DeploymentWithEventDesc          `json:"deployment_model"`
}

type DeploymentMetadata struct {
	Metadata
	SyncInfo
}

type Deployment struct {
	camundamodel.Deployment
	SyncInfo
}

type HistoricProcessInstance struct {
	camundamodel.HistoricProcessInstance
	SyncInfo
}

type Incident struct {
	camundamodel.Incident
	SyncInfo
}

type ProcessDefinition struct {
	camundamodel.ProcessDefinition
	SyncInfo
}

type ProcessInstance struct {
	camundamodel.ProcessInstance
	SyncInfo
}

type StartMessage struct {
	DeploymentId string                 `json:"deployment_id"`
	Parameter    map[string]interface{} `json:"parameter"`
}

type ExtendedDeployment struct {
	Deployment
	Diagram      string `json:"diagram"`
	DefinitionId string `json:"definition_id"`
	Error        string `json:"error"`
}

type HistoryQuery struct {
	State               string
	ProcessDefinitionId string
	Search              string
}

type DeploymentWithEventDesc struct {
	deploymentmodel.Deployment `bson:",inline"`
	DeviceIdToLocalId          map[string]string `json:"device_id_to_local_id"`
	ServiceIdToLocalId         map[string]string `json:"service_id_to_local_id"`
	EventDescriptions          []model.EventDesc `json:"event_descriptions"`
}

type MetadataQuery struct {
	NetworkId           *string `json:"network_id"`
	CamundaDeploymentId *string `json:"camunda_deployment_id"`
	DeploymentId        *string `json:"deployment_id"`
}
