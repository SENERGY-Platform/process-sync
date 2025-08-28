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
	"iter"

	"github.com/SENERGY-Platform/process-sync/pkg/model"
)

type Db struct{}

func (this *Db) ListWardenInfo() (iter.Seq[WardenInfo], error) {
	//TODO implement me
	panic("implement me")
}

func (this *Db) GetWardenInfoForInstance(instance model.ProcessInstance) ([]WardenInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (this *Db) GetWardenInfoForDeploymentId(deploymentId string) ([]WardenInfo, error) {
	//TODO implement me
	panic("implement me")
}

func (this *Db) GetDeployment(info WardenInfo) (result model.Deployment, exist bool, err error) {
	//TODO implement me
	panic("implement me")
}

func (this *Db) SetWardenInfo(info WardenInfo) error {
	//TODO implement me
	panic("implement me")
}

func (this *Db) RemoveWardenInfo(info WardenInfo) error {
	//TODO implement me
	panic("implement me")
}
