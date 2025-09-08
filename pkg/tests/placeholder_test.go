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

package tests

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/SENERGY-Platform/event-deployment/lib/interfaces"
	"github.com/SENERGY-Platform/models/go/models"
	"github.com/SENERGY-Platform/process-deployment/lib/auth"
	"github.com/SENERGY-Platform/process-sync/pkg/api"
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/controller"
	"github.com/SENERGY-Platform/process-sync/pkg/database/mongo"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
	"github.com/SENERGY-Platform/process-sync/pkg/tests/docker"
	"github.com/SENERGY-Platform/process-sync/pkg/tests/mocks"
)

func TestPlaceholderProcessInstanceDelete(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := configuration.Config{
		MqttCleanSession:                  true,
		MqttGroupId:                       "",
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

	networkId := "test-network-id"

	var err error

	config.ApiPort, err = docker.GetFreePortStr()
	if err != nil {
		t.Error(err)
		return
	}

	var camundaPgIp string
	camundaDb, camundaPgIp, _, err := docker.PostgresWithNetwork(ctx, wg, "camunda")
	if err != nil {
		t.Error(err)
		return
	}

	camundaUrl, err := docker.Camunda(ctx, wg, camundaPgIp, "5432")
	if err != nil {
		t.Error(err)
		return
	}

	_, mqttip, err := docker.Mqtt(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	config.Mqtt = []configuration.MqttConfig{{
		Broker: "tcp://" + mqttip + ":1883",
	}}

	mongoPort, mongoIp, err := docker.Mongo(ctx, wg)
	config.MongoUrl = "mongodb://localhost:" + mongoPort
	clientMetadataStorageUrl := "mongodb://" + mongoIp + ":27017/metadata"

	clientCtx, clientStop := context.WithCancel(ctx)

	err = docker.MgwProcessSyncClient(clientCtx, wg, camundaDb, camundaUrl, config.Mqtt[0].Broker, "mgw-test-sync-client", networkId, clientMetadataStorageUrl)
	if err != nil {
		t.Error(err)
		return
	}

	db, err := mongo.New(config)
	if err != nil {
		t.Error(err)
		return
	}
	d := &mocks.Devices{}

	ctrl, err := controller.New(config, ctx, db, mocks.Security(), func(token string, deviceRepoUrl string) interfaces.Devices {
		return d
	}, func(token string, baseUrl string, deviceId string) (result models.Device, err error, code int) {
		return d.GetDevice(auth.Token{Token: token}, deviceId)
	})

	err = api.Start(config, ctx, ctrl)
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("deploy process", testDeployProcessWithParameter(config.ApiPort, networkId))

	deployments := []model.Deployment{}
	t.Run("get deployments", testGetDeployments(config.ApiPort, networkId, &deployments))
	t.Run("check deployments is placeholder", testCheckDeploymentsPlaceholder(&deployments, []bool{true}))
	t.Run("check deployments is marked delete", testCheckDeploymentsMarkedDelete(&deployments, []bool{false}))

	time.Sleep(5 * time.Second)
	t.Run("get deployments", testGetDeployments(config.ApiPort, networkId, &deployments))
	t.Run("check deployments is placeholder", testCheckDeploymentsPlaceholder(&deployments, []bool{false}))
	t.Run("check deployments is marked delete", testCheckDeploymentsMarkedDelete(&deployments, []bool{false}))

	t.Run("check process definitions", testCheckProcessDefinitions(config, networkId))

	deploymentsPreDelete := deployments
	t.Run("check process metadata", testCheckProcessMetadata(config, networkId, &deploymentsPreDelete, 0, true))

	t.Run("check deployment parameter", testGetDeploymentParameter(
		config.ApiPort,
		networkId,
		&deployments,
		0,
		map[string]camundamodel.Variable{
			"targetTemperature": {
				Type:      "String",
				ValueInfo: []interface{}{},
			},
		}))

	clientStop()

	time.Sleep(5 * time.Second)

	t.Run("start deployment", testStartDeploymentWithParameter(config.ApiPort, networkId, &deployments, 0, "targetTemperature=21"))

	instances := []model.ProcessInstance{}
	t.Run("get instances", testGetInstances(config.ApiPort, networkId, &instances))
	t.Run("check instance is placeholder", testCheckInstancesPlaceholder(&instances, []bool{true}))
	t.Run("check instance is marked delete", testCheckInstancesMarkedDelete(&instances, []bool{false}))

	historicInstances := []model.HistoricProcessInstance{}
	t.Run("get historic instances", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
	t.Run("check historic instance is placeholder", testCheckHistoricInstancesPlaceholder(&historicInstances, []bool{true}))
	t.Run("check historic instance is marked delete", testCheckHistoricInstancesMarkedDelete(&historicInstances, []bool{false}))

	t.Run("stop instance", testDeleteInstances(config.ApiPort, networkId, &instances, 0))

	t.Run("get instances", testGetInstances(config.ApiPort, networkId, &instances))
	t.Run("check instance is placeholder", testCheckInstancesPlaceholder(&instances, []bool{}))
	t.Run("check instance is marked delete", testCheckInstancesMarkedDelete(&instances, []bool{}))

	t.Run("get historic instances", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
	t.Run("check historic instance", testCheckHistoricInstancesPlaceholder(&historicInstances, []bool{}))
}

func TestPlaceholderProcessInstanceStopWithHistoryId(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := configuration.Config{
		MqttCleanSession:                  true,
		MqttGroupId:                       "",
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

	networkId := "test-network-id"

	var err error

	config.ApiPort, err = docker.GetFreePortStr()
	if err != nil {
		t.Error(err)
		return
	}

	var camundaPgIp string
	camundaDb, camundaPgIp, _, err := docker.PostgresWithNetwork(ctx, wg, "camunda")
	if err != nil {
		t.Error(err)
		return
	}

	camundaUrl, err := docker.Camunda(ctx, wg, camundaPgIp, "5432")
	if err != nil {
		t.Error(err)
		return
	}

	_, mqttip, err := docker.Mqtt(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	config.Mqtt = []configuration.MqttConfig{{
		Broker: "tcp://" + mqttip + ":1883",
	}}

	mongoPort, mongoIp, err := docker.Mongo(ctx, wg)
	config.MongoUrl = "mongodb://localhost:" + mongoPort
	clientMetadataStorageUrl := "mongodb://" + mongoIp + ":27017/metadata"

	clientCtx, clientStop := context.WithCancel(ctx)

	err = docker.MgwProcessSyncClient(clientCtx, wg, camundaDb, camundaUrl, config.Mqtt[0].Broker, "mgw-test-sync-client", networkId, clientMetadataStorageUrl)
	if err != nil {
		t.Error(err)
		return
	}

	db, err := mongo.New(config)
	if err != nil {
		t.Error(err)
		return
	}
	d := &mocks.Devices{}

	ctrl, err := controller.New(config, ctx, db, mocks.Security(), func(token string, deviceRepoUrl string) interfaces.Devices {
		return d
	}, func(token string, baseUrl string, deviceId string) (result models.Device, err error, code int) {
		return d.GetDevice(auth.Token{Token: token}, deviceId)
	})

	err = api.Start(config, ctx, ctrl)
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("deploy process", testDeployProcessWithParameter(config.ApiPort, networkId))

	deployments := []model.Deployment{}
	t.Run("get deployments", testGetDeployments(config.ApiPort, networkId, &deployments))
	t.Run("check deployments is placeholder", testCheckDeploymentsPlaceholder(&deployments, []bool{true}))
	t.Run("check deployments is marked delete", testCheckDeploymentsMarkedDelete(&deployments, []bool{false}))

	time.Sleep(5 * time.Second)
	t.Run("get deployments", testGetDeployments(config.ApiPort, networkId, &deployments))
	t.Run("check deployments is placeholder", testCheckDeploymentsPlaceholder(&deployments, []bool{false}))
	t.Run("check deployments is marked delete", testCheckDeploymentsMarkedDelete(&deployments, []bool{false}))

	t.Run("check process definitions", testCheckProcessDefinitions(config, networkId))

	deploymentsPreDelete := deployments
	t.Run("check process metadata", testCheckProcessMetadata(config, networkId, &deploymentsPreDelete, 0, true))

	t.Run("check deployment parameter", testGetDeploymentParameter(
		config.ApiPort,
		networkId,
		&deployments,
		0,
		map[string]camundamodel.Variable{
			"targetTemperature": {
				Type:      "String",
				ValueInfo: []interface{}{},
			},
		}))

	clientStop()

	time.Sleep(5 * time.Second)

	t.Run("start deployment", testStartDeploymentWithParameter(config.ApiPort, networkId, &deployments, 0, "targetTemperature=21"))

	instances := []model.ProcessInstance{}
	t.Run("get instances", testGetInstances(config.ApiPort, networkId, &instances))
	t.Run("check instance is placeholder", testCheckInstancesPlaceholder(&instances, []bool{true}))
	t.Run("check instance is marked delete", testCheckInstancesMarkedDelete(&instances, []bool{false}))

	historicInstances := []model.HistoricProcessInstance{}
	t.Run("get historic instances", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
	t.Run("check historic instance is placeholder", testCheckHistoricInstancesPlaceholder(&historicInstances, []bool{true}))
	t.Run("check historic instance is marked delete", testCheckHistoricInstancesMarkedDelete(&historicInstances, []bool{false}))

	if len(instances) == 0 {
		t.Error("missing instances")
		return
	}
	t.Run("stop instance by history id", testDeleteInstancesById(config.ApiPort, networkId, historicInstances[0].Id))

	t.Run("get instances", testGetInstances(config.ApiPort, networkId, &instances))
	t.Run("check instance is placeholder", testCheckInstancesPlaceholder(&instances, []bool{}))
	t.Run("check instance is marked delete", testCheckInstancesMarkedDelete(&instances, []bool{}))

	t.Run("get historic instances", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
	t.Run("check historic instance", testCheckHistoricInstancesPlaceholder(&historicInstances, []bool{}))
}
