/*
 * Copyright (c) 2023 InfAI (CC SES)
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

package canary

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (this *Canary) login() (token string, refreshToken string, err error) {
	this.metrics.AuthCount.Inc()
	defer func() {
		if err != nil {
			log.Println("ERROR: login():", err)
			this.metrics.AuthErr.Inc()
		}
	}()
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	start := time.Now()
	var resp *http.Response
	resp, err = client.PostForm(this.config.AuthEndpoint+"/auth/realms/master/protocol/openid-connect/token", url.Values{
		"client_id":  {this.config.AuthClientId},
		"username":   {this.config.AuthUsername},
		"password":   {this.config.AuthPassword},
		"grant_type": {"password"},
	})
	this.metrics.AuthLatencyMs.Set(float64(time.Since(start).Milliseconds()))
	if err != nil {
		return token, refreshToken, err
	}
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		err = errors.New(resp.Status + ": " + string(b))
		return
	}

	temp := OpenidToken{}

	err = json.NewDecoder(resp.Body).Decode(&temp)
	token = "Bearer " + temp.AccessToken
	refreshToken = temp.RefreshToken
	return
}

func (this *Canary) logout(token string, refreshToken string) (err error) {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	var resp *http.Response
	resp, err = client.PostForm(this.config.AuthEndpoint+"/auth/realms/master/protocol/openid-connect/logout", url.Values{
		"client_id":     {this.config.AuthClientId},
		"refresh_token": {refreshToken},
		"id_token_hint": {strings.TrimPrefix(token, "Bearer ")},
	})
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		err = errors.New(resp.Status + ": " + string(b))
		return
	}
	return
}

type OpenidToken struct {
	AccessToken      string    `json:"access_token"`
	ExpiresIn        float64   `json:"expires_in"`
	RefreshExpiresIn float64   `json:"refresh_expires_in"`
	RefreshToken     string    `json:"refresh_token"`
	TokenType        string    `json:"token_type"`
	RequestTime      time.Time `json:"-"`
}
