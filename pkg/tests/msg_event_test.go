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

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/SENERGY-Platform/process-deployment/lib/model/deploymentmodel"
	"github.com/SENERGY-Platform/process-deployment/lib/model/devicemodel"
	"github.com/SENERGY-Platform/process-deployment/lib/model/deviceselectionmodel"
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/tests/server"
	paho "github.com/eclipse/paho.mqtt.golang"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestMsgEvents(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := configuration.Config{
		Debug:                             true,
		MqttClientId:                      "",
		MqttCleanSession:                  true,
		MqttGroupId:                       "",
		MongoTable:                        "processes",
		MongoProcessDefinitionCollection:  "process_definition",
		MongoDeploymentCollection:         "deployments",
		MongoProcessHistoryCollection:     "histories",
		MongoIncidentCollection:           "incidents",
		MongoProcessInstanceCollection:    "instances",
		MongoDeploymentMetadataCollection: "deployment_metadata",
		MongoLastNetworkContactCollection: "last_network_collection",
	}

	networkId := "test-network-id"
	config, err := server.EnvForEventsCheck(ctx, wg, config, networkId)
	if err != nil {
		t.Error(err)
		return
	}

	options := paho.NewClientOptions().
		SetPassword(config.MqttPw).
		SetUsername(config.MqttUser).
		SetAutoReconnect(true).
		AddBroker(config.MqttBroker)

	mqtt := paho.NewClient(options)
	if token := mqtt.Connect(); token.Wait() && token.Error() != nil {
		t.Error(err)
		return
	}

	messages := []string{}
	mqtt.Subscribe("processes/"+networkId+"/cmd/deployment", 2, func(client paho.Client, message paho.Message) {
		messages = append(messages, string(message.Payload()))
	})

	t.Run("deploy process", testDeployEventProcess(config.ApiPort, networkId))

	time.Sleep(1 * time.Second)

	t.Run("check mqtt deployment message", func(t *testing.T) {
		expected := `{"version":3,"id":"test-id","name":"test-deployment-name","description":"test-description","diagram":{"xml_raw":"\u003c?xml version=\"1.0\" encoding=\"UTF-8\"?\u003e\n\u003cbpmn:definitions xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xmlns:bpmn=\"http://www.omg.org/spec/BPMN/20100524/MODEL\" xmlns:bpmndi=\"http://www.omg.org/spec/BPMN/20100524/DI\" xmlns:dc=\"http://www.omg.org/spec/DD/20100524/DC\" xmlns:camunda=\"http://camunda.org/schema/1.0/bpmn\" xmlns:senergy=\"https://senergy.infai.org\" xmlns:di=\"http://www.omg.org/spec/DD/20100524/DI\" id=\"Definitions_1\" targetNamespace=\"http://bpmn.io/schema/bpmn\"\u003e\n    \u003cbpmn:process id=\"ExampleId\" name=\"ExampleName\" isExecutable=\"true\" senergy:description=\"ExampleDesc\"\u003e\n        \u003cbpmn:startEvent id=\"StartEvent_1\"\u003e\n            \u003cbpmn:outgoing\u003eSequenceFlow_0qjn3dq\u003c/bpmn:outgoing\u003e\n        \u003c/bpmn:startEvent\u003e\n        \u003cbpmn:sequenceFlow id=\"SequenceFlow_0qjn3dq\" sourceRef=\"StartEvent_1\" targetRef=\"Task_1nbnl8y\"/\u003e\n        \u003cbpmn:sequenceFlow id=\"SequenceFlow_15v8030\" sourceRef=\"Task_1nbnl8y\" targetRef=\"Task_1lhzy95\"/\u003e\n        \u003cbpmn:endEvent id=\"EndEvent_1vhaxdr\"\u003e\n            \u003cbpmn:incoming\u003eSequenceFlow_17lypcn\u003c/bpmn:incoming\u003e\n        \u003c/bpmn:endEvent\u003e\n        \u003cbpmn:sequenceFlow id=\"SequenceFlow_17lypcn\" sourceRef=\"Task_1lhzy95\" targetRef=\"EndEvent_1vhaxdr\"/\u003e\n        \u003cbpmn:serviceTask id=\"Task_1nbnl8y\" name=\"Lighting getColorFunction\" camunda:type=\"external\" camunda:topic=\"pessimistic\"\u003e\n            \u003cbpmn:extensionElements\u003e\n                \u003ccamunda:inputOutput\u003e\n                    \u003ccamunda:inputParameter name=\"payload\"\u003e\u003c![CDATA[{\n\t\"function\": {\n\t\t\"id\": \"urn:infai:ses:measuring-function:bdb6a7c8-4a3d-4fe0-bab3-ce02e09b5869\",\n\t\t\"name\": \"\",\n\t\t\"concept_id\": \"\",\n\t\t\"rdf_type\": \"\"\n\t},\n\t\"characteristic_id\": \"urn:infai:ses:characteristic:5b4eea52-e8e5-4e80-9455-0382f81a1b43\",\n\t\"aspect\": {\n\t\t\"id\": \"urn:infai:ses:aspect:a7470d73-dde3-41fc-92bd-f16bb28f2da6\",\n\t\t\"name\": \"\",\n\t\t\"rdf_type\": \"\"\n\t},\n\t\"device_id\": \"hue\",\n\t\"service_id\": \"urn:infai:ses:service:99614933-4734-41b6-a131-3f96f134ee69\",\n\t\"input\": {},\n\t\"retries\": 2\n}]]\u003e\u003c/camunda:inputParameter\u003e\n                    \u003ccamunda:outputParameter name=\"outputs.b\"\u003e${result.b}\u003c/camunda:outputParameter\u003e\n                    \u003ccamunda:outputParameter name=\"outputs.g\"\u003e${result.g}\u003c/camunda:outputParameter\u003e\n                    \u003ccamunda:outputParameter name=\"outputs.r\"\u003e${result.r}\u003c/camunda:outputParameter\u003e\n                \u003c/camunda:inputOutput\u003e\n            \u003c/bpmn:extensionElements\u003e\n            \u003cbpmn:incoming\u003eSequenceFlow_0qjn3dq\u003c/bpmn:incoming\u003e\n            \u003cbpmn:outgoing\u003eSequenceFlow_15v8030\u003c/bpmn:outgoing\u003e\n        \u003c/bpmn:serviceTask\u003e\n        \u003cbpmn:serviceTask id=\"Task_1lhzy95\" name=\"Lamp setColorFunction\" camunda:type=\"external\" camunda:topic=\"optimistic\"\u003e\n            \u003cbpmn:extensionElements\u003e\n                \u003ccamunda:inputOutput\u003e\n                    \u003ccamunda:inputParameter name=\"payload\"\u003e\u003c![CDATA[{\n\t\"function\": {\n\t\t\"id\": \"urn:infai:ses:controlling-function:c54e2a89-1fb8-4ecb-8993-a7b40b355599\",\n\t\t\"name\": \"\",\n\t\t\"concept_id\": \"\",\n\t\t\"rdf_type\": \"\"\n\t},\n\t\"characteristic_id\": \"urn:infai:ses:characteristic:5b4eea52-e8e5-4e80-9455-0382f81a1b43\",\n\t\"device_class\": {\n\t\t\"id\": \"urn:infai:ses:device-class:14e56881-16f9-4120-bb41-270a43070c86\",\n\t\t\"name\": \"\",\n\t\t\"image\": \"\",\n\t\t\"rdf_type\": \"\"\n\t},\n\t\"device_id\": \"hue\",\n\t\"service_id\": \"urn:infai:ses:service:67789396-d1ca-4ea9-9147-0614c6d68a2f\",\n\t\"input\": {\n\t\t\"b\": 0,\n\t\t\"g\": 0,\n\t\t\"r\": 0\n\t}\n}]]\u003e\u003c/camunda:inputParameter\u003e\n                    \u003ccamunda:inputParameter name=\"inputs.b\"\u003e0\u003c/camunda:inputParameter\u003e\n                    \u003ccamunda:inputParameter name=\"inputs.g\"\u003e255\u003c/camunda:inputParameter\u003e\n                    \u003ccamunda:inputParameter name=\"inputs.r\"\u003e100\u003c/camunda:inputParameter\u003e\n                \u003c/camunda:inputOutput\u003e\n            \u003c/bpmn:extensionElements\u003e\n            \u003cbpmn:incoming\u003eSequenceFlow_15v8030\u003c/bpmn:incoming\u003e\n            \u003cbpmn:outgoing\u003eSequenceFlow_17lypcn\u003c/bpmn:outgoing\u003e\n        \u003c/bpmn:serviceTask\u003e\n    \u003c/bpmn:process\u003e\n\u003c/bpmn:definitions\u003e","xml_deployed":"\u003c?xml version=\"1.0\" encoding=\"UTF-8\"?\u003e\n\u003cbpmn:definitions xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xmlns:bpmn=\"http://www.omg.org/spec/BPMN/20100524/MODEL\" xmlns:bpmndi=\"http://www.omg.org/spec/BPMN/20100524/DI\" xmlns:dc=\"http://www.omg.org/spec/DD/20100524/DC\" xmlns:camunda=\"http://camunda.org/schema/1.0/bpmn\" xmlns:senergy=\"https://senergy.infai.org\" xmlns:di=\"http://www.omg.org/spec/DD/20100524/DI\" id=\"Definitions_1\" targetNamespace=\"http://bpmn.io/schema/bpmn\"\u003e\n    \u003cbpmn:process id=\"ExampleId\" name=\"ExampleName\" isExecutable=\"true\" senergy:description=\"ExampleDesc\"\u003e\n        \u003cbpmn:startEvent id=\"StartEvent_1\"\u003e\n            \u003cbpmn:outgoing\u003eSequenceFlow_0qjn3dq\u003c/bpmn:outgoing\u003e\n        \u003c/bpmn:startEvent\u003e\n        \u003cbpmn:sequenceFlow id=\"SequenceFlow_0qjn3dq\" sourceRef=\"StartEvent_1\" targetRef=\"Task_1nbnl8y\"/\u003e\n        \u003cbpmn:sequenceFlow id=\"SequenceFlow_15v8030\" sourceRef=\"Task_1nbnl8y\" targetRef=\"Task_1lhzy95\"/\u003e\n        \u003cbpmn:endEvent id=\"EndEvent_1vhaxdr\"\u003e\n            \u003cbpmn:incoming\u003eSequenceFlow_17lypcn\u003c/bpmn:incoming\u003e\n        \u003c/bpmn:endEvent\u003e\n        \u003cbpmn:sequenceFlow id=\"SequenceFlow_17lypcn\" sourceRef=\"Task_1lhzy95\" targetRef=\"EndEvent_1vhaxdr\"/\u003e\n        \u003cbpmn:serviceTask id=\"Task_1nbnl8y\" name=\"Lighting getColorFunction\" camunda:type=\"external\" camunda:topic=\"pessimistic\"\u003e\n            \u003cbpmn:extensionElements\u003e\n                \u003ccamunda:inputOutput\u003e\n                    \u003ccamunda:inputParameter name=\"payload\"\u003e\u003c![CDATA[{\n\t\"function\": {\n\t\t\"id\": \"urn:infai:ses:measuring-function:bdb6a7c8-4a3d-4fe0-bab3-ce02e09b5869\",\n\t\t\"name\": \"\",\n\t\t\"concept_id\": \"\",\n\t\t\"rdf_type\": \"\"\n\t},\n\t\"characteristic_id\": \"urn:infai:ses:characteristic:5b4eea52-e8e5-4e80-9455-0382f81a1b43\",\n\t\"aspect\": {\n\t\t\"id\": \"urn:infai:ses:aspect:a7470d73-dde3-41fc-92bd-f16bb28f2da6\",\n\t\t\"name\": \"\",\n\t\t\"rdf_type\": \"\"\n\t},\n\t\"device_id\": \"hue\",\n\t\"service_id\": \"urn:infai:ses:service:99614933-4734-41b6-a131-3f96f134ee69\",\n\t\"input\": {},\n\t\"retries\": 2\n}]]\u003e\u003c/camunda:inputParameter\u003e\n                    \u003ccamunda:outputParameter name=\"outputs.b\"\u003e${result.b}\u003c/camunda:outputParameter\u003e\n                    \u003ccamunda:outputParameter name=\"outputs.g\"\u003e${result.g}\u003c/camunda:outputParameter\u003e\n                    \u003ccamunda:outputParameter name=\"outputs.r\"\u003e${result.r}\u003c/camunda:outputParameter\u003e\n                \u003c/camunda:inputOutput\u003e\n            \u003c/bpmn:extensionElements\u003e\n            \u003cbpmn:incoming\u003eSequenceFlow_0qjn3dq\u003c/bpmn:incoming\u003e\n            \u003cbpmn:outgoing\u003eSequenceFlow_15v8030\u003c/bpmn:outgoing\u003e\n        \u003c/bpmn:serviceTask\u003e\n        \u003cbpmn:serviceTask id=\"Task_1lhzy95\" name=\"Lamp setColorFunction\" camunda:type=\"external\" camunda:topic=\"optimistic\"\u003e\n            \u003cbpmn:extensionElements\u003e\n                \u003ccamunda:inputOutput\u003e\n                    \u003ccamunda:inputParameter name=\"payload\"\u003e\u003c![CDATA[{\n\t\"function\": {\n\t\t\"id\": \"urn:infai:ses:controlling-function:c54e2a89-1fb8-4ecb-8993-a7b40b355599\",\n\t\t\"name\": \"\",\n\t\t\"concept_id\": \"\",\n\t\t\"rdf_type\": \"\"\n\t},\n\t\"characteristic_id\": \"urn:infai:ses:characteristic:5b4eea52-e8e5-4e80-9455-0382f81a1b43\",\n\t\"device_class\": {\n\t\t\"id\": \"urn:infai:ses:device-class:14e56881-16f9-4120-bb41-270a43070c86\",\n\t\t\"name\": \"\",\n\t\t\"image\": \"\",\n\t\t\"rdf_type\": \"\"\n\t},\n\t\"device_id\": \"hue\",\n\t\"service_id\": \"urn:infai:ses:service:67789396-d1ca-4ea9-9147-0614c6d68a2f\",\n\t\"input\": {\n\t\t\"b\": 0,\n\t\t\"g\": 0,\n\t\t\"r\": 0\n\t}\n}]]\u003e\u003c/camunda:inputParameter\u003e\n                    \u003ccamunda:inputParameter name=\"inputs.b\"\u003e0\u003c/camunda:inputParameter\u003e\n                    \u003ccamunda:inputParameter name=\"inputs.g\"\u003e255\u003c/camunda:inputParameter\u003e\n                    \u003ccamunda:inputParameter name=\"inputs.r\"\u003e100\u003c/camunda:inputParameter\u003e\n                \u003c/camunda:inputOutput\u003e\n            \u003c/bpmn:extensionElements\u003e\n            \u003cbpmn:incoming\u003eSequenceFlow_15v8030\u003c/bpmn:incoming\u003e\n            \u003cbpmn:outgoing\u003eSequenceFlow_17lypcn\u003c/bpmn:outgoing\u003e\n        \u003c/bpmn:serviceTask\u003e\n    \u003c/bpmn:process\u003e\n\u003c/bpmn:definitions\u003e","svg":"\u003csvg\u003e\u003c/svg\u003e"},"elements":[{"bpmn_id":"bpmnid","group":null,"name":"event-name","order":0,"time_event":null,"notification":null,"message_event":null,"conditional_event":{"script":"x == 42","value_variable":"x","variables":null,"qos":0,"event_id":"1","selection":{"filter_criteria":{"characteristic_id":"cid1","function_id":"urn:infai:ses:measuring-function:fid1","device_class_id":null,"aspect_id":"aid1"},"selection_options":null,"selected_device_id":"did1","selected_service_id":"sid1","selected_device_group_id":null,"selected_import_id":null,"selected_generic_event_source":null,"selected_path":{"path":"path.to.chid2","characteristicId":"cid2","aspectNode":{"id":"","name":"","root_id":"","parent_id":"","child_ids":null,"ancestor_ids":null,"descendent_ids":null},"functionId":"","isVoid":false}}},"task":null},{"bpmn_id":"bpmnid-group","group":null,"name":"event-name-group","order":0,"time_event":null,"notification":null,"message_event":null,"conditional_event":{"script":"x == 42","value_variable":"x","variables":null,"qos":0,"event_id":"1-group","selection":{"filter_criteria":{"characteristic_id":"cid1","function_id":"urn:infai:ses:measuring-function:fid1","device_class_id":null,"aspect_id":"aid1"},"selection_options":null,"selected_device_id":null,"selected_service_id":null,"selected_device_group_id":"gid1","selected_import_id":null,"selected_generic_event_source":null,"selected_path":null}},"task":null}],"executable":true,"device_id_to_local_id":{"did1":"ldid1"},"service_id_to_local_id":{"sid1":"lsid1"},"event_descriptions":[{"user_id":"","deployment_id":"test-id","device_group_id":"","device_id":"did1","service_id":"sid1","import_id":"","script":"x == 42","value_variable":"x","variables":null,"qos":0,"event_id":"1","characteristic_id":"cid1","function_id":"urn:infai:ses:measuring-function:fid1","aspect_id":"aid1","path":"path.to.chid2","service_for_marshaller":{"id":"sid1","local_id":"lsid1","name":"test-service-sid1","description":"","interaction":"event+request","protocol_id":"pid","inputs":null,"outputs":[{"id":"","content_variable":{"id":"","name":"foo","is_void":false,"type":"","sub_content_variables":null,"characteristic_id":"cid2","value":null,"serialization_options":null,"function_id":"urn:infai:ses:measuring-function:fid1","aspect_id":"aid1"},"serialization":"","protocol_segment_id":""}],"attributes":null,"service_group_key":""}}]}`
		if !reflect.DeepEqual(messages, []string{expected}) {
			t.Error("\n", messages, "\n", expected)
			return
		}
	})
}

