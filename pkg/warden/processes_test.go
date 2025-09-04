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
	"context"
	"errors"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/database/mongo"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
	"github.com/SENERGY-Platform/process-sync/pkg/tests/docker"
)

func TestProcesses_AllInstances(t *testing.T) {
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
	}

	db, err := mongo.New(config)
	if err != nil {
		t.Error(err)
		return
	}

	expected := []string{}

	for i := range 100 {
		id := strconv.Itoa(i)
		expected = append(expected, id)
		err = db.SaveProcessInstance(model.ProcessInstance{
			ProcessInstance: camundamodel.ProcessInstance{
				Id:           id,
				DefinitionId: id,
				BusinessKey:  id,
				Ended:        false,
				Suspended:    false,
			},
			SyncInfo: model.SyncInfo{
				NetworkId:       "a",
				IsPlaceholder:   false,
				MarkedForDelete: false,
				MarkedAsMissing: false,
				SyncDate:        time.Now(),
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
	}

	slices.Sort(expected)

	processes := Processes{
		db:        db,
		batchsize: 2,
	}

	i := 0
	for instance, err := range processes.AllInstances() {
		if err != nil {
			t.Error(err)
			return
		}
		if instance.Id != expected[i] {
			t.Error("unexpected instance id:", instance.Id, "expected:", expected[i])
			return
		}
		i++
	}
	if i != 100 {
		t.Error("expected 100 instances")
		return
	}

}

func TestProcesses_GetYoungestProcessInstance(t *testing.T) {
	now := time.Now()
	instances := []model.ProcessInstance{
		{
			ProcessInstance: camundamodel.ProcessInstance{Id: "2"},
			SyncInfo:        model.SyncInfo{SyncDate: now.Add(time.Hour)},
		},
		{
			ProcessInstance: camundamodel.ProcessInstance{Id: "1"},
			SyncInfo:        model.SyncInfo{SyncDate: now},
		},
		{
			ProcessInstance: camundamodel.ProcessInstance{Id: "3"},
			SyncInfo:        model.SyncInfo{SyncDate: now.Add(24 * time.Hour)},
		},
	}
	youngest, err := GetYoungestElement(instances, func(instance model.ProcessInstance) (time.Time, error) {
		return instance.SyncDate, nil
	})
	if err != nil {
		t.Error(err)
		return
	}
	if youngest.Id != "1" {
		t.Error("unexpected youngest instance:", youngest.Id)
		return
	}

}

func TestProcesses_GetYoungestProcessInstance_Error(t *testing.T) {
	now := time.Now()
	instances := []model.ProcessInstance{
		{
			ProcessInstance: camundamodel.ProcessInstance{Id: "2"},
			SyncInfo:        model.SyncInfo{SyncDate: now.Add(time.Hour)},
		},
		{
			ProcessInstance: camundamodel.ProcessInstance{Id: "1"},
			SyncInfo:        model.SyncInfo{SyncDate: now},
		},
		{
			ProcessInstance: camundamodel.ProcessInstance{Id: "3"},
			SyncInfo:        model.SyncInfo{SyncDate: now.Add(24 * time.Hour)},
		},
	}
	_, err := GetYoungestElement(instances, func(instance model.ProcessInstance) (time.Time, error) {
		return instance.SyncDate, errors.New("error")
	})
	if err == nil {
		t.Error(err)
		return
	}
}
