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
	eventmodel "github.com/SENERGY-Platform/event-deployment/lib/model"
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
	"github.com/SENERGY-Platform/process-sync/pkg/model/deploymentmodel"
	"time"
)

type SyncInfo struct {
	NetworkId       string    `json:"network_id"`
	IsPlaceholder   bool      `json:"is_placeholder"`
	MarkedForDelete bool      `json:"marked_for_delete"`
	SyncDate        time.Time `json:"sync_date"`
}

type LastNetworkContact struct {
	NetworkId string    `json:"network_id"`
	Time      time.Time `json:"time"`
}

type Metadata struct {
	CamundaDeploymentId string                           `json:"camunda_deployment_id"`
	ProcessParameter    map[string]camundamodel.Variable `json:"process_parameter"`
	DeploymentModel     deploymentmodel.Deployment       `json:"deployment_model"`
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

type DeploymentWithAnalyticsRecords struct {
	deploymentmodel.Deployment
	AnalyticsRecords   []AnalyticsRecord `json:"analytics_records"`
	DeviceIdToLocalId  map[string]string `json:"device_id_to_local_id"`
	ServiceIdToLocalId map[string]string `json:"service_id_to_local_id"`
}

type DeviceEventAnalyticsRecord struct {
	Label          string `json:"label"`
	DeploymentId   string `json:"deployment_id"`
	FlowId         string `json:"flow_id"`
	EventId        string `json:"event_id"`
	DeviceId       string `json:"device_id"`
	ServiceId      string `json:"service_id"`
	Value          string `json:"value"`
	Path           string `json:"path"`
	PathWithPrefix string `json:"-"` //set json tag to enable
	CastFrom       string `json:"cast_from"`
	CastTo         string `json:"cast_to"`
}

type GroupEventAnalyticsRecord struct {
	Label                                    string                                        `json:"label"`
	Desc                                     eventmodel.GroupEventDescription              `json:"desc"`
	ServiceIds                               []string                                      `json:"service_ids"`
	ServiceToDeviceIdsMapping                map[string][]string                           `json:"service_to_device_ids_mapping"`
	ServiceToPathMapping                     map[string]string                             `json:"service_to_path_mapping"`
	ServiceToPathWithPrefixMapping           map[string]string                             `json:"-"` //set json tag to enable
	ServiceToPathAndCharacteristic           map[string][]eventmodel.PathAndCharacteristic `json:"service_to_path_and_characteristic"`
	ServiceToPathWithPrefixAndCharacteristic map[string][]eventmodel.PathAndCharacteristic `json:"-"` //set json tag to enable
}

type AnalyticsRecord struct {
	DeviceEvent *DeviceEventAnalyticsRecord `json:"device_event"`
	GroupEvent  *GroupEventAnalyticsRecord  `json:"group_event"`
}

type MetadataQuery struct {
	NetworkId           *string `json:"network_id"`
	CamundaDeploymentId *string `json:"camunda_deployment_id"`
	DeploymentId        *string `json:"deployment_id"`
}
