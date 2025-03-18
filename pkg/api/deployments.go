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
	"github.com/SENERGY-Platform/process-deployment/lib/model/deploymentmodel"
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/controller"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func init() {
	endpoints = append(endpoints, &DeploymentEndpoints{})
}

type DeploymentEndpoints struct{}

// GetDeployment godoc
// @Summary      get deployment
// @Description  get deployment
// @Tags         deployment
// @Produce      json
// @Security Bearer
// @Param        networkId path string true "network id"
// @Param        deploymentId path string true "deployment id"
// @Success      200 {object}  model.Deployment
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /deployments/{networkId}/{deploymentId} [GET]
func (this *DeploymentEndpoints) GetDeployment(config configuration.Config, ctrl *controller.Controller, router *http.ServeMux) {
	router.HandleFunc("GET /deployments/{networkId}/{deploymentId}", func(writer http.ResponseWriter, request *http.Request) {
		networkId := request.PathValue("networkId")
		deploymentId := request.PathValue("deploymentId")
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
}

// GetDeploymentMetadata godoc
// @Summary      get deployment metadata
// @Description  get deployment metadata
// @Tags         deployment, metadata
// @Produce      json
// @Security Bearer
// @Param        networkId path string true "network id"
// @Param        deploymentId path string true "deployment id"
// @Success      200 {object}  model.DeploymentMetadata
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /deployments/{networkId}/{deploymentId}/metadata [GET]
func (this *DeploymentEndpoints) GetDeploymentMetadata(config configuration.Config, ctrl *controller.Controller, router *http.ServeMux) {
	router.HandleFunc("GET /deployments/{networkId}/{deploymentId}/metadata", func(writer http.ResponseWriter, request *http.Request) {
		networkId := request.PathValue("networkId")
		deploymentId := request.PathValue("deploymentId")
		err, errCode := ctrl.ApiCheckAccess(request, networkId, "rx")
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		result, err, errCode := ctrl.ApiReadDeploymentMetadata(networkId, deploymentId)
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

// StartDeployment godoc
// @Summary      start deployed process
// @Description  start deployed process; a process may expect parameters on start. these cna be passed as query parameters. swagger allows no arbitrary/dynamic parameter names, which means a query wit parameters must be executed manually
// @Tags         deployment
// @Produce      json
// @Security Bearer
// @Param        networkId path string true "network id"
// @Param        deploymentId path string true "deployment id"
// @Success      200
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /deployments/{networkId}/{deploymentId}/start [GET]
func (this *DeploymentEndpoints) StartDeployment(config configuration.Config, ctrl *controller.Controller, router *http.ServeMux) {
	router.HandleFunc("GET /deployments/{networkId}/{deploymentId}/start", func(writer http.ResponseWriter, request *http.Request) {
		networkId := request.PathValue("networkId")
		deploymentId := request.PathValue("deploymentId")
		err, errCode := ctrl.ApiCheckAccess(request, networkId, "rx")
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}

		inputs := parseQueryParameter(request.URL.Query())

		err, errCode = ctrl.ApiStartDeployment(networkId, deploymentId, inputs)
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

// CreateDeployment godoc
// @Summary      deploy process
// @Description  deploy process; prepared process may be requested from the process-fog-deployment service
// @Tags         deployment
// @Produce      json
// @Security Bearer
// @Param        message body deploymentmodel.Deployment true "deployment"
// @Success      200
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /deployments/{networkId} [POST]
func (this *DeploymentEndpoints) CreateDeployment(config configuration.Config, ctrl *controller.Controller, router *http.ServeMux) {
	router.HandleFunc("POST /deployments/{networkId}", func(writer http.ResponseWriter, request *http.Request) {
		networkId := request.PathValue("networkId")
		deployment := deploymentmodel.Deployment{}
		err := json.NewDecoder(request.Body).Decode(&deployment)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		token, err, errCode := ctrl.ApiCheckAccessReturnToken(request, networkId, "a")
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		err, errCode = ctrl.ApiCreateDeployment(token, networkId, deployment)
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

// DeleteDeployment godoc
// @Summary      delete deployment
// @Description  delete deployment
// @Tags         deployment
// @Produce      json
// @Security Bearer
// @Param        networkId path string true "network id"
// @Param        deploymentId path string true "deployment id"
// @Success      200
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /deployments/{networkId}/{deploymentId} [DELETE]
func (this *DeploymentEndpoints) DeleteDeployment(config configuration.Config, ctrl *controller.Controller, router *http.ServeMux) {
	router.HandleFunc("DELETE /deployments/{networkId}/{deploymentId}", func(writer http.ResponseWriter, request *http.Request) {
		networkId := request.PathValue("networkId")
		deploymentId := request.PathValue("deploymentId")
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
}

// ListDeployments godoc
// @Summary      list deployments
// @Description  list deployments
// @Tags         deployment
// @Produce      json
// @Security Bearer
// @Param        networkId path string true "network id"
// @Param        deploymentId path string true "deployment id"
// @Param        search query string false "search"
// @Param        limit query integer false "default 100"
// @Param        offset query integer false "default 0"
// @Param        sort query string false "default id.asc"
// @Param        extended query bool false "add the fields 'diagram', 'definition_id' and 'error' to the results"
// @Param        network_id query string true "comma seperated list of network-ids used to filter the deployments"
// @Success      200 {array}  model.Deployment
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /deployments [GET]
func (this *DeploymentEndpoints) ListDeployments(config configuration.Config, ctrl *controller.Controller, router *http.ServeMux) {
	router.HandleFunc("GET /deployments", func(writer http.ResponseWriter, request *http.Request) {
		search := request.URL.Query().Get("search")
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

		extended := false
		extendedQueryParam := request.URL.Query().Get("extended")
		if extendedQueryParam != "" {
			extended, err = strconv.ParseBool(extendedQueryParam)
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
		deployments := []model.Deployment{}
		if search == "" {
			deployments, err, errCode = ctrl.ApiListDeployments(networkIds, limit, offset, sort)
		} else {
			deployments, err, errCode = ctrl.ApiSearchDeployments(networkIds, search, limit, offset, sort)
		}
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		var result interface{}
		if extended {
			result = ctrl.ExtendDeployments(deployments)
		} else {
			result = deployments
		}

		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		err = json.NewEncoder(writer).Encode(result)
		if err != nil {
			log.Println("ERROR: unable to encode response", err)
		}
		return
	})
}

func parseQueryParameter(query url.Values) (result map[string]interface{}) {
	if len(query) == 0 {
		return map[string]interface{}{}
	}
	result = map[string]interface{}{}
	for key, _ := range query {
		var val interface{}
		temp := query.Get(key)
		err := json.Unmarshal([]byte(temp), &val)
		if err != nil {
			val = temp
		}
		result[key] = val
	}
	return result
}
