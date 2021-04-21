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

package server

import (
	"context"
	"github.com/SENERGY-Platform/process-sync/pkg/api"
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/controller"
	"github.com/SENERGY-Platform/process-sync/pkg/database/mongo"
	"github.com/SENERGY-Platform/process-sync/pkg/tests/docker"
	"github.com/SENERGY-Platform/process-sync/pkg/tests/mocks"
	"sync"
)

func Env(ctx context.Context, wg *sync.WaitGroup, initConf configuration.Config, networkId string) (config configuration.Config, err error) {
	config = initConf
	config.ApiPort, err = docker.GetFreePortStr()
	if err != nil {
		return config, err
	}

	var camundaPgIp string
	camundaDb, camundaPgIp, _, err := docker.PostgresWithNetwork(ctx, wg, "camunda")
	if err != nil {
		return config, err
	}

	camundaUrl, err := docker.Camunda(ctx, wg, camundaPgIp, "5432")
	if err != nil {
		return config, err
	}

	_, mqttip, err := docker.Mqtt(ctx, wg)
	if err != nil {
		return config, err
	}
	config.MqttBroker = "tcp://" + mqttip + ":1883"

	mongoPort, mongoIp, err := docker.Mongo(ctx, wg)
	config.MongoUrl = "mongodb://localhost:" + mongoPort
	clientMetadataStorageUrl := "mongodb://" + mongoIp + ":27017/metadata"

	err = docker.MgwProcessSyncClient(ctx, wg, camundaDb, camundaUrl, config.MqttBroker, "mgw-test-sync-client", networkId, clientMetadataStorageUrl)
	if err != nil {
		return config, err
	}

	db, err := mongo.New(config)
	if err != nil {
		return config, err
	}
	ctrl, err := controller.New(config, ctx, db, mocks.Security(), nil)

	err = api.Start(config, ctx, ctrl)
	if err != nil {
		return config, err
	}

	return config, nil
}
