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
	"errors"
	"github.com/SENERGY-Platform/permission-search/lib/client"
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"log"
	"time"
)

func New(config configuration.Config) *Security {
	return &Security{config: config, permissionsearch: client.NewClient(config.PermissionsUrl)}
}

type Security struct {
	config           configuration.Config
	openid           *OpenidToken
	permissionsearch client.Client
}

type OpenidToken struct {
	AccessToken      string    `json:"access_token"`
	ExpiresIn        float64   `json:"expires_in"`
	RefreshExpiresIn float64   `json:"refresh_expires_in"`
	RefreshToken     string    `json:"refresh_token"`
	TokenType        string    `json:"token_type"`
	RequestTime      time.Time `json:"-"`
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
	err = this.permissionsearch.CheckUserOrGroup(token, kind, id, rights)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, client.ErrAccessDenied) {
		return false, nil
	}
	if errors.Is(err, client.ErrNotFound) {
		return false, nil
	}
	return allowed, err
}

func (this *Security) CheckMultiple(token string, kind string, ids []string, rights string) (result map[string]bool, err error) {
	result = map[string]bool{}
	if this.IsAdmin(token) {
		for _, id := range ids {
			result[id] = true
		}
		return result, nil
	}
	result, _, err = client.Query[map[string]bool](this.permissionsearch, token, client.QueryMessage{
		Resource: kind,
		CheckIds: &client.QueryCheckIds{
			Ids:    ids,
			Rights: rights,
		},
	})

	return result, err
}
