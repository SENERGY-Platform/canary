/*
 * Copyright 2025 InfAI (CC SES)
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
	"log"
	"testing"
	"time"

	"github.com/SENERGY-Platform/canary/pkg/configuration"
	"github.com/SENERGY-Platform/canary/pkg/metrics"
	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/prometheus/client_golang/prometheus"
)

func TestCertMqtt(t *testing.T) {

	config := configuration.Config{
		AuthEndpoint:           "https://auth.senergy.infai.org",
		AuthUsername:           "canary",
		AuthPassword:           "canary123!.",
		AuthClientId:           "frontend",
		ConnectorMqttBrokerUrl: "tls://certconnector.senergy.infai.org:28888",
		CertAuthorityUrl:       "https://api.senergy.infai.org/ca",
		CertKeyFilePath:        "./key.pem",
		CertFilePath:           "./cert.pem",
	}

	reg := prometheus.NewRegistry()
	m := metrics.NewMetrics(reg)
	canary := Canary{config: config, metrics: m}
	hubId := "test-hub-id"

	token, refresh, err := canary.login()
	if err != nil {
		t.Error(err)
		return
	}
	defer canary.logout(token, refresh)

	tlsConf, err := canary.getTlsConfig(token, hubId, time.Hour)
	if err != nil {
		t.Error(err)
		return
	}

	options := paho.NewClientOptions().
		SetClientID(hubId).
		SetAutoReconnect(true).
		SetCleanSession(true).
		AddBroker(config.ConnectorMqttBrokerUrl).
		SetOnConnectHandler(func(c paho.Client) {
			log.Println("connected")
		}).
		SetConnectionLostHandler(func(c paho.Client, err error) {
			log.Println("lost connection:", hubId, err)
		}).
		SetTLSConfig(tlsConf)

	client := paho.NewClient(options)
	conToken := client.Connect()
	conToken.Wait()
	if conToken.Error() != nil {
		t.Error(conToken.Error())
		return
	}

	time.Sleep(time.Second * 10)

}
