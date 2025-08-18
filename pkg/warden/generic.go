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
)

type WardenInfoInterface interface {
	IsOlderThen(time.Duration) bool
}

type ProcessesInterface[WardenInfo WardenInfoInterface, Deployment any, ProcessInstance any, History any, Incident any] interface {
	AllInstances() (iter.Seq[ProcessInstance], error)
	GetInstances(WardenInfo) ([]ProcessInstance, error)
	GetYoungestProcessInstance(instances []ProcessInstance) (ProcessInstance, error)
	InstanceIsOlderThen(ProcessInstance, time.Duration) bool
	InstanceIsCreatedWithWardenHandlingIntended(instance ProcessInstance) bool

	GetInstanceHistories(WardenInfo) ([]History, error)
	GetYoungestHistory([]History) (History, error) //by start time?
	HistoryIsOlderThen(History, time.Duration) bool

	GetIncidents(History) ([]Incident, error)
	GetYoungestIncident([]Incident) (Incident, error)
	IncidentIsOlderThen(Incident, time.Duration) bool

	Start(WardenInfo) (string, error)
	Stop(ProcessInstance) error

	DeploymentExists(WardenInfo) (exist bool, err error)
	Deploy(Deployment) error
}

type DbInterface[WardenInfo WardenInfoInterface, Deployment any, ProcessInstance any] interface {
	ListWardenInfo() (iter.Seq[WardenInfo], error)
	GetWardenInfoForInstance(ProcessInstance) ([]WardenInfo, error)
	GetWardenInfoForDeploymentId(deploymentId string) ([]WardenInfo, error)
	GetDeployment(WardenInfo) (result Deployment, exist bool, err error)
	SetWardenInfo(WardenInfo) error
	RemoveWardenInfo(WardenInfo) error
}

type Config struct {
	Interval       time.Duration
	AgeGate        time.Duration
	RunDbLoop      bool
	RunProcessLoop bool
	Logger         *slog.Logger
}

type Warden[WardenInfo WardenInfoInterface, Deployment any, ProcessInstance any, History any, Incident any] struct {
	processes ProcessesInterface[WardenInfo, Deployment, ProcessInstance, History, Incident]
	db        DbInterface[WardenInfo, Deployment, ProcessInstance]
	config    Config
}

func NewGeneric[WardenInfo WardenInfoInterface, Deployment any, ProcessInstance any, History any, Incident any](config Config, processes ProcessesInterface[WardenInfo, Deployment, ProcessInstance, History, Incident], db DbInterface[WardenInfo, Deployment, ProcessInstance]) *Warden[WardenInfo, Deployment, ProcessInstance, History, Incident] {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	return &Warden[WardenInfo, Deployment, ProcessInstance, History, Incident]{
		processes: processes,
		db:        db,
		config:    config,
	}
}

func (this *Warden[WardenInfo, Deployment, ProcessInstance, History, Incident]) Add(info WardenInfo) error {
	return this.db.SetWardenInfo(info)
}

func (this *Warden[WardenInfo, Deployment, ProcessInstance, History, Incident]) Remove(info WardenInfo) error {
	return this.db.RemoveWardenInfo(info)
}

