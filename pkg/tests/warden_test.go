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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
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
	"github.com/SENERGY-Platform/process-sync/pkg/multimqtt"
	"github.com/SENERGY-Platform/process-sync/pkg/tests/docker"
	"github.com/SENERGY-Platform/process-sync/pkg/tests/mocks"
	"github.com/SENERGY-Platform/process-sync/pkg/tests/resources"
	paho "github.com/eclipse/paho.mqtt.golang"
)

func TestWardenWithParameterProcess(t *testing.T) {
	t.Skip("TODO") //TODO
}

func TestWardenFailingProcess(t *testing.T) {
	bpmn := resources.IncidentWithDurBpmn
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wardenInterval := time.Second * 5
	wardenAgeGate := time.Second * 2

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

		LogLevel: "debug",

		WardenAgeGate:           wardenInterval.String(),
		WardenInterval:          wardenAgeGate.String(),
		RunWardenDeploymentLoop: true,
		RunWardenProcessLoop:    true,
		RunWardenDbLoop:         true,
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

	err = docker.TaskWorker(ctx, wg, config.Mqtt[0].Broker, camundaUrl, networkId)
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

	mqttMsgMux := sync.Mutex{}
	mqttMessages := map[string][]string{}
	client := multimqtt.NewClient(config.Mqtt, func(options *paho.ClientOptions) {
		options.SetCleanSession(true)
		options.SetResumeSubs(true)
		options.SetConnectionLostHandler(func(c paho.Client, err error) {
			o := c.OptionsReader()
			config.GetLogger().Error("connection to mqtt broker lost", "error", err, "client", o.ClientID())
		})
		options.SetOnConnectHandler(func(c paho.Client) {
			o := c.OptionsReader()
			config.GetLogger().Info("connected to mqtt broker", "client", o.ClientID())
			c.Subscribe("#", 1, func(c paho.Client, msg paho.Message) {
				mqttMsgMux.Lock()
				defer mqttMsgMux.Unlock()
				fmt.Println("mqtt message:", msg.Topic(), string(msg.Payload()))
				mqttMessages[msg.Topic()] = append(mqttMessages[msg.Topic()], string(msg.Payload()))
			})
		})
	})

	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		t.Error(token.Error())
		return
	}

	t.Run("client offline", func(t *testing.T) {
		t.Run("deploy process 1", testDeployProcessWithArgs(config.ApiPort, networkId, "test-1", bpmn))
		t.Run("deploy process 2", testDeployProcessWithArgs(config.ApiPort, networkId, "test-2", bpmn))
		t.Run("deploy process 3", testDeployProcessWithArgs(config.ApiPort, networkId, "test-3", bpmn))

		deployments := []model.Deployment{}
		t.Run("get deployments", testGetDeployments(config.ApiPort, networkId, &deployments))
		t.Run("check deployments count", func(t *testing.T) {
			if len(deployments) != 3 {
				t.Error("expected 3 deployments")
			}
		})

		t.Run("start deployment 1", testStartDeploymentWithKey(config.ApiPort, networkId, &deployments, 0, "inst1"))
		t.Run("start deployment 2", testStartDeploymentWithKey(config.ApiPort, networkId, &deployments, 1, "inst2"))
		t.Run("start deployment 3", testStartDeploymentWithKey(config.ApiPort, networkId, &deployments, 2, "inst3"))

		instances := []model.ProcessInstance{}
		t.Run("get instances", testGetInstances(config.ApiPort, networkId, &instances))
		slices.SortFunc(instances, func(a, b model.ProcessInstance) int {
			return strings.Compare(a.BusinessKey, b.BusinessKey)
		})
		t.Run("check instance count", func(t *testing.T) {
			if len(instances) != 3 {
				t.Error("expected 3 instances")
			}
		})

		historicInstances := []model.HistoricProcessInstance{}
		t.Run("get historic instances", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
		t.Run("check historic instance count", func(t *testing.T) {
			if len(historicInstances) != 3 {
				t.Error("expected 3 historicInstances")
			}
		})

		t.Run("stop instance 2", testDeleteInstances(config.ApiPort, networkId, &instances, 1))

		instances = []model.ProcessInstance{}
		t.Run("get instances after stop", testGetInstances(config.ApiPort, networkId, &instances))
		t.Run("check instance count after stop", func(t *testing.T) {
			if len(instances) != 2 {
				t.Error("expected 2 instances, got", len(instances))
				for _, v := range instances {
					t.Log(v.BusinessKey)
				}
			}
		})

		t.Run("delete deployment 3", testRemoveDeployment(config.ApiPort, networkId, &deployments, 2))

		deployments = []model.Deployment{}
		t.Run("get deployments after delete", testGetDeployments(config.ApiPort, networkId, &deployments))
		t.Run("check deployments count after delete", func(t *testing.T) {
			if len(deployments) != 2 {
				t.Error("expected 2 deployments, got", len(deployments))
			}
		})

		metadata := []model.DeploymentMetadata{}
		t.Run("get metadata", testGetDeploymentMetadata("http://localhost:"+config.ApiPort, networkId, &metadata))
		t.Run("check metadata count", func(t *testing.T) {
			if len(metadata) != 0 {
				t.Error("expected 0 metadata, got", len(metadata))
			}
		})

		instances = []model.ProcessInstance{}
		t.Run("get instances after delete", testGetInstances(config.ApiPort, networkId, &instances))
		t.Run("check instance count after delete", func(t *testing.T) {
			t.Skip("the deployment delete will not delete instances directly, but the warden and camunda will")
			if len(instances) != 1 {
				t.Error("expected 1 instances, got", len(instances))
				for _, v := range instances {
					t.Log(v.BusinessKey)
				}
			}
		})

		historicInstances = []model.HistoricProcessInstance{}
		t.Run("get historic instances after delete", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
		t.Run("check historic instance count after delete", func(t *testing.T) {
			t.Skip("the deployment delete will not delete instances directly, but the warden and camunda will")
			if len(historicInstances) != 1 {
				t.Error("expected 1 historicInstances, got", len(historicInstances))
			}
		})

	})

	t.Run("check backend state", func(t *testing.T) {
		time.Sleep(3 * wardenInterval)

		deployments := []model.Deployment{}
		t.Run("get deployments", testGetDeployments(config.ApiPort, networkId, &deployments))
		t.Run("check deployments count", func(t *testing.T) {
			if len(deployments) != 2 {
				t.Error("expected 2 deployments")
			}
		})

		metadata := []model.DeploymentMetadata{}
		t.Run("get metadata", testGetDeploymentMetadata("http://localhost:"+config.ApiPort, networkId, &metadata))
		t.Run("check metadata count", func(t *testing.T) {
			if len(metadata) != 0 {
				t.Error("expected 0 metadata, got", len(metadata))
			}
		})

		instances := []model.ProcessInstance{}
		t.Run("get instances", testGetInstances(config.ApiPort, networkId, &instances))
		t.Run("check instance count", func(t *testing.T) {
			if len(instances) != 1 {
				t.Error("expected 1 instances, got", len(instances))
				for _, v := range instances {
					t.Log(v.BusinessKey)
				}
			}
		})

		historicInstances := []model.HistoricProcessInstance{}
		t.Run("get historic instances", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
		t.Run("check historic instance count", func(t *testing.T) {
			if len(historicInstances) != 1 {
				t.Error("expected 1 historicInstances, got", len(historicInstances))
			}
		})
	})

	t.Run("start client", func(t *testing.T) {
		err = docker.MgwProcessSyncClient(ctx, wg, camundaDb, camundaUrl, config.Mqtt[0].Broker, "mgw-test-sync-client", networkId, clientMetadataStorageUrl)
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("check state after start", func(t *testing.T) {
		time.Sleep(time.Minute)

		t.Run("backend", func(t *testing.T) {
			deployments := []model.Deployment{}
			t.Run("get deployments", testGetDeployments(config.ApiPort, networkId, &deployments))
			t.Run("check deployments count", func(t *testing.T) {
				if len(deployments) != 2 {
					t.Error("expected 2 deployments, got", len(deployments))
				}
			})

			metadata := []model.DeploymentMetadata{}
			t.Run("get metadata", testGetDeploymentMetadata("http://localhost:"+config.ApiPort, networkId, &metadata))
			t.Run("check metadata count", func(t *testing.T) {
				if len(metadata) != 2 {
					t.Error("expected 2 metadata, got", len(metadata))
				}
			})

			instances := []model.ProcessInstance{}
			t.Run("get instances", testGetInstances(config.ApiPort, networkId, &instances))
			t.Run("check instance count", func(t *testing.T) {
				if len(instances) != 1 {
					t.Error("expected 1 instances, got", len(instances))
				}
			})

			historicInstances := []model.HistoricProcessInstance{}
			t.Run("get historic instances", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
			t.Run("check historic instance count", func(t *testing.T) {
				if len(historicInstances) <= 2 {
					t.Error("expected > 2 historicInstances, got", len(historicInstances))
				}
			})
		})

		t.Run("client", func(t *testing.T) {
			deployments := []model.Deployment{}
			t.Run("get deployments", testCamundaGetDeployments(camundaUrl, &deployments))
			t.Run("check deployments count", func(t *testing.T) {
				if len(deployments) != 2 {
					t.Error("expected 2 deployments, got", len(deployments))
				}
			})

			instances := []model.ProcessInstance{}
			t.Run("get instances", testCamundaGetInstances(camundaUrl, &instances))
			t.Run("check instance count", func(t *testing.T) {
				if len(instances) != 1 {
					t.Error("expected 1 instances, got", len(instances))
				}
			})

			historicInstances := []model.HistoricProcessInstance{}
			t.Run("get historic instances", testCamundaGetHistoricInstances(camundaUrl, &historicInstances))
			t.Run("check historic instance count", func(t *testing.T) {
				if len(historicInstances) <= 2 {
					t.Error("expected > 2 historicInstances, got", len(historicInstances))
				}
			})
		})
	})

	t.Run("mqtt msg log", func(t *testing.T) {
		mqttMsgMux.Lock()
		defer mqttMsgMux.Unlock()
		fmt.Println("mqtt messages")
		for k, v := range mqttMessages {
			fmt.Println("--------------")
			fmt.Println(k)
			for _, vv := range v {
				fmt.Println(vv)
			}
		}
	})
}

func TestWardenStoppingProcess(t *testing.T) {
	bpmn := resources.Finishing
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wardenInterval := time.Second * 5
	wardenAgeGate := time.Second * 2

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

		LogLevel: "debug",

		WardenAgeGate:           wardenInterval.String(),
		WardenInterval:          wardenAgeGate.String(),
		RunWardenDeploymentLoop: true,
		RunWardenProcessLoop:    true,
		RunWardenDbLoop:         true,
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

	err = docker.TaskWorker(ctx, wg, config.Mqtt[0].Broker, camundaUrl, networkId)
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

	mqttMsgMux := sync.Mutex{}
	mqttMessages := map[string][]string{}
	client := multimqtt.NewClient(config.Mqtt, func(options *paho.ClientOptions) {
		options.SetCleanSession(true)
		options.SetResumeSubs(true)
		options.SetConnectionLostHandler(func(c paho.Client, err error) {
			o := c.OptionsReader()
			config.GetLogger().Error("connection to mqtt broker lost", "error", err, "client", o.ClientID())
		})
		options.SetOnConnectHandler(func(c paho.Client) {
			o := c.OptionsReader()
			config.GetLogger().Info("connected to mqtt broker", "client", o.ClientID())
			c.Subscribe("#", 1, func(c paho.Client, msg paho.Message) {
				mqttMsgMux.Lock()
				defer mqttMsgMux.Unlock()
				fmt.Println("mqtt message:", msg.Topic(), string(msg.Payload()))
				mqttMessages[msg.Topic()] = append(mqttMessages[msg.Topic()], string(msg.Payload()))
			})
		})
	})

	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		t.Error(token.Error())
		return
	}

	t.Run("client offline", func(t *testing.T) {
		t.Run("deploy process 1", testDeployProcessWithArgs(config.ApiPort, networkId, "test-1", bpmn))
		t.Run("deploy process 2", testDeployProcessWithArgs(config.ApiPort, networkId, "test-2", bpmn))
		t.Run("deploy process 3", testDeployProcessWithArgs(config.ApiPort, networkId, "test-3", bpmn))

		deployments := []model.Deployment{}
		t.Run("get deployments", testGetDeployments(config.ApiPort, networkId, &deployments))
		t.Run("check deployments count", func(t *testing.T) {
			if len(deployments) != 3 {
				t.Error("expected 3 deployments")
			}
		})

		t.Run("start deployment 1", testStartDeploymentWithKey(config.ApiPort, networkId, &deployments, 0, "inst1"))
		t.Run("start deployment 2", testStartDeploymentWithKey(config.ApiPort, networkId, &deployments, 1, "inst2"))
		t.Run("start deployment 3", testStartDeploymentWithKey(config.ApiPort, networkId, &deployments, 2, "inst3"))

		instances := []model.ProcessInstance{}
		t.Run("get instances", testGetInstances(config.ApiPort, networkId, &instances))
		slices.SortFunc(instances, func(a, b model.ProcessInstance) int {
			return strings.Compare(a.BusinessKey, b.BusinessKey)
		})
		t.Run("check instance count", func(t *testing.T) {
			if len(instances) != 3 {
				t.Error("expected 3 instances")
			}
		})

		historicInstances := []model.HistoricProcessInstance{}
		t.Run("get historic instances", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
		t.Run("check historic instance count", func(t *testing.T) {
			if len(historicInstances) != 3 {
				t.Error("expected 3 historicInstances")
			}
		})

		t.Run("stop instance 2", testDeleteInstances(config.ApiPort, networkId, &instances, 1))

		instances = []model.ProcessInstance{}
		t.Run("get instances after stop", testGetInstances(config.ApiPort, networkId, &instances))
		t.Run("check instance count after stop", func(t *testing.T) {
			if len(instances) != 2 {
				t.Error("expected 2 instances, got", len(instances))
				for _, v := range instances {
					t.Log(v.BusinessKey)
				}
			}
		})

		t.Run("delete deployment 3", testRemoveDeployment(config.ApiPort, networkId, &deployments, 2))

		deployments = []model.Deployment{}
		t.Run("get deployments after delete", testGetDeployments(config.ApiPort, networkId, &deployments))
		t.Run("check deployments count after delete", func(t *testing.T) {
			if len(deployments) != 2 {
				t.Error("expected 2 deployments, got", len(deployments))
			}
		})

		metadata := []model.DeploymentMetadata{}
		t.Run("get metadata", testGetDeploymentMetadata("http://localhost:"+config.ApiPort, networkId, &metadata))
		t.Run("check metadata count", func(t *testing.T) {
			if len(metadata) != 0 {
				t.Error("expected 0 metadata, got", len(metadata))
			}
		})

		instances = []model.ProcessInstance{}
		t.Run("get instances after delete", testGetInstances(config.ApiPort, networkId, &instances))
		t.Run("check instance count after delete", func(t *testing.T) {
			t.Skip("the deployment delete will not delete instances directly, but the warden and camunda will")
			if len(instances) != 1 {
				t.Error("expected 1 instances, got", len(instances))
				for _, v := range instances {
					t.Log(v.BusinessKey)
				}
			}
		})

		historicInstances = []model.HistoricProcessInstance{}
		t.Run("get historic instances after delete", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
		t.Run("check historic instance count after delete", func(t *testing.T) {
			t.Skip("the deployment delete will not delete instances directly, but the warden and camunda will")
			if len(historicInstances) != 1 {
				t.Error("expected 1 historicInstances, got", len(historicInstances))
			}
		})

	})

	t.Run("check backend state", func(t *testing.T) {
		time.Sleep(3 * wardenInterval)

		deployments := []model.Deployment{}
		t.Run("get deployments", testGetDeployments(config.ApiPort, networkId, &deployments))
		t.Run("check deployments count", func(t *testing.T) {
			if len(deployments) != 2 {
				t.Error("expected 2 deployments")
			}
		})

		metadata := []model.DeploymentMetadata{}
		t.Run("get metadata", testGetDeploymentMetadata("http://localhost:"+config.ApiPort, networkId, &metadata))
		t.Run("check metadata count", func(t *testing.T) {
			if len(metadata) != 0 {
				t.Error("expected 0 metadata, got", len(metadata))
			}
		})

		instances := []model.ProcessInstance{}
		t.Run("get instances", testGetInstances(config.ApiPort, networkId, &instances))
		t.Run("check instance count", func(t *testing.T) {
			if len(instances) != 1 {
				t.Error("expected 1 instances, got", len(instances))
				for _, v := range instances {
					t.Log(v.BusinessKey)
				}
			}
		})

		historicInstances := []model.HistoricProcessInstance{}
		t.Run("get historic instances", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
		t.Run("check historic instance count", func(t *testing.T) {
			if len(historicInstances) != 1 {
				t.Error("expected 1 historicInstances, got", len(historicInstances))
			}
		})
	})

	t.Run("start client", func(t *testing.T) {
		err = docker.MgwProcessSyncClient(ctx, wg, camundaDb, camundaUrl, config.Mqtt[0].Broker, "mgw-test-sync-client", networkId, clientMetadataStorageUrl)
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("check state after start", func(t *testing.T) {
		time.Sleep(time.Minute)

		t.Run("backend", func(t *testing.T) {
			deployments := []model.Deployment{}
			t.Run("get deployments", testGetDeployments(config.ApiPort, networkId, &deployments))
			t.Run("check deployments count", func(t *testing.T) {
				if len(deployments) != 2 {
					t.Error("expected 2 deployments, got", len(deployments))
				}
			})

			metadata := []model.DeploymentMetadata{}
			t.Run("get metadata", testGetDeploymentMetadata("http://localhost:"+config.ApiPort, networkId, &metadata))
			t.Run("check metadata count", func(t *testing.T) {
				if len(metadata) != 2 {
					t.Error("expected 2 metadata, got", len(metadata))
				}
			})

			instances := []model.ProcessInstance{}
			t.Run("get instances", testGetInstances(config.ApiPort, networkId, &instances))
			t.Run("check instance count", func(t *testing.T) {
				if len(instances) != 0 {
					t.Error("expected 1 instances, got", len(instances))
				}
			})

			historicInstances := []model.HistoricProcessInstance{}
			t.Run("get historic instances", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
			t.Run("check historic instance count", func(t *testing.T) {
				if len(historicInstances) != 1 {
					t.Error("expected 1 historicInstances, got", len(historicInstances))
				}
			})
		})

		t.Run("client", func(t *testing.T) {
			deployments := []model.Deployment{}
			t.Run("get deployments", testCamundaGetDeployments(camundaUrl, &deployments))
			t.Run("check deployments count", func(t *testing.T) {
				if len(deployments) != 2 {
					t.Error("expected 2 deployments, got", len(deployments))
				}
			})

			instances := []model.ProcessInstance{}
			t.Run("get instances", testCamundaGetInstances(camundaUrl, &instances))
			t.Run("check instance count", func(t *testing.T) {
				if len(instances) != 0 {
					t.Error("expected 0 instances, got", len(instances))
				}
			})

			historicInstances := []model.HistoricProcessInstance{}
			t.Run("get historic instances", testCamundaGetHistoricInstances(camundaUrl, &historicInstances))
			t.Run("check historic instance count", func(t *testing.T) {
				if len(historicInstances) != 1 {
					t.Error("expected 1 historicInstances, got", len(historicInstances))
				}
			})
		})
	})

	t.Run("mqtt msg log", func(t *testing.T) {
		mqttMsgMux.Lock()
		defer mqttMsgMux.Unlock()
		fmt.Println("mqtt messages")
		for k, v := range mqttMessages {
			fmt.Println("--------------")
			fmt.Println(k)
			for _, vv := range v {
				fmt.Println(vv)
			}
		}
	})
}

func TestWardenLongRunningProcess(t *testing.T) {
	bpmn := resources.LongProcess
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wardenInterval := time.Second * 5
	wardenAgeGate := time.Second * 2

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

		LogLevel: "debug",

		WardenAgeGate:           wardenInterval.String(),
		WardenInterval:          wardenAgeGate.String(),
		RunWardenDeploymentLoop: true,
		RunWardenProcessLoop:    true,
		RunWardenDbLoop:         true,
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

	mqttMsgMux := sync.Mutex{}
	mqttMessages := map[string][]string{}
	client := multimqtt.NewClient(config.Mqtt, func(options *paho.ClientOptions) {
		options.SetCleanSession(true)
		options.SetResumeSubs(true)
		options.SetConnectionLostHandler(func(c paho.Client, err error) {
			o := c.OptionsReader()
			config.GetLogger().Error("connection to mqtt broker lost", "error", err, "client", o.ClientID())
		})
		options.SetOnConnectHandler(func(c paho.Client) {
			o := c.OptionsReader()
			config.GetLogger().Info("connected to mqtt broker", "client", o.ClientID())
			c.Subscribe("#", 1, func(c paho.Client, msg paho.Message) {
				mqttMsgMux.Lock()
				defer mqttMsgMux.Unlock()
				fmt.Println("mqtt message:", msg.Topic(), string(msg.Payload()))
				mqttMessages[msg.Topic()] = append(mqttMessages[msg.Topic()], string(msg.Payload()))
			})
		})
	})

	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		t.Error(token.Error())
		return
	}

	t.Run("client offline", func(t *testing.T) {
		t.Run("deploy process 1", testDeployProcessWithArgs(config.ApiPort, networkId, "test-1", bpmn))
		t.Run("deploy process 2", testDeployProcessWithArgs(config.ApiPort, networkId, "test-2", bpmn))
		t.Run("deploy process 3", testDeployProcessWithArgs(config.ApiPort, networkId, "test-3", bpmn))

		deployments := []model.Deployment{}
		t.Run("get deployments", testGetDeployments(config.ApiPort, networkId, &deployments))
		t.Run("check deployments count", func(t *testing.T) {
			if len(deployments) != 3 {
				t.Error("expected 3 deployments")
			}
		})

		t.Run("start deployment 1", testStartDeploymentWithKey(config.ApiPort, networkId, &deployments, 0, "inst1"))
		t.Run("start deployment 2", testStartDeploymentWithKey(config.ApiPort, networkId, &deployments, 1, "inst2"))
		t.Run("start deployment 3", testStartDeploymentWithKey(config.ApiPort, networkId, &deployments, 2, "inst3"))

		instances := []model.ProcessInstance{}
		t.Run("get instances", testGetInstances(config.ApiPort, networkId, &instances))
		slices.SortFunc(instances, func(a, b model.ProcessInstance) int {
			return strings.Compare(a.BusinessKey, b.BusinessKey)
		})
		t.Run("check instance count", func(t *testing.T) {
			if len(instances) != 3 {
				t.Error("expected 3 instances")
			}
		})

		historicInstances := []model.HistoricProcessInstance{}
		t.Run("get historic instances", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
		t.Run("check historic instance count", func(t *testing.T) {
			if len(historicInstances) != 3 {
				t.Error("expected 3 historicInstances")
			}
		})

		t.Run("stop instance 2", testDeleteInstances(config.ApiPort, networkId, &instances, 1))

		instances = []model.ProcessInstance{}
		t.Run("get instances after stop", testGetInstances(config.ApiPort, networkId, &instances))
		t.Run("check instance count after stop", func(t *testing.T) {
			if len(instances) != 2 {
				t.Error("expected 2 instances, got", len(instances))
				for _, v := range instances {
					t.Log(v.BusinessKey)
				}
			}
		})

		t.Run("delete deployment 3", testRemoveDeployment(config.ApiPort, networkId, &deployments, 2))

		deployments = []model.Deployment{}
		t.Run("get deployments after delete", testGetDeployments(config.ApiPort, networkId, &deployments))
		t.Run("check deployments count after delete", func(t *testing.T) {
			if len(deployments) != 2 {
				t.Error("expected 2 deployments, got", len(deployments))
			}
		})

		metadata := []model.DeploymentMetadata{}
		t.Run("get metadata", testGetDeploymentMetadata("http://localhost:"+config.ApiPort, networkId, &metadata))
		t.Run("check metadata count", func(t *testing.T) {
			if len(metadata) != 0 {
				t.Error("expected 0 metadata, got", len(metadata))
			}
		})

		instances = []model.ProcessInstance{}
		t.Run("get instances after delete", testGetInstances(config.ApiPort, networkId, &instances))
		t.Run("check instance count after delete", func(t *testing.T) {
			t.Skip("the deployment delete will not delete instances directly, but the warden and camunda will")
			if len(instances) != 1 {
				t.Error("expected 1 instances, got", len(instances))
				for _, v := range instances {
					t.Log(v.BusinessKey)
				}
			}
		})

		historicInstances = []model.HistoricProcessInstance{}
		t.Run("get historic instances after delete", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
		t.Run("check historic instance count after delete", func(t *testing.T) {
			t.Skip("the deployment delete will not delete instances directly, but the warden and camunda will")
			if len(historicInstances) != 1 {
				t.Error("expected 1 historicInstances, got", len(historicInstances))
			}
		})

	})

	t.Run("check backend state", func(t *testing.T) {
		time.Sleep(3 * wardenInterval)

		deployments := []model.Deployment{}
		t.Run("get deployments", testGetDeployments(config.ApiPort, networkId, &deployments))
		t.Run("check deployments count", func(t *testing.T) {
			if len(deployments) != 2 {
				t.Error("expected 2 deployments")
			}
		})

		metadata := []model.DeploymentMetadata{}
		t.Run("get metadata", testGetDeploymentMetadata("http://localhost:"+config.ApiPort, networkId, &metadata))
		t.Run("check metadata count", func(t *testing.T) {
			if len(metadata) != 0 {
				t.Error("expected 0 metadata, got", len(metadata))
			}
		})

		instances := []model.ProcessInstance{}
		t.Run("get instances", testGetInstances(config.ApiPort, networkId, &instances))
		t.Run("check instance count", func(t *testing.T) {
			if len(instances) != 1 {
				t.Error("expected 1 instances, got", len(instances))
				for _, v := range instances {
					t.Log(v.BusinessKey)
				}
			}
		})

		historicInstances := []model.HistoricProcessInstance{}
		t.Run("get historic instances", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
		t.Run("check historic instance count", func(t *testing.T) {
			if len(historicInstances) != 1 {
				t.Error("expected 1 historicInstances, got", len(historicInstances))
			}
		})
	})

	t.Run("start client", func(t *testing.T) {
		err = docker.MgwProcessSyncClient(ctx, wg, camundaDb, camundaUrl, config.Mqtt[0].Broker, "mgw-test-sync-client", networkId, clientMetadataStorageUrl)
		if err != nil {
			t.Error(err)
			return
		}
	})

	t.Run("check state after start", func(t *testing.T) {
		time.Sleep(3 * wardenInterval)

		t.Run("backend", func(t *testing.T) {
			deployments := []model.Deployment{}
			t.Run("get deployments", testGetDeployments(config.ApiPort, networkId, &deployments))
			t.Run("check deployments count", func(t *testing.T) {
				if len(deployments) != 2 {
					t.Error("expected 2 deployments, got", len(deployments))
				}
			})

			metadata := []model.DeploymentMetadata{}
			t.Run("get metadata", testGetDeploymentMetadata("http://localhost:"+config.ApiPort, networkId, &metadata))
			t.Run("check metadata count", func(t *testing.T) {
				if len(metadata) != 2 {
					t.Error("expected 2 metadata, got", len(metadata))
				}
			})

			instances := []model.ProcessInstance{}
			t.Run("get instances", testGetInstances(config.ApiPort, networkId, &instances))
			t.Run("check instance count", func(t *testing.T) {
				if len(instances) != 1 {
					t.Error("expected 1 instances, got", len(instances))
				}
			})

			historicInstances := []model.HistoricProcessInstance{}
			t.Run("get historic instances", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
			t.Run("check historic instance count", func(t *testing.T) {
				if len(historicInstances) != 1 {
					t.Error("expected 1 historicInstances, got", len(historicInstances))
				}
			})
		})

		t.Run("client", func(t *testing.T) {
			deployments := []model.Deployment{}
			t.Run("get deployments", testCamundaGetDeployments(camundaUrl, &deployments))
			t.Run("check deployments count", func(t *testing.T) {
				if len(deployments) != 2 {
					t.Error("expected 2 deployments, got", len(deployments))
				}
			})

			instances := []model.ProcessInstance{}
			t.Run("get instances", testCamundaGetInstances(camundaUrl, &instances))
			t.Run("check instance count", func(t *testing.T) {
				if len(instances) != 1 {
					t.Error("expected 1 instances, got", len(instances))
				}
			})

			historicInstances := []model.HistoricProcessInstance{}
			t.Run("get historic instances", testCamundaGetHistoricInstances(camundaUrl, &historicInstances))
			t.Run("check historic instance count", func(t *testing.T) {
				if len(historicInstances) != 1 {
					t.Error("expected 1 historicInstances, got", len(historicInstances))
				}
			})
		})
	})

	t.Run("client online", func(t *testing.T) {
		deployments := []model.Deployment{}
		t.Run("get deployments", testGetDeployments(config.ApiPort, networkId, &deployments))
		t.Run("check deployments count", func(t *testing.T) {
			if len(deployments) != 2 {
				t.Error("expected 2 deployments")
			}
		})
		t.Run("delete deployment 1", testRemoveDeployment(config.ApiPort, networkId, &deployments, 0))
		t.Run("delete deployment 2", testRemoveDeployment(config.ApiPort, networkId, &deployments, 1))
	})

	t.Run("check state", func(t *testing.T) {
		time.Sleep(3 * wardenInterval)

		t.Run("backend", func(t *testing.T) {
			deployments := []model.Deployment{}
			t.Run("get deployments", testGetDeployments(config.ApiPort, networkId, &deployments))
			t.Run("check deployments count", func(t *testing.T) {
				if len(deployments) != 0 {
					t.Error("expected 0 deployments")
				}
			})

			metadata := []model.DeploymentMetadata{}
			t.Run("get metadata", testGetDeploymentMetadata("http://localhost:"+config.ApiPort, networkId, &metadata))
			t.Run("check metadata count", func(t *testing.T) {
				if len(metadata) != 0 {
					t.Error("expected 0 metadata")
				}
			})

			instances := []model.ProcessInstance{}
			t.Run("get instances", testGetInstances(config.ApiPort, networkId, &instances))
			t.Run("check instance count", func(t *testing.T) {
				if len(instances) != 0 {
					t.Error("expected 0 instances")
				}
			})
		})

		t.Run("client", func(t *testing.T) {
			deployments := []model.Deployment{}
			t.Run("get deployments", testCamundaGetDeployments(camundaUrl, &deployments))
			t.Run("check deployments count", func(t *testing.T) {
				if len(deployments) != 0 {
					t.Error("expected 0 deployments")
				}
			})

			instances := []model.ProcessInstance{}
			t.Run("get instances", testCamundaGetInstances(camundaUrl, &instances))
			t.Run("check instance count", func(t *testing.T) {
				if len(instances) != 0 {
					t.Error("expected 0 instances")
				}
			})

			historicInstances := []model.HistoricProcessInstance{}
			t.Run("get historic instances", testCamundaGetHistoricInstances(camundaUrl, &historicInstances))
			t.Run("check historic instance count", func(t *testing.T) {
				if len(historicInstances) != 0 {
					t.Error("expected 0 historicInstances")
				}
			})
		})
	})

	t.Run("mqtt msg log", func(t *testing.T) {
		mqttMsgMux.Lock()
		defer mqttMsgMux.Unlock()
		fmt.Println("mqtt messages")
		for k, v := range mqttMessages {
			fmt.Println("--------------")
			fmt.Println(k)
			for _, vv := range v {
				fmt.Println(vv)
			}
		}
	})

}

func testGetDeploymentMetadata(syncUrl string, networkId string, metadata *[]model.DeploymentMetadata) func(t *testing.T) {
	return func(t *testing.T) {
		resp, err := http.Get(syncUrl + "/metadata/" + url.PathEscape(networkId))
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			pl, _ := io.ReadAll(resp.Body)
			err = fmt.Errorf("http error: %d %s", resp.StatusCode, string(pl))
			t.Error(err)
			return
		}
		err = json.NewDecoder(resp.Body).Decode(metadata)
		if err != nil {
			t.Error(err)
			return
		}
	}
}

func testCamundaGetHistoricInstances(camundaUrl string, instances *[]model.HistoricProcessInstance) func(t *testing.T) {
	return func(t *testing.T) {
		resp, err := http.Get(camundaUrl + "/engine-rest/history/process-instance")
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			pl, _ := io.ReadAll(resp.Body)
			err = fmt.Errorf("http error: %d %s", resp.StatusCode, string(pl))
			t.Error(err)
			return
		}
		err = json.NewDecoder(resp.Body).Decode(instances)
		if err != nil {
			t.Error(err)
			return
		}
	}
}

func testCamundaGetInstances(camundaUrl string, instances *[]model.ProcessInstance) func(t *testing.T) {
	return func(t *testing.T) {
		resp, err := http.Get(camundaUrl + "/engine-rest/process-instance")
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			pl, _ := io.ReadAll(resp.Body)
			err = fmt.Errorf("http error: %d %s", resp.StatusCode, string(pl))
			t.Error(err)
			return
		}
		err = json.NewDecoder(resp.Body).Decode(instances)
		if err != nil {
			t.Error(err)
			return
		}
	}
}

func testCamundaGetDeployments(camundaUrl string, deployments *[]model.Deployment) func(t *testing.T) {
	return func(t *testing.T) {
		resp, err := http.Get(camundaUrl + "/engine-rest/deployment")
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			pl, _ := io.ReadAll(resp.Body)
			err = fmt.Errorf("http error: %d %s", resp.StatusCode, string(pl))
			t.Error(err)
			return
		}
		err = json.NewDecoder(resp.Body).Decode(deployments)
		if err != nil {
			t.Error(err)
			return
		}
	}
}
