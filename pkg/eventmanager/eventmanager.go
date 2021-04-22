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

package eventmanager

import (
	"context"
	"errors"
	"github.com/SENERGY-Platform/event-deployment/lib/config"
	"github.com/SENERGY-Platform/event-deployment/lib/devices"
	"github.com/SENERGY-Platform/event-deployment/lib/events"
	"github.com/SENERGY-Platform/event-deployment/lib/interfaces"
	"github.com/SENERGY-Platform/event-deployment/lib/marshaller"
	"github.com/SENERGY-Platform/process-deployment/lib/model/deploymentmodel/v2"
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"net/http"
)

var DefaultMarshallerFactory interfaces.MarshallerFactory = marshaller.Factory
var DefaultDevicesFactory func(config config.Config, auth devices.Auth) interfaces.Devices = func(config config.Config, auth devices.Auth) interfaces.Devices {
	return devices.NewWithAuth(config, auth)
}

var MarshallerFactory interfaces.MarshallerFactory = DefaultMarshallerFactory
var DevicesFactory func(config config.Config, auth devices.Auth) interfaces.Devices = DefaultDevicesFactory

func GetAnalyticsDeploymentsForMessageEvents(conf configuration.Config, token string, deployment deploymentmodel.Deployment) (result []model.AnalyticsRecord, err error) {
	var eventConfig = &config.ConfigStruct{
		MarshallerUrl:    conf.MarshallerUrl,
		PermSearchUrl:    conf.PermissionsUrl,
		Debug:            conf.Debug,
		AuthClientId:     "ignore",
		AuthClientSecret: "ignore",
		AuthEndpoint:     "ignore",
	}
	analyticsRecorder := &AnalyticsRecorder{}
	deviceRepo := DevicesFactory(eventConfig, TokenAuth(token))
	m, err := MarshallerFactory.New(context.Background(), eventConfig)
	if err != nil {
		return result, err
	}
	eventsInterface, err := events.Factory.New(context.Background(),
		eventConfig,
		analyticsRecorder,
		m,
		deviceRepo,
		ImportsPlaceholder{},
	)
	events, ok := eventsInterface.(*events.Events)
	if !ok {

	}
	if err != nil {
		return result, err
	}
	err = events.Deploy("", deployment)
	result = analyticsRecorder.Records
	return
}

type ImportsPlaceholder struct{}

func (_ ImportsPlaceholder) GetTopic(user string, importId string) (topic string, err error, code int) {
	return "", errors.New("no imports for fog deployments possible"), http.StatusBadRequest
}
