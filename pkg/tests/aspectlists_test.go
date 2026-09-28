/*
 * Copyright 2026 InfAI (CC SES)
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
	"io"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"testing"
	"time"

	eventmodel "github.com/SENERGY-Platform/event-deployment/lib/model"
	eventworkermodel "github.com/SENERGY-Platform/event-worker/pkg/model"
	"github.com/SENERGY-Platform/process-deployment/lib/model/deploymentmodel"
	"github.com/SENERGY-Platform/process-deployment/lib/model/devicemodel"
	"github.com/SENERGY-Platform/process-deployment/lib/model/deviceselectionmodel"
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"github.com/SENERGY-Platform/process-sync/pkg/tests/mocks"
	"github.com/SENERGY-Platform/process-sync/pkg/tests/server"
	paho "github.com/eclipse/paho.mqtt.golang"
)

// aspectSpelling is one way a conditional event can name its aspects. AspectId is deprecated
// and an alias for an AspectIds list with a single element.
type aspectSpelling struct {
	eventId   string
	aspectId  *string
	aspectIds []string
	// folded is what a reader of the event description sees after joining both spellings.
	folded []string
}

var aspectSpellings = []aspectSpelling{
	{eventId: "aspect-id-only", aspectId: strptr("aid1"), folded: []string{"aid1"}},
	{eventId: "aspect-ids-single", aspectIds: []string{"aid1"}, folded: []string{"aid1"}},
	{eventId: "aspect-ids-multiple", aspectIds: []string{"aid1", "aid2"}, folded: []string{"aid1", "aid2"}},
	{eventId: "aspect-id-and-ids", aspectId: strptr("aid3"), aspectIds: []string{"aid1", "aid2"}, folded: []string{"aid1", "aid2", "aid3"}},
}

// TestAspectListsInFogDeployment deploys conditional events that name their aspects in every
// supported spelling and checks what reaches the mgw and the device-repository.
// The service stores the deployment and forwards it, so both spellings have to pass through
// unchanged: a mgw that only knows aspect_id must keep getting it, and one that knows the list
// must get the list.
func TestAspectListsInFogDeployment(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := configuration.Config{
		MqttCleanSession:                  true,
		MqttGroupId:                       "",
		MongoDatabase:                     "processes",
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
	devices := &mocks.Devices{}
	config, err := server.EnvForEventsCheckWithDevices(ctx, wg, config, networkId, devices)
	if err != nil {
		t.Error(err)
		return
	}

	options := paho.NewClientOptions().
		SetPassword(config.Mqtt[0].Pw).
		SetUsername(config.Mqtt[0].User).
		SetAutoReconnect(true).
		AddBroker(config.Mqtt[0].Broker)

	mqtt := paho.NewClient(options)
	if token := mqtt.Connect(); token.Wait() && token.Error() != nil {
		t.Error(token.Error())
		return
	}
	defer mqtt.Disconnect(0)

	mux := sync.Mutex{}
	messages := []string{}
	token := mqtt.Subscribe("processes/"+networkId+"/cmd/deployment", 2, func(client paho.Client, message paho.Message) {
		mux.Lock()
		defer mux.Unlock()
		messages = append(messages, string(message.Payload()))
	})
	if token.Wait() && token.Error() != nil {
		t.Error(token.Error())
		return
	}

	t.Run("deploy process", testDeployAspectListProcess(config.ApiPort, networkId))

	var message string
	for start := time.Now(); message == "" && time.Since(start) < 10*time.Second; {
		time.Sleep(200 * time.Millisecond)
		mux.Lock()
		if len(messages) > 0 {
			message = messages[0]
		}
		mux.Unlock()
	}
	if message == "" {
		t.Error("no deployment message received")
		return
	}

	deployment := model.DeploymentWithEventDesc{}
	err = json.Unmarshal([]byte(message), &deployment)
	if err != nil {
		t.Error(err)
		return
	}

	t.Run("event descriptions keep both spellings as selected", func(t *testing.T) {
		for _, spelling := range aspectSpellings {
			desc, ok := findEventDesc(deployment, spelling.eventId)
			if !ok {
				t.Errorf("%v: no event description", spelling.eventId)
				continue
			}
			expectedAspectId := ""
			if spelling.aspectId != nil {
				expectedAspectId = *spelling.aspectId
			}
			if desc.AspectId != expectedAspectId {
				t.Errorf("%v: aspect_id %#v, expected %#v", spelling.eventId, desc.AspectId, expectedAspectId)
			}
			if !reflect.DeepEqual(desc.AspectIds, spelling.aspectIds) {
				t.Errorf("%v: aspect_ids %#v, expected %#v", spelling.eventId, desc.AspectIds, spelling.aspectIds)
			}
		}
	})

	t.Run("deprecated aspect id is an alias for a single element list", func(t *testing.T) {
		for _, spelling := range aspectSpellings {
			desc, ok := findEventDesc(deployment, spelling.eventId)
			if !ok {
				t.Errorf("%v: no event description", spelling.eventId)
				continue
			}
			if !reflect.DeepEqual(desc.GetAspectIds(), spelling.folded) {
				t.Errorf("%v: folded aspects %#v, expected %#v", spelling.eventId, desc.GetAspectIds(), spelling.folded)
			}
		}
		idOnly, _ := findEventDesc(deployment, "aspect-id-only")
		listSingle, _ := findEventDesc(deployment, "aspect-ids-single")
		if !reflect.DeepEqual(idOnly.GetAspectIds(), listSingle.GetAspectIds()) {
			t.Errorf("aspect_id %#v and single element aspect_ids %#v are read differently", idOnly.GetAspectIds(), listSingle.GetAspectIds())
		}
	})

	t.Run("forwarded deployment keeps the filter criteria as selected", func(t *testing.T) {
		for _, spelling := range aspectSpellings {
			criteria, ok := findFilterCriteria(deployment, spelling.eventId)
			if !ok {
				t.Errorf("%v: no element", spelling.eventId)
				continue
			}
			if !reflect.DeepEqual(criteria.AspectId, spelling.aspectId) {
				t.Errorf("%v: aspect_id %#v, expected %#v", spelling.eventId, criteria.AspectId, spelling.aspectId)
			}
			if !reflect.DeepEqual(criteria.AspectIds, spelling.aspectIds) {
				t.Errorf("%v: aspect_ids %#v, expected %#v", spelling.eventId, criteria.AspectIds, spelling.aspectIds)
			}
		}
	})

	t.Run("message for a single aspect id is unchanged for old mgw clients", func(t *testing.T) {
		raw := struct {
			EventDescriptions []map[string]interface{} `json:"event_descriptions"`
		}{}
		err := json.Unmarshal([]byte(message), &raw)
		if err != nil {
			t.Error(err)
			return
		}
		found := false
		for _, desc := range raw.EventDescriptions {
			if desc["event_id"] != "aspect-id-only" {
				continue
			}
			found = true
			if desc["aspect_id"] != "aid1" {
				t.Errorf("aspect_id %#v", desc["aspect_id"])
			}
			if _, ok := desc["aspect_ids"]; ok {
				t.Errorf("unexpected aspect_ids %#v", desc["aspect_ids"])
			}
		}
		if !found {
			t.Error("no event description")
		}
	})

	t.Run("device-repository is asked with the aspect list of a device group event", func(t *testing.T) {
		expected := []eventmodel.FilterCriteria{{
			FunctionId: devicemodel.MEASURING_FUNCTION_PREFIX + "fid1",
			AspectId:   "aid3",
			AspectIds:  []string{"aid1", "aid2"},
		}}
		calls := devices.DeviceTypeSelectableCriteria()
		for _, criteria := range calls {
			if reflect.DeepEqual(criteria, expected) {
				return
			}
		}
		t.Errorf("no GetDeviceTypeSelectables call with %#v; got %#v", expected, calls)
	})
}

func findEventDesc(deployment model.DeploymentWithEventDesc, eventId string) (desc eventworkermodel.EventDesc, found bool) {
	for _, element := range deployment.EventDescriptions {
		if element.EventId == eventId {
			return element, true
		}
	}
	return desc, false
}

func findFilterCriteria(deployment model.DeploymentWithEventDesc, eventId string) (criteria deploymentmodel.FilterCriteria, found bool) {
	for _, element := range deployment.Elements {
		if element.ConditionalEvent != nil && element.ConditionalEvent.EventId == eventId {
			return element.ConditionalEvent.Selection.FilterCriteria, true
		}
	}
	return criteria, false
}

func testDeployAspectListProcess(port string, networkId string) func(t *testing.T) {
	return func(t *testing.T) {
		elements := []deploymentmodel.Element{}
		for _, spelling := range aspectSpellings {
			elements = append(elements, deploymentmodel.Element{
				BpmnId: "bpmnid-" + spelling.eventId,
				Name:   "event-name-" + spelling.eventId,
				ConditionalEvent: &deploymentmodel.ConditionalEvent{
					Script:        "x == 42",
					ValueVariable: "x",
					EventId:       spelling.eventId,
					Selection: deploymentmodel.Selection{
						FilterCriteria: deploymentmodel.FilterCriteria{
							CharacteristicId: strptr("cid1"),
							FunctionId:       strptr(devicemodel.MEASURING_FUNCTION_PREFIX + "fid1"),
							AspectId:         spelling.aspectId,
							AspectIds:        spelling.aspectIds,
						},
						SelectedDeviceId:  strptr("did1"),
						SelectedServiceId: strptr("sid1"),
						SelectedPath: &deviceselectionmodel.PathOption{
							Path:             "path.to.chid2",
							CharacteristicId: "cid2",
						},
					},
				},
			})
		}
		elements = append(elements, deploymentmodel.Element{
			BpmnId: "bpmnid-group",
			Name:   "event-name-group",
			ConditionalEvent: &deploymentmodel.ConditionalEvent{
				Script:        "x == 42",
				ValueVariable: "x",
				EventId:       "group",
				Selection: deploymentmodel.Selection{
					FilterCriteria: deploymentmodel.FilterCriteria{
						CharacteristicId: strptr("cid1"),
						FunctionId:       strptr(devicemodel.MEASURING_FUNCTION_PREFIX + "fid1"),
						AspectId:         strptr("aid3"),
						AspectIds:        []string{"aid1", "aid2"},
					},
					SelectedDeviceGroupId: strptr("gid1"),
				},
			},
		})

		requestBody := new(bytes.Buffer)
		err := json.NewEncoder(requestBody).Encode(deploymentmodel.Deployment{
			Id:          "test-aspect-lists",
			Version:     deploymentmodel.CurrentVersion,
			Name:        "test-aspect-lists",
			Description: "test-description",
			Diagram: deploymentmodel.Diagram{
				XmlDeployed: deploymentExampleXml,
				Svg:         "<svg></svg>",
				XmlRaw:      deploymentExampleXml,
			},
			Executable: true,
			Elements:   elements,
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
			temp, _ := io.ReadAll(resp.Body)
			t.Error(resp.StatusCode, string(temp))
		}
	}
}
