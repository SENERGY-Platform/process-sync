/*
 * Copyright 2026 InfAI (CC SES)
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

package mgw

import (
	"encoding/json"
	"runtime/debug"

	"github.com/SENERGY-Platform/process-sync/pkg/model"
	paho "github.com/eclipse/paho.mqtt.golang"
)

func (this *Mgw) handleErrorMessage(message paho.Message) {
	networkId, err := this.getNetworkId(message.Topic())
	if err != nil {
		this.config.GetLogger().Error("error", "error", err, "stack", debug.Stack())
	}
	this.handler.LogNetworkInteraction(networkId)

	msg := model.ErrorMessage{}
	err = json.Unmarshal(message.Payload(), &msg)
	if err != nil {
		//old message format --> handle as string
		this.handler.HandleProcessSyncNetworkError(networkId, string(message.Payload()))
		return
	}
	this.handler.HandleProcessSyncError(networkId, msg)
}
