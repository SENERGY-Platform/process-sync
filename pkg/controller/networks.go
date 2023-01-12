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
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

type SearchHub struct {
	Id             string   `json:"id"`
	Name           string   `json:"name"`
	DeviceLocalIds []string `json:"device_local_ids"`
	DeviceIds      []string `json:"device_ids"`
}

func (this *Controller) ApiListNetworks(request *http.Request) (result []SearchHub, err error, errCode int) {
	token := request.Header.Get("Authorization")
	all := []SearchHub{}
	err = this.security.ListElements(token, "hubs", "10000", "0", "r", &all)
	if err != nil {
		return result, err, http.StatusInternalServerError
	}
	allIds := []string{}
	hubIndex := map[string]SearchHub{}
	for _, element := range all {
		allIds = append(allIds, element.Id)
		hubIndex[element.Id] = element
	}
	if len(allIds) == 0 {
		return []SearchHub{}, nil, http.StatusOK
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
