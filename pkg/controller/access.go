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

func (this *Controller) ApiCheckAccess(request *http.Request, networkId string) (err error, errCode int) {
	token := request.Header.Get("Authorization")
	allowed, err := this.security.CheckBool(token, "hubs", networkId, "rx")
	if err != nil {
		return err, http.StatusInternalServerError
	}
	if !allowed {
		return errors.New("not allowed"), http.StatusForbidden
	}
	return nil, http.StatusOK
}
