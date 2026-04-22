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

package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/controller"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
)

// TODO: add warden endpoints
func init() {
	endpoints = append(endpoints, &WardenEndpoints{})
}

type WardenEndpoints struct{}

// ListWardens godoc
// @Summary      list wardens
// @Description  list wardens
// @Tags         warden
// @Produce      json
// @Security Bearer
// @Param        networkId path string true "network id"
// @Param        deployment_ids query string false "filter by comma-separated deployment ids"
// @Param        business_keys query string false "filter by comma-separated business keys"
// @Param        limit query integer false "default 100"
// @Param        offset query integer false "default 0"
// @Success      200 {array} model.WardenInfo
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /wardens/{networkId} [GET]
func (this *WardenEndpoints) ListWardens(config configuration.Config, ctrl *controller.Controller, router *http.ServeMux) {
	router.HandleFunc("GET /wardens/{networkId}", func(writer http.ResponseWriter, request *http.Request) {
		networkId := request.PathValue("networkId")
		err, errCode := ctrl.ApiCheckAccess(request, networkId, "r")
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}

		query := model.WardenInfoQuery{
			NetworkIds: []string{networkId},
		}

		deploymentIds := request.URL.Query().Get("deployment_ids")
		if deploymentIds != "" {
			query.ProcessDeploymentIds = strings.Split(deploymentIds, ",")
		}

		businessKeys := request.URL.Query().Get("business_keys")
		if businessKeys != "" {
			query.BusinessKeys = strings.Split(businessKeys, ",")
		}

		limitStr := request.URL.Query().Get("limit")
		if limitStr == "" {
			limitStr = "100"
		}
		query.Limit, err = strconv.ParseInt(limitStr, 10, 64)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		offsetStr := request.URL.Query().Get("offset")
		if offsetStr == "" {
			offsetStr = "0"
		}
		query.Offset, err = strconv.ParseInt(offsetStr, 10, 64)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		result, err := ctrl.ListWardens(query)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
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

// DeleteWarden godoc
// @Summary      delete warden
// @Description  delete warden, alias for DELETE /process-instances-by-business-key/{networkId}/{business_key}
// @Tags         warden
// @Produce      json
// @Security Bearer
// @Param        networkId path string true "network id"
// @Param        businessKey path string true "businessKey"
// @Success      200
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /wardens/{networkId}/{businessKey} [DELETE]
func (this *WardenEndpoints) DeleteWarden(config configuration.Config, ctrl *controller.Controller, router *http.ServeMux) {
	router.HandleFunc("DELETE /wardens/{networkId}/{businessKey}", func(writer http.ResponseWriter, request *http.Request) {
		networkId := request.PathValue("networkId")
		businessKey := request.PathValue("businessKey")
		err, errCode := ctrl.ApiCheckAccess(request, networkId, "w")
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}
		err, errCode = ctrl.DeleteProcessInstanceByBusinessKey(networkId, businessKey)
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

// ListDeploymentWardens godoc
// @Summary      list deployment wardens
// @Description  list deployment wardens
// @Tags         warden
// @Produce      json
// @Security Bearer
// @Param        networkId path string true "network id"
// @Param        deployment_ids query string false "filter by comma-separated deployment ids"
// @Param        limit query integer false "default 100"
// @Param        offset query integer false "default 0"
// @Success      200 {array} model.DeploymentWardenInfo
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /deployment-wardens/{networkId} [GET]
func (this *WardenEndpoints) ListDeploymentWardens(config configuration.Config, ctrl *controller.Controller, router *http.ServeMux) {
	router.HandleFunc("GET /deployment-wardens/{networkId}", func(writer http.ResponseWriter, request *http.Request) {
		networkId := request.PathValue("networkId")
		err, errCode := ctrl.ApiCheckAccess(request, networkId, "r")
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}

		query := model.DeploymentWardenInfoQuery{
			NetworkIds: []string{networkId},
		}

		deploymentIds := request.URL.Query().Get("deployment_ids")
		if deploymentIds != "" {
			query.ProcessDeploymentIds = strings.Split(deploymentIds, ",")
		}

		limitStr := request.URL.Query().Get("limit")
		if limitStr == "" {
			limitStr = "100"
		}
		query.Limit, err = strconv.ParseInt(limitStr, 10, 64)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		offsetStr := request.URL.Query().Get("offset")
		if offsetStr == "" {
			offsetStr = "0"
		}
		query.Offset, err = strconv.ParseInt(offsetStr, 10, 64)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		result, err := ctrl.ListDeploymentWardens(query)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
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

// DeleteDeploymentWarden godoc
// @Summary      delete deployment warden
// @Description  delete deployment warden, alias for DELETE /deployments/{networkId}/{deploymentId}
// @Tags         warden
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
// @Router       /deployment-wardens/{networkId}/{deploymentId} [DELETE]
func (this *WardenEndpoints) DeleteDeploymentWarden(config configuration.Config, ctrl *controller.Controller, router *http.ServeMux) {
	router.HandleFunc("DELETE /deployment-wardens/{networkId}/{deploymentId}", func(writer http.ResponseWriter, request *http.Request) {
		networkId := request.PathValue("networkId")
		deploymentId := request.PathValue("deploymentId")
		err, errCode := ctrl.ApiCheckAccess(request, networkId, "w")
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
			config.GetLogger().Error("unable to encode response", "error", err)
		}
		return
	})
}

//TODO: implement warden webhooks
