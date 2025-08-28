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

package security

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
)

func (this *Security) GetAdminToken() (token string, err error) {
	if this.openid == nil {
		this.openid = &OpenidToken{}
	}
	duration := time.Since(this.openid.RequestTime).Seconds()

	if this.openid.AccessToken != "" && this.openid.ExpiresIn-this.config.AuthExpirationTimeBuffer > duration {
		token = "Bearer " + this.openid.AccessToken
		return
	}

	if this.openid.RefreshToken != "" && this.openid.RefreshExpiresIn-this.config.AuthExpirationTimeBuffer > duration {
		this.config.GetLogger().Info("refresh token", "expires_in", this.openid.RefreshExpiresIn, "duration", duration)
		err = refreshOpenidToken(this.openid, this.config)
		if err != nil {
			this.config.GetLogger().Warn("unable to refresh token", "error", err)
		} else {
			token = "Bearer " + this.openid.AccessToken
			return
		}
	}

	this.config.GetLogger().Info("get new access token")
	err = getOpenidToken(this.openid, this.config)
	if err != nil {
		this.config.GetLogger().Error("unable to get new access token", "error", err)
		this.openid = &OpenidToken{}
	}
	token = "Bearer " + this.openid.AccessToken
	return
}

func getOpenidToken(token *OpenidToken, config configuration.Config) (err error) {
	requesttime := time.Now()
	resp, err := http.PostForm(config.AuthEndpoint+"/auth/realms/master/protocol/openid-connect/token", url.Values{
		"client_id":     {config.AuthClientId},
		"client_secret": {config.AuthClientSecret},
		"grant_type":    {"client_credentials"},
	})

	if err != nil {
		config.GetLogger().Error("getOpenidToken::PostForm()", "error", err)
		return err
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		config.GetLogger().Error("getOpenidToken()", "status_code", resp.StatusCode, "payload", string(body))
		err = errors.New("access denied")
		resp.Body.Close()
		return
	}
	err = json.NewDecoder(resp.Body).Decode(token)
	token.RequestTime = requesttime
	return
}

func refreshOpenidToken(token *OpenidToken, config configuration.Config) (err error) {
	requesttime := time.Now()
	resp, err := http.PostForm(config.AuthEndpoint+"/auth/realms/master/protocol/openid-connect/token", url.Values{
		"client_id":     {config.AuthClientId},
		"client_secret": {config.AuthClientSecret},
		"refresh_token": {token.RefreshToken},
		"grant_type":    {"refresh_token"},
	})

	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		config.GetLogger().Error("refreshOpenidToken()", "status_code", resp.StatusCode, "payload", string(body))
		err = errors.New("access denied")
		resp.Body.Close()
		return
	}
	err = json.NewDecoder(resp.Body).Decode(token)
	token.RequestTime = requesttime
	return
}
