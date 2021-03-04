package mgw

import (
	"encoding/json"
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
	"github.com/SENERGY-Platform/process-sync/pkg/model/deploymentmodel"
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
	this.handler.UpdateDeployment(networkId, deployment)
}

func (this *Mgw) handleDeploymentDelete(message paho.Message) {
	networkId, err := this.getNetworkId(message.Topic())
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
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
	this.handler.DeleteUnknownDeployments(networkId, knownIds)
}

func (this *Mgw) SendDeploymentCommand(networkId string, deployment deploymentmodel.Deployment) error {
	return this.send(this.getCommandTopic(networkId, deploymentTopic), deployment)
}

func (this *Mgw) SendDeploymentDeleteCommand(networkId string, deploymentId string) error {
	return this.send(this.getCommandTopic(networkId, deploymentTopic, "delete"), deploymentId)
}

func (this *Mgw) SendDeploymentStartCommand(networkId string, deploymentId string) error {
	return this.send(this.getCommandTopic(networkId, deploymentTopic, "start"), deploymentId)
}
