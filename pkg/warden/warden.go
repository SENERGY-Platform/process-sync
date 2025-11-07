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

package warden

import (
	"github.com/SENERGY-Platform/process-sync/pkg/database"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"github.com/SENERGY-Platform/service-commons/pkg/cache"
)

type WardenInfo = model.WardenInfo
type DeploymentWardenInfo = model.DeploymentWardenInfo

type Warden = *GenericWarden[WardenInfo, DeploymentWardenInfo, model.ProcessInstance, model.HistoricProcessInstance, model.Incident]

func New(config Config, ctrl Controller, db database.Database) (Warden, error) {
	c, err := cache.New(cache.Config{})
	if err != nil {
		return nil, err
	}
	return NewGeneric(config, &Processes{
		db:        db,
		batchsize: 100,
		config:    config,
		cache:     c,
		ctrl:      ctrl,
	}, &WardenDb{
		db:        db,
		batchsize: 100,
		config:    config,
	}), nil
}
