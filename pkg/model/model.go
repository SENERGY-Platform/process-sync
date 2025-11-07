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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/SENERGY-Platform/event-worker/pkg/model"
	"github.com/SENERGY-Platform/process-deployment/lib/model/deploymentmodel"
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
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

type IncidentQuery struct {
	NetworkIds         []string
	ProcessInstanceIds []string
	Sort               string
	Limit              int64
	Offset             int64
}

type ProcessDefinition struct {
	camundamodel.ProcessDefinition
	SyncInfo
}

type ProcessInstance struct {
	camundamodel.ProcessInstance
	SyncInfo
}

type InstanceQuery struct {
	NetworkIds   []string
	BusinessKeys []string
	Sort         string
	Limit        int64
	Offset       int64
}

type StartMessage struct {
	DeploymentId string                 `json:"deployment_id"`
	Parameter    map[string]interface{} `json:"parameter"`
	BusinessKey  string                 `json:"business_key"`
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

type WardenInfo struct {
	CreationTime        int64                  `json:"creation_time" bson:"creation_time"`                 //unix timestamp
	NetworkId           string                 `json:"network_id" bson:"network_id"`                       //must be the same as the process-instance network-id
	BusinessKey         string                 `json:"business_key" bson:"business_key"`                   //must be the same as the process-instance business-key and start with WardenBusinessKeyPrefix; the prefix may be set by Warden.MarkInstanceBusinessKeyAsWardenHandled
	ProcessDeploymentId string                 `json:"process_deployment_id" bson:"process_deployment_id"` //must be the same as the process-instance process-deployment-id
	StartParameters     map[string]interface{} `json:"start_parameters" bson:"start_parameters"`
}

const WardenBusinessKeyPrefix = "wardened:"

func (this WardenInfo) Validate() error {
	if this.CreationTime == 0 {
		return errors.New("creation-time must not be 0")
	}
	if this.NetworkId == "" {
		return errors.New("network-id must not be empty")
	}
	if !strings.HasPrefix(this.BusinessKey, WardenBusinessKeyPrefix) {
		return fmt.Errorf("business-key must start with '%s'", WardenBusinessKeyPrefix)
	}
	if this.ProcessDeploymentId == "" {
		return errors.New("process-deployment-id must not be empty")
	}
	return nil
}

func (this WardenInfo) IsOlderThen(duration time.Duration) bool {
	return time.Unix(this.CreationTime, 0).Add(duration).After(time.Now())
}

type WardenInfoQuery struct {
	NetworkIds           []string
	BusinessKeys         []string
	ProcessDeploymentIds []string
	Sort                 string
	Limit                int64
	Offset               int64
}

type DeploymentWardenInfo struct {
	DeploymentId string                  `json:"deployment_id" bson:"deployment_id"`
	NetworkId    string                  `json:"network_id" bson:"network_id"`
	Deployment   DeploymentWithEventDesc `json:"deployment" bson:"deployment"`
}

type DeploymentWardenInfoQuery struct {
	NetworkIds           []string
	ProcessDeploymentIds []string
	Sort                 string
	Limit                int64
	Offset               int64
}

func NormalizeBpmnDeploymentId(id string) string {
	return "deplid_" + strings.NewReplacer("-", "_", ":", "_", "#", "_").Replace(id)
}
