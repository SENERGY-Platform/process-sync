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
	"strconv"
	"strings"
)

func init() {
	endpoints = append(endpoints, HistoryEndpoints)
}

func HistoryEndpoints(config configuration.Config, ctrl *controller.Controller, router *httprouter.Router) {
	resource := "/history/process-instances"

	router.GET(resource+"/:networkId/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		networkId := params.ByName("networkId")
		id := params.ByName("id")
		err, errCode := ctrl.ApiCheckAccess(request, networkId, "rx")
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		result, err, errCode := ctrl.ApiReadHistoricProcessInstance(networkId, id)
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

	router.DELETE(resource+"/:networkId/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		networkId := params.ByName("networkId")
		id := params.ByName("id")
		err, errCode := ctrl.ApiCheckAccess(request, networkId, "a")
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		err, errCode = ctrl.ApiDeleteHistoricProcessInstance(networkId, id)
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
			offsetStr = "0"
		}
		offset, err := strconv.ParseInt(offsetStr, 10, 64)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		withTotal := false
		extendedQueryParam := request.URL.Query().Get("with_total")
		if extendedQueryParam != "" {
			withTotal, err = strconv.ParseBool(extendedQueryParam)
		}
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
		result, total, err, errCode := ctrl.ApiListHistoricProcessInstance(networkIds, limit, offset, sort)
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}

		writer.Header().Set("Content-Type", "application/json; charset=utf-8")

		if withTotal {
			err = json.NewEncoder(writer).Encode(map[string]interface{}{
				"data":  result,
				"total": total,
			})
		} else {
			err = json.NewEncoder(writer).Encode(result)
		}

		if err != nil {
			log.Println("ERROR: unable to encode response", err)
		}
		return
	})
}
