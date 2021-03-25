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
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"github.com/SENERGY-Platform/process-sync/pkg/model/deploymentmodel"
	"github.com/SENERGY-Platform/process-sync/pkg/tests/server"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"
)

func TestSync(t *testing.T) {
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
	}

	networkId := "test-network-id"
	config, err := server.Env(ctx, wg, config, networkId)
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("deploy process", testDeployProcess(config.ApiPort, networkId))

	deployments := []model.Deployment{}
	t.Run("search deployments", testFindDeployments(config.ApiPort, "test", networkId, &deployments))
	t.Run("check deployments search", testCheckDeploymentsPlaceholder(&deployments, []bool{true}))

	t.Run("get deployments", testGetDeployments(config.ApiPort, networkId, &deployments))
	t.Run("check deployments is placeholder", testCheckDeploymentsPlaceholder(&deployments, []bool{true}))
	t.Run("check deployments is marked delete", testCheckDeploymentsMarkedDelete(&deployments, []bool{false}))

	time.Sleep(5 * time.Second)
	t.Run("get deployments", testGetDeployments(config.ApiPort, networkId, &deployments))
	t.Run("check deployments is placeholder", testCheckDeploymentsPlaceholder(&deployments, []bool{false}))
	t.Run("check deployments is marked delete", testCheckDeploymentsMarkedDelete(&deployments, []bool{false}))

	t.Run("check process definitions", testCheckProcessDefinitions(config, networkId))
	t.Run("check extended deployment", testCheckExtendedDeployment(config, networkId, 1))

	deploymentsPreDelete := deployments
	t.Run("check process metadata", testCheckProcessMetadata(config, networkId, &deploymentsPreDelete, 0, true))

	t.Run("start deployment", testStartDeployment(config.ApiPort, networkId, &deployments, 0))

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

func testCheckExtendedDeployment(config configuration.Config, networkId string, count int) func(t *testing.T) {
	return func(t *testing.T) {
		req, err := http.NewRequest("GET", "http://localhost:"+config.ApiPort+"/deployments?extended=true&network_id="+networkId, nil)
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
		result := []model.ExtendedDeployment{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			t.Error(err)
			return
		}

		if len(result) != count {
			t.Error(result)
			return
		}

		for _, element := range result {
			if element.Error != "" {
				t.Error(element.Error)
				return
			}
			if element.DefinitionId == "" {
				t.Error(element)
				return
			}
			if element.Diagram != "<svg></svg>" {
				t.Error(element.Diagram, element)
				return
			}
		}
	}
}

func testCheckProcessDefinitions(config configuration.Config, networkId string) func(t *testing.T) {
	return func(t *testing.T) {
		definitions := []model.ProcessDefinition{}
		t.Run("get definitions", testGetDefinitions(config.ApiPort, networkId, &definitions))
		t.Run("check definitions is placeholder", testCheckDefinitionsPlaceholder(&definitions, []bool{false}))
		t.Run("check definitions is marked delete", testCheckDefinitionsMarkedDelete(&definitions, []bool{false}))
	}
}

func testCheckProcessMetadata(config configuration.Config, networkId string, list *[]model.Deployment, index int, expectedExistence bool) func(t *testing.T) {
	return func(t *testing.T) {
		if index >= len(*list) {
			t.Error(len(*list), index)
			return
		}
		deployment := (*list)[index]
		req, err := http.NewRequest("GET", "http://localhost:"+config.ApiPort+"/deployments/"+networkId+"/"+deployment.Id+"/metadata", nil)
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
		exists := true
		if resp.StatusCode >= 300 {
			exists = false
		}
		if exists != expectedExistence {
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			t.Error(exists, expectedExistence, resp.StatusCode, buf.String())
			return
		}

		if expectedExistence {
			temp := model.DeploymentMetadata{}
			err = json.NewDecoder(resp.Body).Decode(&temp)
			if err != nil {
				t.Error(err)
				return
			}
			if temp.NetworkId != networkId {
				t.Error(temp.NetworkId, networkId, temp)
			}
			if temp.CamundaDeploymentId != deployment.Id {
				t.Error(temp.CamundaDeploymentId, deployment.Id, temp)
			}
			if temp.DeploymentModel.Name != deployment.Name {
				t.Error(temp.DeploymentModel.Name, deployment.Name, temp)
			}
		}
	}
}

