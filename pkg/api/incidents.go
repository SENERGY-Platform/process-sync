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
	"strconv"
	"strings"
)

func init() {
	endpoints = append(endpoints, &IncidentEndpoints{})
}

type IncidentEndpoints struct{}

// GetIncident godoc
// @Summary      get incident
// @Description  get incident
// @Tags         incidents
// @Produce      json
// @Security Bearer
// @Param        networkId path string true "network id"
// @Param        id path string true "incident id"
// @Success      200 {object}  model.Incident
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /incidents/{networkId}/{id} [GET]
func (this *IncidentEndpoints) GetIncident(config configuration.Config, ctrl *controller.Controller, router *http.ServeMux) {
	router.HandleFunc("GET /incidents/{networkId}/{id}", func(writer http.ResponseWriter, request *http.Request) {
		networkId := request.PathValue("networkId")
		id := request.PathValue("id")
		err, errCode := ctrl.ApiCheckAccess(request, networkId, "rx")
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		result, err, errCode := ctrl.ApiReadIncident(networkId, id)
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

// DeleteIncident godoc
// @Summary      delete incident
// @Description  delete incident
// @Tags         incidents
// @Produce      json
// @Security Bearer
// @Param        networkId path string true "network id"
// @Param        id path string true "incident id"
// @Success      200
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /incidents/{networkId}/{id} [DELETE]
func (this *IncidentEndpoints) DeleteIncident(config configuration.Config, ctrl *controller.Controller, router *http.ServeMux) {
	router.HandleFunc("DELETE /incidents/{networkId}/{id}", func(writer http.ResponseWriter, request *http.Request) {
		networkId := request.PathValue("networkId")
		id := request.PathValue("id")
		err, errCode := ctrl.ApiCheckAccess(request, networkId, "a")
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		err, errCode = ctrl.ApiDeleteIncident(networkId, id)
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

// ListIncidents godoc
// @Summary      list incidents
// @Description  list incidents
// @Tags         incidents
// @Produce      json
// @Security Bearer
// @Param        limit query integer false "default 100"
// @Param        offset query integer false "default 0"
// @Param        sort query string false "default id.asc"
// @Param        network_id query string true "comma seperated list of network-ids, used to filter the result"
// @Param        process_instance_id query string false "process-instance-id, used to filter the result"
// @Success      200 {array}  model.Incident
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /incidents [GET]
func (this *IncidentEndpoints) ListIncidents(config configuration.Config, ctrl *controller.Controller, router *http.ServeMux) {
	router.HandleFunc("GET /incidents", func(writer http.ResponseWriter, request *http.Request) {
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

		instanceId := request.URL.Query().Get("process_instance_id")

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
		result, err, errCode := ctrl.ApiListIncidents(networkIds, instanceId, limit, offset, sort)
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
