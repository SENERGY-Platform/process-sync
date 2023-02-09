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
