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
	"encoding/json"
	"runtime/debug"

	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
	paho "github.com/eclipse/paho.mqtt.golang"
)

func (this *Mgw) handleProcessInstanceUpdate(message paho.Message) {
	processInstance := camundamodel.ProcessInstance{}
	err := json.Unmarshal(message.Payload(), &processInstance)
	if err != nil {
		this.config.GetLogger().Error("error", "error", err, "stack", debug.Stack())
	}
	networkId, err := this.getNetworkId(message.Topic())
	if err != nil {
		this.config.GetLogger().Error("error", "error", err, "stack", debug.Stack())
	}
	this.handler.LogNetworkInteraction(networkId)
	this.handler.UpdateProcessInstance(networkId, processInstance)
}

func (this *Mgw) handleProcessInstanceDelete(message paho.Message) {
	networkId, err := this.getNetworkId(message.Topic())
	if err != nil {
		this.config.GetLogger().Error("error", "error", err, "stack", debug.Stack())
	}
	this.handler.LogNetworkInteraction(networkId)
	this.handler.DeleteProcessInstance(networkId, string(message.Payload()))
}

func (this *Mgw) handleProcessInstanceKnown(message paho.Message) {
	knownIds := []string{}
	err := json.Unmarshal(message.Payload(), &knownIds)
	if err != nil {
		this.config.GetLogger().Error("error", "error", err, "stack", debug.Stack())
	}
	networkId, err := this.getNetworkId(message.Topic())
	if err != nil {
		this.config.GetLogger().Error("error", "error", err, "stack", debug.Stack())
	}
	this.handler.LogNetworkInteraction(networkId)
	this.handler.DeleteUnknownProcessInstances(networkId, knownIds)
}

func (this *Mgw) SendProcessStopCommand(networkId string, processInstanceId string) error {
	return this.sendStr(this.getCommandTopic(networkId, processInstanceTopic, "delete"), processInstanceId)
}
