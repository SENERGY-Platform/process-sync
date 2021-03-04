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
	this.handler.UpdateHistoricProcessInstance(networkId, historicProcessInstance)
}

func (this *Mgw) handleHistoricProcessInstanceDelete(message paho.Message) {
	networkId, err := this.getNetworkId(message.Topic())
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
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
	this.handler.DeleteUnknownHistoricProcessInstances(networkId, knownIds)
}

func (this *Mgw) SendProcessHistoryDeleteCommand(networkId string, processInstanceHistoryId string) error {
	return this.send(this.getCommandTopic(networkId, processInstanceHistoryTopic, "delete"), processInstanceHistoryId)
}
