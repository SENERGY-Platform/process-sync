package controller

import (
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
	"log"
	"runtime/debug"
)

func (this *Controller) UpdateHistoricProcessInstance(networkId string, historicProcessInstance camundamodel.HistoricProcessInstance) {
	err := this.db.SaveHistoricProcessInstance(model.HistoricProcessInstance{
		HistoricProcessInstance: historicProcessInstance,
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

func (this *Controller) DeleteHistoricProcessInstance(networkId string, historicInstanceId string) {
	err := this.db.RemoveHistoricProcessInstance(networkId, historicInstanceId)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
}

func (this *Controller) DeleteUnknownHistoricProcessInstances(networkId string, knownIds []string) {
	err := this.db.RemoveUnknownHistoricProcessInstances(networkId, knownIds)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
}
