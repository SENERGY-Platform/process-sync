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

package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/SENERGY-Platform/process-sync/pkg/model"
)

func (this *Controller) TriggerWebhooks(webhooks []model.Webhook, on model.WebhookTrigger, resourceType string, networkId string, id string, message string) {
	for _, wh := range webhooks {
		if wh.On == on {
			whm := model.WebhookMessage{
				ResourceType: resourceType,
				NetworkId:    networkId,
				Id:           id,
				Trigger:      on,
				Message:      message,
			}
			pl, err := json.Marshal(whm)
			if err != nil {
				this.logger.Error("unable to marshal webhook message", "error", err.Error(), "webhookMessage", fmt.Sprintf("%#v", whm))
				continue
			}
			req, err := http.NewRequest(wh.Method, wh.Url, bytes.NewBuffer(pl))
			if err != nil {
				this.logger.Error("unable to create webhook request", "error", err.Error(), "webhookMessage", fmt.Sprintf("%#v", whm))
				continue
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				this.logger.Error("unable to send webhook request", "error", err.Error(), "webhookMessage", fmt.Sprintf("%#v", whm))
				continue
			}
			defer resp.Body.Close()
			if resp.StatusCode >= 300 && resp.StatusCode != 404 {
				respMsg, _ := io.ReadAll(resp.Body)
				this.logger.Error("webhook request returned unexpected status code", "error", string(respMsg), "status", resp.Status, "webhookMessage", fmt.Sprintf("%#v", whm))
			}
		}
	}
}
