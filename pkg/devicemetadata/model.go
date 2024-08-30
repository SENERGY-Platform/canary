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

import "github.com/SENERGY-Platform/models/go/models"

type DeviceInfo = models.Device
type DeviceTypeInfo struct {
	Id       string   `json:"id"`
	Services []string `json:"service"`
}

type PermDevice struct {
	Id          string                 `json:"id"`
	Name        string                 `json:"name"`
	Annotations map[string]interface{} `json:"annotations"`
}

const AttributeUsedForCanaryDevice = "senergy/canary-device"
const AttributeUsedForCanaryDeviceType = "senergy/canary-device-type"
const SensorServiceLocalId = "sensor"
const CmdServiceLocalId = "cmd"
