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

package api

import (
	"encoding/json"
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/controller"
	"log"
	"net/http"
)

func init() {
	endpoints = append(endpoints, &SyncEndpoints{})
}

type SyncEndpoints struct{}

// ReSyncDeployments godoc
// @Summary      resync deployments
// @Description  resync deployments that are registered as lost on the mgw side. can only be tried once.
// @Tags         deployment
// @Produce      json
// @Security Bearer
// @Param        networkId path string true "network id"
// @Success      200
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /sync/deployments/{networkId} [POST]
func (this *SyncEndpoints) ReSyncDeployments(config configuration.Config, ctrl *controller.Controller, router *http.ServeMux) {
	router.HandleFunc("POST /sync/deployments/{networkId}", func(writer http.ResponseWriter, request *http.Request) {
		networkId := request.PathValue("networkId")
		token, err, errCode := ctrl.ApiCheckAccessReturnToken(request, networkId, "a")
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		err, errCode = ctrl.ApiSyncDeployments(token, networkId)
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		err = json.NewEncoder(writer).Encode(true)
		if err != nil {
			log.Println("ERROR: unable to encode response", err)
		}
		return
	})

}