func testCheckInstances(config configuration.Config, networkId string) func(t *testing.T) {
	return func(t *testing.T) {
		instances := []model.ProcessInstance{}
		t.Run("get instances", testGetInstances(config.ApiPort, networkId, &instances))
		t.Run("check instance is placeholder", testCheckInstancesPlaceholder(&instances, []bool{true}))
		t.Run("check instance is marked delete", testCheckInstancesMarkedDelete(&instances, []bool{false}))

		time.Sleep(5 * time.Second)
		t.Run("get instances", testGetInstances(config.ApiPort, networkId, &instances))
		t.Run("check instance is placeholder", testCheckInstancesPlaceholder(&instances, []bool{false}))
		t.Run("check instance is marked delete", testCheckInstancesMarkedDelete(&instances, []bool{false}))

		t.Run("stop instance", testDeleteInstances(config.ApiPort, networkId, &instances, 0))

		t.Run("get instances", testGetInstances(config.ApiPort, networkId, &instances))
		t.Run("check instance is placeholder", testCheckInstancesPlaceholder(&instances, []bool{false}))
		t.Run("check instance is marked delete", testCheckInstancesMarkedDelete(&instances, []bool{true}))

		time.Sleep(5 * time.Second)
		t.Run("get instances", testGetInstances(config.ApiPort, networkId, &instances))
		t.Run("check instance is placeholder", testCheckInstancesPlaceholder(&instances, []bool{}))
		t.Run("check instance is marked delete", testCheckInstancesMarkedDelete(&instances, []bool{}))
	}
}

