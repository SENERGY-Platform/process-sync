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
	"sync"

	"github.com/SENERGY-Platform/event-deployment/lib/interfaces"
	"github.com/SENERGY-Platform/models/go/models"
	"github.com/SENERGY-Platform/process-deployment/lib/auth"
	"github.com/SENERGY-Platform/process-sync/pkg/api"
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/controller"
	"github.com/SENERGY-Platform/process-sync/pkg/database/mongo"
	"github.com/SENERGY-Platform/process-sync/pkg/tests/docker"
	"github.com/SENERGY-Platform/process-sync/pkg/tests/mocks"
)

func Env(ctx context.Context, wg *sync.WaitGroup, initConf configuration.Config, networkId string) (config configuration.Config, err error) {
	config = initConf
	config.ApiPort, err = docker.GetFreePortStr()
	if err != nil {
		return config, err
	}
	if config.WardenAgeGate == "" {
		config.WardenAgeGate = "2s"
	}
	if config.WardenInterval == "" {
		config.WardenInterval = "5s"
	}
	config.RunWardenDbLoop = true
	config.RunWardenProcessLoop = true
	config.RunWardenDeploymentLoop = true

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
	config.Mqtt = []configuration.MqttConfig{{
		Broker: "tcp://" + mqttip + ":1883",
	}}

	mongoPort, mongoIp, err := docker.Mongo(ctx, wg)
	config.MongoUrl = "mongodb://localhost:" + mongoPort
	clientMetadataStorageUrl := "mongodb://" + mongoIp + ":27017/metadata"

	err = docker.MgwProcessSyncClient(ctx, wg, camundaDb, camundaUrl, config.Mqtt[0].Broker, "mgw-test-sync-client", networkId, clientMetadataStorageUrl)
	if err != nil {
		return config, err
	}

	db, err := mongo.New(config)
	if err != nil {
		return config, err
	}
	d := &mocks.Devices{}

	ctrl, err := controller.New(config, ctx, db, mocks.Security(), func(token string, deviceRepoUrl string) interfaces.Devices {
		return d
	}, func(token string, baseUrl string, deviceId string) (result models.Device, err error, code int) {
		return d.GetDevice(auth.Token{Token: token}, deviceId)
	})
	if err != nil {
		return config, err
	}
	err = api.Start(config, ctx, ctrl)
	if err != nil {
		return config, err
	}

	return config, nil
}

func EnvForEventsCheck(ctx context.Context, wg *sync.WaitGroup, initConf configuration.Config, networkId string) (conf configuration.Config, err error) {
	conf = initConf
	conf.ApiPort, err = docker.GetFreePortStr()
	if err != nil {
		return conf, err
	}
	conf.DeviceRepoUrl = "placeholder"
	if conf.WardenAgeGate == "" {
		conf.WardenAgeGate = "2s"
	}
	if conf.WardenInterval == "" {
		conf.WardenInterval = "5s"
	}
	conf.RunWardenDbLoop = true
	conf.RunWardenProcessLoop = true
	conf.RunWardenDeploymentLoop = true

	_, mqttip, err := docker.Mqtt(ctx, wg)
	if err != nil {
		return conf, err
	}
	conf.Mqtt = []configuration.MqttConfig{{
		Broker: "tcp://" + mqttip + ":1883",
	}}

	mongoPort, _, err := docker.Mongo(ctx, wg)
	conf.MongoUrl = "mongodb://localhost:" + mongoPort

	db, err := mongo.New(conf)
	if err != nil {
		return conf, err
	}

	d := &mocks.Devices{}

	ctrl, err := controller.New(conf, ctx, db, mocks.Security(), func(token string, deviceRepoUrl string) interfaces.Devices {
		return d
	}, func(token string, baseUrl string, deviceId string) (result models.Device, err error, code int) {
		return d.GetDevice(auth.Token{Token: token}, deviceId)
	})
	if err != nil {
		return conf, err
	}

	err = api.Start(conf, ctx, ctrl)
	if err != nil {
		return conf, err
	}

	return conf, nil
}
