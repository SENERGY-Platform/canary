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
	"errors"
	devicemodel "github.com/SENERGY-Platform/device-repository/lib/model"
	"github.com/SENERGY-Platform/models/go/models"
	"github.com/SENERGY-Platform/permission-search/lib/client"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	"github.com/google/uuid"
	"io"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
	"sync"
	"time"
)

type DeviceInfo = models.Device
type DeviceTypeInfo struct {
	Id       string   `json:"id"`
	Services []string `json:"service"`
}

const AttributeUsedForCanaryDevice = "senergy/canary-device"
const AttributeUsedForCanaryDeviceType = "senergy/canary-device-type"
const SensorServiceLocalId = "sensor"

func (this *Canary) testMetadata(wg *sync.WaitGroup, token string, info DeviceInfo) {
	wg.Add(1)
	go func() {
		defer wg.Done()

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
		req, err := http.NewRequest(http.MethodPut, this.config.DeviceManagerUrl+"/devices/"+url.PathEscape(d.Id), buf)
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

		time.Sleep(this.getChangeGuaranteeDuration()) //wait for cqrs

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
	}()
}

func (this *Canary) ensureDevice(token string) (device DeviceInfo, err error) {
	canaryDevices, err := this.listCanaryDevices(token)
	if err != nil {
		return device, err
	}
	if len(canaryDevices) > 0 {
		return canaryDevices[0], nil
	} else {
		return this.createCanaryDevice(token)
	}
}

func (this *Canary) listCanaryDevices(token string) (devices []DeviceInfo, err error) {
	start := time.Now()
	devices, _, err = client.Query[[]DeviceInfo](this.permissions, token, client.QueryMessage{
		Resource: "devices",
		Find: &client.QueryFind{
			QueryListCommons: client.QueryListCommons{
				Limit:  1,
				Offset: 0,
				Rights: "r",
				SortBy: "name",
			},
			Filter: &client.Selection{
				Condition: client.ConditionConfig{
					Feature:   "features.attributes.key",
					Operation: client.QueryEqualOperation,
					Value:     AttributeUsedForCanaryDevice,
				},
			},
		},
	})
	this.metrics.PermissionsRequestCount.Inc()
	this.metrics.PermissionsRequestLatencyMs.Set(float64(time.Since(start).Milliseconds()))
	if err != nil {
		this.metrics.PermissionsRequestErr.Inc()
	}
	return devices, err
}

