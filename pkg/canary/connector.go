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
	"bytes"
	"encoding/json"
	"github.com/SENERGY-Platform/permission-search/lib/client"
	"github.com/SENERGY-Platform/permission-search/lib/model"
	paho "github.com/eclipse/paho.mqtt.golang"
	"log"
	"math/rand"
	"net/http"
	"reflect"
	"runtime/debug"
	"strconv"
	"sync"
	"time"
)

func (this *Canary) testDeviceConnection(wg *sync.WaitGroup, token string, info DeviceInfo) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		this.checkDeviceConnState(token, info, false)

		hubId, err := this.ensureHub(token, info)
		if err != nil {
			return
		}

		conn, err := this.connect(hubId)
		if err != nil {
			return
		}

		this.subscribe(info, conn)

		value := rand.Int()

		this.publish(info, conn, value)

		time.Sleep(this.getChangeGuaranteeDuration())

		this.checkDeviceConnState(token, info, true)

		this.checkDeviceValue(token, info, value)

		this.disconnect(conn)

		time.Sleep(this.getChangeGuaranteeDuration())

		this.checkDeviceConnState(token, info, false)
	}()
}

type PermDevice struct {
	Id          string                 `json:"id"`
	Name        string                 `json:"name"`
	Annotations map[string]interface{} `json:"annotations"`
}

func (this *Canary) checkDeviceConnState(token string, info DeviceInfo, expectedConnState bool) {
	this.metrics.PermissionsRequestCount.Inc()
	start := time.Now()
	result, _, err := client.Query[[]PermDevice](this.permissions, token, client.QueryMessage{
		Resource: "devices",
		ListIds: &client.QueryListIds{
			QueryListCommons: model.QueryListCommons{
				Limit:  1,
				Offset: 0,
				Rights: "r",
			},
			Ids: []string{info.Id},
		},
	})
	this.metrics.PermissionsRequestLatencyMs.Set(float64(time.Since(start).Milliseconds()))
	if err != nil {
		this.metrics.PermissionsRequestErr.Inc()
		return
	}
	if len(result) == 0 {
		this.metrics.UncategorizedErr.Inc()
		log.Printf("Unexpected conn state result: \n%#v\n", result)
		debug.PrintStack()
		return
	}
	if result[0].Annotations["connected"] != expectedConnState {
		if expectedConnState {
			this.metrics.UnexpectedPermissionsDeviceOfflineStateErr.Inc()
		} else {
			this.metrics.UnexpectedPermissionsDeviceOnlineStateErr.Inc()
		}
	}
}

type Conn struct {
	Client paho.Client
}

func (this *Canary) connect(hubId string) (conn *Conn, err error) {
	conn = &Conn{}

	options := paho.NewClientOptions().
		SetClientID(hubId).
		SetUsername(this.config.AuthUsername).
		SetPassword(this.config.AuthPassword).
		SetAutoReconnect(true).
		SetCleanSession(true).
		AddBroker(this.config.ConnectorMqttBrokerUrl).
		SetConnectionLostHandler(func(c paho.Client, err error) {
			log.Println("lost connection:", hubId, err)
		})

	this.metrics.ConnectorLoginCount.Inc()
	conn.Client = paho.NewClient(options)
	start := time.Now()
	token := conn.Client.Connect()
	token.Wait()
	this.metrics.ConnectorLoginLatencyMs.Set(float64(time.Since(start).Milliseconds()))
	if token.Error() != nil {
		log.Println("Error on Client.Connect(): ", token.Error())
		this.metrics.ConnectorLoginErr.Inc()
		return conn, token.Error()
	}
	return conn, nil
}

func (this *Canary) disconnect(conn *Conn) {
	conn.Client.Disconnect(250)
}

func (this *Canary) subscribe(info DeviceInfo, conn *Conn) {
	this.metrics.ConnectorSubscribeCount.Inc()
	topic := "command/" + info.LocalId + "/+"
	start := time.Now()
	token := conn.Client.Subscribe(topic, 2, func(c paho.Client, message paho.Message) {})
	token.Wait()
	this.metrics.ConnectorSubscribeLatencyMs.Set(float64(time.Since(start).Milliseconds()))
	if token.Error() != nil {
		log.Println("Error on Client.Subscribe(): ", token.Error())
		this.metrics.ConnectorSubscribeErr.Inc()
		return
	}
}

func (this *Canary) publish(info DeviceInfo, conn *Conn, value int) {
	payload, err := json.Marshal(map[string]string{this.config.CanaryProtocolSegmentName: strconv.Itoa(value)})
	if err != nil {
		this.metrics.UncategorizedErr.Inc()
		return
	}

	this.metrics.ConnectorPublishCount.Inc()
	topic := "event/" + info.LocalId + "/sensor"

	start := time.Now()
	token := conn.Client.Publish(topic, 2, false, payload)
	token.Wait()
	this.metrics.ConnectorPublishLatencyMs.Set(float64(time.Since(start).Milliseconds()))
	if token.Error() != nil {
		log.Println("Error on Client.Subscribe(): ", token.Error())
		this.metrics.ConnectorPublishErr.Inc()
		return
	}
}

type LastValue struct {
	Time  string      `json:"time"`
	Value interface{} `json:"value"`
}

func (this *Canary) checkDeviceValue(token string, info DeviceInfo, value int) {
	this.metrics.DeviceRepoRequestCount.Inc()
	start := time.Now()
	dt, err, _ := this.devicerepo.ReadDeviceType(info.DeviceTypeId, token)
	this.metrics.DeviceRepoRequestLatencyMs.Set(float64(time.Since(start).Milliseconds()))
	if err != nil {
		this.metrics.DeviceRepoRequestErr.Inc()
		log.Println("ERROR:", err)
		debug.PrintStack()
		return
	}

	serviceId := ""
	for _, s := range dt.Services {
		if s.LocalId == SensorServiceLocalId {
			serviceId = s.Id
			break
		}
	}

	buf := &bytes.Buffer{}
	err = json.NewEncoder(buf).Encode([]map[string]interface{}{{
		"deviceId":   info.Id,
		"serviceId":  serviceId,
		"columnName": "value",
	}})
	if err != nil {
		return
	}
	this.metrics.DeviceDataRequestCount.Inc()
	req, err := http.NewRequest(http.MethodPost, this.config.LastValueQueryUrl, buf)
	if err != nil {
		this.metrics.UncategorizedErr.Inc()
		log.Println("ERROR:", err)
		debug.PrintStack()
		return
	}
	req.Header.Set("Authorization", token)
	start = time.Now()
	lastValues, _, err := Do[[]LastValue](req)
	this.metrics.DeviceDataRequestLatencyMs.Set(float64(time.Since(start).Milliseconds()))
	if err != nil {
		this.metrics.DeviceDataRequestErr.Inc()
		log.Println("ERROR:", err)
		debug.PrintStack()
	}

	expected := jsonNormalize(value)

	if len(lastValues) != 1 {
		this.metrics.UnexpectedDeviceDataErr.Inc()
		log.Printf("UnexpectedDeviceDataErr: %#v\n", lastValues)
		return
	}

	if !reflect.DeepEqual(lastValues[0].Value, expected) {
		this.metrics.UnexpectedDeviceDataErr.Inc()
		log.Printf("UnexpectedDeviceDataErr: %#v, %#v\n", lastValues[0].Value, expected)
		return
	}
}

func jsonNormalize(in interface{}) (out interface{}) {
	temp, _ := json.Marshal(in)
	json.Unmarshal(temp, &out)
	return
}
