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
	"github.com/SENERGY-Platform/process-sync/pkg/model/deploymentmodel"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func init() {
	endpoints = append(endpoints, DeploymentEndpoints)
}

func DeploymentEndpoints(config configuration.Config, ctrl *controller.Controller, router *httprouter.Router) {
	resource := "/deployments"

	router.GET(resource+"/:networkId/:deploymentId", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		networkId := params.ByName("networkId")
		deploymentId := params.ByName("deploymentId")
		err, errCode := ctrl.ApiCheckAccess(request, networkId, "rx")
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

	router.GET(resource+"/:networkId/:deploymentId/start", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		networkId := params.ByName("networkId")
		deploymentId := params.ByName("deploymentId")
		err, errCode := ctrl.ApiCheckAccess(request, networkId, "rx")
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		err, errCode = ctrl.ApiStartDeployment(networkId, deploymentId)
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

	router.POST(resource+"/:networkId", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		networkId := params.ByName("networkId")
		deployment := deploymentmodel.Deployment{}
		err := json.NewDecoder(request.Body).Decode(&deployment)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		err, errCode := ctrl.ApiCheckAccess(request, networkId, "a")
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		err, errCode = ctrl.ApiCreateDeployment(networkId, deployment)
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

	router.DELETE(resource+"/:networkId/:deploymentId", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		networkId := params.ByName("networkId")
		deploymentId := params.ByName("deploymentId")
		err, errCode := ctrl.ApiCheckAccess(request, networkId, "a")
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		err, errCode = ctrl.ApiDeleteDeployment(networkId, deploymentId)
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

	router.GET(resource, func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		sort := request.URL.Query().Get("sort")
		if sort == "" {
			sort = "id.asc"
		}
		limitStr := request.URL.Query().Get("limit")
		if limitStr == "" {
			limitStr = "100"
		}
		limit, err := strconv.ParseInt(limitStr, 10, 64)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		offsetStr := request.URL.Query().Get("offset")
		if offsetStr == "" {
			offsetStr = "100"
		}
		offset, err := strconv.ParseInt(offsetStr, 10, 64)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		networkIdsStr := request.URL.Query().Get("network_id")
		if networkIdsStr == "" {
			http.Error(writer, "expect network_id query parameter", http.StatusBadRequest)
			return
		}
		networkIds := strings.Split(networkIdsStr, ",")
		err, errCode := ctrl.ApiCheckAccessMultiple(request, networkIds, "rx")
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		result, err, errCode := ctrl.ApiListDeployments(networkIds, limit, offset, sort)
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
