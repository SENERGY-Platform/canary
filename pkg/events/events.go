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
	"errors"
	"github.com/SENERGY-Platform/canary/pkg/configuration"
	"github.com/SENERGY-Platform/canary/pkg/devicemetadata"
	"github.com/SENERGY-Platform/canary/pkg/metrics"
	devicerepo "github.com/SENERGY-Platform/device-repository/lib/client"
	"log"
	"time"
)

type Events struct {
	config               configuration.Config
	devicerepo           devicerepo.Interface
	guaranteeChangeAfter time.Duration
	metrics              *metrics.Metrics
}

type DeviceInfo = devicemetadata.DeviceInfo

func New(config configuration.Config, devicerepo devicerepo.Interface, metrics *metrics.Metrics, guaranteeChangeAfter time.Duration) *Events {
	return &Events{
		config:               config,
		devicerepo:           devicerepo,
		guaranteeChangeAfter: guaranteeChangeAfter,
		metrics:              metrics,
	}
}

func (this *Events) getChangeGuaranteeDuration() time.Duration {
	return this.guaranteeChangeAfter
}

func (this *Events) ProcessStartup(token string, info DeviceInfo) error {
	ids, err := this.ListCanaryProcessDeployments(token)
	if err != nil {
		this.metrics.UncategorizedErr.Inc()
		log.Println("ERROR: unexpected event process deployment list count")
		return err
	}
	//cleanup
	for _, id := range ids {
		err = this.DeleteProcess(token, id)
		if err != nil {
			this.metrics.UncategorizedErr.Inc()
			log.Println("ERROR: DeleteProcess()", err)
			return err
		}
	}

	dt, err, _ := this.devicerepo.ReadDeviceType(info.DeviceTypeId, token)
	if err != nil {
		this.metrics.UncategorizedErr.Inc()
		log.Println("ERROR: ReadDeviceType()", err)
		return err
	}
	serviceId := ""
	for _, s := range dt.Services {
		if s.LocalId == devicemetadata.SensorServiceLocalId {
			serviceId = s.Id
			break
		}
	}
	if serviceId == "" {
		return errors.New("event no sensor service id found")
	}

	//check prepared deployment
	preparedDepl, err := this.PrepareProcessDeployment(token)
	if err != nil {
		this.metrics.EventProcessPreparedDeploymentErr.Inc()
		log.Println("ERROR: event EventProcessPreparedDeploymentErr", err)
	} else {
		foundService := false
		foundDevice := false
		for _, e := range preparedDepl.Elements {
			if e.BpmnId == "StartEvent_1" && e.ConditionalEvent != nil {
				for _, o := range e.ConditionalEvent.Selection.SelectionOptions {
					if o.Device != nil && o.Device.Id == info.Id {
						foundDevice = true
					}
					for _, s := range o.Services {
						if s.Id == serviceId {
							foundService = true
						}
					}
				}
			}
		}
		if !foundDevice {
			this.metrics.EventProcessUnexpectedPreparedDeploymentSelectablesErr.Inc()
			log.Println("ERROR: EventProcessUnexpectedPreparedDeploymentSelectablesErr !foundDevice", info.Id)
		}
		if !foundService {
			this.metrics.EventProcessUnexpectedPreparedDeploymentSelectablesErr.Inc()
			temp, _ := json.Marshal(preparedDepl)
			log.Printf("ERROR: EventProcessUnexpectedPreparedDeploymentSelectablesErr !foundService %v \n %#v \n", serviceId, string(temp))
		}
	}

	_, err = this.DeployProcess(token, info.Id, serviceId)
	if err != nil {
		this.metrics.EventProcessDeploymentErr.Inc()
		log.Println("ERROR: EventProcessDeploymentErr", err)
		return err
	}

	time.Sleep(this.getChangeGuaranteeDuration())

	return nil
}

func (this *Events) ProcessTeardown(token string) error {
	ids, err := this.ListCanaryProcessDeployments(token)
	if err != nil {
		this.metrics.UncategorizedErr.Inc()
		return err
	}
	if len(ids) != 1 {
		this.metrics.UncategorizedErr.Inc()
		log.Println("ERROR: unexpected process deployment list count")
	}

	unfilteredInstances, err := this.GetProcessInstances(token)
	instances := []ProcessInstance{}
	for _, e := range unfilteredInstances {
		if e.ProcessDefinitionName == ExpectedCanaryDeploymentName {
			instances = append(instances, e)
		}
	}

	if err != nil {
		this.metrics.UncategorizedErr.Inc()
		log.Println("ERROR: unexpected process list count", err)
	} else {
		if len(instances) != 1 {
			this.metrics.UncategorizedErr.Inc()
			log.Println("ERROR: unexpected process instance list count", len(instances))
		} else {
			if instances[0].State != "COMPLETED" {
				this.metrics.UnexpectedEventProcessInstanceStateErr.Inc()
				log.Printf("ERROR: UnexpectedProcessInstanceStateErr %#v \n", instances)
			} else {
				this.metrics.EventProcessInstanceDurationMs.Set(float64(instances[0].DurationInMillis))
			}
		}
	}

	//cleanup
	for _, id := range ids {
		err = this.DeleteProcess(token, id)
		if err != nil {
			this.metrics.UncategorizedErr.Inc()
			log.Println("ERROR: DeleteProcess()", err)
			return err
		}
	}

	return nil
}
