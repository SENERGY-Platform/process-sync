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
	permv2 "github.com/SENERGY-Platform/permissions-v2/pkg/client"
	"github.com/SENERGY-Platform/permissions-v2/pkg/model"
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"log"
	"time"
)

func New(config configuration.Config) *Security {
	return &Security{config: config, permv2: permv2.New(config.PermissionsV2Url)}
}

type Security struct {
	config configuration.Config
	openid *OpenidToken
	permv2 permv2.Client
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
	permList, err := model.PermissionListFromString(rights)
	if err != nil {
		return false, err
	}
	allowed, err, _ = this.permv2.CheckPermission(token, kind, id, permList...)
	if err != nil {
		return false, err
	}
	return allowed, nil
}

func (this *Security) CheckMultiple(token string, kind string, ids []string, rights string) (result map[string]bool, err error) {
	result = map[string]bool{}
	if this.IsAdmin(token) {
		for _, id := range ids {
			result[id] = true
		}
		return result, nil
	}
	permList, err := model.PermissionListFromString(rights)
	if err != nil {
		return nil, err
	}
	result, err, _ = this.permv2.CheckMultiplePermissions(token, kind, ids, permList...)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return map[string]bool{}, nil
	}
	return result, nil
}
