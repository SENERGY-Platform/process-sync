/*
 * Copyright 2019 InfAI (CC SES)
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

package security

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
)

func New(config configuration.Config) *Security {
	return &Security{config: config}
}

type Security struct {
	config configuration.Config
}

type IdWrapper struct {
	Id string `json:"id"`
}

func (this *Security) IsAdmin(token string) bool {
	roles, err := ReadTokenRoles(token)
	if err != nil {
		log.Println("ERROR: unable to parse auth token to check if user is admin:", err)
		return false
	}
	return contains(roles, "admin")
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func (this *Security) CheckBool(token string, kind string, id string, rights string) (allowed bool, err error) {
	if this.IsAdmin(token) {
		return true, nil
	}
	req, err := http.NewRequest("GET", this.config.PermissionsUrl+"/v3/resources/"+url.QueryEscape(kind)+"/"+url.QueryEscape(id)+"/access?rights="+rights, nil)
	if err != nil {
		debug.PrintStack()
		return false, err
	}
	req.Header.Set("Authorization", token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		debug.PrintStack()
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return false, errors.New(buf.String())
	}
	err = json.NewDecoder(resp.Body).Decode(&allowed)
	if err != nil {
		debug.PrintStack()
		return false, err
	}
	return true, nil
}

func (this *Security) CheckMultiple(token string, kind string, ids []string, rights string) (result map[string]bool, err error) {
	result = map[string]bool{}
	if this.IsAdmin(token) {
		for _, id := range ids {
			result[id] = true
		}
		return result, nil
	}
	requestBody := new(bytes.Buffer)
	err = json.NewEncoder(requestBody).Encode(QueryMessage{
		Resource: kind,
		CheckIds: &QueryCheckIds{
			Ids:    ids,
			Rights: rights,
		},
	})
	if err != nil {
		return result, err
	}
	req, err := http.NewRequest("POST", this.config.PermissionsUrl+"/v3/query", requestBody)
	if err != nil {
		debug.PrintStack()
		return result, err
	}
	req.Header.Set("Authorization", token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		debug.PrintStack()
		return result, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return result, errors.New(buf.String())
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		debug.PrintStack()
		return result, err
	}
	return result, err
}
