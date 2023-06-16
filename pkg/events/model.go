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

type ProcessInstance struct {
	Id                    string `json:"id"`
	ProcessDefinitionName string `json:"processDefinitionName"`
	StartTime             string `json:"startTime"`
	EndTime               string `json:"endTime"`
	DurationInMillis      int    `json:"durationInMillis"`
	State                 string `json:"state"`
}

type PreparedDeployment struct {
	Id       string    `json:"id"`
	Name     string    `json:"name"`
	Elements []Element `json:"elements"`
}

type Element struct {
	BpmnId           string            `json:"bpmn_id"`
	ConditionalEvent *ConditionalEvent `json:"conditional_event"`
}

type ConditionalEvent struct {
	Script        string            `json:"script"`
	ValueVariable string            `json:"value_variable"`
	Variables     map[string]string `json:"variables"`
	Qos           int               `json:"qos"`
	EventId       string            `json:"event_id"`
	Selection     Selection         `json:"selection"`
}

type Selection struct {
	SelectionOptions []SelectionOption `json:"selection_options"`
}

type SelectionOption struct {
	Device   *Device   `json:"device"`
	Services []Service `json:"services"`
}

type Device struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type DeviceGroup struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type Service struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}
