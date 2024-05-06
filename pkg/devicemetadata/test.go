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

package devicemetadata

import (
	"bytes"
	"encoding/json"
	devicemodel "github.com/SENERGY-Platform/device-repository/lib/model"
	"github.com/SENERGY-Platform/permission-search/lib/client"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
	"time"
)

func (this *DeviceMetaData) TestMetadata(token string, info DeviceInfo) {
	//read current device
	this.metrics.DeviceRepoRequestCount.Inc()
	start := time.Now()
	d, err, _ := this.devicerepo.ReadDevice(info.Id, token, devicemodel.READ)
	this.metrics.DeviceRepoRequestLatencyMs.Set(float64(time.Since(start).Milliseconds()))
	if err != nil {
		this.metrics.DeviceRepoRequestErr.Inc()
		log.Println("ERROR:", err)
		debug.PrintStack()
		return
	}

	//set name
	d.Name = "canary-" + time.Now().String()

	//save device with changed name
	buf := &bytes.Buffer{}
	err = json.NewEncoder(buf).Encode(d)
	if err != nil {
		this.metrics.UncategorizedErr.Inc()
		log.Println("ERROR:", err)
		debug.PrintStack()
		return
	}
	this.metrics.DeviceMetaUpdateCount.Inc()
	req, err := http.NewRequest(http.MethodPut, this.config.DeviceManagerUrl+"/devices/"+url.PathEscape(d.Id)+"?wait=true", buf)
	if err != nil {
		this.metrics.UncategorizedErr.Inc()
		log.Println("ERROR:", err)
		debug.PrintStack()
		return
	}
	req.Header.Set("Authorization", token)
	start = time.Now()
	_, _, err = Do[DeviceInfo](req)
	this.metrics.DeviceMetaUpdateLatencyMs.Set(float64(time.Since(start).Milliseconds()))
	if err != nil {
		this.metrics.DeviceMetaUpdateErr.Inc()
		log.Println("ERROR:", err)
		debug.PrintStack()
	}

	//check device-repo for name change
	this.metrics.DeviceRepoRequestCount.Inc()
	start = time.Now()
	repoDevice, err, _ := this.devicerepo.ReadDevice(info.Id, token, devicemodel.READ)
	this.metrics.DeviceRepoRequestLatencyMs.Set(float64(time.Since(start).Milliseconds()))
	if err != nil {
		this.metrics.DeviceRepoRequestErr.Inc()
		log.Println("ERROR:", err)
		debug.PrintStack()
		return
	}

	if repoDevice.Name != d.Name {
		this.metrics.UnexpectedDeviceRepoMetadataErr.Inc()
		log.Printf("UnexpectedDeviceRepoMetadataErr: %#v != %#v\n", repoDevice.Name, d.Name)
	}

	//check permission search for name change
	this.metrics.PermissionsRequestCount.Inc()
	start = time.Now()
	permDevice, _, err := client.Query[[]PermDevice](this.permissions, token, client.QueryMessage{
		Resource: "devices",
		ListIds: &client.QueryListIds{
			QueryListCommons: model.QueryListCommons{
				Limit:  1,
				Offset: 0,
				Rights: "r",
			},
			Ids: []string{info.Id},
		},
	})
	this.metrics.PermissionsRequestLatencyMs.Set(float64(time.Since(start).Milliseconds()))
	if err != nil {
		this.metrics.PermissionsRequestErr.Inc()
		return
	}
	if len(permDevice) == 0 {
		this.metrics.UncategorizedErr.Inc()
		log.Printf("Unexpected conn state result: \n%#v\n", permDevice)
		debug.PrintStack()
		return
	}

	if permDevice[0].Name != d.Name {
		this.metrics.UnexpectedPermissionsMetadataErr.Inc()
		log.Printf("UnexpectedDeviceRepoMetadataErr: %#v != %#v\n", permDevice[0].Name, d.Name)
	}
}
