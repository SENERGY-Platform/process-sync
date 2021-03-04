package mgw

import (
	"encoding/json"
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
	paho "github.com/eclipse/paho.mqtt.golang"
	"log"
	"runtime/debug"
)

func (this *Mgw) handleProcessInstanceUpdate(message paho.Message) {
	processInstance := camundamodel.ProcessInstance{}
	err := json.Unmarshal(message.Payload(), &processInstance)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
	networkId, err := this.getNetworkId(message.Topic())
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
	this.handler.UpdateProcessInstance(networkId, processInstance)
}

func (this *Mgw) handleProcessInstanceDelete(message paho.Message) {
	networkId, err := this.getNetworkId(message.Topic())
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
	this.handler.DeleteProcessInstance(networkId, string(message.Payload()))
}

func (this *Mgw) handleProcessInstanceKnown(message paho.Message) {
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
	this.handler.DeleteUnknownProcessInstances(networkId, knownIds)
}

func (this *Mgw) SendProcessStopCommand(networkId string, processInstanceId string) error {
	return this.send(this.getCommandTopic(networkId, processInstanceTopic, "delete"), processInstanceId)
}
