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
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
	"github.com/SENERGY-Platform/process-sync/pkg/tests/docker"
	"reflect"
	"sync"
	"testing"
)

func TestDeployment(t *testing.T) {
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
		Debug:                            true,
		MongoUrl:                         "mongodb://localhost:" + mongoPort,
		MongoTable:                       "processes",
		MongoProcessDefinitionCollection: "process_definition",
		MongoDeploymentCollection:        "deployments",
		MongoProcessHistoryCollection:    "histories",
		MongoIncidentCollection:          "incidents",
		MongoProcessInstanceCollection:   "instances",
	}

	db, err := New(config)
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("create deployment 1 n1 true", testCreateDeployment(db, "n1", "def1", true))
	t.Run("create deployment 2 n1 false", testCreateDeployment(db, "n1", "def2", false))
	t.Run("create deployment 1 n2 true", testCreateDeployment(db, "n2", "def1", true))
	t.Run("create deployment 2 n2 false", testCreateDeployment(db, "n2", "def2", false))

	t.Run("list n1 n2", testListDeployments(db, []string{"n1", "n2"}, []model.Deployment{
		{
			Deployment: camundamodel.Deployment{
				Id:   "def1",
				Name: "def1",
			},
			SyncInfo: model.SyncInfo{
				NetworkId:     "n1",
				IsPlaceholder: true,
			},
		},
		{
			Deployment: camundamodel.Deployment{
				Id:   "def1",
				Name: "def1",
			},
			SyncInfo: model.SyncInfo{
				NetworkId:     "n2",
				IsPlaceholder: true,
			},
		},
		{
			Deployment: camundamodel.Deployment{
				Id:   "def2",
				Name: "def2",
			},
			SyncInfo: model.SyncInfo{
				NetworkId:     "n1",
				IsPlaceholder: false,
			},
		},
		{
			Deployment: camundamodel.Deployment{
				Id:   "def2",
				Name: "def2",
			},
			SyncInfo: model.SyncInfo{
				NetworkId:     "n2",
				IsPlaceholder: false,
			},
		},
	}))

	t.Run("n1 remove placeholder deployments", testRemovePlaceholderDeployments(db, "n1"))

	t.Run("list n1 n2 after placeholder remove", testListDeployments(db, []string{"n1", "n2"}, []model.Deployment{
		{
			Deployment: camundamodel.Deployment{
				Id:   "def1",
				Name: "def1",
			},
			SyncInfo: model.SyncInfo{
				NetworkId:     "n2",
				IsPlaceholder: true,
			},
		},
		{
			Deployment: camundamodel.Deployment{
				Id:   "def2",
				Name: "def2",
			},
			SyncInfo: model.SyncInfo{
				NetworkId:     "n1",
				IsPlaceholder: false,
			},
		},
		{
			Deployment: camundamodel.Deployment{
				Id:   "def2",
				Name: "def2",
			},
			SyncInfo: model.SyncInfo{
				NetworkId:     "n2",
				IsPlaceholder: false,
			},
		},
	}))

}

func testRemovePlaceholderDeployments(db *Mongo, networkId string) func(t *testing.T) {
	return func(t *testing.T) {
		err := db.RemovePlaceholderDeployments(networkId)
		if err != nil {
			t.Error(err)
			return
		}
	}
}

func testCreateDeployment(db *Mongo, networkId string, defId string, placeholder bool) func(t *testing.T) {
	return func(t *testing.T) {
		err := db.SaveDeployment(model.Deployment{
			Deployment: camundamodel.Deployment{
				Id:   defId,
				Name: defId,
			},
			SyncInfo: model.SyncInfo{
				NetworkId:     networkId,
				IsPlaceholder: placeholder,
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
	}
}

func testListDeployments(db *Mongo, networkIds []string, expected []model.Deployment) func(t *testing.T) {
	return func(t *testing.T) {
		actual, err := db.ListDeployments(networkIds, 10, 0, "id")
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(actual, expected) {
			t.Error(actual, expected)
			return
		}
	}
}
