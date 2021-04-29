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

package controller

import (
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"github.com/SENERGY-Platform/process-sync/pkg/security"
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

func (this *Controller) ApiListNetworks(request *http.Request) (result []security.ListElement, err error, errCode int) {
	token := request.Header.Get("Authorization")
	all, err := this.security.List(token, "hubs", "10000", "0", "r", "id.asc")
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	allIds := []string{}
	hubIndex := map[string]security.ListElement{}
	for _, element := range all {
		allIds = append(allIds, element.Id)
		hubIndex[element.Id] = element
	}
	filteredIds, err := this.db.FilterNetworkIds(allIds)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	for _, id := range filteredIds {
		result = append(result, hubIndex[id])
	}
	return result, nil, http.StatusOK
}

func (this *Controller) LogNetworkInteraction(networkId string) {
	err := this.db.SaveLastContact(model.LastNetworkContact{
		NetworkId: networkId,
		Time:      time.Now(),
	})
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
}
