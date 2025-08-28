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
	"time"

	"github.com/SENERGY-Platform/process-sync/pkg/model"
)

type WardenInfo struct {
	CreationTime int64 //unix timestamp

}

func (this WardenInfo) IsOlderThen(duration time.Duration) bool {
	return time.Unix(this.CreationTime, 0).Add(duration).After(time.Now())
}

type Warden = *GenericWarden[WardenInfo, model.Deployment, model.ProcessInstance, model.HistoricProcessInstance, model.Incident]

func New(config Config) Warden {
	return NewGeneric(config, &Processes{}, &Db{})
}
