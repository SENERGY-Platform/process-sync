package model

import (
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
	"time"
)

type SyncInfo struct {
	NetworkId       string    `json:"network_id"`
	IsPlaceholder   bool      `json:"is_placeholder"`
	MarkedForDelete bool      `json:"marked_for_delete"`
	SyncDate        time.Time `json:"sync_date"`
}

type Deployment struct {
	camundamodel.Deployment
	SyncInfo
}

type HistoricProcessInstance struct {
	camundamodel.HistoricProcessInstance
	SyncInfo
}

type Incident struct {
	camundamodel.Incident
	SyncInfo
}

type ProcessDefinition struct {
	camundamodel.ProcessDefinition
	SyncInfo
}

type ProcessInstance struct {
	camundamodel.ProcessInstance
	SyncInfo
}
