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
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"time"

	"github.com/SENERGY-Platform/process-sync/pkg/database"
)

type WardenInfoInterface interface {
	IsOlderThen(time.Duration) bool
	Validate() error
}

type ProcessesInterface[WardenInfo WardenInfoInterface, DeploymentWardenInfo, ProcessInstance any, History any, Incident any] interface {
	AllInstances() iter.Seq2[ProcessInstance, error]
	GetInstances(WardenInfo) ([]ProcessInstance, error)
	GetYoungestProcessInstance(instances []ProcessInstance) (ProcessInstance, error)
	InstanceIsOlderThen(ProcessInstance, time.Duration) (bool, error)
	InstanceIsCreatedWithWardenHandlingIntended(instance ProcessInstance) bool
	InstanceIsOldPlaceholder(instance ProcessInstance) (bool, error)

	MarkInstanceBusinessKeyAsWardenHandled(businessKey string) string

	GetInstanceHistories(WardenInfo) ([]History, error)
	GetYoungestHistory([]History) (History, error) //by start time?
	HistoryIsOlderThen(History, time.Duration) (bool, error)

	GetIncidents(History) ([]Incident, error)
	GetYoungestIncident([]Incident) (Incident, error)
	IncidentIsOlderThen(Incident, time.Duration) bool

	Start(WardenInfo) error
	Stop(ProcessInstance) error

	DeploymentExistsForWarden(WardenInfo) (exist bool, err error)
	DeploymentExistsForDeploymentWarden(DeploymentWardenInfo) (exist bool, err error)
	Redeploy(DeploymentWardenInfo) error
}

type DbInterface[WardenInfo WardenInfoInterface, DeploymentWardenInfo any, ProcessInstance any] interface {
	ListWardenInfo() iter.Seq2[WardenInfo, error]
	GetWardenInfoForInstance(ProcessInstance) ([]WardenInfo, error)
	GetWardenInfoForDeploymentId(deploymentId string) ([]WardenInfo, error)
	GetDeploymentWardenInfo(WardenInfo) (result DeploymentWardenInfo, exist bool, err error)
	SetWardenInfo(WardenInfo) error
	RemoveWardenInfo(WardenInfo) error
	SetDeploymentWardenInfo(info DeploymentWardenInfo) error
	RemoveDeploymentWardenById(networkId string, deploymentId string) error
	RemoveWardenInfoByBusinessKey(networkId string, businessKey string) error
	ListDeploymentWardenInfo() iter.Seq2[DeploymentWardenInfo, error]
	UpdateWardenInfoDeploymentId(networkId string, oldDeploymentId string, newDeploymentId string) error
}

type Config struct {
	Interval          time.Duration
	AgeGate           time.Duration
	RunDbLoop         bool
	RunProcessLoop    bool
	RunDeploymentLoop bool
	Logger            *slog.Logger
}

type GenericWarden[WardenInfo WardenInfoInterface, DeploymentWardenInfo any, ProcessInstance any, History any, Incident any] struct {
	processes ProcessesInterface[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident]
	wardendb  DbInterface[WardenInfo, DeploymentWardenInfo, ProcessInstance]
	config    Config
}

func NewGeneric[WardenInfo WardenInfoInterface, DeploymentWardenInfo any, ProcessInstance any, History any, Incident any](config Config, processes ProcessesInterface[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident], db DbInterface[WardenInfo, DeploymentWardenInfo, ProcessInstance]) *GenericWarden[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident] {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	return &GenericWarden[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident]{
		processes: processes,
		wardendb:  db,
		config:    config,
	}
}

func (this *GenericWarden[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident]) AddDeploymentWarden(info DeploymentWardenInfo) error {
	return this.wardendb.SetDeploymentWardenInfo(info)
}

func (this *GenericWarden[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident]) RemoveDeploymentWardenById(networkId string, deploymentId string) error {
	return this.wardendb.RemoveDeploymentWardenById(networkId, deploymentId)
}

// MarkInstanceAsWardenHandled is intended to be used before the instance is handled by other services
func (this *GenericWarden[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident]) MarkInstanceBusinessKeyAsWardenHandled(businessKey string) string {
	return this.processes.MarkInstanceBusinessKeyAsWardenHandled(businessKey)
}

func (this *GenericWarden[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident]) InstanceIsCreatedWithWardenHandlingIntended(instance ProcessInstance) bool {
	return this.processes.InstanceIsCreatedWithWardenHandlingIntended(instance)
}

func (this *GenericWarden[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident]) AddInstanceWarden(info WardenInfo) error {
	err := info.Validate()
	if err != nil {
		return err
	}
	return this.wardendb.SetWardenInfo(info)
}

func (this *GenericWarden[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident]) RemoveInstanceWardenByBusinessKey(networkId string, businessKey string) error {
	return this.wardendb.RemoveWardenInfoByBusinessKey(networkId, businessKey)
}

