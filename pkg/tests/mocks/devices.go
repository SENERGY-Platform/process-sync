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

package mocks

import (
	eventmodel "github.com/SENERGY-Platform/event-deployment/lib/model"
	"github.com/SENERGY-Platform/process-deployment/lib/auth"
	"github.com/SENERGY-Platform/process-deployment/lib/model/devicemodel"
)

type Devices struct{}

func (this *Devices) GetDeviceInfosOfGroup(groupId string) (devices []eventmodel.Device, deviceTypeIds []string, err error, code int) {
	return []eventmodel.Device{
		{
			Id:           "did1",
			Name:         "test-device-did1",
			DeviceTypeId: "dt1",
		},
		{
			Id:           "did2",
			Name:         "test-device-did2",
			DeviceTypeId: "dt1",
		},
	}, []string{"dt1"}, nil, 200
}

func (this *Devices) GetDeviceInfosOfDevices(deviceIds []string) (devices []eventmodel.Device, deviceTypeIds []string, err error, code int) {
	for _, id := range deviceIds {
		devices = append(devices, eventmodel.Device{
			Id:           id,
			Name:         "test-device-" + id,
			DeviceTypeId: "dt1",
		})
	}
	return devices, []string{"dt1"}, nil, 200
}

func (this *Devices) GetDevice(token auth.Token, id string) (devicemodel.Device, error, int) {
	return devicemodel.Device{
		Id:           id,
		LocalId:      "l" + id,
		Name:         "test-device-" + id,
		DeviceTypeId: "dt1",
	}, nil, 200
}

func (this *Devices) GetService(token auth.Token, id string) (devicemodel.Service, error, int) {
	return devicemodel.Service{
		Id:          id,
		LocalId:     "l" + id,
		Name:        "test-service-" + id,
		Interaction: devicemodel.EVENT_AND_REQUEST,
		ProtocolId:  "pid",
		Outputs: []devicemodel.Content{
			{
				ContentVariable: devicemodel.ContentVariable{
					CharacteristicId: "cid2",
					Name:             "foo",
					FunctionId:       devicemodel.MEASURING_FUNCTION_PREFIX + "fid1",
					AspectId:         "aid1",
				},
			},
		},
	}, nil, 200
}

func (this *Devices) GetDeviceTypeSelectables(criteria []eventmodel.FilterCriteria) (result []eventmodel.DeviceTypeSelectable, err error, code int) {
	result = []eventmodel.DeviceTypeSelectable{
		{
			DeviceTypeId: "dt1",
			ServicePathOptions: map[string][]eventmodel.ServicePathOption{
				"sid1": {
					{
						ServiceId:        "sid1",
						Path:             "path.to.chid2",
						CharacteristicId: "cid1",
						FunctionId:       devicemodel.MEASURING_FUNCTION_PREFIX + "fid1",
					},
				},
				"sid2": {
					{
						ServiceId:        "sid2",
						Path:             "path.to.chid3",
						CharacteristicId: "cid2",
						FunctionId:       devicemodel.MEASURING_FUNCTION_PREFIX + "fid1",
					},
					{
						ServiceId:        "sid2",
						Path:             "path.to.chid4",
						CharacteristicId: "cid3",
						FunctionId:       devicemodel.MEASURING_FUNCTION_PREFIX + "fid1",
					},
				},
			},
		},
	}
	return result, nil, 200
}
