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
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	developerNotifications "github.com/SENERGY-Platform/developer-notifications/pkg/client"
	eventinterfaces "github.com/SENERGY-Platform/event-deployment/lib/interfaces"
	"github.com/SENERGY-Platform/event-deployment/lib/model"
	"github.com/SENERGY-Platform/models/go/models"
	"github.com/SENERGY-Platform/process-deployment/lib/auth"
	"github.com/SENERGY-Platform/process-deployment/lib/interfaces"
	"github.com/SENERGY-Platform/process-deployment/lib/model/devicemodel"
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/database"
	"github.com/SENERGY-Platform/process-sync/pkg/database/mongo"
	"github.com/SENERGY-Platform/process-sync/pkg/devices"
	"github.com/SENERGY-Platform/process-sync/pkg/kafka"
	"github.com/SENERGY-Platform/process-sync/pkg/mgw"
	"github.com/SENERGY-Platform/process-sync/pkg/security"
	"github.com/SENERGY-Platform/process-sync/pkg/warden"
)

type Controller struct {
	config                 configuration.Config
	mgw                    *mgw.Mgw
	db                     database.Database
	security               Security
	baseDeviceRepoFactory  BaseDeviceRepoFactory
	devicerepo             Devices
	deploymentDoneNotifier interfaces.Producer
	devNotifications       developerNotifications.Client
	logger                 *slog.Logger
	warden                 warden.Warden
}

type BaseDeviceRepoFactory = func(token string, deviceRepoUrl string) eventinterfaces.Devices

type DeviceProvider = func(token string, baseUrl string, deviceId string) (result models.Device, err error, code int)

type Devices interface {
	GetDeviceInfosOfGroup(token auth.Token, groupId string) (devices []model.Device, deviceTypeIds []string, err error, code int)
	GetDeviceInfosOfDevices(token auth.Token, deviceIds []string) (devices []model.Device, deviceTypeIds []string, err error, code int)
	GetDeviceTypeSelectables(token auth.Token, criteria []model.FilterCriteria) (result []model.DeviceTypeSelectable, err error, code int)
	GetConcept(token auth.Token, conceptId string) (result model.Concept, err error, code int)
	GetFunction(token auth.Token, functionId string) (result model.Function, err error, code int)
	GetService(token auth.Token, serviceId string) (result models.Service, err error, code int)
	GetDevice(token auth.Token, id string) (devicemodel.Device, error, int)
}

type Security interface {
	GetAdminToken() (token string, err error)
	CheckBool(token string, kind string, id string, rights string) (allowed bool, err error)
	CheckMultiple(token string, kind string, ids []string, rights string) (result map[string]bool, err error)
}

func NewDefault(conf configuration.Config, ctx context.Context) (ctrl *Controller, err error) {
	db, err := mongo.New(conf)
	if err != nil {
		return ctrl, err
	}
	return New(conf, ctx, db, security.New(conf), devices.DefaultBaseDeviceRepoFactory, devices.DefaultDeviceProvider)
}

func New(config configuration.Config, ctx context.Context, db database.Database, security Security, baseDeviceRepoFactory BaseDeviceRepoFactory, deviceProvider DeviceProvider) (ctrl *Controller, err error) {
	d, err := devices.New(config, baseDeviceRepoFactory, deviceProvider)
	if err != nil {
		return ctrl, err
	}
	logger := config.GetLogger()
	if info, ok := debug.ReadBuildInfo(); ok {
		logger = logger.With("go-module", info.Path)
	}

	wardenInterval, err := time.ParseDuration(config.WardenInterval)
	if err != nil {
		return ctrl, err
	}
	wardenAgeGate, err := time.ParseDuration(config.WardenAgeGate)
	if err != nil {
		return ctrl, err
	}

	ctrl = &Controller{config: config, db: db, security: security, baseDeviceRepoFactory: baseDeviceRepoFactory, devicerepo: d, logger: logger}
	w, err := warden.New(warden.Config{
		Interval:          wardenInterval,
		AgeGate:           wardenAgeGate,
		RunDbLoop:         config.RunWardenDbLoop,
		RunProcessLoop:    config.RunWardenProcessLoop,
		RunDeploymentLoop: config.RunWardenDeploymentLoop,
		Logger:            config.GetLogger(),
	}, ctrl, db)
	if err != nil {
		return ctrl, err
	}
	ctrl.warden = w
	if config.KafkaUrl != "" && config.KafkaUrl != "-" {
		ctrl.deploymentDoneNotifier, err = kafka.NewProducer(ctx, config.KafkaUrl, config.ProcessDeploymentDoneTopic, config.GetLogger(), config.InitTopics)
		if err != nil {
			return ctrl, err
		}
	}
	if config.DeveloperNotificationUrl != "" && config.DeveloperNotificationUrl != "-" {
		ctrl.devNotifications = developerNotifications.New(config.DeveloperNotificationUrl)
	}
	ctrl.mgw, err = mgw.New(config, ctx, ctrl)
	if err != nil {
		return ctrl, err
	}
	err = ctrl.initDeviceGroupWatcher(ctx)
	if err != nil {
		return ctrl, err
	}

	err = w.Start(ctx)
	if err != nil {
		return ctrl, err
	}

	return ctrl, nil
}

var IsPlaceholderProcessErr = errors.New("is placeholder process")
var IsMarkedForDeleteErr = errors.New("is market for deletion")
var HistoryMayOnlyDeletedIfFinishedOrPlaceholderErr = errors.New("history may only deleted if the process instance is finished or the element is a placeholder")
var IsMarkedAsMissingErr = errors.New("is market as missing (you may try to redeploy)")

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