func (this *GenericWarden[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident]) RemoveInstanceWarden(info WardenInfo) error {
	return this.wardendb.RemoveWardenInfo(info)
}

func (this *GenericWarden[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident]) RemoveWardenInfoByInstance(instance ProcessInstance) error {
	infos, err := this.wardendb.GetWardenInfoForInstance(instance)
	if err != nil {
		return err
	}
	for _, info := range infos {
		err = this.RemoveInstanceWarden(info)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *GenericWarden[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident]) Start(ctx context.Context) error {
	if !this.config.RunDbLoop && !this.config.RunProcessLoop && !this.config.RunDeploymentLoop {
		return errors.New("no warden loops enabled")
	}
	if this.config.Interval == 0 {
		return errors.New("invalid warden interval")
	}
	ticker := time.NewTicker(this.config.Interval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				now := time.Now()
				this.config.Logger.Debug("start warden loop")
				if this.config.RunDeploymentLoop {
					err := this.LoopDeploymentWardenDb()
					if err != nil {
						this.config.Logger.Error("error in deployment loop", "error", err)
					}
				}
				if this.config.RunDbLoop {
					err := this.LoopWardenDb()
					if err != nil {
						this.config.Logger.Error("error in wardendb loop", "error", err)
					}
				}
				if this.config.RunProcessLoop {
					err := this.LoopProcesses()
					if err != nil {
						this.config.Logger.Error("error in process loop", "error", err)
					}
				}

				this.config.Logger.Debug("warden loop end", "loop-duration", time.Since(now).String())
			}
		}
	}()
	return nil
}

func (this *GenericWarden[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident]) LoopDeploymentWardenDb() error {
	it := this.wardendb.ListDeploymentWardenInfo()
	for info, err := range it {
		if err != nil {
			return err
		}
		err = this.CheckDeploymentWardenInfo(info)
		if err != nil {
			this.config.Logger.Error("error in warden info check", "error", err)
		}
	}
	return nil
}

func (this *GenericWarden[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident]) LoopWardenDb() error {
	it := this.wardendb.ListWardenInfo()
	for info, err := range it {
		if err != nil {
			return err
		}
		err = this.CheckWardenInfo(info)
		if err != nil {
			this.config.Logger.Error("error in warden info check", "error", err)
		}
	}
	return nil
}

func (this *GenericWarden[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident]) LoopProcesses() error {
	it := this.processes.AllInstances()
	for instance, err := range it {
		if err != nil {
			return err
		}
		err = this.CheckProcessInstance(instance)
		if err != nil {
			this.config.Logger.Error("error in process instance check", "error", err)
		}
	}
	return nil
}

func (this *GenericWarden[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident]) CheckWardenInfo(info WardenInfo) error {
	this.config.Logger.Debug("check warden info", "info", fmt.Sprintf("%+v", info))
	if info.IsOlderThen(this.config.AgeGate) {
		this.config.Logger.Debug("warden age is immature --> no action", "info", fmt.Sprintf("%+v", info))
		return nil
	}
	instances, err := this.processes.GetInstances(info)
	if err != nil {
		return err
	}
	switch len(instances) {
	case 0:
		return this.missingInstance(info)
	case 1:
		isOldPlaceholder, err := this.processes.InstanceIsOldPlaceholder(instances[0])
		if err != nil {
			return err
		}
		if isOldPlaceholder {
			return this.oldPlaceholderInstance(info, instances[0])
		}
		return nil // expected state; incidents would cause the stopping of the process-instance, but that is handled by other services
	default:
		return this.duplicateInstances(info, instances)
	}
}

func (this *GenericWarden[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident]) CheckProcessInstance(instance ProcessInstance) error {
	infos, err := this.wardendb.GetWardenInfoForInstance(instance)
	if err != nil {
		return err
	}
	if len(infos) == 0 {
		if !this.processes.InstanceIsCreatedWithWardenHandlingIntended(instance) {
			this.config.Logger.Debug("process instance without warden info, but is recognised as legacy instance --> no handling", "instance", fmt.Sprintf("%+v", instance))
			return nil
		}
		this.config.Logger.Debug("process instance without warden info --> remove", "instance", fmt.Sprintf("%+v", instance))
		return this.processes.Stop(instance)
	}
	return nil
}

func (this *GenericWarden[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident]) oldPlaceholderInstance(info WardenInfo, instance ProcessInstance) error {
	this.config.Logger.Debug("old placeholder instance --> start process instance", "info", fmt.Sprintf("%+v", info))
	err := this.processes.Stop(instance)
	if err != nil {
		this.config.Logger.Error("unable to stop old placeholder instance", "error", err)
	}
	return this.processes.Start(info)
}

