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

package mgw

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
	"github.com/SENERGY-Platform/process-sync/pkg/multimqtt"
	paho "github.com/eclipse/paho.mqtt.golang"
)

type Mgw struct {
	mqtt    paho.Client
	config  configuration.Config
	handler Handler
}

type Handler interface {
	UpdateDeployment(networkId string, deployment camundamodel.Deployment)
	DeleteDeployment(networkId string, deploymentId string)
	DeleteUnknownDeployments(networkId string, knownIds []string)
	UpdateIncident(networkId string, incident camundamodel.Incident)
	DeleteIncident(networkId string, incidentId string)
	DeleteUnknownIncidents(networkId string, knownIds []string)
	UpdateHistoricProcessInstance(networkId string, historicProcessInstance camundamodel.HistoricProcessInstance)
	DeleteHistoricProcessInstance(networkId string, historicInstanceId string)
	DeleteUnknownHistoricProcessInstances(networkId string, knownIds []string)
	UpdateProcessDefinition(networkId string, processDefinition camundamodel.ProcessDefinition)
	DeleteProcessDefinition(networkId string, definitionId string)
	DeleteUnknownProcessDefinitions(networkId string, knownIds []string)
	UpdateProcessInstance(networkId string, instance camundamodel.ProcessInstance)
	DeleteProcessInstance(networkId string, instanceId string)
	DeleteUnknownProcessInstances(networkId string, knownIds []string)
	UpdateDeploymentMetadata(networkId string, metadata model.Metadata)
	LogNetworkInteraction(networkId string)
}

func New(config configuration.Config, ctx context.Context, handler Handler) (*Mgw, error) {
	client := &Mgw{
		config:  config,
		handler: handler,
	}

	client.mqtt = multimqtt.NewClient(config.Mqtt, func(options *paho.ClientOptions) {
		options.SetResumeSubs(true).
			SetCleanSession(config.MqttCleanSession).
			SetConnectionLostHandler(func(c paho.Client, err error) {
				o := c.OptionsReader()
				config.GetLogger().Error("connection to mqtt broker lost", "error", err, "client", o.ClientID())
			}).
			SetOnConnectHandler(func(c paho.Client) {
				o := c.OptionsReader()
				config.GetLogger().Info("connected to mqtt broker", "client", o.ClientID())
				client.subscribe(c)
			})
	})
	if token := client.mqtt.Connect(); token.Wait() && token.Error() != nil {
		config.GetLogger().Error("unable to connect to mqtt broker", "error", token.Error())
		return nil, token.Error()
	}

	go func() {
		<-ctx.Done()
		client.mqtt.Disconnect(0)
	}()

	return client, nil
}

const deploymentTopic = "deployment"
const incidentTopic = "incident"
const processDefinitionTopic = "process-definition"
const processInstanceTopic = "process-instance"
const processInstanceHistoryTopic = "process-instance-history"

