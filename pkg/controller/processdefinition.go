package controller

import (
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
	"log"
	"runtime/debug"
)

func (this *Controller) UpdateProcessDefinition(networkId string, processDefinition camundamodel.ProcessDefinition) {
	err := this.db.SaveProcessDefinition(model.ProcessDefinition{
		ProcessDefinition: processDefinition,
		SyncInfo: model.SyncInfo{
			NetworkId:       networkId,
			IsPlaceholder:   false,
			MarkedForDelete: false,
			SyncDate:        configuration.TimeNow(),
		},
	})
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
}

func (this *Controller) DeleteProcessDefinition(networkId string, definitionId string) {
	err := this.db.RemoveProcessDefinition(networkId, definitionId)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
}

func (this *Controller) DeleteUnknownProcessDefinitions(networkId string, knownIds []string) {
	err := this.db.RemoveUnknownProcessDefinitions(networkId, knownIds)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
}
