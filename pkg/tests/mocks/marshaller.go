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
	"context"
	"github.com/SENERGY-Platform/event-deployment/lib/config"
	"github.com/SENERGY-Platform/event-deployment/lib/interfaces"
	eventmodel "github.com/SENERGY-Platform/event-deployment/lib/model"
)

type Marshaller struct {
}

func (this *Marshaller) FindPath(serviceId string, characteristicId string) (path string, serviceCharacteristicId string, err error) {
	return "path.to.chid2", "cid2", nil
}

func (this *Marshaller) FindPathOptions(deviceTypeIds []string, functionId string, aspectId string, characteristicsIdFilter []string, withEnvelope bool) (result map[string][]eventmodel.PathOptionsResultElement, err error) {
	result = map[string][]eventmodel.PathOptionsResultElement{}
	for _, dt := range deviceTypeIds {
		result[dt] = []eventmodel.PathOptionsResultElement{
			{
				ServiceId: "sid1",
				JsonPath:  []string{"path.to.chid2"},
				PathToCharacteristicId: map[string]string{
					"path.to.chid2": "cid1",
				},
			},
			{
				ServiceId: "sid2",
				JsonPath:  []string{"path.to.chid3", "path.to.chid4"},
				PathToCharacteristicId: map[string]string{
					"path.to.chid3": "cid2",
					"path.to.chid4": "cid3",
				},
			},
		}
	}
	return result, nil
}

func (this *Marshaller) New(ctx context.Context, config config.Config) (interfaces.Marshaller, error) {
	return this, nil
}
