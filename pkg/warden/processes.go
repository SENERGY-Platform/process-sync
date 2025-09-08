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
	"errors"
	"fmt"
	"iter"
	"slices"
	"strings"
	"time"

	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/controller"
	"github.com/SENERGY-Platform/process-sync/pkg/database"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"github.com/SENERGY-Platform/process-sync/pkg/model/camundamodel"
	"github.com/SENERGY-Platform/service-commons/pkg/cache"
)

type Processes struct {
	db        database.Database
	batchsize int64
	config    configuration.Config
	cache     *cache.Cache
	ctrl      *controller.Controller
}

func (this *Processes) AllInstances() iter.Seq2[model.ProcessInstance, error] {
	var offset int64 = 0
	return func(yield func(model.ProcessInstance, error) bool) {
		finished := false
		for !finished {
			batch, err := this.db.FindProcessInstances(model.InstanceQuery{
				Limit:  this.batchsize,
				Offset: offset,
			})
			if err != nil {
				yield(model.ProcessInstance{}, err)
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

func (this *Processes) GetInstances(info WardenInfo) ([]model.ProcessInstance, error) {
	return this.db.FindProcessInstances(model.InstanceQuery{
		NetworkIds:   []string{info.NetworkId},
		BusinessKeys: []string{info.BusinessKey},
	})
}

func (this *Processes) getInstanceDate(instance model.ProcessInstance) (time.Time, error) {
	return cache.Use(this.cache, "process-instance-age."+instance.Id, func() (time.Time, error) {
		history, err := this.db.ReadHistoricProcessInstance(instance.NetworkId, instance.Id)
		if err != nil {
			this.config.GetLogger().Error("unable to read historic process instance to determine instance date --> use instance.SyncDate", "error", err, "instanceId", instance.Id, "networkId", instance.NetworkId)
			return instance.SyncDate, nil
		}
		result, err := time.Parse(history.StartTime, camundamodel.CamundaTimeFormat)
		if err != nil {
			this.config.GetLogger().Error("unable to parse historic process instance start time to determine instance date --> use instance.SyncDate", "error", err, "instanceId", instance.Id, "networkId", instance.NetworkId)
			return instance.SyncDate, nil
		}
		return result, nil
	}, cache.NoValidation, time.Minute)
}

func GetYoungestElement[T any](list []T, elementTimeProvider func(T) (time.Time, error)) (result T, err error) {
	if len(list) == 0 {
		return result, errors.New("expect at least one element in GetYoungestElement()")
	}
	var sortErr error
	slices.SortFunc(list, func(a, b T) int {
		aTime, err := elementTimeProvider(a)
		sortErr = errors.Join(sortErr, err)
		bTime, err := elementTimeProvider(b)
		sortErr = errors.Join(sortErr, err)
		return int(aTime.Sub(bTime))
	})
	return list[0], sortErr
}

func (this *Processes) GetYoungestProcessInstance(instances []model.ProcessInstance) (model.ProcessInstance, error) {
	return GetYoungestElement(instances, this.getInstanceDate)
}

func (this *Processes) InstanceIsOlderThen(instance model.ProcessInstance, duration time.Duration) (bool, error) {
	instanceDate, err := this.getInstanceDate(instance)
	if err != nil {
		return false, err
	}
	return instanceDate.Add(duration).Before(time.Now()), nil
}

func (this *Processes) InstanceIsCreatedWithWardenHandlingIntended(instance model.ProcessInstance) bool {
	return strings.HasPrefix(instance.BusinessKey, "wardened:") //TODO: add prefix to business-key of instances when warden is used
}

func (this *Processes) GetInstanceHistories(info WardenInfo) ([]model.HistoricProcessInstance, error) {
	return this.db.FindHistoricProcessInstances(model.InstanceQuery{
		NetworkIds:   []string{info.NetworkId},
		BusinessKeys: []string{info.BusinessKey},
	})
}

func (this *Processes) getHistoryDate(history model.HistoricProcessInstance) (time.Time, error) {
	result, err := time.Parse(history.StartTime, camundamodel.CamundaTimeFormat)
	if err != nil {
		this.config.GetLogger().Error("unable to parse historic process instance start time to determine history date --> use history.SyncDate", "error", err, "historyId", history.Id, "networkId", history.NetworkId)
		return history.SyncDate, nil
	}
	return result, nil
}

func (this *Processes) GetYoungestHistory(histories []model.HistoricProcessInstance) (model.HistoricProcessInstance, error) {
	return GetYoungestElement(histories, this.getHistoryDate)
}

func (this *Processes) HistoryIsOlderThen(history model.HistoricProcessInstance, duration time.Duration) (bool, error) {
	date, err := this.getHistoryDate(history)
	if err != nil {
		return false, err
	}
	return date.Add(duration).Before(time.Now()), nil
}

func (this *Processes) GetIncidents(history model.HistoricProcessInstance) ([]model.Incident, error) {
	return this.db.FindIncidents(model.IncidentQuery{
		NetworkIds:         []string{history.NetworkId},
		ProcessInstanceIds: []string{history.Id},
	})
}

func (this *Processes) GetYoungestIncident(incidents []model.Incident) (model.Incident, error) {
	return GetYoungestElement(incidents, func(incident model.Incident) (time.Time, error) {
		return incident.Time, nil
	})
}

func (this *Processes) IncidentIsOlderThen(incident model.Incident, duration time.Duration) bool {
	return incident.Time.Add(duration).Before(time.Now())
}

func (this *Processes) Start(info WardenInfo) (err error) {
	err, _ = this.ctrl.StartDeploymentWithoutWardenHandling(info.NetworkId, info.ProcessDeploymentId, info.BusinessKey, info.StartParameters)
	return
}

func (this *Processes) Stop(instance model.ProcessInstance) (err error) {
	err, _ = this.ctrl.StopProcessInstanceWithoutWardenHandling(instance.NetworkId, instance.Id)
	return
}

func (this *Processes) DeploymentExistsForWarden(info WardenInfo) (exist bool, err error) {
	depl, err := this.db.ReadDeployment(info.NetworkId, info.ProcessDeploymentId)
	if errors.Is(err, database.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return !depl.MarkedForDelete && !depl.MarkedAsMissing, nil
}

func (this *Processes) DeploymentExistsForDeploymentWarden(info DeploymentWardenInfo) (exist bool, err error) {
	depl, err := this.db.ReadDeployment(info.NetworkId, info.DeploymentId)
	if errors.Is(err, database.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return !depl.MarkedForDelete && !depl.MarkedAsMissing, nil
}

func (this *Processes) Redeploy(info DeploymentWardenInfo) error {
	var exists, markedForDelete, markedAsMissing bool
	exists = true
	depl, err := this.db.ReadDeployment(info.NetworkId, info.DeploymentId)
	if errors.Is(err, database.ErrNotFound) {
		exists = false
	}
	if err != nil {
		return errors.Join(fmt.Errorf("unable to redeploy process %v %v (unable to read existing): %w", info.Deployment, info.Deployment.Name, err), ErrRetry)
	}
	markedForDelete = depl.MarkedForDelete
	markedAsMissing = depl.MarkedAsMissing
	if !exists {
		err = this.ctrl.DeployProcessWithoutWardenHandling(info.NetworkId, info.Deployment)
		if err != nil {
			return errors.Join(fmt.Errorf("unable to redeploy process %v %v: %w", info.Deployment, info.Deployment.Name, err), ErrRetry)
		}
		return nil
	}
	if markedForDelete {
		//this is a bad state (user has selected to delete this deployment) -> rectify by deleting DeploymentWardenInfo and returning ErrFinal (deletes instance WardenInfo)
		err = this.db.RemoveDeploymentWardenInfo(info.NetworkId, info.DeploymentId)
		if err != nil && !errors.Is(err, database.ErrNotFound) {
			return errors.Join(fmt.Errorf("error in removing deployment warden info for %v %v: %w", info.Deployment, info.Deployment.Name, err), ErrRetry) //retry to find the bad state again, in the hope to rectify it
		}
		return fmt.Errorf("unable to redeploy because deployment is marked for delete (%w)", ErrFinal) //signal instance WardenInfo removal
	}
	if markedAsMissing {
		err = this.db.RemoveDeploymentMetadata(info.NetworkId, info.DeploymentId)
		if err != nil {
			return errors.Join(fmt.Errorf("unable to redeploy process %v %v (unable to remove old metadata): %w", info.Deployment, info.Deployment.Name, err), ErrRetry)
		}
		err = this.ctrl.DeployProcessWithoutWardenHandling(info.NetworkId, info.Deployment)
		if err != nil {
			return errors.Join(fmt.Errorf("unable to redeploy process %v %v: %w", info.Deployment, info.Deployment.Name, err), ErrRetry)
		}
	}
	return nil
}
