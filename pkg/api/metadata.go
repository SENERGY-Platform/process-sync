/*
 * Copyright 2023 InfAI (CC SES)
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
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"net/http"
)

func init() {
	endpoints = append(endpoints, &MetadataEndpoints{})
}

type MetadataEndpoints struct{}

// ListMetadata godoc
// @Summary      list deployment metadata
// @Description  list deployment metadata
// @Tags         deployment, metadata
// @Produce      json
// @Security Bearer
// @Param        networkId path string true "network id"
// @Param        deployment_id query string false "deployment id"
// @Param        camunda_deployment_id query string false "camunda deployment id"
// @Success      200 {array}  model.DeploymentMetadata
// @Failure      400
// @Failure      401
// @Failure      403
// @Failure      404
// @Failure      500
// @Router       /metadata/{networkId} [GET]
func (this *MetadataEndpoints) ListMetadata(config configuration.Config, ctrl *controller.Controller, router *http.ServeMux) {
	router.HandleFunc("GET /metadata/{networkId}", func(writer http.ResponseWriter, request *http.Request) {
		networkId := request.PathValue("networkId")
		err, errCode := ctrl.ApiCheckAccess(request, networkId, "rx")
		if err != nil {
			http.Error(writer, err.Error(), errCode)
			return
		}

		query := model.MetadataQuery{
			NetworkId: &networkId,
		}

		deploymentId := request.URL.Query().Get("deployment_id")
		if deploymentId != "" {
			query.DeploymentId = &deploymentId
		}

		camundaDeploymentId := request.URL.Query().Get("camunda_deployment_id")
		if camundaDeploymentId != "" {
			query.CamundaDeploymentId = &camundaDeploymentId
		}

		metadata, err := ctrl.ListDeploymentMetadata(query)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		err = json.NewEncoder(writer).Encode(metadata)
		if err != nil {
			config.GetLogger().Error("unable to encode response", "error", err)
		}
		return
	})
}