func (this *Warden[WardenInfo, Deployment, ProcessInstance, History, Incident]) RemoveWardenInfoByInstance(instance ProcessInstance) error {
	infos, err := this.db.GetWardenInfoForInstance(instance)
	if err != nil {
		return err
	}
	for _, info := range infos {
		err = this.Remove(info)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *Warden[WardenInfo, Deployment, ProcessInstance, History, Incident]) Start(ctx context.Context) error {
	if !this.config.RunDbLoop && !this.config.RunProcessLoop {
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
				if this.config.RunDbLoop {
					err := this.LoopDb()
					if err != nil {
						this.config.Logger.Error("error in db loop", "error", err)
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

func (this *Warden[WardenInfo, Deployment, ProcessInstance, History, Incident]) LoopDb() error {
	it, err := this.db.ListWardenInfo()
	if err != nil {
		return err
	}
	for info := range it {
		err = this.CheckWardenInfo(info)
		if err != nil {
			this.config.Logger.Error("error in warden info check", "error", err)
		}
	}
	return nil
}

func (this *Warden[WardenInfo, Deployment, ProcessInstance, History, Incident]) LoopProcesses() error {
	it, err := this.processes.AllInstances()
	if err != nil {
		return err
	}
	for instance := range it {
		err = this.CheckProcessInstance(instance)
		if err != nil {
			this.config.Logger.Error("error in process instance check", "error", err)
		}
	}
	return nil
}

func (this *Warden[WardenInfo, Deployment, ProcessInstance, History, Incident]) CheckWardenInfo(info WardenInfo) error {
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
		// expected state; incidents would cause the stopping of the process-instance, but that is handled by other services
		return nil
	default:
		return this.duplicateInstances(info, instances)
	}
}

func (this *Warden[WardenInfo, Deployment, ProcessInstance, History, Incident]) CheckProcessInstance(instance ProcessInstance) error {
	infos, err := this.db.GetWardenInfoForInstance(instance)
	if err != nil {
		return err
	}
	if len(infos) == 0 {
		if !this.processes.InstanceIsCreatedWithWardenHandlingIntended(instance) {
			this.config.Logger.Debug("process instance without warden info, but is recognised as legacy instance --> no handling", "instance", fmt.Sprintf("%+v", instance))
			return nil
		}
		this.config.Logger.Debug("process instance without warden info --> remove", "instance", fmt.Sprintf("%+v", instance))
		return this.RemoveWardenInfoByInstance(instance)
	}
	return nil
}

func (this *Warden[WardenInfo, Deployment, ProcessInstance, History, Incident]) missingInstance(info WardenInfo) error {
	histories, err := this.processes.GetInstanceHistories(info)
	if err != nil {
		return err
	}
	if len(histories) == 0 {
		exists, err := this.processes.DeploymentExists(info)
		if err != nil {
			this.config.Logger.Error("unable to get deployment for warden info", "error", err)
			return nil
		}
		if !exists {
			this.config.Logger.Debug("missing process instance but warden info has become invalid, this may happen if a process-deployment is deleted --> remove info from warden", "info", fmt.Sprintf("%+v", info), "validation-result", err)
			err = this.tryRedeployProcess(info)
			if err != nil {
				this.config.Logger.Error("unable to redeploy missing process-deployment --> remove warden", "error", err)
				switch {
				case errors.Is(err, ErrRetry):
					return nil
				case errors.Is(err, ErrFinal):
					return this.Remove(info)
				default:
					return this.Remove(info)
				}
			}
		}
		this.config.Logger.Debug("missing process instance --> start process instance", "info", fmt.Sprintf("%+v", info))
		_, err = this.processes.Start(info)
		return err
	}
	history, err := this.processes.GetYoungestHistory(histories)
	if err != nil {
		return err
	}
	if !this.processes.HistoryIsOlderThen(history, this.config.AgeGate) {
		this.config.Logger.Debug("youngest process history change is immature --> no action", "info", fmt.Sprintf("%+v", info))
		return nil
	}
	incidents, err := this.processes.GetIncidents(history)
	if err != nil {
		return err
	}
	if len(incidents) == 0 {
		this.config.Logger.Debug("process finished without incident --> remove info from warden", "info", fmt.Sprintf("%+v", info))
		return this.Remove(info)
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
	_, err = this.processes.Start(info)
	return err
}

func (this *Warden[WardenInfo, Deployment, ProcessInstance, History, Incident]) duplicateInstances(info WardenInfo, instances []ProcessInstance) error {
	youngest, err := this.processes.GetYoungestProcessInstance(instances)
	if err != nil {
		return err
	}
	if !this.processes.InstanceIsOlderThen(youngest, this.config.AgeGate) {
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

func (this *Warden[WardenInfo, Deployment, ProcessInstance, History, Incident]) RemoveDeployment(deploymentId string) error {
	infos, err := this.db.GetWardenInfoForDeploymentId(deploymentId)
	if err != nil {
		return err
	}
	for _, info := range infos {
		err = this.Remove(info)
		if err != nil {
			return err
		}
	}
	return nil
}

func (this *Warden[WardenInfo, Deployment, ProcessInstance, History, Incident]) tryRedeployProcess(info WardenInfo) error {
	depl, exists, err := this.db.GetDeployment(info)
	if err != nil {
		return errors.Join(err, ErrRetry)
	}
	if !exists {
		return fmt.Errorf("no deployment found for warden info (%w)", ErrFinal)
	}
	return this.processes.Deploy(depl)
}
