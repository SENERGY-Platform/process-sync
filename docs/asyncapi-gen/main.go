/*
 * Copyright 2025 InfAI (CC SES)
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

package main

import (
	"flag"
	"fmt"
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/controller"
	"github.com/SENERGY-Platform/process-sync/pkg/mgw"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
	"log"
	"os"

	"github.com/swaggest/go-asyncapi/reflector/asyncapi-2.4.0"
	"github.com/swaggest/go-asyncapi/spec-2.4.0"
)

//go:generate go run main.go

func main() {
	configLocation := flag.String("config", "../../config.json", "configuration file")
	flag.Parse()

	conf, err := configuration.Load(*configLocation)
	if err != nil {
		log.Fatal("ERROR: unable to load config", err)
	}

	asyncAPI := spec.AsyncAPI{}
	asyncAPI.Info.Title = "Process-Sync"
	asyncAPI.Info.Description = "topics or parts of topics in '[]' are placeholders"

	asyncAPI.AddServer("kafka", spec.Server{
		URL:      conf.KafkaUrl,
		Protocol: "kafka",
	})

	asyncAPI.AddServer("mqtt", spec.Server{
		URL:         conf.MqttBroker,
		Protocol:    "mqtt",
		Description: "this service subscribes with a '$share/[group-id]/' topic prefix, to enable service scaling without duplicate message handling. this prefix is transparent to other mqtt clients.",
	})

	reflector := asyncapi.Reflector{}
	reflector.Schema = &asyncAPI

	mustNotFail := func(err error) {
		if err != nil {
			panic(err.Error())
		}
	}

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: conf.DeviceGroupTopic,
		BaseChannelItem: &spec.ChannelItem{
			Description: "topic may be configured by config.DeviceGroupTopic; messages will trigger updates to fog process deployments",
			Servers:     []string{"kafka"},
		},
		Publish: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "DeviceGroupCommand",
				Title: "DeviceGroupCommand",
			},
			MessageSample: new(controller.DeviceGroupCommand),
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: conf.ProcessDeploymentDoneTopic,
		BaseChannelItem: &spec.ChannelItem{
			Description: "topic may be configured by config.ProcessDeploymentDoneTopic; send when mgw has signaled finished process deployment",
			Servers:     []string{"kafka"},
		},
		Subscribe: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "DoneNotification",
				Title: "DoneNotification",
			},
			MessageSample: new(controller.DoneNotification),
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "processes/[network-id]/state/deployment",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"mqtt"},
			Description: "informs about deployed processes",
		},
		Publish: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "Deployment",
				Title: "Deployment",
			},
			MessageSample: new(camundamodel.Deployment),
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "processes/[network-id]/state/deployment/delete",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"mqtt"},
			Description: "signal that process deployment has been deleted at mgw",
		},
		Publish: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "DeleteProcessDeployment",
				Title: "DeleteProcessDeployment",
			},
			MessageSample: "process-deployment-id",
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "processes/[network-id]/state/deployment/known",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"mqtt"},
			Description: "informs about known deployments; resources on platform, that are not in this list will be deleted",
		},
		Publish: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "KnownDeploymentIds",
				Title: "KnownDeploymentIds",
			},
			MessageSample: []string{},
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "processes/[network-id]/state/deployment/metadata",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"mqtt"},
			Description: "informs about metadata of a process deployment",
		},
		Publish: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "Metadata",
				Title: "Metadata",
			},
			MessageSample: new(model.Metadata),
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "processes/[network-id]/state/process-instance",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"mqtt"},
			Description: "informs about a running process instance",
		},
		Publish: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "ProcessInstance",
				Title: "ProcessInstance",
			},
			MessageSample: new(camundamodel.ProcessInstance),
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "processes/[network-id]/state/process-instance/delete",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"mqtt"},
			Description: "signal that process instance has been deleted at mgw",
		},
		Publish: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "DeleteProcessInstance",
				Title: "DeleteProcess",
			},
			MessageSample: "process-instance-id",
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "processes/[network-id]/state/process-instance/known",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"mqtt"},
			Description: "informs about known process-instances; resources on platform, that are not in this list will be deleted",
		},
		Publish: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "KnownInstanceIds",
				Title: "KnownInstanceIds",
			},
			MessageSample: []string{},
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "processes/[network-id]/state/incident",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"mqtt"},
			Description: "informs about a new incident",
		},
		Publish: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "Incident",
				Title: "Incident",
			},
			MessageSample: new(camundamodel.Incident),
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "processes/[network-id]/state/incident/delete",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"mqtt"},
			Description: "signal that an incident has been deleted at mgw",
		},
		Publish: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "DeleteProcessIncident",
				Title: "DeleteProcessIncident",
			},
			MessageSample: "incident-id",
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "processes/[network-id]/state/incident/known",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"mqtt"},
			Description: "informs about known incidents; resources on platform, that are not in this list will be deleted",
		},
		Publish: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "KnownIncidentIds",
				Title: "KnownIncidentIds",
			},
			MessageSample: []string{},
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "processes/[network-id]/state/process-definition",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"mqtt"},
			Description: "informs about deployed process definitions",
		},
		Publish: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "ProcessDefinition",
				Title: "ProcessDefinition",
			},
			MessageSample: new(camundamodel.ProcessDefinition),
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "processes/[network-id]/state/process-definition/delete",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"mqtt"},
			Description: "signal that a process-definition has been deleted at mgw",
		},
		Publish: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "DeleteDefinitionId",
				Title: "DeleteDefinitionId",
			},
			MessageSample: "process-definition-id",
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "processes/[network-id]/state/process-definition/known",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"mqtt"},
			Description: "informs about known process-definitions; resources on platform, that are not in this list will be deleted",
		},
		Publish: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "KnownProcessDefinitionIds",
				Title: "KnownProcessDefinitionIds",
			},
			MessageSample: []string{},
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "processes/[network-id]/state/process-instance-history",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"mqtt"},
			Description: "informs about process instance history",
		},
		Publish: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "HistoricProcessInstance",
				Title: "HistoricProcessInstance",
			},
			MessageSample: new(camundamodel.HistoricProcessInstance),
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "processes/[network-id]/state/process-instance-history/delete",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"mqtt"},
			Description: "signal that process-instance-history has been deleted at mgw",
		},
		Publish: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "DeleteHistoryId",
				Title: "DeleteHistoryId",
			},
			MessageSample: "history-id",
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "processes/[network-id]/state/process-instance-history/known",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"mqtt"},
			Description: "informs about known process-instance-history; resources on platform, that are not in this list will be deleted",
		},
		Publish: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "KnownHistoryIds",
				Title: "KnownHistoryIds",
			},
			MessageSample: []string{},
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "processes/[network-id]/cmd/deployment",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"mqtt"},
			Description: "send process deployment to mgw",
		},
		Subscribe: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "DeploymentWithEventDesc",
				Title: "DeploymentWithEventDesc",
			},
			MessageSample: new(model.DeploymentWithEventDesc),
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "processes/[network-id]/cmd/deployment/event-descriptions",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"mqtt"},
			Description: "send event descriptions to mgw",
		},
		Subscribe: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "EventDescriptionsUpdate",
				Title: "EventDescriptionsUpdate",
			},
			MessageSample: new(mgw.EventDescriptionsUpdate),
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "processes/[network-id]/cmd/deployment/start",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"mqtt"},
			Description: "send process start request to a mgw",
		},
		Subscribe: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "StartMessage",
				Title: "StartMessage",
			},
			MessageSample: new(model.StartMessage),
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "processes/[network-id]/cmd/deployment/delete",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"mqtt"},
			Description: "send process deployment delete request to a mgw",
		},
		Subscribe: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "DeploymentId",
				Title: "DeploymentId",
			},
			MessageSample: "deployment-id",
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "processes/[network-id]/cmd/process-instance-history/delete",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"mqtt"},
			Description: "send process-instance-history delete request to a mgw",
		},
		Subscribe: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "HistoryId",
				Title: "HistoryId",
			},
			MessageSample: "instance-history-id",
		},
	}))

	mustNotFail(reflector.AddChannel(asyncapi.ChannelInfo{
		Name: "processes/[network-id]/cmd/process-instance/delete",
		BaseChannelItem: &spec.ChannelItem{
			Servers:     []string{"mqtt"},
			Description: "send process-instance delete request to a mgw",
		},
		Subscribe: &asyncapi.MessageSample{
			MessageEntity: spec.MessageEntity{
				Name:  "InstanceId",
				Title: "InstanceId",
			},
			MessageSample: "instance-id",
		},
	}))

	buff, err := reflector.Schema.MarshalJSON()
	mustNotFail(err)

	fmt.Println(string(buff))
	mustNotFail(os.WriteFile("asyncapi.json", buff, 0o600))
}
