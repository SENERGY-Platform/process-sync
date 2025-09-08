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

package mongo

import (
	"github.com/SENERGY-Platform/process-sync/pkg/model"
)

func (this *Mongo) SetDeploymentWardenInfo(info model.DeploymentWardenInfo) error {
	//TODO
	panic("implement me")
}

func (this *Mongo) RemoveDeploymentWardenInfo(networkId string, deploymentId string) error {
	//TODO
	panic("implement me")
}

func (this *Mongo) GetDeploymentWardenInfoByDeploymentId(networkId string, deploymentId string) (info model.DeploymentWardenInfo, exists bool, err error) {
	//TODO
	panic("implement me")
}

func (this *Mongo) SetWardenInfo(info model.WardenInfo) error {
	//TODO
	panic("implement me")
}

func (this *Mongo) RemoveWardenInfo(networkId string, businessKey string) error {
	//TODO
	panic("implement me")
}

func (this *Mongo) FindWardenInfo(query model.WardenInfoQuery) ([]model.WardenInfo, error) {
	//TODO
	panic("implement me")
}
