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
	"bytes"
	_ "embed"
	"text/template"

	"github.com/SENERGY-Platform/automation-engine/pkg/model"
	"github.com/SENERGY-Platform/canary/pkg/automation"
)

const ExpectedCanaryDeploymentName = "canary_process"

// TaskId is the service task that sends the command to the canary device.
const TaskId = "Task_0fa1ff0"

// selectionCriteria are the ids the service task filters on. they have to match
// the cmd service that devicemetadata creates, so they come from the same config.
func (this *Process) selectionCriteria() map[string]string {
	return map[string]string{
		"CmdFunctionId":       this.config.CanaryCmdFunctionId,
		"CmdCharacteristicId": this.config.CanaryCmdCharacteristicId,
		"DeviceClassId":       this.config.CanaryDeviceClassId,
	}
}

// getDeploymentInput selects the canary device for the task and sets the command value to 42.
// the bpmn stays in the format of the old designer, the engine converts it on deploy.
func (this *Process) getDeploymentInput(deviceId string, serviceId string) (input model.DeploymentInput, err error) {
	bpmn, err := this.getProcessBpmn()
	if err != nil {
		return input, err
	}
	return model.DeploymentInput{
		Name:             ExpectedCanaryDeploymentName,
		Description:      "no description",
		Xml:              bpmn,
		Svg:              ProcessSvg,
		Source:           model.DefaultDeploymentSource,
		IncidentHandling: &model.IncidentHandling{Restart: false, Notify: true},
		Selections: map[string]model.Selection{
			TaskId: {
				DeviceId:  deviceId,
				ServiceId: serviceId,
				Path:      &model.SelectedPath{Path: "value", CharacteristicId: this.config.CanaryCmdCharacteristicId},
			},
		},
		Parameters: map[string]model.ElementParameters{
			TaskId: {Inputs: map[string]string{"inputs": "42"}},
		},
	}, nil
}

func (this *Process) DeployProcess(token string, deviceId string, serviceId string) (deploymentId string, err error) {
	input, err := this.getDeploymentInput(deviceId, serviceId)
	if err != nil {
		return "", err
	}
	ctx, cancel := automation.Context()
	defer cancel()
	depl, err := this.engine.CreateDeployment(ctx, token, automation.Namespace, input)
	if err != nil {
		return "", err
	}
	return depl.Id, nil
}

func (this *Process) ListCanaryProcessDeployments(token string) (ids []string, err error) {
	return automation.ListDeploymentIds(this.engine, token, ExpectedCanaryDeploymentName)
}

func (this *Process) DeleteProcess(token string, deploymentId string) (err error) {
	ctx, cancel := automation.Context()
	defer cancel()
	return this.engine.DeleteDeployment(ctx, token, automation.Namespace, deploymentId)
}

func (this *Process) StartProcess(token string, deploymentId string) (err error) {
	ctx, cancel := automation.Context()
	defer cancel()
	_, err = this.engine.StartInstance(ctx, token, automation.Namespace, model.StartInstanceInput{DeploymentId: deploymentId})
	return err
}

func (this *Process) GetProcessInstances(token string, deploymentIds []string) (result []model.Instance, err error) {
	return automation.ListInstances(this.engine, token, deploymentIds)
}

//go:embed canary_process.bpmn
var ProcessBpmnTemplate string

//go:embed canary_process.svg
var ProcessSvg string

func (this *Process) getProcessBpmn() (result string, err error) {
	templ, err := template.New("bpmn").Parse(ProcessBpmnTemplate)
	if err != nil {
		return result, err
	}
	buff := &bytes.Buffer{}
	err = templ.Execute(buff, this.selectionCriteria())
	return buff.String(), err
}

func (this *Process) PrepareProcessDeployment(token string) (result model.PreparedDeployment, err error) {
	bpmn, err := this.getProcessBpmn()
	if err != nil {
		return result, err
	}
	ctx, cancel := automation.Context()
	defer cancel()
	return this.engine.PrepareDeployment(ctx, token, automation.Namespace, model.PrepareInput{Xml: bpmn})
}
