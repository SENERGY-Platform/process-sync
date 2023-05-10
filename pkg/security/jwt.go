/*
 * Copyright 2020 InfAI (CC SES)
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
	"github.com/golang-jwt/jwt"
	"strings"
)

func ReadTokenRoles(token string) (roles []string, err error) {
	if token == "" {
		err = errors.New("missing Authorization token")
	}
	authParts := strings.Split(token, " ")
	if len(authParts) != 2 {
		return roles, errors.New("expect auth string format like '<type> <token>'")
	}
	claims := jwt.MapClaims{}
	parser := jwt.Parser{}
	_, _, err = parser.ParseUnverified(strings.Join(authParts[1:], " "), &claims)
	if err != nil {
		return roles, err
	}
	temp, ok := claims["roles"].([]interface{})
	if !ok {
		return roles, errors.New("missing jwt roles")
	}
	for _, roleInterface := range temp {
		role, ok := roleInterface.(string)
		if ok {
			roles = append(roles, role)
		} else {
			return roles, errors.New("expect string in jwt roles")
		}
	}
	return
}
