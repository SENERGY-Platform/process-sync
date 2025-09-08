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

package mongo

import (
	"context"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	model2 "github.com/SENERGY-Platform/event-worker/pkg/model"
	"github.com/SENERGY-Platform/process-deployment/lib/model/deploymentmodel"
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
	"github.com/SENERGY-Platform/process-sync/pkg/tests/docker"
)

func TestLastNetworkContact(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mongoPort, _, err := docker.Mongo(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}

	config := configuration.Config{
		MongoUrl:                          "mongodb://localhost:" + mongoPort,
		MongoTable:                        "processes",
		MongoProcessDefinitionCollection:  "process_definition",
		MongoDeploymentCollection:         "deployments",
		MongoProcessHistoryCollection:     "histories",
		MongoIncidentCollection:           "incidents",
		MongoProcessInstanceCollection:    "instances",
		MongoDeploymentMetadataCollection: "deployment_metadata",
		MongoLastNetworkContactCollection: "last_network_collection",
		MongoWardenCollection:             "warden",
		MongoDeploymentWardenCollection:   "deployment_warden",
	}

	db, err := New(config)
	if err != nil {
		t.Error(err)
		return
	}

	maxAge, err := time.ParseDuration("1h")
	if err != nil {
		t.Error(err)
		return
	}
	network1 := model.LastNetworkContact{NetworkId: "n1", Time: time.Now()}
	network2 := model.LastNetworkContact{NetworkId: "n2", Time: time.Now().Add(-(maxAge * 10))}

	t.Run("create elements", func(t *testing.T) {
		t.Run("create last contacts", func(t *testing.T) {
			err = db.SaveLastContact(network1)
			if err != nil {
				t.Error(err)
				return
			}
			err = db.SaveLastContact(network2)
			if err != nil {
				t.Error(err)
				return
			}
		})
		t.Run("create deployments", func(t *testing.T) {
			err = db.SaveDeployment(model.Deployment{
				Deployment: camundamodel.Deployment{Id: "1"},
				SyncInfo:   model.SyncInfo{NetworkId: network1.NetworkId},
			})
			if err != nil {
				t.Error(err)
				return
			}
			err = db.SaveDeployment(model.Deployment{
				Deployment: camundamodel.Deployment{Id: "2"},
				SyncInfo:   model.SyncInfo{NetworkId: network2.NetworkId},
			})
			if err != nil {
				t.Error(err)
				return
			}
		})
		t.Run("create definitions", func(t *testing.T) {
			err = db.SaveProcessDefinition(model.ProcessDefinition{
				ProcessDefinition: camundamodel.ProcessDefinition{Id: "1"},
				SyncInfo:          model.SyncInfo{NetworkId: network1.NetworkId},
			})
			if err != nil {
				t.Error(err)
				return
			}
			err = db.SaveProcessDefinition(model.ProcessDefinition{
				ProcessDefinition: camundamodel.ProcessDefinition{Id: "2"},
				SyncInfo:          model.SyncInfo{NetworkId: network2.NetworkId},
			})
			if err != nil {
				t.Error(err)
				return
			}
		})
		t.Run("create metadata", func(t *testing.T) {
			err = db.SaveDeploymentMetadata(model.DeploymentMetadata{
				Metadata: model.Metadata{
					CamundaDeploymentId: "1",
					DeploymentModel: model.DeploymentWithEventDesc{
						Deployment:        deploymentmodel.Deployment{Id: "1"},
						EventDescriptions: []model2.EventDesc{{DeviceGroupId: "dg1"}},
					},
				},
				SyncInfo: model.SyncInfo{NetworkId: network1.NetworkId},
			})
			if err != nil {
				t.Error(err)
				return
			}
			err = db.SaveDeploymentMetadata(model.DeploymentMetadata{
				Metadata: model.Metadata{
					CamundaDeploymentId: "2",
					DeploymentModel: model.DeploymentWithEventDesc{
						Deployment:        deploymentmodel.Deployment{Id: "2"},
						EventDescriptions: []model2.EventDesc{{DeviceGroupId: "dg1"}},
					},
				},
				SyncInfo: model.SyncInfo{NetworkId: network2.NetworkId},
			})
			if err != nil {
				t.Error(err)
				return
			}
			err = db.SaveDeploymentMetadata(model.DeploymentMetadata{
				Metadata: model.Metadata{
					CamundaDeploymentId: "3",
					DeploymentModel: model.DeploymentWithEventDesc{
						Deployment:        deploymentmodel.Deployment{Id: "3"},
						EventDescriptions: []model2.EventDesc{{DeviceGroupId: "dg2"}},
					},
				},
				SyncInfo: model.SyncInfo{NetworkId: "nope"},
			})
			if err != nil {
				t.Error(err)
				return
			}
		})

		t.Run("get deployments by device-group", func(t *testing.T) {
			list, err := db.ListDeploymentMetadataByEventDeviceGroupId("dg1")
			if err != nil {
				t.Error(err)
				return
			}
			sort.Slice(list, func(i, j int) bool {
				return list[i].CamundaDeploymentId < list[j].CamundaDeploymentId
			})
			expected := []model.DeploymentMetadata{
				{
					Metadata: model.Metadata{
						CamundaDeploymentId: "1",
						DeploymentModel: model.DeploymentWithEventDesc{
							Deployment:        deploymentmodel.Deployment{Id: "1"},
							EventDescriptions: []model2.EventDesc{{DeviceGroupId: "dg1"}},
						},
					},
					SyncInfo: model.SyncInfo{NetworkId: network1.NetworkId},
				},
				{
					Metadata: model.Metadata{
						CamundaDeploymentId: "2",
						DeploymentModel: model.DeploymentWithEventDesc{
							Deployment:        deploymentmodel.Deployment{Id: "2"},
							EventDescriptions: []model2.EventDesc{{DeviceGroupId: "dg1"}},
						},
					},
					SyncInfo: model.SyncInfo{NetworkId: network2.NetworkId},
				},
			}
			if !reflect.DeepEqual(expected, list) {
				t.Errorf("%#v\n%#v\n", expected, list)
			}
		})

		t.Run("create histories", func(t *testing.T) {
			err = db.SaveHistoricProcessInstance(model.HistoricProcessInstance{
				HistoricProcessInstance: camundamodel.HistoricProcessInstance{Id: "1"},
				SyncInfo:                model.SyncInfo{NetworkId: network1.NetworkId},
			})
			if err != nil {
				t.Error(err)
				return
			}
			err = db.SaveHistoricProcessInstance(model.HistoricProcessInstance{
				HistoricProcessInstance: camundamodel.HistoricProcessInstance{Id: "2"},
				SyncInfo:                model.SyncInfo{NetworkId: network2.NetworkId},
			})
			if err != nil {
				t.Error(err)
				return
			}
		})
		t.Run("create incidents", func(t *testing.T) {
			newDoc, err := db.SaveIncident(model.Incident{
				Incident: camundamodel.Incident{Id: "1"},
				SyncInfo: model.SyncInfo{NetworkId: network1.NetworkId},
			})
			if err != nil {
				t.Error(err)
				return
			}
			if !newDoc {
				t.Error("doc should be new")
				return
			}

			//check newDoc result
			newDoc, err = db.SaveIncident(model.Incident{
				Incident: camundamodel.Incident{Id: "1"},
				SyncInfo: model.SyncInfo{NetworkId: network1.NetworkId},
			})
			if err != nil {
				t.Error(err)
				return
			}
			if newDoc {
				t.Error("doc should not be new")
				return
			}

			newDoc, err = db.SaveIncident(model.Incident{
				Incident: camundamodel.Incident{Id: "2"},
				SyncInfo: model.SyncInfo{NetworkId: network2.NetworkId},
			})
			if err != nil {
				t.Error(err)
				return
			}
			if !newDoc {
				t.Error("doc should be new")
				return
			}
		})
		t.Run("create instance", func(t *testing.T) {
			err = db.SaveProcessInstance(model.ProcessInstance{
				ProcessInstance: camundamodel.ProcessInstance{Id: "1"},
				SyncInfo:        model.SyncInfo{NetworkId: network1.NetworkId},
			})
			if err != nil {
				t.Error(err)
				return
			}
			err = db.SaveProcessInstance(model.ProcessInstance{
				ProcessInstance: camundamodel.ProcessInstance{Id: "2"},
				SyncInfo:        model.SyncInfo{NetworkId: network2.NetworkId},
			})
			if err != nil {
				t.Error(err)
				return
			}
		})
	})

	t.Run("check FilterNetworkIds()", func(t *testing.T) {
		result, err := db.FilterNetworkIds([]string{"n1", "n2"})
		if err != nil {
			t.Error(err)
			return
		}
		sort.Strings(result)
		if !reflect.DeepEqual(result, []string{"n1", "n2"}) {
			t.Error(result)
			return
		}
		result, err = db.FilterNetworkIds([]string{"n1", "n3"})
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(result, []string{"n1"}) {
			t.Error(result)
			return
		}
		result, err = db.FilterNetworkIds([]string{"n3"})
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(result, []string{}) {
			t.Error(result)
			return
		}
	})

	t.Run("check GetOldNetworkIds()", func(t *testing.T) {
		result, err := db.GetOldNetworkIds(maxAge)
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(result, []string{"n2"}) {
			t.Error(result)
			return
		}
	})

	t.Run("check RemoveOldElements()", func(t *testing.T) {
		err := db.RemoveOldElements(maxAge)
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("check RemoveOldElements result", func(t *testing.T) {
		t.Run("check networks", func(t *testing.T) {
			result, err := db.FilterNetworkIds([]string{"n1", "n2"})
			if err != nil {
				t.Error(err)
				return
			}
			if !reflect.DeepEqual(result, []string{"n1"}) {
				t.Error(result)
				return
			}
		})

		t.Run("check definitions", func(t *testing.T) {
			result, err := db.ListProcessDefinitions([]string{"n1", "n2"}, 100, 0, "id.asc")
			if err != nil {
				t.Error(err)
				return
			}
			ids := []string{}
			for _, e := range result {
				ids = append(ids, e.Id)
			}
			if !reflect.DeepEqual(ids, []string{"1"}) {
				t.Error(ids)
				return
			}
		})
		t.Run("check deployments", func(t *testing.T) {
			result, err := db.ListDeployments([]string{"n1", "n2"}, 100, 0, "id.asc")
			if err != nil {
				t.Error(err)
				return
			}
			ids := []string{}
			for _, e := range result {
				ids = append(ids, e.Id)
			}
			if !reflect.DeepEqual(ids, []string{"1"}) {
				t.Error(ids)
				return
			}
		})
		t.Run("check metadata", func(t *testing.T) {
			ids := []string{}
			result, err := db.GetDeploymentMetadataOfDeploymentIdList("n1", []string{"1", "2"})
			if err != nil {
				t.Error(err)
				return
			}
			for _, e := range result {
				ids = append(ids, e.CamundaDeploymentId)
			}
			result, err = db.GetDeploymentMetadataOfDeploymentIdList("n2", []string{"1", "2"})
			if err != nil {
				t.Error(err)
				return
			}
			for _, e := range result {
				ids = append(ids, e.CamundaDeploymentId)
			}
			if !reflect.DeepEqual(ids, []string{"1"}) {
				t.Error(ids)
				return
			}
		})
		t.Run("check history", func(t *testing.T) {
			result, _, err := db.ListHistoricProcessInstances([]string{"n1", "n2"}, model.HistoryQuery{}, 100, 0, "id.asc")
			if err != nil {
				t.Error(err)
				return
			}
			ids := []string{}
			for _, e := range result {
				ids = append(ids, e.Id)
			}
			if !reflect.DeepEqual(ids, []string{"1"}) {
				t.Error(ids)
				return
			}
		})
		t.Run("check incident", func(t *testing.T) {
			ids := []string{}
			result, err := db.ListIncidents([]string{"n1", "n2"}, "", 100, 0, "id.asc")
			if err != nil {
				t.Error(err)
				return
			}
			for _, e := range result {
				ids = append(ids, e.Id)
			}
			if !reflect.DeepEqual(ids, []string{"1"}) {
				t.Error(ids)
				return
			}
		})
		t.Run("check instance", func(t *testing.T) {
			result, err := db.ListProcessInstances([]string{"n1", "n2"}, 100, 0, "id.asc")
			if err != nil {
				t.Error(err)
				return
			}
			ids := []string{}
			for _, e := range result {
				ids = append(ids, e.Id)
			}
			if !reflect.DeepEqual(ids, []string{"1"}) {
				t.Error(ids)
				return
			}
		})
	})
}
