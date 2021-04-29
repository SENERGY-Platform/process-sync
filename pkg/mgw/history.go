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
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
	paho "github.com/eclipse/paho.mqtt.golang"
	"log"
	"runtime/debug"
)

func (this *Mgw) handleHistoricProcessInstanceUpdate(message paho.Message) {
	historicProcessInstance := camundamodel.HistoricProcessInstance{}
	err := json.Unmarshal(message.Payload(), &historicProcessInstance)
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
	this.handler.UpdateHistoricProcessInstance(networkId, historicProcessInstance)
}

func (this *Mgw) handleHistoricProcessInstanceDelete(message paho.Message) {
	networkId, err := this.getNetworkId(message.Topic())
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
	this.handler.LogNetworkInteraction(networkId)
	this.handler.DeleteHistoricProcessInstance(networkId, string(message.Payload()))
}

func (this *Mgw) handleHistoricProcessInstanceKnown(message paho.Message) {
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
	this.handler.DeleteUnknownHistoricProcessInstances(networkId, knownIds)
}

func (this *Mgw) SendProcessHistoryDeleteCommand(networkId string, processInstanceHistoryId string) error {
	return this.sendStr(this.getCommandTopic(networkId, processInstanceHistoryTopic, "delete"), processInstanceHistoryId)
}
