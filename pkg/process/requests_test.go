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

package process

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/SENERGY-Platform/canary/pkg/configuration"
)

const (
	testCmdFunctionId       = "urn:infai:ses:controlling-function:test-function"
	testCmdCharacteristicId = "urn:infai:ses:characteristic:test-characteristic"
	testDeviceClassId       = "urn:infai:ses:device-class:test-device-class"
)

func testProcess() *Process {
	return &Process{config: configuration.Config{
		CanaryCmdFunctionId:       testCmdFunctionId,
		CanaryCmdCharacteristicId: testCmdCharacteristicId,
		CanaryDeviceClassId:       testDeviceClassId,
	}}
}

// the service task has to filter on the same ids the cmd service is created with,
// otherwise the prepared deployment contains no selectables
func TestProcessBpmnUsesConfiguredCriteria(t *testing.T) {
	bpmn, err := testProcess().getProcessBpmn()
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{testCmdFunctionId, testCmdCharacteristicId, testDeviceClassId} {
		if !strings.Contains(bpmn, expected) {
			t.Error("missing", expected)
		}
	}
	if strings.Contains(bpmn, "{{") {
		t.Error("unresolved template action in bpmn")
	}
}

func TestDeploymentMessageUsesConfiguredCriteria(t *testing.T) {
	process := testProcess()
	buff, err := process.getDeploymentMessage("device-id", "service-id")
	if err != nil {
		t.Fatal(err)
	}
	depl := map[string]interface{}{}
	err = json.Unmarshal(buff.Bytes(), &depl)
	if err != nil {
		t.Fatal(err)
	}

	bpmn, err := process.getProcessBpmn()
	if err != nil {
		t.Fatal(err)
	}
	diagram := depl["diagram"].(map[string]interface{})
	if diagram["xml_raw"] != bpmn {
		t.Error("xml_raw does not match the prepared bpmn")
	}

	element := depl["elements"].([]interface{})[0].(map[string]interface{})
	selection := element["task"].(map[string]interface{})["selection"].(map[string]interface{})
	criteria := selection["filter_criteria"].(map[string]interface{})
	if criteria["function_id"] != testCmdFunctionId {
		t.Error("unexpected function_id", criteria["function_id"])
	}
	if criteria["device_class_id"] != testDeviceClassId {
		t.Error("unexpected device_class_id", criteria["device_class_id"])
	}
	if criteria["characteristic_id"] != testCmdCharacteristicId {
		t.Error("unexpected characteristic_id", criteria["characteristic_id"])
	}

	//the selected path is what the task writes the command to, it describes the same variable
	path := selection["selected_path"].(map[string]interface{})
	if path["functionId"] != testCmdFunctionId {
		t.Error("unexpected path functionId", path["functionId"])
	}
	if path["characteristicId"] != testCmdCharacteristicId {
		t.Error("unexpected path characteristicId", path["characteristicId"])
	}

	if selection["selected_device_id"] != "device-id" {
		t.Error("unexpected selected_device_id", selection["selected_device_id"])
	}
	if selection["selected_service_id"] != "service-id" {
		t.Error("unexpected selected_service_id", selection["selected_service_id"])
	}
}
