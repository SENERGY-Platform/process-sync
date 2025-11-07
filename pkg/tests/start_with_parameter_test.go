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
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/SENERGY-Platform/process-deployment/lib/model/deploymentmodel"
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
	"github.com/SENERGY-Platform/process-sync/pkg/tests/server"
)

func TestStartWithParameter(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := configuration.Config{
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
		MongoWardenCollection:             "warden",
		MongoDeploymentWardenCollection:   "deployment_warden",
	}

	networkId := "test-network-id"
	config, err := server.Env(ctx, wg, config, networkId)
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(1 * time.Second)

	t.Run("deploy process", testDeployProcessWithParameter(config.ApiPort, networkId))

	deployments := []model.Deployment{}
	t.Run("get deployments", testGetDeployments(config.ApiPort, networkId, &deployments))
	t.Run("check deployments is placeholder", testCheckDeploymentsPlaceholder(&deployments, []bool{true}))
	t.Run("check deployments is marked delete", testCheckDeploymentsMarkedDelete(&deployments, []bool{false}))

	time.Sleep(5 * time.Second)
	t.Run("get deployments", testGetDeployments(config.ApiPort, networkId, &deployments))
	t.Run("check deployments is placeholder", testCheckDeploymentsPlaceholder(&deployments, []bool{false}))
	t.Run("check deployments is marked delete", testCheckDeploymentsMarkedDelete(&deployments, []bool{false}))

	t.Run("check process definitions", testCheckProcessDefinitions(config, networkId))

	deploymentsPreDelete := deployments
	t.Run("check process metadata", testCheckProcessMetadata(config, networkId, &deploymentsPreDelete, 0, true))

	t.Run("check deployment parameter", testGetDeploymentParameter(
		config.ApiPort,
		networkId,
		&deployments,
		0,
		map[string]camundamodel.Variable{
			"targetTemperature": {
				Type:      "String",
				ValueInfo: []interface{}{},
			},
		}))

	t.Run("start deployment", testStartDeploymentWithParameter(config.ApiPort, networkId, &deployments, 0, "targetTemperature=21"))

	t.Run("check instance and history", testCheckInstancesAndHistory(config, networkId))

	t.Run("stop deployment", testRemoveDeployment(config.ApiPort, networkId, &deployments, 0))

	t.Run("get deployments", testGetDeployments(config.ApiPort, networkId, &deployments))
	t.Run("check deployments is placeholder", testCheckDeploymentsPlaceholder(&deployments, []bool{false}))
	t.Run("check deployments is marked delete", testCheckDeploymentsMarkedDelete(&deployments, []bool{true}))

	time.Sleep(5 * time.Second)
	t.Run("get deployments", testGetDeployments(config.ApiPort, networkId, &deployments))
	t.Run("check deployments is placeholder", testCheckDeploymentsPlaceholder(&deployments, []bool{}))
	t.Run("check deployments is marked delete", testCheckDeploymentsMarkedDelete(&deployments, []bool{}))

	t.Run("check process metadata after delete", testCheckProcessMetadata(config, networkId, &deploymentsPreDelete, 0, false))
}

