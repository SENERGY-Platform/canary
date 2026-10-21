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
	"context"
	"strings"
	"testing"

	"github.com/SENERGY-Platform/automation-engine/pkg/client"
	"github.com/SENERGY-Platform/canary/pkg/automation"
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

func TestDeploymentInputUsesConfiguredCriteria(t *testing.T) {
	events := testEvents()
	input, err := events.getDeploymentInput("device-id", "service-id")
	if err != nil {
		t.Fatal(err)
	}

	bpmn, err := events.getProcessBpmn()
	if err != nil {
		t.Fatal(err)
	}
	if input.Xml != bpmn {
		t.Error("xml does not match the prepared bpmn")
	}
	if input.Name != ExpectedCanaryDeploymentName {
		t.Error("unexpected name", input.Name)
	}

	selection, ok := input.Selections[EventId]
	if !ok || len(input.Selections) != 1 {
		t.Fatal("unexpected selections", input.Selections)
	}
	if selection.DeviceId != "device-id" {
		t.Error("unexpected device id", selection.DeviceId)
	}
	if selection.ServiceId != "service-id" {
		t.Error("unexpected service id", selection.ServiceId)
	}
	//the selected path is what the deployment reads the event value from, it describes the same variable
	if selection.Path == nil || selection.Path.Path != "value" || selection.Path.CharacteristicId != testSensorCharacteristicId {
		t.Error("unexpected path", selection.Path)
	}
}

// the engine converts the bpmn of the old designer on deploy; nothing of it may be unsupported
// and the start event the selection refers to has to survive the conversion
func TestEngineConvertsEventProcessBpmn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	testEngine, err := client.NewTestClient(ctx)
	if err != nil {
		t.Fatal(err)
	}
	bpmn, err := testEvents().getProcessBpmn()
	if err != nil {
		t.Fatal(err)
	}
	conversion, err := testEngine.Convert(ctx, client.TestToken("canary"), automation.Namespace, bpmn)
	if err != nil {
		t.Fatal(err)
	}
	if len(conversion.Unsupported) > 0 {
		t.Error("unsupported", conversion.Unsupported)
	}
	if issues := automation.BlockingIssues(conversion.Issues); len(issues) > 0 {
		t.Error("issues", issues)
	}
	if !strings.Contains(conversion.Xml, `id="`+EventId+`"`) {
		t.Error("start event missing in converted bpmn")
	}
}
