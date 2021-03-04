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
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/database"
	"github.com/SENERGY-Platform/process-sync/pkg/database/mongo"
	"github.com/SENERGY-Platform/process-sync/pkg/mgw"
	"github.com/SENERGY-Platform/process-sync/pkg/security"
	"net/http"
)

type Controller struct {
	config   configuration.Config
	mgw      *mgw.Mgw
	db       database.Database
	security Security
}

type Security interface {
	CheckBool(token string, kind string, id string, rights string) (allowed bool, err error)
}

func NewDefault(config configuration.Config, ctx context.Context) (ctrl *Controller, err error) {
	db, err := mongo.New(config)
	if err != nil {
		return ctrl, err
	}
	return New(config, ctx, db, security.New(config))
}

func New(config configuration.Config, ctx context.Context, db database.Database, security Security) (ctrl *Controller, err error) {
	ctrl = &Controller{config: config, db: db, security: security}
	if err != nil {
		return ctrl, err
	}
	ctrl.mgw, err = mgw.New(config, ctx, ctrl)
	if err != nil {
		return ctrl, err
	}
	return ctrl, nil
}

func (this *Controller) SetErrCode(err error) int {
	switch err {
	case nil:
		return http.StatusOK
	case database.ErrNotFound:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}
