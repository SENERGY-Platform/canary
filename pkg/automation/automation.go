/*
 * Copyright (c) 2026 InfAI (CC SES)
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

// Package automation holds the calls to the automation-engine that the process and the event check share.
package automation

import (
	"context"
	"time"

	"github.com/SENERGY-Platform/automation-engine/pkg/client"
	"github.com/SENERGY-Platform/automation-engine/pkg/model"
)

// Namespace is where the engine keeps the processes of users; the canary user deploys there.
const Namespace = "processes"

const timeout = time.Minute

type Client = client.Client

func New(url string) Client {
	return client.New(url)
}

func Context() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// ListDeploymentIds returns the ids of all deployments of the user named name.
func ListDeploymentIds(c Client, token string, name string) (ids []string, err error) {
	ctx, cancel := Context()
	defer cancel()
	list := func(page model.PageOptions) (model.Page[model.Deployment], error) {
		return c.ListDeployments(ctx, token, Namespace, model.DeploymentListOptions{PageOptions: page, Search: name})
	}
	for depl, err := range client.All(list) {
		if err != nil {
			return ids, err
		}
		if depl.Name == name {
			ids = append(ids, depl.Id)
		}
	}
	return ids, nil
}

// ListInstances returns the instances of the given deployments, running and finished.
func ListInstances(c Client, token string, deploymentIds []string) (instances []model.Instance, err error) {
	ctx, cancel := Context()
	defer cancel()
	for _, id := range deploymentIds {
		list := func(page model.PageOptions) (model.Page[model.Instance], error) {
			return c.ListInstances(ctx, token, Namespace, model.InstanceListOptions{PageOptions: page, DeploymentId: id})
		}
		for instance, err := range client.All(list) {
			if err != nil {
				return instances, err
			}
			instances = append(instances, instance)
		}
	}
	return instances, nil
}

// DurationMs is the time from the start to the end of a finished instance.
func DurationMs(instance model.Instance) float64 {
	if instance.EndedAt == nil {
		return 0
	}
	return float64(instance.EndedAt.Sub(instance.StartedAt).Milliseconds())
}

// FindOption reports whether the prepared element offers the device and the service.
func FindOption(prepared model.PreparedDeployment, elementId string, kind string, deviceId string, serviceId string) (foundDevice bool, foundService bool) {
	for _, e := range prepared.Elements {
		if e.ElementId != elementId || e.Kind != kind {
			continue
		}
		for _, o := range e.Options {
			if o.Device != nil && o.Device.Id == deviceId {
				foundDevice = true
			}
			for _, s := range o.Services {
				if s.Id == serviceId {
					foundService = true
				}
			}
		}
	}
	return foundDevice, foundService
}

// BlockingIssues are the findings of a prepared deployment that keep it from deploying.
func BlockingIssues(issues []model.Issue) (result []model.Issue) {
	for _, issue := range issues {
		if !issue.Hint {
			result = append(result, issue)
		}
	}
	return result
}
