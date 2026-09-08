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

package events

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/SENERGY-Platform/canary/pkg/configuration"
)

const (
	testSensorFunctionId       = "urn:infai:ses:measuring-function:test-function"
	testSensorAspectId         = "urn:infai:ses:aspect:test-aspect"
	testSensorCharacteristicId = "urn:infai:ses:characteristic:test-characteristic"
)

func testEvents() *Events {
	return &Events{config: configuration.Config{
		CanarySensorFunctionId:       testSensorFunctionId,
		CanarySensorAspectId:         testSensorAspectId,
		CanarySensorCharacteristicId: testSensorCharacteristicId,
	}}
}

// the conditional event has to filter on the same ids the sensor service is created with,
// otherwise the prepared deployment contains no selectables
func TestProcessBpmnUsesConfiguredCriteria(t *testing.T) {
	bpmn, err := testEvents().getProcessBpmn()
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		`senergy:aspect="` + testSensorAspectId + `"`,
		`senergy:function="` + testSensorFunctionId + `"`,
		`senergy:characteristic="` + testSensorCharacteristicId + `"`,
	} {
		if !strings.Contains(bpmn, expected) {
			t.Error("missing", expected)
		}
	}
	if strings.Contains(bpmn, "{{") {
		t.Error("unresolved template action in bpmn")
	}
}

func TestDeploymentMessageUsesConfiguredCriteria(t *testing.T) {
	events := testEvents()
	buff, err := events.getDeploymentMessage("device-id", "service-id")
	if err != nil {
		t.Fatal(err)
	}
	depl := map[string]interface{}{}
	err = json.Unmarshal(buff.Bytes(), &depl)
	if err != nil {
		t.Fatal(err)
	}

	bpmn, err := events.getProcessBpmn()
	if err != nil {
		t.Fatal(err)
	}
	diagram := depl["diagram"].(map[string]interface{})
	if diagram["xml_raw"] != bpmn {
		t.Error("xml_raw does not match the prepared bpmn")
	}

	element := depl["elements"].([]interface{})[0].(map[string]interface{})
	selection := element["conditional_event"].(map[string]interface{})["selection"].(map[string]interface{})
	criteria := selection["filter_criteria"].(map[string]interface{})
	if criteria["function_id"] != testSensorFunctionId {
		t.Error("unexpected function_id", criteria["function_id"])
	}
	if criteria["aspect_id"] != testSensorAspectId {
		t.Error("unexpected aspect_id", criteria["aspect_id"])
	}
	if criteria["characteristic_id"] != testSensorCharacteristicId {
		t.Error("unexpected characteristic_id", criteria["characteristic_id"])
	}

	//the selected path is what the deployment reads the event value from, it describes the same variable
	path := selection["selected_path"].(map[string]interface{})
	if path["functionId"] != testSensorFunctionId {
		t.Error("unexpected path functionId", path["functionId"])
	}
	if path["characteristicId"] != testSensorCharacteristicId {
		t.Error("unexpected path characteristicId", path["characteristicId"])
	}
	if path["aspectNode"].(map[string]interface{})["id"] != testSensorAspectId {
		t.Error("unexpected path aspectNode id")
	}

	if selection["selected_device_id"] != "device-id" {
		t.Error("unexpected selected_device_id", selection["selected_device_id"])
	}
	if selection["selected_service_id"] != "service-id" {
		t.Error("unexpected selected_service_id", selection["selected_service_id"])
	}
}