func (this *Mgw) subscribe(client paho.Client) {
	sharedSubscriptionPrefix := ""
	if this.config.MqttGroupId != "" {
		sharedSubscriptionPrefix = "$share/" + this.config.MqttGroupId + "/"
	}

	client.Subscribe(sharedSubscriptionPrefix+this.getStateTopic("+", deploymentTopic), 2, func(client paho.Client, message paho.Message) {
		this.config.GetLogger().Debug("receive", "topic", message.Topic(), "payload", string(message.Payload()))
		this.handleDeploymentUpdate(message)
	})
	client.Subscribe(sharedSubscriptionPrefix+this.getStateTopic("+", deploymentTopic, "delete"), 2, func(client paho.Client, message paho.Message) {
		this.config.GetLogger().Debug("receive", "topic", message.Topic(), "payload", string(message.Payload()))
		this.handleDeploymentDelete(message)
	})
	client.Subscribe(sharedSubscriptionPrefix+this.getStateTopic("+", deploymentTopic, "known"), 2, func(client paho.Client, message paho.Message) {
		this.config.GetLogger().Debug("receive", "topic", message.Topic(), "payload", string(message.Payload()))
		this.handleDeploymentKnown(message)
	})
	client.Subscribe(sharedSubscriptionPrefix+this.getStateTopic("+", deploymentTopic, "metadata"), 2, func(client paho.Client, message paho.Message) {
		this.config.GetLogger().Debug("receive", "topic", message.Topic(), "payload", string(message.Payload()))
		this.handleDeploymentMetadata(message)
	})
	client.Subscribe(sharedSubscriptionPrefix+this.getStateTopic("+", incidentTopic), 2, func(client paho.Client, message paho.Message) {
		this.config.GetLogger().Debug("receive", "topic", message.Topic(), "payload", string(message.Payload()))
		this.handleIncidentUpdate(message)
	})
	client.Subscribe(sharedSubscriptionPrefix+this.getStateTopic("+", incidentTopic, "delete"), 2, func(client paho.Client, message paho.Message) {
		this.config.GetLogger().Debug("receive", "topic", message.Topic(), "payload", string(message.Payload()))
		this.handleIncidentDelete(message)
	})
	client.Subscribe(sharedSubscriptionPrefix+this.getStateTopic("+", incidentTopic, "known"), 2, func(client paho.Client, message paho.Message) {
		this.config.GetLogger().Debug("receive", "topic", message.Topic(), "payload", string(message.Payload()))
		this.handleIncidentKnown(message)
	})
	client.Subscribe(sharedSubscriptionPrefix+this.getStateTopic("+", processDefinitionTopic), 2, func(client paho.Client, message paho.Message) {
		this.config.GetLogger().Debug("receive", "topic", message.Topic(), "payload", string(message.Payload()))
		this.handleProcessDefinitionUpdate(message)
	})
	client.Subscribe(sharedSubscriptionPrefix+this.getStateTopic("+", processDefinitionTopic, "delete"), 2, func(client paho.Client, message paho.Message) {
		this.config.GetLogger().Debug("receive", "topic", message.Topic(), "payload", string(message.Payload()))
		this.handleProcessDefinitionDelete(message)
	})
	client.Subscribe(sharedSubscriptionPrefix+this.getStateTopic("+", processDefinitionTopic, "known"), 2, func(client paho.Client, message paho.Message) {
		this.config.GetLogger().Debug("receive", "topic", message.Topic(), "payload", string(message.Payload()))
		this.handleProcessDefinitionKnown(message)
	})
	client.Subscribe(sharedSubscriptionPrefix+this.getStateTopic("+", processInstanceTopic), 2, func(client paho.Client, message paho.Message) {
		this.config.GetLogger().Debug("receive", "topic", message.Topic(), "payload", string(message.Payload()))
		this.handleProcessInstanceUpdate(message)
	})
	client.Subscribe(sharedSubscriptionPrefix+this.getStateTopic("+", processInstanceTopic, "delete"), 2, func(client paho.Client, message paho.Message) {
		this.config.GetLogger().Debug("receive", "topic", message.Topic(), "payload", string(message.Payload()))
		this.handleProcessInstanceDelete(message)
	})
	client.Subscribe(sharedSubscriptionPrefix+this.getStateTopic("+", processInstanceTopic, "known"), 2, func(client paho.Client, message paho.Message) {
		this.config.GetLogger().Debug("receive", "topic", message.Topic(), "payload", string(message.Payload()))
		this.handleProcessInstanceKnown(message)
	})
	client.Subscribe(sharedSubscriptionPrefix+this.getStateTopic("+", processInstanceHistoryTopic), 2, func(client paho.Client, message paho.Message) {
		this.config.GetLogger().Debug("receive", "topic", message.Topic(), "payload", string(message.Payload()))
		this.handleHistoricProcessInstanceUpdate(message)
	})
	client.Subscribe(sharedSubscriptionPrefix+this.getStateTopic("+", processInstanceHistoryTopic, "delete"), 2, func(client paho.Client, message paho.Message) {
		this.config.GetLogger().Debug("receive", "topic", message.Topic(), "payload", string(message.Payload()))
		this.handleHistoricProcessInstanceDelete(message)
	})
	client.Subscribe(sharedSubscriptionPrefix+this.getStateTopic("+", processInstanceHistoryTopic, "known"), 2, func(client paho.Client, message paho.Message) {
		this.config.GetLogger().Debug("receive", "topic", message.Topic(), "payload", string(message.Payload()))
		this.handleHistoricProcessInstanceKnown(message)
	})
}

func (this *Mgw) getNetworkId(topic string) (networkId string, err error) {
	parts := strings.Split(topic, "/")
	if len(parts) < 2 {
		return "", errors.New("expect topic to have at least 2 levels (" + topic + ")")
	}
	if parts[0] != "processes" {
		return "", errors.New("expect 'processes' as top level topic (" + topic + ")")
	}
	return parts[1], nil
}

func (this *Mgw) getBaseTopic(networkId string) string {
	return "processes/" + networkId
}

func (this *Mgw) getCommandTopic(networkId string, entity string, subcommand ...string) (topic string) {
	topic = this.getBaseTopic(networkId) + "/cmd/" + entity
	for _, sub := range subcommand {
		topic = topic + "/" + sub
	}
	return
}

func (this *Mgw) getStateTopic(networkId string, entity string, substate ...string) (topic string) {
	topic = this.getBaseTopic(networkId) + "/state/" + entity
	for _, sub := range substate {
		topic = topic + "/" + sub
	}
	return
}

func (this *Mgw) sendObj(topic string, message interface{}) error {
	msg, err := json.Marshal(message)
	if err != nil {
		return err
	}
	this.config.GetLogger().Debug("send", "topic", topic, "payload", string(msg))
	token := this.mqtt.Publish(topic, 2, false, msg)
	token.Wait()
	return token.Error()
}

func (this *Mgw) sendStr(topic string, message string) error {
	this.config.GetLogger().Debug("send", "topic", topic, "payload", message)
	token := this.mqtt.Publish(topic, 2, false, message)
	token.Wait()
	return token.Error()
}
