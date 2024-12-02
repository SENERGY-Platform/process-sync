/*
 * Copyright 2023 InfAI (CC SES)
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

package devices

import (
	"encoding/json"
	"fmt"
	auth2 "github.com/SENERGY-Platform/event-deployment/lib/auth"
	"github.com/SENERGY-Platform/event-deployment/lib/config"
	"github.com/SENERGY-Platform/event-deployment/lib/devices"
	"github.com/SENERGY-Platform/event-deployment/lib/interfaces"
	"github.com/SENERGY-Platform/event-deployment/lib/model"
	"github.com/SENERGY-Platform/models/go/models"
	"github.com/SENERGY-Platform/process-deployment/lib/auth"
	syncconf "github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"io"
	"net/http"
	"net/url"
)

type DeviceRepo struct {
	config         syncconf.Config
	baseRepFactory BaseDeviceRepoFactory
	deviceProvider DeviceProvider
}

func New(config syncconf.Config, baseRepFactory BaseDeviceRepoFactory, deviceProvider DeviceProvider) (*DeviceRepo, error) {
	if baseRepFactory == nil {
		baseRepFactory = DefaultBaseDeviceRepoFactory
	}
	return &DeviceRepo{config: config, baseRepFactory: baseRepFactory, deviceProvider: deviceProvider}, nil
}

type BaseDeviceRepoFactory = func(token string, deviceRepoUrl string) interfaces.Devices

func DefaultBaseDeviceRepoFactory(token string, deviceRepoUrl string) interfaces.Devices {
	return devices.NewWithAuth(&config.ConfigStruct{DeviceRepositoryUrl: deviceRepoUrl}, AuthFromToken{Token: token})
}

type DeviceProvider = func(token string, baseUrl string, deviceId string) (result models.Device, err error, code int)

func DefaultDeviceProvider(token string, baseUrl string, deviceId string) (result models.Device, err error, code int) {
	req, err := http.NewRequest(http.MethodGet, baseUrl+"/devices/"+url.PathEscape(deviceId), nil)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	req.Header.Set("Authorization", token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		temp, _ := io.ReadAll(resp.Body) //read error response end ensure that resp.Body is read to EOF
		return result, fmt.Errorf("unexpected statuscode %v: %v", resp.StatusCode, string(temp)), resp.StatusCode
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		_, _ = io.ReadAll(resp.Body) //ensure resp.Body is read to EOF
		return result, err, http.StatusInternalServerError
	}
	return
}

type AuthFromToken struct {
	Token string
}

func (this AuthFromToken) Ensure() (token auth2.AuthToken, err error) {
	return auth2.AuthToken(this.Token), nil
}

func (this *DeviceRepo) WithPresetToken(token auth.Token) interfaces.Devices {
	return this.baseRepFactory(token.Token, this.config.DeviceRepoUrl)
}

func (this *DeviceRepo) GetDeviceInfosOfGroup(token auth.Token, groupId string) (devices []model.Device, deviceTypeIds []string, err error, code int) {
	return this.WithPresetToken(token).GetDeviceInfosOfGroup(groupId)
}

func (this *DeviceRepo) GetDeviceInfosOfDevices(token auth.Token, deviceIds []string) (devices []model.Device, deviceTypeIds []string, err error, code int) {
	return this.WithPresetToken(token).GetDeviceInfosOfDevices(deviceIds)
}

func (this *DeviceRepo) GetDeviceTypeSelectables(token auth.Token, criteria []model.FilterCriteria) (result []model.DeviceTypeSelectable, err error, code int) {
	return this.WithPresetToken(token).GetDeviceTypeSelectables(criteria)
}

func (this *DeviceRepo) GetConcept(token auth.Token, conceptId string) (result model.Concept, err error, code int) {
	return this.WithPresetToken(token).GetConcept(conceptId)
}

func (this *DeviceRepo) GetFunction(token auth.Token, functionId string) (result model.Function, err error, code int) {
	return this.WithPresetToken(token).GetFunction(functionId)
}

func (this *DeviceRepo) GetService(token auth.Token, serviceId string) (result models.Service, err error, code int) {
	return this.WithPresetToken(token).GetService(serviceId)
}

func (this *DeviceRepo) GetDevice(token auth.Token, deviceId string) (result models.Device, err error, code int) {
	return this.deviceProvider(token.Token, this.config.DeviceRepoUrl, deviceId)
}
