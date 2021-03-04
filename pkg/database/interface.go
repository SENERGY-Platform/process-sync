package database

import "github.com/SENERGY-Platform/process-sync/pkg/model"

type Database interface {
	SaveDeployment(deployment model.Deployment) error
	RemoveDeployment(networkId string, deploymentId string) error
	RemovePlaceholderDeployments(networkId string) error
	RemoveUnknownDeployments(networkId string, knownIds []string) error
	ReadDeployment(networkId string, deploymentId string) (deployment model.Deployment, err error)
	ListDeployments(networkIds []string, limit int64, offset int64, sort string) (deployment []model.Deployment, err error)

	SaveHistoricProcessInstance(historicProcessInstance model.HistoricProcessInstance) error
	RemoveHistoricProcessInstance(networkId string, historicProcessInstanceId string) error
	RemoveUnknownHistoricProcessInstances(networkId string, knownIds []string) error
	ReadHistoricProcessInstance(networkId string, historicProcessInstanceId string) (historicProcessInstance model.HistoricProcessInstance, err error)
	ListHistoricProcessInstances(networkIds []string, limit int64, offset int64, sort string) (historicProcessInstance []model.HistoricProcessInstance, err error)

	SaveProcessInstance(processInstance model.ProcessInstance) error
	RemoveProcessInstance(networkId string, processInstanceId string) error
	RemovePlaceholderProcessInstances(networkId string) error
	RemoveUnknownProcessInstances(networkId string, knownIds []string) error
	ReadProcessInstance(networkId string, processInstanceId string) (processInstance model.ProcessInstance, err error)
	ListProcessInstances(networkIds []string, limit int64, offset int64, sort string) (processInstance []model.ProcessInstance, err error)

	SaveProcessDefinition(processDefinition model.ProcessDefinition) error
	RemoveProcessDefinition(networkId string, processDefinitionId string) error
	RemoveUnknownProcessDefinitions(networkId string, knownIds []string) error
	ReadProcessDefinition(networkId string, processDefinitionId string) (processDefinition model.ProcessDefinition, err error)
	ListProcessDefinitions(networkIds []string, limit int64, offset int64, sort string) (processDefinition []model.ProcessDefinition, err error)

	SaveIncident(incident model.Incident) error
	RemoveIncident(networkId string, incidentId string) error
	RemoveUnknownIncidents(networkId string, knownIds []string) error
	ReadIncident(networkId string, incidentId string) (incident model.Incident, err error)
	ListIncidents(networkIds []string, limit int64, offset int64, sort string) (incident []model.Incident, err error)
}
