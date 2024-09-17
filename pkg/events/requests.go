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
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"text/template"
)

//go:embed deployment.json
var DeploymentModelTemplate string

func getDeploymentMessage(deviceId string, serviceId string) (buff *bytes.Buffer, err error) {
	templ, err := template.New("deployment").Parse(DeploymentModelTemplate)
	if err != nil {
		return buff, err
	}
	buff = &bytes.Buffer{}
	err = templ.Execute(buff, map[string]string{"DeviceId": deviceId, "ServiceId": serviceId})
	return buff, err
}

func (this *Events) DeployProcess(token string, deviceId string, serviceId string) (deploymentId string, err error) {
	endpoint := this.config.ProcessDeploymentUrl + "/v3/deployments?source=sepl"
	method := "POST"

	buff, err := getDeploymentMessage(deviceId, serviceId)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(method, endpoint, buff)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		temp, _ := io.ReadAll(resp.Body) //read error response end ensure that resp.Body is read to EOF
		return "", errors.New("unable to deploy process: " + string(temp))
	}
	wrapper := Wrapper{}
	err = json.NewDecoder(resp.Body).Decode(&wrapper)
	if err != nil {
		_, _ = io.ReadAll(resp.Body) //ensure resp.Body is read to EOF
		return "", err
	}
	return wrapper.Id, nil
}

type Wrapper struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

const ExpectedCanaryDeploymentName = "canary_event_process"

func (this *Events) ListCanaryProcessDeployments(token string) (ids []string, err error) {
	limit := 200
	offset := 0
	for {
		sub, err := this.listCanaryProcessDeployments(token, limit, offset)
		if err != nil {
			return ids, err
		}
		for _, w := range sub {
			if w.Name == ExpectedCanaryDeploymentName {
				ids = append(ids, w.Id)
			}
		}
		if len(sub) < limit {
			return ids, nil
		}
		offset = limit + offset
		log.Println("ListCanaryProcessDeployments offset =", offset)
	}
}

func (this *Events) listCanaryProcessDeployments(token string, limit int, offset int) (wrappers []Wrapper, err error) {
	query := url.Values{"maxResults": {strconv.Itoa(limit)}}
	if offset > 0 {
		query.Set("firstResult", strconv.Itoa(offset))
	}
	endpoint := this.config.ProcessEngineWrapperUrl + "/v2/deployments?" + query.Encode()
	method := "GET"

	req, err := http.NewRequest(method, endpoint, nil)
	if err != nil {
		return wrappers, err
	}
	req.Header.Set("Authorization", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return wrappers, err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		temp, _ := io.ReadAll(resp.Body) //read error response end ensure that resp.Body is read to EOF
		return wrappers, errors.New("unable to list process deployments: " + string(temp))
	}
	err = json.NewDecoder(resp.Body).Decode(&wrappers)
	if err != nil {
		_, _ = io.ReadAll(resp.Body) //ensure resp.Body is read to EOF
		return wrappers, err
	}
	return wrappers, nil
}

func (this *Events) DeleteProcess(token string, deploymentId string) (err error) {
	endpoint := this.config.ProcessDeploymentUrl + "/v3/deployments/" + url.PathEscape(deploymentId)
	method := "DELETE"

	req, err := http.NewRequest(method, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		temp, _ := io.ReadAll(resp.Body) //read error response end ensure that resp.Body is read to EOF
		return errors.New("unable to delete event process: " + string(temp))
	}
	return nil
}

func (this *Events) GetProcessInstances(token string) (result []ProcessInstance, err error) {
	endpoint := this.config.ProcessEngineWrapperUrl + "/v2/history/process-instances?maxResults=20"
	method := "GET"

	req, err := http.NewRequest(method, endpoint, nil)
	if err != nil {
		return result, err
	}
	req.Header.Set("Authorization", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		temp, _ := io.ReadAll(resp.Body) //read error response end ensure that resp.Body is read to EOF
		return result, errors.New("unable to list process deployments: " + string(temp))
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		_, _ = io.ReadAll(resp.Body) //ensure resp.Body is read to EOF
		return result, err
	}
	return result, nil
}

//go:embed canary_event_process.bpmn
var ProcessBpmn string

//go:embed canary_event_process.svg
var ProcessSvg string

func (this *Events) PrepareProcessDeployment(token string) (result PreparedDeployment, err error) {
	endpoint := this.config.ProcessDeploymentUrl + "/v3/prepared-deployments"
	method := "POST"

	msg, err := json.Marshal(map[string]interface{}{
		"xml": ProcessBpmn,
		"svg": ProcessSvg,
	})
	if err != nil {
		return result, err
	}

	req, err := http.NewRequest(method, endpoint, bytes.NewBuffer(msg))
	if err != nil {
		return result, err
	}
	req.Header.Set("Authorization", token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 299 {
		temp, _ := io.ReadAll(resp.Body) //read error response end ensure that resp.Body is read to EOF
		return result, errors.New("unable to deploy process: " + string(temp))
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		_, _ = io.ReadAll(resp.Body) //ensure resp.Body is read to EOF
		return result, err
	}
	return result, nil
}
