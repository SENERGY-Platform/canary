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
	"bytes"
	"encoding/json"
	"github.com/SENERGY-Platform/canary/pkg/devicemetadata"
	"github.com/SENERGY-Platform/device-repository/lib/client"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
	"time"
)

func (this *Canary) ensureHub(token string, device DeviceInfo) (hubId string, err error) {
	canaryHubs, err := this.listCanaryHubs(token)
	if err != nil {
		return "", err
	}
	if len(canaryHubs) > 0 {
		hub := canaryHubs[0]
		if contains(hub.DeviceIds, device.Id) && contains(hub.DeviceLocalIds, device.LocalId) {
			return hub.Id, nil
		} else {
			err = this.updateCanaryHub(token, hub.Id, device)
			return hub.Id, err
		}
	} else {
		return this.createCanaryHub(token, device)
	}
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

type HubInfo struct {
	Id             string   `json:"id"`
	Name           string   `json:"name"`
	DeviceIds      []string `json:"device_ids,omitempty"`
	DeviceLocalIds []string `json:"device_local_ids,omitempty"`
}

func (this *Canary) listCanaryHubs(token string) (hubs []HubInfo, err error) {
	start := time.Now()
	temp, err, _ := this.devicerepo.ListHubs(token, client.HubListOptions{
		Search: this.config.CanaryHubName,
		Limit:  1,
		Offset: 0,
	})
	this.metrics.DeviceRepoRequestCount.Inc()
	this.metrics.DeviceRepoRequestLatencyMs.Set(float64(time.Since(start).Milliseconds()))
	if err != nil {
		this.metrics.DeviceRepoRequestErr.Inc()
		log.Println("ERROR:", err)
		debug.PrintStack()
		return hubs, err
	}
	for _, hub := range temp {
		hubs = append(hubs, HubInfo{
			Id:             hub.Id,
			Name:           hub.Name,
			DeviceIds:      hub.DeviceIds,
			DeviceLocalIds: hub.DeviceLocalIds,
		})
	}
	return hubs, err
}

func (this *Canary) createCanaryHub(token string, device DeviceInfo) (hubId string, err error) {
	hub := HubInfo{
		Name:           this.config.CanaryHubName,
		DeviceLocalIds: []string{device.LocalId},
	}

	buf := &bytes.Buffer{}
	err = json.NewEncoder(buf).Encode(hub)
	if err != nil {
		return "", err
	}
	this.metrics.DeviceMetaUpdateCount.Inc()
	req, err := http.NewRequest(http.MethodPost, this.config.DeviceManagerUrl+"/hubs?wait=true", buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", token)
	start := time.Now()
	hub, _, err = devicemetadata.Do[HubInfo](req)
	this.metrics.DeviceMetaUpdateLatencyMs.Set(float64(time.Since(start).Milliseconds()))
	if err != nil {
		this.metrics.DeviceMetaUpdateErr.Inc()
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
	time.Sleep(this.getChangeGuaranteeDuration())
	return hub.Id, err
}

func (this *Canary) updateCanaryHub(token string, hubId string, device DeviceInfo) (err error) {
	hub := HubInfo{
		Id:             hubId,
		Name:           this.config.CanaryHubName,
		DeviceLocalIds: []string{device.LocalId},
	}

	buf := &bytes.Buffer{}
	err = json.NewEncoder(buf).Encode(hub)
	if err != nil {
		return err
	}
	this.metrics.DeviceMetaUpdateCount.Inc()
	req, err := http.NewRequest(http.MethodPut, this.config.DeviceManagerUrl+"/hubs/"+url.PathEscape(hub.Id)+"?wait=true", buf)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", token)
	start := time.Now()
	hub, _, err = devicemetadata.Do[HubInfo](req)
	this.metrics.DeviceMetaUpdateLatencyMs.Set(float64(time.Since(start).Milliseconds()))
	if err != nil {
		this.metrics.DeviceMetaUpdateErr.Inc()
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
	time.Sleep(this.getChangeGuaranteeDuration())
	return err
}
