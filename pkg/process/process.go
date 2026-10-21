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
	"errors"
	"sync/atomic"
	"time"

	"github.com/SENERGY-Platform/automation-engine/pkg/model"
	"github.com/SENERGY-Platform/canary/pkg/automation"
	"github.com/SENERGY-Platform/canary/pkg/configuration"
	"github.com/SENERGY-Platform/canary/pkg/devicemetadata"
	"github.com/SENERGY-Platform/canary/pkg/metrics"
	devicerepo "github.com/SENERGY-Platform/device-repository/v3/lib/client"
)

type Process struct {
	config               configuration.Config
	devicerepo           devicerepo.Interface
	engine               automation.Client
	guaranteeChangeAfter time.Duration
	receivedCommands     atomic.Int64
	metrics              *metrics.Metrics
}

type DeviceInfo = devicemetadata.DeviceInfo

func New(config configuration.Config, devicerepo devicerepo.Interface, engine automation.Client, metrics *metrics.Metrics, guaranteeChangeAfter time.Duration) *Process {
	return &Process{
		config:               config,
		devicerepo:           devicerepo,
		engine:               engine,
		guaranteeChangeAfter: guaranteeChangeAfter,
		metrics:              metrics,
	}
}

func (this *Process) getChangeGuaranteeDuration() time.Duration {
	return this.guaranteeChangeAfter
}

func (this *Process) ProcessStartup(token string, info DeviceInfo) error {
	this.receivedCommands.Store(0)
	ids, err := this.ListCanaryProcessDeployments(token)
	if err != nil {
		this.metrics.UncategorizedErr.Inc()
		this.config.GetLogger().Error("unable to list canary process deployments", "error", err)
		return err
	}
	//cleanup
	for _, id := range ids {
		err = this.DeleteProcess(token, id)
		if err != nil {
			this.metrics.UncategorizedErr.Inc()
			this.config.GetLogger().Error("unable to delete canary process deployment", "error", err)
			return err
		}
	}

	dt, err, _ := this.devicerepo.ReadDeviceType(info.DeviceTypeId, token)
	if err != nil {
		this.metrics.UncategorizedErr.Inc()
		this.config.GetLogger().Error("unable to read device-type", "error", err)
		return err
	}
	serviceId := ""
	for _, s := range dt.Services {
		if s.LocalId == devicemetadata.CmdServiceLocalId {
			serviceId = s.Id
			break
		}
	}
	if serviceId == "" {
		return errors.New("ProcessStartup(): no cmd service id found")
	}

	//check prepared deployment
	preparedDepl, err := this.PrepareProcessDeployment(token)
	if err != nil {
		this.metrics.ProcessPreparedDeploymentErr.Inc()
		this.config.GetLogger().Error("unable to prepare process deployment", "error", err)
	} else {
		if issues := automation.BlockingIssues(preparedDepl.Issues); len(issues) > 0 {
			this.config.GetLogger().Error("prepared process deployment has issues", "issues", issues)
		}
		foundDevice, foundService := automation.FindOption(preparedDepl, TaskId, model.PreparedTask, info.Id, serviceId)
		if !foundDevice {
			this.metrics.ProcessUnexpectedPreparedDeploymentSelectablesErr.Inc()
			this.config.GetLogger().Error("device not found in prepared process selection options", "device", info.Id)
		}
		if !foundService {
			this.metrics.ProcessUnexpectedPreparedDeploymentSelectablesErr.Inc()
			this.config.GetLogger().Error("service not found in prepared process selection options", "service", serviceId)
		}
	}

	deplId, err := this.DeployProcess(token, info.Id, serviceId)
	if err != nil {
		this.metrics.ProcessDeploymentErr.Inc()
		this.config.GetLogger().Error("unable to deploy process", "error", err)
		return err
	}

	time.Sleep(this.getChangeGuaranteeDuration())

	err = this.StartProcess(token, deplId)
	if err != nil {
		this.metrics.ProcessStartErr.Inc()
		this.config.GetLogger().Error("unable to start process", "error", err)
		return err
	}

	return nil
}

func (this *Process) ProcessTeardown(token string) error {
	ids, err := this.ListCanaryProcessDeployments(token)
	if err != nil {
		this.metrics.UncategorizedErr.Inc()
		return err
	}
	if len(ids) != 1 {
		this.metrics.UncategorizedErr.Inc()
		this.config.GetLogger().Error("unexpected process deployment list count", "count", len(ids))
	}

	instances, err := this.GetProcessInstances(token, ids)
	if err != nil {
		this.metrics.UncategorizedErr.Inc()
		this.config.GetLogger().Error("unable to get process instances", "error", err)
	} else {
		if len(instances) != 1 {
			this.metrics.UncategorizedErr.Inc()
			this.config.GetLogger().Error("unexpected process instance list count", "count", len(instances))
		} else {
			if instances[0].State != model.StateCompleted {
				this.metrics.UnexpectedProcessInstanceStateErr.Inc()
				this.config.GetLogger().Error("unexpected process instance state", "state", instances[0].State)
			} else {
				this.metrics.ProcessInstanceDurationMs.Set(automation.DurationMs(instances[0]))
			}
		}
	}

	//cleanup
	for _, id := range ids {
		err = this.DeleteProcess(token, id)
		if err != nil {
			this.metrics.UncategorizedErr.Inc()
			this.config.GetLogger().Error("unable to delete canary process deployment", "error", err)
			return err
		}
	}

	if this.receivedCommands.Load() == 0 {
		this.metrics.ProcessUnexpectedCommandCountError.Inc()
		this.config.GetLogger().Error("unexpected command count", "count", this.receivedCommands.Load())
	}
	return nil
}

func (this *Process) NotifyCommand(topic string, payload []byte) {
	this.receivedCommands.Add(1)
}