func testStartDeploymentWithKey(port string, networkId string, list *[]model.Deployment, index int, businessKey string) func(t *testing.T) {
	return func(t *testing.T) {
		if index >= len(*list) {
			t.Error(len(*list), index)
			return
		}
		deployment := (*list)[index]
		req, err := http.NewRequest("GET", "http://localhost:"+port+"/deployments/"+networkId+"/"+deployment.Id+"/start?business_key="+businessKey, nil)
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

func testStartDeploymentWithParameter(port string, networkId string, list *[]model.Deployment, index int, queryEncodedParameter string) func(t *testing.T) {
	return func(t *testing.T) {
		if index >= len(*list) {
			t.Error(len(*list), index)
			return
		}
		deployment := (*list)[index]
		req, err := http.NewRequest("GET", "http://localhost:"+port+"/deployments/"+networkId+"/"+deployment.Id+"/start?"+queryEncodedParameter, nil)
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

func testGetDeploymentParameter(port string, networkId string, list *[]model.Deployment, index int, expectedParameter map[string]camundamodel.Variable) func(t *testing.T) {
	return func(t *testing.T) {
		if index >= len(*list) {
			t.Error(len(*list), index)
			return
		}
		deployment := (*list)[index]
		req, err := http.NewRequest("GET", "http://localhost:"+port+"/deployments/"+networkId+"/"+deployment.Id+"/metadata", nil)
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

		metadata := model.DeploymentMetadata{}
		err = json.NewDecoder(resp.Body).Decode(&metadata)
		if err != nil {
			t.Error(err)
			return
		}
		if metadata.NetworkId != networkId {
			t.Error(metadata.NetworkId, networkId, metadata)
		}
		if metadata.CamundaDeploymentId != deployment.Id {
			t.Error(metadata.CamundaDeploymentId, deployment.Id, metadata)
		}
		if metadata.DeploymentModel.Name != deployment.Name {
			t.Error(metadata.DeploymentModel.Name, deployment.Name, metadata)
		}
		if !reflect.DeepEqual(metadata.Metadata.ProcessParameter, expectedParameter) {
			temp, _ := json.Marshal(metadata)
			t.Error(string(temp))
		}
	}
}

func testDeployProcessWithArgs(port string, networkId string, deploymentId string, bpmn string) func(t *testing.T) {
	return func(t *testing.T) {
		requestBody := new(bytes.Buffer)
		err := json.NewEncoder(requestBody).Encode(deploymentmodel.Deployment{
			Version:     deploymentmodel.CurrentVersion,
			Id:          deploymentId,
			Name:        deploymentId + "-name",
			Description: "test-description",
			Diagram: deploymentmodel.Diagram{
				XmlDeployed: bpmn,
				Svg:         "<svg></svg>",
				XmlRaw:      bpmn,
			},
			Executable: true,
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

func testDeployProcessWithParameter(port string, networkId string) func(t *testing.T) {
	return testDeployProcessWithArgs(port, networkId, "test-id", processWithParameter)
}

const processWithParameter = "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<bpmn:definitions xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xmlns:bpmn=\"http://www.omg.org/spec/BPMN/20100524/MODEL\" xmlns:bpmndi=\"http://www.omg.org/spec/BPMN/20100524/DI\" xmlns:dc=\"http://www.omg.org/spec/DD/20100524/DC\" xmlns:camunda=\"http://camunda.org/schema/1.0/bpmn\" xmlns:di=\"http://www.omg.org/spec/DD/20100524/DI\" id=\"Definitions_1\" targetNamespace=\"http://bpmn.io/schema/bpmn\"><bpmn:process id=\"SetThermostatDouble_Input\" isExecutable=\"true\"><bpmn:startEvent id=\"StartEvent_1\"><bpmn:extensionElements><camunda:formData><camunda:formField id=\"targetTemperature\" type=\"string\" /></camunda:formData></bpmn:extensionElements><bpmn:outgoing>SequenceFlow_08n5i1u</bpmn:outgoing></bpmn:startEvent><bpmn:endEvent id=\"EndEvent_0vfh9qd\"><bpmn:incoming>SequenceFlow_05a7edq</bpmn:incoming></bpmn:endEvent><bpmn:serviceTask id=\"Task_0qasfc7\" name=\"Thermostat setTemperatureFunction\" camunda:type=\"external\" camunda:topic=\"pessimistic\"><bpmn:extensionElements><camunda:inputOutput><camunda:inputParameter name=\"payload\">{\n    \"function\": {\n        \"id\": \"urn:infai:ses:controlling-function:99240d90-02dd-4d4f-a47c-069cfe77629c\",\n        \"name\": \"setTemperatureFunction\",\n        \"concept_id\": \"urn:infai:ses:concept:0bc81398-3ed6-4e2b-a6c4-b754583aac37\",\n        \"rdf_type\": \"https://senergy.infai.org/ontology/ControllingFunction\"\n    },\n    \"device_class\": {\n        \"id\": \"urn:infai:ses:device-class:997937d6-c5f3-4486-b67c-114675038393\",\n        \"image\": \"\",\n        \"name\": \"Thermostat\",\n        \"rdf_type\": \"https://senergy.infai.org/ontology/DeviceClass\"\n    },\n    \"aspect\": null,\n    \"label\": \"setTemperatureFunction\",\n    \"input\": 0,\n    \"characteristic_id\": \"urn:infai:ses:characteristic:5ba31623-0ccb-4488-bfb7-f73b50e03b5a\",\n    \"retries\": 2\n}</camunda:inputParameter><camunda:inputParameter name=\"inputs\">${targetTemperature}</camunda:inputParameter></camunda:inputOutput></bpmn:extensionElements><bpmn:incoming>SequenceFlow_0a4xeky</bpmn:incoming><bpmn:outgoing>SequenceFlow_0bunaeu</bpmn:outgoing></bpmn:serviceTask><bpmn:serviceTask id=\"Task_0o3bteo\" name=\"Thermostat setTemperatureFunction\" camunda:type=\"external\" camunda:topic=\"pessimistic\"><bpmn:extensionElements><camunda:inputOutput><camunda:inputParameter name=\"payload\">{\n    \"function\": {\n        \"id\": \"urn:infai:ses:controlling-function:99240d90-02dd-4d4f-a47c-069cfe77629c\",\n        \"name\": \"setTemperatureFunction\",\n        \"concept_id\": \"urn:infai:ses:concept:0bc81398-3ed6-4e2b-a6c4-b754583aac37\",\n        \"rdf_type\": \"https://senergy.infai.org/ontology/ControllingFunction\"\n    },\n    \"device_class\": {\n        \"id\": \"urn:infai:ses:device-class:997937d6-c5f3-4486-b67c-114675038393\",\n        \"image\": \"\",\n        \"name\": \"Thermostat\",\n        \"rdf_type\": \"https://senergy.infai.org/ontology/DeviceClass\"\n    },\n    \"aspect\": null,\n    \"label\": \"setTemperatureFunction\",\n    \"input\": 0,\n    \"characteristic_id\": \"urn:infai:ses:characteristic:5ba31623-0ccb-4488-bfb7-f73b50e03b5a\",\n    \"retries\": 2\n}</camunda:inputParameter><camunda:inputParameter name=\"inputs\">${targetTemperature}</camunda:inputParameter></camunda:inputOutput></bpmn:extensionElements><bpmn:incoming>SequenceFlow_1dgbuzc</bpmn:incoming><bpmn:outgoing>SequenceFlow_0bqyg2r</bpmn:outgoing></bpmn:serviceTask><bpmn:parallelGateway id=\"ExclusiveGateway_0ds5o6b\"><bpmn:incoming>SequenceFlow_08n5i1u</bpmn:incoming><bpmn:outgoing>SequenceFlow_1dgbuzc</bpmn:outgoing><bpmn:outgoing>SequenceFlow_0a4xeky</bpmn:outgoing></bpmn:parallelGateway><bpmn:parallelGateway id=\"ExclusiveGateway_138ca0d\"><bpmn:incoming>SequenceFlow_0bqyg2r</bpmn:incoming><bpmn:incoming>SequenceFlow_0bunaeu</bpmn:incoming><bpmn:outgoing>SequenceFlow_05a7edq</bpmn:outgoing></bpmn:parallelGateway><bpmn:sequenceFlow id=\"SequenceFlow_0bqyg2r\" sourceRef=\"Task_0o3bteo\" targetRef=\"ExclusiveGateway_138ca0d\" /><bpmn:sequenceFlow id=\"SequenceFlow_0bunaeu\" sourceRef=\"Task_0qasfc7\" targetRef=\"ExclusiveGateway_138ca0d\" /><bpmn:sequenceFlow id=\"SequenceFlow_05a7edq\" sourceRef=\"ExclusiveGateway_138ca0d\" targetRef=\"EndEvent_0vfh9qd\" /><bpmn:sequenceFlow id=\"SequenceFlow_08n5i1u\" sourceRef=\"StartEvent_1\" targetRef=\"ExclusiveGateway_0ds5o6b\" /><bpmn:sequenceFlow id=\"SequenceFlow_1dgbuzc\" sourceRef=\"ExclusiveGateway_0ds5o6b\" targetRef=\"Task_0o3bteo\" /><bpmn:sequenceFlow id=\"SequenceFlow_0a4xeky\" sourceRef=\"ExclusiveGateway_0ds5o6b\" targetRef=\"Task_0qasfc7\" /></bpmn:process><bpmndi:BPMNDiagram id=\"BPMNDiagram_1\"><bpmndi:BPMNPlane id=\"BPMNPlane_1\" bpmnElement=\"SetThermostatDouble_Input\"><bpmndi:BPMNShape id=\"_BPMNShape_StartEvent_2\" bpmnElement=\"StartEvent_1\"><dc:Bounds x=\"112\" y=\"162\" width=\"36\" height=\"36\" /></bpmndi:BPMNShape><bpmndi:BPMNShape id=\"EndEvent_0vfh9qd_di\" bpmnElement=\"EndEvent_0vfh9qd\"><dc:Bounds x=\"532\" y=\"162\" width=\"36\" height=\"36\" /></bpmndi:BPMNShape><bpmndi:BPMNShape id=\"ServiceTask_17o74jp_di\" bpmnElement=\"Task_0qasfc7\"><dc:Bounds x=\"280\" y=\"80\" width=\"100\" height=\"80\" /></bpmndi:BPMNShape><bpmndi:BPMNShape id=\"ServiceTask_1wr65xs_di\" bpmnElement=\"Task_0o3bteo\"><dc:Bounds x=\"280\" y=\"200\" width=\"100\" height=\"80\" /></bpmndi:BPMNShape><bpmndi:BPMNShape id=\"ParallelGateway_1yjkf3k_di\" bpmnElement=\"ExclusiveGateway_0ds5o6b\"><dc:Bounds x=\"185\" y=\"155\" width=\"50\" height=\"50\" /></bpmndi:BPMNShape><bpmndi:BPMNShape id=\"ParallelGateway_0swaeew_di\" bpmnElement=\"ExclusiveGateway_138ca0d\"><dc:Bounds x=\"435\" y=\"155\" width=\"50\" height=\"50\" /></bpmndi:BPMNShape><bpmndi:BPMNEdge id=\"SequenceFlow_0bqyg2r_di\" bpmnElement=\"SequenceFlow_0bqyg2r\"><di:waypoint x=\"380\" y=\"240\" /><di:waypoint x=\"460\" y=\"240\" /><di:waypoint x=\"460\" y=\"205\" /></bpmndi:BPMNEdge><bpmndi:BPMNEdge id=\"SequenceFlow_0bunaeu_di\" bpmnElement=\"SequenceFlow_0bunaeu\"><di:waypoint x=\"380\" y=\"120\" /><di:waypoint x=\"460\" y=\"120\" /><di:waypoint x=\"460\" y=\"155\" /></bpmndi:BPMNEdge><bpmndi:BPMNEdge id=\"SequenceFlow_05a7edq_di\" bpmnElement=\"SequenceFlow_05a7edq\"><di:waypoint x=\"485\" y=\"180\" /><di:waypoint x=\"532\" y=\"180\" /></bpmndi:BPMNEdge><bpmndi:BPMNEdge id=\"SequenceFlow_08n5i1u_di\" bpmnElement=\"SequenceFlow_08n5i1u\"><di:waypoint x=\"148\" y=\"180\" /><di:waypoint x=\"185\" y=\"180\" /></bpmndi:BPMNEdge><bpmndi:BPMNEdge id=\"SequenceFlow_1dgbuzc_di\" bpmnElement=\"SequenceFlow_1dgbuzc\"><di:waypoint x=\"210\" y=\"205\" /><di:waypoint x=\"210\" y=\"240\" /><di:waypoint x=\"280\" y=\"240\" /></bpmndi:BPMNEdge><bpmndi:BPMNEdge id=\"SequenceFlow_0a4xeky_di\" bpmnElement=\"SequenceFlow_0a4xeky\"><di:waypoint x=\"210\" y=\"155\" /><di:waypoint x=\"210\" y=\"120\" /><di:waypoint x=\"280\" y=\"120\" /></bpmndi:BPMNEdge></bpmndi:BPMNPlane></bpmndi:BPMNDiagram></bpmn:definitions>"
