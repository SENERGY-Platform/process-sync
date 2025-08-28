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

package controller

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/SENERGY-Platform/models/go/models"
	"github.com/SENERGY-Platform/process-sync/pkg/kafka"
)

func (this *Controller) initDeviceGroupWatcher(ctx context.Context) (err error) {
	if this.config.KafkaUrl == "" || this.config.KafkaUrl == "-" {
		this.config.GetLogger().Info("skip device-group handler: missing kafka url config")
		return nil
	}
	if this.config.AuthEndpoint == "" || this.config.AuthEndpoint == "-" {
		this.config.GetLogger().Info("skip device-group handler: missing auth url config")
		return nil
	}
	return kafka.NewConsumer(ctx, this.config, this.config.DeviceGroupTopic, func(msg []byte) error {
		this.config.GetLogger().Debug("receive device-group command", "msg", string(msg))
		cmd := DeviceGroupCommand{}
		err := json.Unmarshal(msg, &cmd)
		if err != nil {
			this.config.GetLogger().Warn("unable to interpret device group command", "error", err)
			return nil //ignore uninterpretable commands
		}
		if cmd.Command != "PUT" {
			return nil // ignore unhandled command types
		}
		token, err := this.security.GetAdminToken()
		if err != nil {
			return err
		}
		return this.UpdateDeviceGroup(token, cmd.Id)
	}, func(err error) (fatal bool) {
		if errors.Is(err, kafka.FetchError) {
			this.config.GetLogger().Error("kafka fetch error", "error", err)
			log.Fatal(err)
			return true
		}
		if errors.Is(err, kafka.CommitError) {
			this.config.GetLogger().Error("kafka commit error", "error", err)
			log.Fatal(err)
			return true
		}
		if errors.Is(err, kafka.HandlerError) {
			this.config.GetLogger().Error("kafka handler error", "error", err)
			log.Fatal(err)
			return true
		}
		return true
	})
}

func (this *Controller) UpdateDeviceGroup(token string, deviceGroupId string) error {
	list, err := this.db.ListDeploymentMetadataByEventDeviceGroupId(deviceGroupId)
	if err != nil {
		return err
	}
	for _, element := range list {
		withEvents, err := this.deploymentModelWithEventDescriptions(token, element.DeploymentModel.Deployment)
		if err != nil {
			return err
		}
		err = this.mgw.SendDeploymentEventUpdateCommand(element.NetworkId,
			element.CamundaDeploymentId,
			withEvents.EventDescriptions,
			withEvents.DeviceIdToLocalId,
			withEvents.ServiceIdToLocalId)
		if err != nil {
			return err
		}
	}
	return nil
}

type DeviceGroupCommand struct {
	Command     string             `json:"command"`
	Id          string             `json:"id"`
	Owner       string             `json:"owner"`
	DeviceGroup models.DeviceGroup `json:"device_group"`
}
