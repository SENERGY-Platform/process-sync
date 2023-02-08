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

package transformer

import (
	"errors"
	"github.com/SENERGY-Platform/event-deployment/lib/events/conditionalevents"
	"github.com/SENERGY-Platform/event-deployment/lib/interfaces"
	"github.com/SENERGY-Platform/process-deployment/lib/model/importmodel"
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"net/http"
)

type DeviceRepoFactory = func(token string, deviceRepoUrl string, permissionsSearchUrl string) interfaces.Devices

func New(config configuration.Config, repoFactory DeviceRepoFactory, token string) *conditionalevents.Transformer {
	return conditionalevents.NewTransformer(repoFactory(token, config.DeviceRepoUrl, config.PermissionsUrl), Imports{})
}

type Imports struct {
}

func (this Imports) GetTopic(user string, importId string) (topic string, err error, code int) {
	return "", errors.New("imports not supported for fog deployments"), http.StatusBadRequest
}

func (this Imports) GetImportInstance(user string, importId string) (importInstance importmodel.Import, err error, code int) {
	return importInstance, errors.New("imports not supported for fog deployments"), http.StatusBadRequest
}

func (this Imports) GetImportType(user string, importTypeId string) (importType importmodel.ImportType, err error, code int) {
	return importType, errors.New("imports not supported for fog deployments"), http.StatusBadRequest
}
