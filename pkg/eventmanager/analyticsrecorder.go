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

package eventmanager

import (
	"errors"
	eventmodel "github.com/SENERGY-Platform/event-deployment/lib/model"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
)

type AnalyticsRecorder struct {
	Records []model.AnalyticsRecord
}

func (this *AnalyticsRecorder) DeployGroup(label string, user string, desc eventmodel.GroupEventDescription, serviceIds []string, serviceToDeviceIdsMapping map[string][]string, serviceToPathMapping map[string]string) (pipelineId string, err error) {
	this.Records = append(this.Records, model.AnalyticsRecord{
		GroupEvent: &model.GroupEventAnalyticsRecord{
			Label:                     label,
			Desc:                      desc,
			ServiceIds:                serviceIds,
			ServiceToDeviceIdsMapping: serviceToDeviceIdsMapping,
			ServiceToPathMapping:      serviceToPathMapping,
		},
	})
	return "placeholder", nil
}

func (this *AnalyticsRecorder) Deploy(label string, user string, deploymentId string, flowId string, eventId string, deviceId string, serviceId string, value string, path string, castFrom string, castTo string) (pipelineId string, err error) {
	this.Records = append(this.Records, model.AnalyticsRecord{
		DeviceEvent: &model.DeviceEventAnalyticsRecord{},
	})
	return "placeholder", nil
}

func (this *AnalyticsRecorder) UpdateGroupDeployment(pipelineId string, label string, owner string, desc eventmodel.GroupEventDescription, serviceIds []string, serviceToDeviceIdsMapping map[string][]string, serviceToPathMapping map[string]string) (err error) {
	return errors.New("updates not supported")
}

func (this *AnalyticsRecorder) DeployImport(label string, user string, desc eventmodel.GroupEventDescription, topic string, path string, castFrom string, castTo string) (pipelineId string, err error) {
	return "", errors.New("imports not supported")
}

func (this *AnalyticsRecorder) Remove(user string, pipelineId string) error {
	return errors.New("remove not supported")
}

func (this *AnalyticsRecorder) GetPipelinesByDeploymentId(owner string, deploymentId string) (pipelineIds []string, err error) {
	return []string{}, nil
}

func (this *AnalyticsRecorder) GetPipelineByEventId(owner string, eventId string) (pipelineId string, exists bool, err error) {
	return "", false, errors.New("not implemented")
}

func (this *AnalyticsRecorder) GetPipelinesByDeviceGroupId(owner string, groupId string) (pipelineIds []string, pipelineToGroupDescription map[string]eventmodel.GroupEventDescription, pipelineNames map[string]string, err error) {
	err = errors.New("not implemented")
	return
}

func (this *AnalyticsRecorder) GetEventStates(userId string, eventIds []string) (states map[string]bool, err error) {
	err = errors.New("not implemented")
	return
}
