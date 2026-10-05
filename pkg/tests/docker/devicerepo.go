/*
 * Copyright (c) 2026 InfAI (CC SES)
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

package docker

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
)

// DeviceRepo serves the device-repository endpoints the mgw-external-task-worker reads
// and returns the port it listens on; it stops when ctx is done.
// entities is a json object keyed like the former task-worker fallback file:
// "concept-ids", "list-functions", "concept.<id>" and "characteristics.<id>"
func DeviceRepo(ctx context.Context, wg *sync.WaitGroup, entities string) (port int, err error) {
	values := map[string]json.RawMessage{}
	err = json.Unmarshal([]byte(entities), &values)
	if err != nil {
		return 0, err
	}
	conceptIds := []string{}
	if raw, ok := values["concept-ids"]; ok {
		err = json.Unmarshal(raw, &conceptIds)
		if err != nil {
			return 0, err
		}
	}
	functions := []json.RawMessage{}
	if raw, ok := values["list-functions"]; ok {
		err = json.Unmarshal(raw, &functions)
		if err != nil {
			return 0, err
		}
	}
	concepts := []json.RawMessage{}
	for _, id := range conceptIds {
		concept, ok := values["concept."+id]
		if !ok {
			concept, _ = json.Marshal(map[string]string{"id": id})
		}
		concepts = append(concepts, concept)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var result interface{}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/concepts":
			limit, _ := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64)
			offset, _ := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64)
			w.Header().Set("X-Total-Count", strconv.Itoa(len(concepts)))
			result = page(concepts, limit, offset)
		case r.Method == http.MethodPost && r.URL.Path == "/query/functions":
			options := struct {
				Limit  int64 `json:"limit"`
				Offset int64 `json:"offset"`
			}{}
			err := json.NewDecoder(r.Body).Decode(&options)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("X-Total-Count", strconv.Itoa(len(functions)))
			result = page(functions, options.Limit, options.Offset)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/concepts/"):
			value, ok := values["concept."+strings.TrimPrefix(r.URL.Path, "/concepts/")]
			if !ok {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			result = value
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/characteristics/"):
			value, ok := values["characteristics."+strings.TrimPrefix(r.URL.Path, "/characteristics/")]
			if !ok {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			result = value
		default:
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	}))
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		server.Close()
	}()
	return server.Listener.Addr().(*net.TCPAddr).Port, nil
}

// page returns the elements of a list request; a limit of 0 means no limit, like the device-repository client omits it
func page[T any](list []T, limit int64, offset int64) []T {
	if offset >= int64(len(list)) {
		return []T{}
	}
	list = list[offset:]
	if limit > 0 && limit < int64(len(list)) {
		list = list[:limit]
	}
	return list
}