func testDeployEventProcess(port string, networkId string) func(t *testing.T) {
	return func(t *testing.T) {
		requestBody := new(bytes.Buffer)
		err := json.NewEncoder(requestBody).Encode(deploymentmodel.Deployment{
			Id:          "test-id",
			Version:     deploymentmodel.CurrentVersion,
			Name:        "test-deployment-name",
			Description: "test-description",
			Diagram: deploymentmodel.Diagram{
				XmlDeployed: deploymentExampleXml,
				Svg:         "<svg></svg>",
				XmlRaw:      deploymentExampleXml,
			},
			Executable: true,
			Elements: []deploymentmodel.Element{
				{
					BpmnId: "bpmnid",
					Name:   "event-name",
					ConditionalEvent: &deploymentmodel.ConditionalEvent{
						Script:        "x == 42",
						ValueVariable: "x",
						EventId:       "1",
						Selection: deploymentmodel.Selection{
							FilterCriteria: deploymentmodel.FilterCriteria{
								CharacteristicId: strptr("cid1"),
								FunctionId:       strptr(devicemodel.MEASURING_FUNCTION_PREFIX + "fid1"),
								AspectId:         strptr("aid1"),
							},
							SelectionOptions:  nil,
							SelectedDeviceId:  strptr("did1"),
							SelectedServiceId: strptr("sid1"),
							SelectedPath: &deviceselectionmodel.PathOption{
								Path:             "path.to.chid2",
								CharacteristicId: "cid2",
							},
						},
					},
				},
				{
					BpmnId: "bpmnid-group",
					Name:   "event-name-group",
					ConditionalEvent: &deploymentmodel.ConditionalEvent{
						Script:        "x == 42",
						ValueVariable: "x",
						EventId:       "1-group",
						Selection: deploymentmodel.Selection{
							FilterCriteria: deploymentmodel.FilterCriteria{
								CharacteristicId: strptr("cid1"),
								FunctionId:       strptr(devicemodel.MEASURING_FUNCTION_PREFIX + "fid1"),
								AspectId:         strptr("aid1"),
							},
							SelectedDeviceGroupId: strptr("gid1"),
						},
					},
				},
			},
		})
		if err != nil {
			t.Error(err)
			return
		}
		req, err := http.NewRequest("POST", "http://localhost:"+port+"/deployments/"+url.PathEscape(networkId), requestBody)
		if err != nil {
			t.Error(err)
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			err = errors.New(buf.String())
		}
		if err != nil {
			t.Error(err)
			return
		}
	}
}

func strptr(s string) *string {
	return &s
}
