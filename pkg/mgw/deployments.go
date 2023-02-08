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
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
	paho "github.com/eclipse/paho.mqtt.golang"
	"log"
	"runtime/debug"
)

func (this *Mgw) handleDeploymentUpdate(message paho.Message) {
	deployment := camundamodel.Deployment{}
	err := json.Unmarshal(message.Payload(), &deployment)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
	networkId, err := this.getNetworkId(message.Topic())
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
	this.handler.LogNetworkInteraction(networkId)
	this.handler.UpdateDeployment(networkId, deployment)
}

func (this *Mgw) handleDeploymentMetadata(message paho.Message) {
	metadata := model.Metadata{}
	err := json.Unmarshal(message.Payload(), &metadata)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
	networkId, err := this.getNetworkId(message.Topic())
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
	this.handler.LogNetworkInteraction(networkId)
	this.handler.UpdateDeploymentMetadata(networkId, metadata)
}

func (this *Mgw) handleDeploymentDelete(message paho.Message) {
	networkId, err := this.getNetworkId(message.Topic())
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
	this.handler.LogNetworkInteraction(networkId)
	this.handler.DeleteDeployment(networkId, string(message.Payload()))
}

func (this *Mgw) handleDeploymentKnown(message paho.Message) {
	knownIds := []string{}
	err := json.Unmarshal(message.Payload(), &knownIds)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
	networkId, err := this.getNetworkId(message.Topic())
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
	this.handler.LogNetworkInteraction(networkId)
	this.handler.DeleteUnknownDeployments(networkId, knownIds)
}

func (this *Mgw) SendDeploymentCommand(networkId string, deployment model.DeploymentWithEventDesc) error {
	return this.sendObj(this.getCommandTopic(networkId, deploymentTopic), deployment)
}

func (this *Mgw) SendDeploymentDeleteCommand(networkId string, deploymentId string) error {
	return this.sendStr(this.getCommandTopic(networkId, deploymentTopic, "delete"), deploymentId)
}

func (this *Mgw) SendDeploymentStartCommand(networkId string, deploymentId string, parameter map[string]interface{}) error {
	return this.sendObj(this.getCommandTopic(networkId, deploymentTopic, "start"), model.StartMessage{
		DeploymentId: deploymentId,
		Parameter:    parameter,
	})
}
