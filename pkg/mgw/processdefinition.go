package mgw

import (
	"encoding/json"
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
	paho "github.com/eclipse/paho.mqtt.golang"
	"log"
	"runtime/debug"
)

func (this *Mgw) handleProcessDefinitionUpdate(message paho.Message) {
	processDefinition := camundamodel.ProcessDefinition{}
	err := json.Unmarshal(message.Payload(), &processDefinition)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
	networkId, err := this.getNetworkId(message.Topic())
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
	this.handler.UpdateProcessDefinition(networkId, processDefinition)
}

func (this *Mgw) handleProcessDefinitionDelete(message paho.Message) {
	networkId, err := this.getNetworkId(message.Topic())
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
	this.handler.DeleteProcessDefinition(networkId, string(message.Payload()))
}

func (this *Mgw) handleProcessDefinitionKnown(message paho.Message) {
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
	this.handler.DeleteUnknownProcessDefinitions(networkId, knownIds)
}
