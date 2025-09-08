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
	"sync"
	"testing"
	"time"

	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
	"github.com/SENERGY-Platform/process-sync/pkg/tests/docker"
)

func TestHistorySearch(t *testing.T) {
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

	t.Run("add running 1", testAddHistory(db, camundamodel.HistoricProcessInstance{
		Id:                    "r1",
		ProcessDefinitionName: "test foo bar",
		ProcessDefinitionId:   "pdid_1",
		EndTime:               "",
	}))

	t.Run("add running 2", testAddHistory(db, camundamodel.HistoricProcessInstance{
		Id:                    "r2",
		ProcessDefinitionName: "test foo",
		ProcessDefinitionId:   "pdid_2",
		EndTime:               "",
	}))

	t.Run("add running 3", testAddHistory(db, camundamodel.HistoricProcessInstance{
		Id:                    "r3",
		ProcessDefinitionName: "test bar",
		ProcessDefinitionId:   "pdid_3",
		EndTime:               "",
	}))

	t.Run("add stopped 1", testAddHistory(db, camundamodel.HistoricProcessInstance{
		Id:                    "s1",
		ProcessDefinitionName: "test foo bar",
		ProcessDefinitionId:   "pdid_1",
		EndTime:               "something",
	}))

	t.Run("add stopped 2", testAddHistory(db, camundamodel.HistoricProcessInstance{
		Id:                    "s2",
		ProcessDefinitionName: "test foo",
		ProcessDefinitionId:   "pdid_2",
		EndTime:               "something",
	}))

	t.Run("add stopped 3", testAddHistory(db, camundamodel.HistoricProcessInstance{
		Id:                    "s3",
		ProcessDefinitionName: "test bar",
		ProcessDefinitionId:   "pdid_3",
		EndTime:               "something",
	}))

	t.Run("find all", testFind(db, "", "", "", []string{"r1", "r2", "r3", "s1", "s2", "s3"}))
	t.Run("find running", testFind(db, "unfinished", "", "", []string{"r1", "r2", "r3"}))
	t.Run("find stopped", testFind(db, "finished", "", "", []string{"s1", "s2", "s3"}))

	t.Run("find pdid_2", testFind(db, "", "", "pdid_2", []string{"r2", "s2"}))
	t.Run("find running pdid_2", testFind(db, "unfinished", "", "pdid_2", []string{"r2"}))
	t.Run("find stopped pdid_2", testFind(db, "finished", "", "pdid_2", []string{"s2"}))

	t.Run("search foo", testFind(db, "", "foo", "", []string{"r1", "r2", "s1", "s2"}))
	t.Run("search running foo", testFind(db, "unfinished", "foo", "", []string{"r1", "r2"}))
	t.Run("search stopped foo", testFind(db, "finished", "foo", "", []string{"s1", "s2"}))
}

func testFind(db *Mongo, state string, search string, definitionId string, expectedResultIds []string) func(t *testing.T) {
	return func(t *testing.T) {
		result, total, err := db.ListHistoricProcessInstances(
			[]string{"test"},
			model.HistoryQuery{
				State:               state,
				ProcessDefinitionId: definitionId,
				Search:              search,
			},
			100,
			0,
			"id.asc")

		if err != nil {
			t.Error(err)
			return
		}

		if int(total) != len(expectedResultIds) {
			t.Error(total, len(expectedResultIds))
		}

		actualIds := []string{}
		for _, element := range result {
			actualIds = append(actualIds, element.Id)
		}
		if !reflect.DeepEqual(actualIds, expectedResultIds) {
			t.Error(actualIds, expectedResultIds)
		}
	}
}

func testAddHistory(db *Mongo, instance camundamodel.HistoricProcessInstance) func(t *testing.T) {
	return func(t *testing.T) {
		err := db.SaveHistoricProcessInstance(model.HistoricProcessInstance{
			HistoricProcessInstance: instance,
			SyncInfo: model.SyncInfo{
				NetworkId:       "test",
				IsPlaceholder:   false,
				MarkedForDelete: false,
				SyncDate:        time.Time{},
			},
		})
		if err != nil {
			t.Error(err)
		}
	}
}
