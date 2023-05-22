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

package canary

import (
	"context"
	"github.com/SENERGY-Platform/canary/pkg/configuration"
	devicerepo "github.com/SENERGY-Platform/device-repository/lib/client"
	"github.com/SENERGY-Platform/permission-search/lib/client"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"sync"
	"time"
)

type Canary struct {
	metrics              *Metrics
	reg                  *prometheus.Registry
	config               configuration.Config
	promHttpHandler      http.Handler
	isRunningMux         sync.Mutex
	isRunning            bool
	permissions          client.Client
	guaranteeChangeAfter time.Duration
	devicerepo           devicerepo.Interface
}

func New(ctx context.Context, wg *sync.WaitGroup, config configuration.Config) (canary *Canary, err error) {
	guaranteeChangeAfter, err := time.ParseDuration(config.GuaranteeChangeAfter)
	reg := prometheus.NewRegistry()
	return &Canary{
		reg:                  reg,
		metrics:              NewMetrics(reg),
		config:               config,
		permissions:          client.NewClient(config.PermissionSearchUrl),
		devicerepo:           devicerepo.NewClient(config.DeviceRepositoryUrl),
		guaranteeChangeAfter: guaranteeChangeAfter,
	}, nil
}

func (this *Canary) GetMetricsHandler() (h http.Handler, err error) {
	return this, nil
}

func (this *Canary) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	log.Printf("%v [%v] %v \n", request.RemoteAddr, request.Method, request.URL)
	if this.promHttpHandler == nil {
		this.promHttpHandler = promhttp.HandlerFor(
			this.reg,
			promhttp.HandlerOpts{
				Registry: this.reg,
			},
		)
	}
	this.promHttpHandler.ServeHTTP(writer, request)
	this.StartTests()
}

func (this *Canary) StartTests() {
	go func() {
		log.Println("start canary tests")
		isCurrentlyRunning, done := this.running()
		if isCurrentlyRunning {
			log.Println("test are currently already running")
			return
		}
		defer done()
		defer log.Println("canary tests are finished")
		wg := &sync.WaitGroup{}

		token, refresh, err := this.login()
		if err != nil {
			return
		}
		defer this.logout(token, refresh)

		deviceInfo, err := this.ensureDevice(token)
		if err != nil {
			return
		}

		this.testDeviceConnection(wg, token, deviceInfo)

		this.testMetadata(wg, token, deviceInfo)

		this.testNotification(wg, token)

		//TODO: tests for device-selection, process-model-repo, process-deployment, process-execution

		wg.Wait()

	}()
}

// running() responds with isRunning==true if a test is already running.
// if not, the function also returns a done callback, to let the caller say when he is finished
// if the caller receives isRunning==false, subsequent calls to running() will return isRunning==true until done is called
func (this *Canary) running() (isRunning bool, done func()) {
	this.isRunningMux.Lock()
	defer this.isRunningMux.Unlock()
	if this.isRunning {
		return true, void
	}
	this.isRunning = true
	return false, func() {
		this.isRunningMux.Lock()
		defer this.isRunningMux.Unlock()
		this.isRunning = false
	}
}

func (this *Canary) getChangeGuaranteeDuration() time.Duration {
	return this.guaranteeChangeAfter
}

func void() {}
