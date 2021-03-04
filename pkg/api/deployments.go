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
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
)

func init() {
	endpoints = append(endpoints, DeploymentEndpoints)
}

func DeploymentEndpoints(config configuration.Config, ctrl *controller.Controller, router *httprouter.Router) {
	resource := "/deployments"

	router.GET(resource+"/:networkId/:deploymentId", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		networkId := params.ByName("networkId")
		deploymentId := params.ByName("deploymentId")
		err, errCode := ctrl.ApiCheckAccess(request, networkId)
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		result, err, errCode := ctrl.ApiReadDeployment(networkId, deploymentId)
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		err = json.NewEncoder(writer).Encode(result)
		if err != nil {
			log.Println("ERROR: unable to encode response", err)
		}
		return
	})
}
