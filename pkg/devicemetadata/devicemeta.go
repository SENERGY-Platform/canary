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
	"errors"
	"github.com/SENERGY-Platform/canary/pkg/configuration"
	"github.com/SENERGY-Platform/canary/pkg/metrics"
	devicerepo "github.com/SENERGY-Platform/device-repository/lib/client"
	"github.com/SENERGY-Platform/device-repository/lib/model"
	"github.com/SENERGY-Platform/models/go/models"
	"github.com/google/uuid"
	"io"
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

type DeviceMetaData struct {
	devicerepo           devicerepo.Interface
	metrics              *metrics.Metrics
	config               configuration.Config
	guaranteeChangeAfter time.Duration
}

func NewDeviceMetaData(devicerepo devicerepo.Interface, metrics *metrics.Metrics, config configuration.Config, guaranteeChangeAfter time.Duration) *DeviceMetaData {
	return &DeviceMetaData{devicerepo: devicerepo, metrics: metrics, config: config, guaranteeChangeAfter: guaranteeChangeAfter}
}

func (this *DeviceMetaData) getChangeGuaranteeDuration() time.Duration {
	return this.guaranteeChangeAfter
}

func (this *DeviceMetaData) EnsureDevice(token string) (device DeviceInfo, err error) {
	canaryDevices, err := this.ListCanaryDevices(token)
	if err != nil {
		return device, err
	}
	if len(canaryDevices) > 0 {
		return canaryDevices[0], nil
	} else {
		return this.CreateCanaryDevice(token)
	}
}

func (this *DeviceMetaData) ListCanaryDevices(token string) (devices []DeviceInfo, err error) {
	start := time.Now()
	devices, err, _ = this.devicerepo.ListDevices(token, model.DeviceListOptions{Limit: 1, AttributeKeys: []string{AttributeUsedForCanaryDevice}})
	this.metrics.DeviceRepoRequestCount.Inc()
	this.metrics.DeviceRepoRequestLatencyMs.Set(float64(time.Since(start).Milliseconds()))
	if err != nil {
		log.Println("ERROR: ListCanaryDevices()", err)
		this.metrics.DeviceRepoRequestErr.Inc()
	}
	return devices, err
}

func (this *DeviceMetaData) CreateCanaryDevice(token string) (device DeviceInfo, err error) {
	dt, err := this.EnsureDeviceType(token)
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
	req, err := http.NewRequest(http.MethodPost, this.config.DeviceManagerUrl+"/devices?wait=true", buf)
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
	time.Sleep(this.getChangeGuaranteeDuration())
	return device, err
}

func (this *DeviceMetaData) EnsureDeviceType(token string) (result DeviceTypeInfo, err error) {
	canaryDeviceTypes, err := this.ListCanaryDeviceTypes(token)
	if err != nil {
		return result, err
	}
	if len(canaryDeviceTypes) > 0 {
		return canaryDeviceTypes[0], nil
	} else {
		return this.CreateCanaryDeviceType(token)
	}
}

func (this *DeviceMetaData) ListCanaryDeviceTypes(token string) (result []DeviceTypeInfo, err error) {
	start := time.Now()
	deviceTypes, _, err, _ := this.devicerepo.ListDeviceTypesV3(token, model.DeviceTypeListOptions{
		Limit:         1,
		Offset:        0,
		SortBy:        "name",
		AttributeKeys: []string{AttributeUsedForCanaryDeviceType},
	})
	this.metrics.DeviceRepoRequestCount.Inc()
	this.metrics.DeviceRepoRequestLatencyMs.Set(float64(time.Since(start).Milliseconds()))
	if err != nil {
		this.metrics.DeviceRepoRequestErr.Inc()
		log.Println("ERROR:", err)
		debug.PrintStack()
	}
	for _, dt := range deviceTypes {
		services := []string{}
		for _, s := range dt.Services {
			services = append(services, s.Id)
		}
		result = append(result, DeviceTypeInfo{
			Id:       dt.Id,
			Services: services,
		})
	}
	return result, err
}

func (this *DeviceMetaData) CreateCanaryDeviceType(token string) (deviceType DeviceTypeInfo, err error) {
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
				LocalId:     CmdServiceLocalId,
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
	req, err := http.NewRequest(http.MethodPost, this.config.DeviceManagerUrl+"/device-types?wait=true", buf)
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
	time.Sleep(this.getChangeGuaranteeDuration())
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
