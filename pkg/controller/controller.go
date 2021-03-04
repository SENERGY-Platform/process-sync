package controller

import (
	"context"
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/database"
	"github.com/SENERGY-Platform/process-sync/pkg/database/mongo"
	"github.com/SENERGY-Platform/process-sync/pkg/mgw"
)

type Controller struct {
	config configuration.Config
	mgw    *mgw.Mgw
	db     database.Database
}

func New(config configuration.Config, ctx context.Context) (ctrl *Controller, err error) {
	ctrl = &Controller{config: config}
	if err != nil {
		return ctrl, err
	}
	ctrl.mgw, err = mgw.New(config, ctx, ctrl)
	if err != nil {
		return ctrl, err
	}
	ctrl.db, err = mongo.New(config)
	if err != nil {
		return ctrl, err
	}
	return ctrl, nil
}
