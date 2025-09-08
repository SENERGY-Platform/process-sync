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

	"github.com/SENERGY-Platform/process-sync/pkg/database"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
)

type WardenDb struct {
	db        database.Database
	batchsize int64
}

func (this *WardenDb) ListDeploymentWardenInfo() iter.Seq2[DeploymentWardenInfo, error] {
	var offset int64 = 0
	return func(yield func(DeploymentWardenInfo, error) bool) {
		finished := false
		for !finished {
			batch, err := this.db.FindDeploymentWardenInfo(model.DeploymentWardenInfoQuery{
				Limit:  this.batchsize,
				Offset: offset,
			})
			if err != nil {
				yield(model.DeploymentWardenInfo{}, err)
				return
			}
			for _, depl := range batch {
				if !yield(depl, nil) {
					return
				}
			}
			offset += this.batchsize
			if len(batch) < int(this.batchsize) {
				finished = true
			}
		}
	}
}

func (this *WardenDb) ListWardenInfo() iter.Seq2[WardenInfo, error] {
	var offset int64 = 0
	return func(yield func(WardenInfo, error) bool) {
		finished := false
		for !finished {
			batch, err := this.db.FindWardenInfo(model.WardenInfoQuery{
				Limit:  this.batchsize,
				Offset: offset,
			})
			if err != nil {
				yield(model.WardenInfo{}, err)
				return
			}
			for _, instance := range batch {
				if !yield(instance, nil) {
					return
				}
			}
			offset += this.batchsize
			if len(batch) < int(this.batchsize) {
				finished = true
			}
		}
	}
}

func (this *WardenDb) GetWardenInfoForInstance(instance model.ProcessInstance) ([]WardenInfo, error) {
	return this.db.FindWardenInfo(model.WardenInfoQuery{NetworkIds: []string{instance.NetworkId}, BusinessKeys: []string{instance.BusinessKey}})
}

func (this *WardenDb) GetWardenInfoForDeploymentId(deploymentId string) ([]WardenInfo, error) {
	return this.db.FindWardenInfo(model.WardenInfoQuery{ProcessDeploymentIds: []string{deploymentId}})
}

func (this *WardenDb) GetDeploymentWardenInfo(info WardenInfo) (result DeploymentWardenInfo, exists bool, err error) {
	deplInfo, exists, err := this.db.GetDeploymentWardenInfoByDeploymentId(info.NetworkId, info.ProcessDeploymentId)
	if err != nil {
		return result, exists, err
	}
	if !exists {
		return result, false, nil
	}
	return deplInfo, true, nil
}

func (this *WardenDb) SetWardenInfo(info WardenInfo) error {
	return this.db.SetWardenInfo(info)
}

func (this *WardenDb) RemoveWardenInfo(info WardenInfo) error {
	return this.RemoveWardenInfoByBusinessKey(info.NetworkId, info.BusinessKey)
}

func (this *WardenDb) RemoveWardenInfoByBusinessKey(networkId string, businessKey string) error {
	return this.db.RemoveWardenInfo(networkId, businessKey)
}

func (this *WardenDb) SetDeploymentWardenInfo(info model.DeploymentWardenInfo) error {
	return this.db.SetDeploymentWardenInfo(info)
}

func (this *WardenDb) RemoveDeploymentWardenById(networkId string, deploymentId string) error {
	return this.db.RemoveDeploymentWardenInfo(networkId, deploymentId)
}