func (this *Canary) createCanaryDevice(token string) (device DeviceInfo, err error) {
	dt, err := this.ensureDeviceType(token)
	if err != nil {
		return device, err
	}
	device = DeviceInfo{
		LocalId: "canary_" + uuid.NewString(),
		Name:    "canary-" + time.Now().String(),
		Attributes: []models.Attribute{{
			Key:    AttributeUsedForCanaryDevice,
			Value:  "true",
			Origin: "canary",
		}},
		DeviceTypeId: dt.Id,
	}

	buf := &bytes.Buffer{}
	err = json.NewEncoder(buf).Encode(device)
	if err != nil {
		this.metrics.UncategorizedErr.Inc()
		log.Println("ERROR:", err)
		debug.PrintStack()
		return device, err
	}
	this.metrics.DeviceMetaUpdateCount.Inc()
	req, err := http.NewRequest(http.MethodPost, this.config.DeviceManagerUrl+"/devices", buf)
	if err != nil {
		this.metrics.UncategorizedErr.Inc()
		log.Println("ERROR:", err)
		debug.PrintStack()
		return device, err
	}
	req.Header.Set("Authorization", token)
	start := time.Now()
	device, _, err = Do[DeviceInfo](req)
	this.metrics.DeviceMetaUpdateLatencyMs.Set(float64(time.Since(start).Milliseconds()))
	if err != nil {
		this.metrics.DeviceMetaUpdateErr.Inc()
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
	time.Sleep(this.getChangeGuaranteeDuration()) //ensure device is finished creating
	return device, err
}

func (this *Canary) ensureDeviceType(token string) (result DeviceTypeInfo, err error) {
	canaryDeviceTypes, err := this.listCanaryDeviceTypes(token)
	if err != nil {
		return result, err
	}
	if len(canaryDeviceTypes) > 0 {
		return canaryDeviceTypes[0], nil
	} else {
		return this.createCanaryDeviceType(token)
	}
}

func (this *Canary) listCanaryDeviceTypes(token string) (dts []DeviceTypeInfo, err error) {
	start := time.Now()
	dts, _, err = client.Query[[]DeviceTypeInfo](this.permissions, token, client.QueryMessage{
		Resource: "device-types",
		Find: &client.QueryFind{
			QueryListCommons: client.QueryListCommons{
				Limit:  1,
				Offset: 0,
				Rights: "r",
				SortBy: "name",
			},
			Filter: &client.Selection{
				Condition: client.ConditionConfig{
					Feature:   "features.attributes.key",
					Operation: client.QueryEqualOperation,
					Value:     AttributeUsedForCanaryDeviceType,
				},
			},
		},
	})
	this.metrics.PermissionsRequestCount.Inc()
	this.metrics.PermissionsRequestLatencyMs.Set(float64(time.Since(start).Milliseconds()))
	if err != nil {
		this.metrics.PermissionsRequestErr.Inc()
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
	return dts, err
}

func (this *Canary) createCanaryDeviceType(token string) (deviceType DeviceTypeInfo, err error) {
	dt := models.DeviceType{
		Name:          "canary-device-type",
		Description:   "used for canary service github.com/SENERGY-Platform/canary",
		DeviceClassId: this.config.CanaryDeviceClassId,
		Attributes: []models.Attribute{{
			Key:    AttributeUsedForCanaryDeviceType,
			Value:  "true",
			Origin: "canary",
		}},
		Services: []models.Service{
			{
				LocalId:     "cmd",
				Name:        "cmd",
				Description: "canary cmd service, needed to test online state by subscription",
				Interaction: models.REQUEST,
				ProtocolId:  this.config.CanaryProtocolId,
				Inputs: []models.Content{
					{
						ContentVariable: models.ContentVariable{
							Name:             "value",
							Type:             models.Type(this.config.CanaryCmdValueType),
							CharacteristicId: this.config.CanaryCmdCharacteristicId,
							FunctionId:       this.config.CanaryCmdFunctionId,
						},
						Serialization:     models.JSON,
						ProtocolSegmentId: this.config.CanaryProtocolSegmentId,
					},
				},
			},
			{
				LocalId:     SensorServiceLocalId,
				Name:        "sensor",
				Description: "canary sensor service, needed to test device data handling",
				Interaction: models.EVENT,
				ProtocolId:  this.config.CanaryProtocolId,
				Outputs: []models.Content{
					{
						ContentVariable: models.ContentVariable{
							Name:             "value",
							Type:             models.Type(this.config.CanarySensorValueType),
							CharacteristicId: this.config.CanarySensorCharacteristicId,
							FunctionId:       this.config.CanarySensorFunctionId,
							AspectId:         this.config.CanarySensorAspectId,
						},
						Serialization:     models.JSON,
						ProtocolSegmentId: this.config.CanaryProtocolSegmentId,
					},
				},
			},
		},
	}

	buf := &bytes.Buffer{}
	err = json.NewEncoder(buf).Encode(dt)
	if err != nil {
		return deviceType, err
	}
	this.metrics.DeviceMetaUpdateCount.Inc()
	req, err := http.NewRequest(http.MethodPost, this.config.DeviceManagerUrl+"/device-types", buf)
	if err != nil {
		this.metrics.UncategorizedErr.Inc()
		log.Println("ERROR:", err)
		debug.PrintStack()
		return deviceType, err
	}
	req.Header.Set("Authorization", token)
	start := time.Now()
	deviceType, _, err = Do[DeviceTypeInfo](req)
	this.metrics.DeviceMetaUpdateLatencyMs.Set(float64(time.Since(start).Milliseconds()))
	if err != nil {
		this.metrics.DeviceMetaUpdateErr.Inc()
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
	time.Sleep(this.getChangeGuaranteeDuration()) //ensure device-type is finished creating
	return deviceType, err
}

func Do[T any](req *http.Request) (result T, code int, err error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, http.StatusInternalServerError, err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		temp, _ := io.ReadAll(resp.Body) //read error response end ensure that resp.Body is read to EOF
		return result, resp.StatusCode, errors.New(string(temp))
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		_, _ = io.ReadAll(resp.Body) //ensure resp.Body is read to EOF
		return result, http.StatusInternalServerError, err
	}
	return result, resp.StatusCode, nil
}
