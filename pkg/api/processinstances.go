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
	"net/http"
	"strconv"
	"strings"
)

func init() {
	endpoints = append(endpoints, &ProcessInstanceEndpoints{})
}

type ProcessInstanceEndpoints struct{}

// GetProcessInstance godoc
// @Summary      get process-instances
// @Description  get process-instances
// @Tags         process-instance
// @Produce      json
// @Security Bearer
// @Param        networkId path string true "network id"
// @Param        id path string true "instance id"
// @Success      200 {object}  model.HistoricProcessInstance
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /process-instances/{networkId}/{id} [GET]
func (this *ProcessInstanceEndpoints) GetProcessInstance(config configuration.Config, ctrl *controller.Controller, router *http.ServeMux) {
	router.HandleFunc("GET /process-instances/{networkId}/{id}", func(writer http.ResponseWriter, request *http.Request) {
		networkId := request.PathValue("networkId")
		id := request.PathValue("id")
		err, errCode := ctrl.ApiCheckAccess(request, networkId, "rx")
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		result, err, errCode := ctrl.ApiReadProcessInstance(networkId, id)
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		err = json.NewEncoder(writer).Encode(result)
		if err != nil {
			config.GetLogger().Error("unable to encode response", "error", err)
		}
		return
	})
}

// DeleteProcessInstance godoc
// @Summary      get process-instances
// @Description  get process-instances
// @Tags         process-instance
// @Produce      json
// @Security Bearer
// @Param        networkId path string true "network id"
// @Param        id path string true "instance id"
// @Success      200
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /process-instances/{networkId}/{id} [DELETE]
func (this *ProcessInstanceEndpoints) DeleteProcessInstance(config configuration.Config, ctrl *controller.Controller, router *http.ServeMux) {
	router.HandleFunc("DELETE /process-instances/{networkId}/{id}", func(writer http.ResponseWriter, request *http.Request) {
		networkId := request.PathValue("networkId")
		id := request.PathValue("id")
		err, errCode := ctrl.ApiCheckAccess(request, networkId, "a")
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		err, errCode = ctrl.ApiDeleteProcessInstance(networkId, id)
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		err = json.NewEncoder(writer).Encode(true)
		if err != nil {
			config.GetLogger().Error("unable to encode response", "error", err)
		}
		return
	})
}

// ListProcessInstances godoc
// @Summary      list process-instances
// @Description  list process-instances
// @Tags         process-instance
// @Produce      json
// @Security Bearer
// @Param        limit query integer false "default 100"
// @Param        offset query integer false "default 0"
// @Param        sort query string false "default id.asc"
// @Param        network_id query string true "comma seperated list of network-ids used to filter"
// @Success      200 {array}  model.ProcessInstance
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /process-instances [GET]
func (this *ProcessInstanceEndpoints) ListProcessInstances(config configuration.Config, ctrl *controller.Controller, router *http.ServeMux) {
	router.HandleFunc("GET /process-instances", func(writer http.ResponseWriter, request *http.Request) {
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
			offsetStr = "0"
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
		result, err, errCode := ctrl.ApiListProcessInstances(networkIds, limit, offset, sort)
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		err = json.NewEncoder(writer).Encode(result)
		if err != nil {
			config.GetLogger().Error("unable to encode response", "error", err)
		}
		return
	})
}
