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
	"errors"
	"net/http"
)

func (this *Controller) ApiCheckAccess(request *http.Request, networkId string, rights string) (err error, errCode int) {
	token := request.Header.Get("Authorization")
	allowed, err := this.security.CheckBool(token, "hubs", networkId, rights)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	if !allowed {
		return errors.New("not allowed"), http.StatusForbidden
	}
	return nil, http.StatusOK
}

func (this *Controller) ApiCheckAccessReturnToken(request *http.Request, networkId string, rights string) (token string, err error, errCode int) {
	token = request.Header.Get("Authorization")
	allowed, err := this.security.CheckBool(token, "hubs", networkId, rights)
	if err != nil {
		return token, err, http.StatusInternalServerError
	}
	if !allowed {
		return token, errors.New("not allowed"), http.StatusForbidden
	}
	return token, nil, http.StatusOK
}

func (this *Controller) ApiCheckAccessMultiple(request *http.Request, networkIds []string, rights string) (err error, errCode int) {
	token := request.Header.Get("Authorization")
	allowed, err := this.security.CheckMultiple(token, "hubs", networkIds, rights)
	if err != nil {
		return err, http.StatusInternalServerError
	}
	for _, id := range networkIds {
		if !allowed[id] {
			return errors.New("not allowed"), http.StatusForbidden
		}
	}
	return nil, http.StatusOK
}