func (this *GenericWarden[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident]) missingInstance(info WardenInfo) error {
	histories, err := this.processes.GetInstanceHistories(info)
	if err != nil {
		return err
	}
	if len(histories) == 0 {
		exists, err := this.processes.DeploymentExistsForWarden(info)
		if err != nil {
			this.config.Logger.Error("unable to get deployment for warden info", "error", err)
			return nil
		}
		if !exists {
			this.config.Logger.Debug("missing process instance but warden info has become invalid, this may happen if a process-deployment is deleted --> remove info from warden", "info", fmt.Sprintf("%+v", info), "validation-result", err)
			err = this.tryRedeployProcess(info)
			if err != nil {
				switch {
				case errors.Is(err, ErrRetry):
					this.config.Logger.Error("unable to redeploy missing process-deployment --> retry later", "error", err)
					return nil
				case errors.Is(err, ErrFinal):
					this.config.Logger.Error("unable to redeploy missing process-deployment --> remove warden", "error", err)
					return this.RemoveInstanceWarden(info)
				default:
					this.config.Logger.Error("unable to redeploy missing process-deployment --> remove warden", "error", err)
					return this.RemoveInstanceWarden(info)
				}
			}
		}
		this.config.Logger.Debug("missing process instance --> start process instance", "info", fmt.Sprintf("%+v", info))
		err = this.processes.Start(info)
		return err
	}
	history, err := this.processes.GetYoungestHistory(histories)
	if err != nil {
		return err
	}
	isOlder, err := this.processes.HistoryIsOlderThen(history, this.config.AgeGate)
	if err != nil {
		return err
	}
	if !isOlder {
		this.config.Logger.Debug("youngest process history change is immature --> no action", "info", fmt.Sprintf("%+v", info))
		return nil
	}
	incidents, err := this.processes.GetIncidents(history)
	if err != nil {
		return err
	}
	if len(incidents) == 0 {
		this.config.Logger.Debug("process finished without incident --> remove info from warden", "info", fmt.Sprintf("%+v", info))
		return this.RemoveInstanceWarden(info)
	}
	youngest, err := this.processes.GetYoungestIncident(incidents)
	if err != nil {
		return err
	}
	if !this.processes.IncidentIsOlderThen(youngest, this.config.AgeGate) {
		this.config.Logger.Debug("youngest process incident change is immature --> no action", "info", fmt.Sprintf("%+v", info))
		return nil
	}
	this.config.Logger.Debug("process finished with incident --> restart", "info", fmt.Sprintf("%+v", info))
	err = this.processes.Start(info)
	return err
}

func (this *GenericWarden[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident]) duplicateInstances(info WardenInfo, instances []ProcessInstance) error {
	youngest, err := this.processes.GetYoungestProcessInstance(instances)
	if err != nil {
		return err
	}
	isOlder, err := this.processes.InstanceIsOlderThen(youngest, this.config.AgeGate)
	if err != nil {
		return err
	}
	if !isOlder {
		this.config.Logger.Debug("youngest process instance change is immature --> no action", "info", fmt.Sprintf("%+v", info))
		return nil
	}
	for i, instance := range instances {
		if i == 0 {
			continue
		}
		this.config.Logger.Debug("duplicate process instance --> stop", "info", fmt.Sprintf("%+v", info), "instance", fmt.Sprintf("%+v", instance))
		err := this.processes.Stop(instance)
		if err != nil {
			return err
		}
	}
	return nil
}

var ErrRetry = fmt.Errorf("will be retried")
var ErrFinal = fmt.Errorf("will not be retried")

func (this *GenericWarden[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident]) RemoveDeployment(deploymentId string) error {
	infos, err := this.wardendb.GetWardenInfoForDeploymentId(deploymentId)
	if err != nil {
		return err
	}
	for _, info := range infos {
		err = this.RemoveInstanceWarden(info)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *GenericWarden[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident]) tryRedeployProcess(info WardenInfo) error {
	depl, exists, err := this.wardendb.GetDeploymentWardenInfo(info)
	if errors.Is(err, database.ErrNotFound) {
		return errors.Join(err, ErrFinal)
	}
	if err != nil {
		return errors.Join(err, ErrRetry)
	}
	if !exists {
		return fmt.Errorf("no deployment found for warden info (%w)", ErrFinal)
	}
	return this.processes.Redeploy(depl)
}

func (this *GenericWarden[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident]) CheckDeploymentWardenInfo(info DeploymentWardenInfo) error {
	exists, err := this.processes.DeploymentExistsForDeploymentWarden(info)
	if err != nil {
		return err
	}
	if !exists {
		return this.processes.Redeploy(info)
	}
	return nil
}

func (this *GenericWarden[WardenInfo, DeploymentWardenInfo, ProcessInstance, History, Incident]) UpdateWardenInfoDeploymentId(networkId string, oldDeploymentId string, newDeploymentId string) error {
	return this.wardendb.UpdateWardenInfoDeploymentId(networkId, oldDeploymentId, newDeploymentId)
}
