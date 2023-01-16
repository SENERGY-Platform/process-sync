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

package controller

import (
	"context"
	"errors"
	"github.com/SENERGY-Platform/process-deployment/lib/auth"
	"github.com/SENERGY-Platform/process-deployment/lib/config"
	"github.com/SENERGY-Platform/process-deployment/lib/devices"
	"github.com/SENERGY-Platform/process-deployment/lib/interfaces"
	"github.com/SENERGY-Platform/process-deployment/lib/model/devicemodel"
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/database"
	"github.com/SENERGY-Platform/process-sync/pkg/database/mongo"
	"github.com/SENERGY-Platform/process-sync/pkg/kafka"
	"github.com/SENERGY-Platform/process-sync/pkg/mgw"
	"github.com/SENERGY-Platform/process-sync/pkg/security"
	"net/http"
	"time"
)

type Controller struct {
	config                 configuration.Config
	mgw                    *mgw.Mgw
	db                     database.Database
	security               Security
	devicerepo             Devices
	deploymentDoneNotifier interfaces.Producer
}

type Devices interface {
	GetDevice(token auth.Token, id string) (devicemodel.Device, error, int)
	GetService(token auth.Token, id string) (devicemodel.Service, error, int)
}

type Security interface {
	CheckBool(token string, kind string, id string, rights string) (allowed bool, err error)
	CheckMultiple(token string, kind string, ids []string, rights string) (result map[string]bool, err error)
	List(token string, resource string, limit string, offset string, rights string) (result []security.ListElement, err error)
	ListElements(token string, resource string, limit string, offset string, rights string, result interface{}) (err error)
}

func NewDefault(conf configuration.Config, ctx context.Context) (ctrl *Controller, err error) {
	db, err := mongo.New(conf)
	if err != nil {
		return ctrl, err
	}
	d, err := devices.Factory.New(context.Background(), &config.ConfigStruct{DeviceRepoUrl: conf.DeviceRepoUrl})
	if err != nil {
		return ctrl, err
	}
	return New(conf, ctx, db, security.New(conf), d)
}

func New(config configuration.Config, ctx context.Context, db database.Database, security Security, devicerepo Devices) (ctrl *Controller, err error) {
	ctrl = &Controller{config: config, db: db, security: security, devicerepo: devicerepo}
	if err != nil {
		return ctrl, err
	}
	if config.KafkaUrl != "" && config.KafkaUrl != "-" {
		ctrl.deploymentDoneNotifier, err = kafka.NewProducer(ctx, config.KafkaUrl, config.ProcessDeploymentDoneTopic, config.Debug)
		if err != nil {
			return ctrl, err
		}
	}
	ctrl.mgw, err = mgw.New(config, ctx, ctrl)
	if err != nil {
		return ctrl, err
	}
	return ctrl, nil
}

var IsPlaceholderProcessErr = errors.New("is placeholder process")
var IsMarkedForDeleteErr = errors.New("is market for deletion")
var HistoryMayOnlyDeletedIfFinishedOrPlaceholderErr = errors.New("history may only deleted if the process instance is finished or the element is a placeholder")

func (this *Controller) SetErrCode(err error) int {
	switch err {
	case nil:
		return http.StatusOK
	case database.ErrNotFound:
		return http.StatusNotFound
	case IsPlaceholderProcessErr:
		return http.StatusBadRequest
	case IsMarkedForDeleteErr:
		return http.StatusBadRequest
	case HistoryMayOnlyDeletedIfFinishedOrPlaceholderErr:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func (this *Controller) RemoveOldEntities(maxAge time.Duration) error {
	return this.db.RemoveOldElements(maxAge)
}