func testCheckInstancesAndHistory(config configuration.Config, networkId string) func(t *testing.T) {
	return func(t *testing.T) {
		instances := []model.ProcessInstance{}
		t.Run("get instances", testGetInstances(config.ApiPort, networkId, &instances))
		t.Run("check instance is placeholder", testCheckInstancesPlaceholder(&instances, []bool{true}))
		t.Run("check instance is marked delete", testCheckInstancesMarkedDelete(&instances, []bool{false}))

		historicInstances := []model.HistoricProcessInstance{}
		t.Run("get historic instances", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
		t.Run("check historic instance is placeholder", testCheckHistoricInstancesPlaceholder(&historicInstances, []bool{true}))
		t.Run("check historic instance is marked delete", testCheckHistoricInstancesMarkedDelete(&historicInstances, []bool{false}))

		time.Sleep(5 * time.Second)

		t.Run("get instances", testGetInstances(config.ApiPort, networkId, &instances))
		t.Run("check instance is placeholder", testCheckInstancesPlaceholder(&instances, []bool{false}))
		t.Run("check instance is marked delete", testCheckInstancesMarkedDelete(&instances, []bool{false}))

		t.Run("get historicInstances", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
		t.Run("check historic instance is placeholder", testCheckHistoricInstancesPlaceholder(&historicInstances, []bool{false}))
		t.Run("check historic instance is marked delete", testCheckHistoricInstancesMarkedDelete(&historicInstances, []bool{false}))

		t.Run("stop instance", testDeleteInstances(config.ApiPort, networkId, &instances, 0))

		t.Run("get instances", testGetInstances(config.ApiPort, networkId, &instances))
		t.Run("check instance is placeholder", testCheckInstancesPlaceholder(&instances, []bool{false}))
		t.Run("check instance is marked delete", testCheckInstancesMarkedDelete(&instances, []bool{true}))

		time.Sleep(5 * time.Second)
		t.Run("get instances", testGetInstances(config.ApiPort, networkId, &instances))
		t.Run("check instance is placeholder", testCheckInstancesPlaceholder(&instances, []bool{}))
		t.Run("check instance is marked delete", testCheckInstancesMarkedDelete(&instances, []bool{}))

		t.Run("get historic instances", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
		t.Run("check historic instance is placeholder", testCheckHistoricInstancesPlaceholder(&historicInstances, []bool{false}))
		t.Run("check historic instance is marked delete", testCheckHistoricInstancesMarkedDelete(&historicInstances, []bool{false}))

		t.Run("remove historic instance", testDeleteHistoricInstances(config.ApiPort, networkId, &historicInstances, 0))

		t.Run("get historic instances", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
		t.Run("check historic instance is placeholder", testCheckHistoricInstancesPlaceholder(&historicInstances, []bool{false}))
		t.Run("check historic instance is marked delete", testCheckHistoricInstancesMarkedDelete(&historicInstances, []bool{true}))

		time.Sleep(5 * time.Second)
		t.Run("get historic instances", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
		t.Run("check historic instance is placeholder", testCheckHistoricInstancesPlaceholder(&historicInstances, []bool{}))
		t.Run("check historic instance is marked delete", testCheckHistoricInstancesMarkedDelete(&historicInstances, []bool{}))
	}
}

func testCheckHistoricInstances(config configuration.Config, networkId string) func(t *testing.T) {
	return func(t *testing.T) {
		historicInstances := []model.HistoricProcessInstance{}
		t.Run("get historic instances", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
		t.Run("check historic instance is placeholder", testCheckHistoricInstancesPlaceholder(&historicInstances, []bool{true}))
		t.Run("check historic instance is marked delete", testCheckHistoricInstancesMarkedDelete(&historicInstances, []bool{false}))

		time.Sleep(5 * time.Second)
		t.Run("get historicInstances", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
		t.Run("check historic instance is placeholder", testCheckHistoricInstancesPlaceholder(&historicInstances, []bool{false}))
		t.Run("check historic instance is marked delete", testCheckHistoricInstancesMarkedDelete(&historicInstances, []bool{false}))

		t.Run("stop historic instance", testDeleteHistoricInstances(config.ApiPort, networkId, &historicInstances, 0))

		t.Run("get historic instances", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
		t.Run("check historic instance is placeholder", testCheckHistoricInstancesPlaceholder(&historicInstances, []bool{false}))
		t.Run("check historic instance is marked delete", testCheckHistoricInstancesMarkedDelete(&historicInstances, []bool{true}))

		time.Sleep(5 * time.Second)
		t.Run("get historic instances", testGetHistoricInstances(config.ApiPort, networkId, &historicInstances))
		t.Run("check historic instance is placeholder", testCheckHistoricInstancesPlaceholder(&historicInstances, []bool{}))
		t.Run("check historic instance is marked delete", testCheckHistoricInstancesMarkedDelete(&historicInstances, []bool{}))
	}
}

func testRemoveDeployment(port string, networkId string, list *[]model.Deployment, index int) func(t *testing.T) {
	return func(t *testing.T) {
		if index >= len(*list) {
			t.Error(len(*list), index)
			return
		}
		deployment := (*list)[index]
		req, err := http.NewRequest("DELETE", "http://localhost:"+port+"/deployments/"+networkId+"/"+deployment.Id, nil)
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

func testDeleteInstances(port string, networkId string, list *[]model.ProcessInstance, index int) func(t *testing.T) {
	return func(t *testing.T) {
		if index >= len(*list) {
			t.Error(len(*list), index)
			return
		}
		instance := (*list)[index]
		req, err := http.NewRequest("DELETE", "http://localhost:"+port+"/process-instances/"+networkId+"/"+instance.Id, nil)
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

func testCheckInstancesMarkedDelete(list *[]model.ProcessInstance, bools []bool) func(t *testing.T) {
	return func(t *testing.T) {
		if len(*list) != len(bools) {
			temp, _ := json.Marshal(*list)
			t.Error(len(*list), string(temp))
			return
		}
		for i, b := range bools {
			element := (*list)[i]
			if element.MarkedForDelete != b {
				temp, _ := json.Marshal(element)
				t.Error(string(temp))
				return
			}
		}
	}
}

func testCheckInstancesPlaceholder(list *[]model.ProcessInstance, bools []bool) func(t *testing.T) {
	return func(t *testing.T) {
		if len(*list) != len(bools) {
			temp, _ := json.Marshal(*list)
			t.Error(len(*list), string(temp))
			return
		}
		for i, b := range bools {
			element := (*list)[i]
			if element.IsPlaceholder != b {
				temp, _ := json.Marshal(element)
				t.Error(string(temp))
				return
			}
		}
	}
}

func testGetInstances(port string, networkId string, result *[]model.ProcessInstance) func(t *testing.T) {
	return func(t *testing.T) {
		req, err := http.NewRequest("GET", "http://localhost:"+port+"/process-instances?network_id="+networkId, nil)
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
		temp := []model.ProcessInstance{}
		err = json.NewDecoder(resp.Body).Decode(&temp)
		if err != nil {
			t.Error(err)
			return
		}
		*result = temp
	}
}

func testDeleteHistoricInstances(port string, networkId string, list *[]model.HistoricProcessInstance, index int) func(t *testing.T) {
	return func(t *testing.T) {
		if index >= len(*list) {
			t.Error(len(*list), index)
			return
		}
		instance := (*list)[index]
		req, err := http.NewRequest("DELETE", "http://localhost:"+port+"/history/process-instances/"+networkId+"/"+instance.Id, nil)
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

func testCheckHistoricInstancesMarkedDelete(list *[]model.HistoricProcessInstance, bools []bool) func(t *testing.T) {
	return func(t *testing.T) {
		if len(*list) != len(bools) {
			temp, _ := json.Marshal(*list)
			t.Error(len(*list), string(temp))
			return
		}
		for i, b := range bools {
			element := (*list)[i]
			if element.MarkedForDelete != b {
				temp, _ := json.Marshal(element)
				t.Error(string(temp))
				return
			}
		}
	}
}

func testCheckHistoricInstancesPlaceholder(list *[]model.HistoricProcessInstance, bools []bool) func(t *testing.T) {
	return func(t *testing.T) {
		if len(*list) != len(bools) {
			temp, _ := json.Marshal(*list)
			t.Error(bools, len(*list), string(temp))
			return
		}
		for i, b := range bools {
			element := (*list)[i]
			if element.IsPlaceholder != b {
				temp, _ := json.Marshal(element)
				t.Error(string(temp))
				return
			}
		}
	}
}

func testGetHistoricInstances(port string, networkId string, result *[]model.HistoricProcessInstance) func(t *testing.T) {
	return func(t *testing.T) {
		req, err := http.NewRequest("GET", "http://localhost:"+port+"/history/process-instances?network_id="+networkId, nil)
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
		temp := []model.HistoricProcessInstance{}
		err = json.NewDecoder(resp.Body).Decode(&temp)
		if err != nil {
			t.Error(err)
			return
		}
		*result = temp
	}
}

func testStartDeployment(port string, networkId string, list *[]model.Deployment, index int) func(t *testing.T) {
	return func(t *testing.T) {
		if index >= len(*list) {
			t.Error(len(*list), index)
			return
		}
		deployment := (*list)[index]
		req, err := http.NewRequest("GET", "http://localhost:"+port+"/deployments/"+networkId+"/"+deployment.Id+"/start", nil)
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

func testCheckDefinitionsMarkedDelete(list *[]model.ProcessDefinition, bools []bool) func(t *testing.T) {
	return func(t *testing.T) {
		if len(*list) != len(bools) {
			temp, _ := json.Marshal(*list)
			t.Error(len(*list), string(temp))
			return
		}
		for i, b := range bools {
			element := (*list)[i]
			if element.MarkedForDelete != b {
				temp, _ := json.Marshal(element)
				t.Error(string(temp))
				return
			}
		}
	}
}

func testCheckDefinitionsPlaceholder(list *[]model.ProcessDefinition, bools []bool) func(t *testing.T) {
	return func(t *testing.T) {
		if len(*list) != len(bools) {
			temp, _ := json.Marshal(*list)
			t.Error(len(*list), string(temp))
			return
		}
		for i, b := range bools {
			element := (*list)[i]
			if element.IsPlaceholder != b {
				temp, _ := json.Marshal(element)
				t.Error(string(temp))
				return
			}
		}
	}
}

func testGetDefinitions(port string, networkId string, result *[]model.ProcessDefinition) func(t *testing.T) {
	return func(t *testing.T) {
		req, err := http.NewRequest("GET", "http://localhost:"+port+"/process-definitions?network_id="+networkId, nil)
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
		temp := []model.ProcessDefinition{}
		err = json.NewDecoder(resp.Body).Decode(&temp)
		if err != nil {
			t.Error(err)
			return
		}
		*result = temp
	}
}

func testCheckDeploymentsMarkedDelete(deployments *[]model.Deployment, bools []bool) func(t *testing.T) {
	return func(t *testing.T) {
		if len(*deployments) != len(bools) {
			t.Error(len(*deployments), *deployments)
			return
		}
		for i, b := range bools {
			depl := (*deployments)[i]
			if depl.MarkedForDelete != b {
				t.Error(depl)
				return
			}
		}
	}
}

func testCheckDeploymentsPlaceholder(deployments *[]model.Deployment, bools []bool) func(t *testing.T) {
	return func(t *testing.T) {
		if len(*deployments) != len(bools) {
			t.Error(len(*deployments), *deployments)
			return
		}
		for i, b := range bools {
			depl := (*deployments)[i]
			if depl.IsPlaceholder != b {
				t.Error(depl)
				return
			}
		}
	}
}

func testGetDeployments(port string, networkId string, result *[]model.Deployment) func(t *testing.T) {
	return func(t *testing.T) {
		req, err := http.NewRequest("GET", "http://localhost:"+port+"/deployments?network_id="+networkId, nil)
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
		temp := []model.Deployment{}
		err = json.NewDecoder(resp.Body).Decode(&temp)
		if err != nil {
			t.Error(err)
			return
		}
		*result = temp
	}
}

func testFindDeployments(port string, searchtext string, networkId string, result *[]model.Deployment) func(t *testing.T) {
	return func(t *testing.T) {
		req, err := http.NewRequest("GET", "http://localhost:"+port+"/deployments?network_id="+networkId+"&search="+searchtext, nil)
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
		temp := []model.Deployment{}
		err = json.NewDecoder(resp.Body).Decode(&temp)
		if err != nil {
			t.Error(err)
			return
		}
		*result = temp
	}
}

func testDeployProcess(port string, networkId string) func(t *testing.T) {
	return func(t *testing.T) {
		requestBody := new(bytes.Buffer)
		err := json.NewEncoder(requestBody).Encode(deploymentmodel.Deployment{
			Name:        "test-deployment-name",
			Description: "test-description",
			Diagram: deploymentmodel.Diagram{
				XmlDeployed: deploymentExampleXml,
				Svg:         "<svg></svg>",
				XmlRaw:      deploymentExampleXml,
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

const deploymentExampleXml = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL" xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI" xmlns:dc="http://www.omg.org/spec/DD/20100524/DC" xmlns:camunda="http://camunda.org/schema/1.0/bpmn" xmlns:senergy="https://senergy.infai.org" xmlns:di="http://www.omg.org/spec/DD/20100524/DI" id="Definitions_1" targetNamespace="http://bpmn.io/schema/bpmn">
    <bpmn:process id="ExampleId" name="ExampleName" isExecutable="true" senergy:description="ExampleDesc">
        <bpmn:startEvent id="StartEvent_1">
            <bpmn:outgoing>SequenceFlow_0qjn3dq</bpmn:outgoing>
        </bpmn:startEvent>
        <bpmn:sequenceFlow id="SequenceFlow_0qjn3dq" sourceRef="StartEvent_1" targetRef="Task_1nbnl8y"/>
        <bpmn:sequenceFlow id="SequenceFlow_15v8030" sourceRef="Task_1nbnl8y" targetRef="Task_1lhzy95"/>
        <bpmn:endEvent id="EndEvent_1vhaxdr">
            <bpmn:incoming>SequenceFlow_17lypcn</bpmn:incoming>
        </bpmn:endEvent>
        <bpmn:sequenceFlow id="SequenceFlow_17lypcn" sourceRef="Task_1lhzy95" targetRef="EndEvent_1vhaxdr"/>
        <bpmn:serviceTask id="Task_1nbnl8y" name="Lighting getColorFunction" camunda:type="external" camunda:topic="pessimistic">
            <bpmn:extensionElements>
                <camunda:inputOutput>
                    <camunda:inputParameter name="payload"><![CDATA[{
	"function": {
		"id": "urn:infai:ses:measuring-function:bdb6a7c8-4a3d-4fe0-bab3-ce02e09b5869",
		"name": "",
		"concept_id": "",
		"rdf_type": ""
	},
	"characteristic_id": "urn:infai:ses:characteristic:5b4eea52-e8e5-4e80-9455-0382f81a1b43",
	"aspect": {
		"id": "urn:infai:ses:aspect:a7470d73-dde3-41fc-92bd-f16bb28f2da6",
		"name": "",
		"rdf_type": ""
	},
	"device_id": "hue",
	"service_id": "urn:infai:ses:service:99614933-4734-41b6-a131-3f96f134ee69",
	"input": {},
	"retries": 2
}]]></camunda:inputParameter>
                    <camunda:outputParameter name="outputs.b">${result.b}</camunda:outputParameter>
                    <camunda:outputParameter name="outputs.g">${result.g}</camunda:outputParameter>
                    <camunda:outputParameter name="outputs.r">${result.r}</camunda:outputParameter>
                </camunda:inputOutput>
            </bpmn:extensionElements>
            <bpmn:incoming>SequenceFlow_0qjn3dq</bpmn:incoming>
            <bpmn:outgoing>SequenceFlow_15v8030</bpmn:outgoing>
        </bpmn:serviceTask>
        <bpmn:serviceTask id="Task_1lhzy95" name="Lamp setColorFunction" camunda:type="external" camunda:topic="optimistic">
            <bpmn:extensionElements>
                <camunda:inputOutput>
                    <camunda:inputParameter name="payload"><![CDATA[{
	"function": {
		"id": "urn:infai:ses:controlling-function:c54e2a89-1fb8-4ecb-8993-a7b40b355599",
		"name": "",
		"concept_id": "",
		"rdf_type": ""
	},
	"characteristic_id": "urn:infai:ses:characteristic:5b4eea52-e8e5-4e80-9455-0382f81a1b43",
	"device_class": {
		"id": "urn:infai:ses:device-class:14e56881-16f9-4120-bb41-270a43070c86",
		"name": "",
		"image": "",
		"rdf_type": ""
	},
	"device_id": "hue",
	"service_id": "urn:infai:ses:service:67789396-d1ca-4ea9-9147-0614c6d68a2f",
	"input": {
		"b": 0,
		"g": 0,
		"r": 0
	}
}]]></camunda:inputParameter>
                    <camunda:inputParameter name="inputs.b">0</camunda:inputParameter>
                    <camunda:inputParameter name="inputs.g">255</camunda:inputParameter>
                    <camunda:inputParameter name="inputs.r">100</camunda:inputParameter>
                </camunda:inputOutput>
            </bpmn:extensionElements>
            <bpmn:incoming>SequenceFlow_15v8030</bpmn:incoming>
            <bpmn:outgoing>SequenceFlow_17lypcn</bpmn:outgoing>
        </bpmn:serviceTask>
    </bpmn:process>
</bpmn:definitions>`
