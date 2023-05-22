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
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	AuthCount     prometheus.Counter
	AuthLatencyMs prometheus.Gauge
	AuthErr       prometheus.Counter

	DeviceMetaUpdateCount     prometheus.Counter
	DeviceMetaUpdateLatencyMs prometheus.Gauge
	DeviceMetaUpdateErr       prometheus.Counter

	DeviceRepoRequestCount     prometheus.Counter
	DeviceRepoRequestLatencyMs prometheus.Gauge
	DeviceRepoRequestErr       prometheus.Counter

	PermissionsRequestCount     prometheus.Counter
	PermissionsRequestLatencyMs prometheus.Gauge
	PermissionsRequestErr       prometheus.Counter

	DeviceDataRequestCount     prometheus.Counter
	DeviceDataRequestLatencyMs prometheus.Gauge
	DeviceDataRequestErr       prometheus.Counter

	ConnectorLoginCount     prometheus.Counter
	ConnectorLoginLatencyMs prometheus.Gauge
	ConnectorLoginErr       prometheus.Counter

	ConnectorSubscribeCount     prometheus.Counter
	ConnectorSubscribeLatencyMs prometheus.Gauge
	ConnectorSubscribeErr       prometheus.Counter

	ConnectorPublishCount     prometheus.Counter
	ConnectorPublishLatencyMs prometheus.Gauge
	ConnectorPublishErr       prometheus.Counter

	NotificationPublishCount     prometheus.Counter
	NotificationPublishLatencyMs prometheus.Gauge
	NotificationPublishErr       prometheus.Counter

	NotificationReadCount     prometheus.Counter
	NotificationReadLatencyMs prometheus.Gauge
	NotificationReadErr       prometheus.Counter

	NotificationDeleteCount     prometheus.Counter
	NotificationDeleteLatencyMs prometheus.Gauge
	NotificationDeleteErr       prometheus.Counter

	UnexpectedPermissionsDeviceOnlineStateErr  prometheus.Counter
	UnexpectedPermissionsDeviceOfflineStateErr prometheus.Counter
	UnexpectedPermissionsMetadataErr           prometheus.Counter
	UnexpectedDeviceRepoMetadataErr            prometheus.Counter
	UnexpectedDeviceDataErr                    prometheus.Counter
	UnexpectedNotificationStateErr             prometheus.Counter
	UncategorizedErr                           prometheus.Counter
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	const countHelpMsg = "how often has this test ben started. this value is used to indicate if a test has ben started and no error has ben found ore no test has ben started."
	m := &Metrics{
		AuthCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_auth_count",
			Help: countHelpMsg,
		}),
		AuthLatencyMs: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "canary_auth_latency_ms",
			Help: "latency of auth request",
		}),
		AuthErr: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_auth_err",
			Help: "total count of auth errors since canary startup",
		}),

		DeviceMetaUpdateCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_device_meta_update_count",
			Help: countHelpMsg,
		}),
		DeviceMetaUpdateLatencyMs: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "canary_device_meta_update_latency_ms",
			Help: "latency of device meta update request",
		}),
		DeviceMetaUpdateErr: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_device_meta_update_err",
			Help: "total count of device meta update errors since canary startup",
		}),

		DeviceRepoRequestCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_device_repo_request_count",
			Help: countHelpMsg,
		}),
		DeviceRepoRequestLatencyMs: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "canary_device_repo_request_latency_ms",
			Help: "latency of device repo request",
		}),
		DeviceRepoRequestErr: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_device_repo_request_update_err",
			Help: "total count of device repo request errors since canary startup",
		}),

		PermissionsRequestCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_permissions_request_count",
			Help: countHelpMsg,
		}),
		PermissionsRequestLatencyMs: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "canary_permissions_request_latency_ms",
			Help: "latency of permissions request",
		}),
		PermissionsRequestErr: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_permissions_request_update_err",
			Help: "total count of permissions request errors since canary startup",
		}),

		DeviceDataRequestCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_device_data_request_count",
			Help: countHelpMsg,
		}),
		DeviceDataRequestLatencyMs: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "canary_device_data_request_latency_ms",
			Help: "latency of device data request",
		}),
		DeviceDataRequestErr: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_device_data_request_update_err",
			Help: "total count of device data request errors since canary startup",
		}),

		ConnectorLoginCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_connector_login_count",
			Help: countHelpMsg,
		}),
		ConnectorLoginLatencyMs: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "canary_connector_login_latency_ms",
			Help: "latency of connector login",
		}),
		ConnectorLoginErr: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_connector_login_err",
			Help: "total count of connector login errors since canary startup",
		}),

		ConnectorSubscribeCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_connector_subscribe_count",
			Help: countHelpMsg,
		}),
		ConnectorSubscribeLatencyMs: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "canary_connector_subscribe_latency_ms",
			Help: "latency of connector subscribe",
		}),
		ConnectorSubscribeErr: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_connector_subscribe_err",
			Help: "total count of connector subscribe errors since canary startup",
		}),

		ConnectorPublishCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_connector_publish_count",
			Help: countHelpMsg,
		}),
		ConnectorPublishLatencyMs: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "canary_connector_publish_latency_ms",
			Help: "latency of connector publish",
		}),
		ConnectorPublishErr: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_connector_publish_err",
			Help: "total count of connector publish errors since canary startup",
		}),

		NotificationPublishCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_notification_publish_count",
			Help: countHelpMsg,
		}),
		NotificationPublishLatencyMs: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "canary_notification_publish_latency_ms",
			Help: "latency of notification publish",
		}),
		NotificationPublishErr: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_notification_publish_err",
			Help: "total count of notification publish errors since canary startup",
		}),

		NotificationReadCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_notification_read_count",
			Help: countHelpMsg,
		}),
		NotificationReadLatencyMs: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "canary_notification_read_latency_ms",
			Help: "latency of notification read",
		}),
		NotificationReadErr: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_notification_read_err",
			Help: "total count of notification read errors since canary startup",
		}),

		NotificationDeleteCount: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_notification_delete_count",
			Help: countHelpMsg,
		}),
		NotificationDeleteLatencyMs: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "canary_notification_delete_latency_ms",
			Help: "latency of notification delete",
		}),
		NotificationDeleteErr: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_notification_delete_err",
			Help: "total count of notification delete errors since canary startup",
		}),

		UnexpectedPermissionsDeviceOnlineStateErr: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_unexpected_permissions_device_online_state_err",
			Help: "total count of unexpected permission device online state errors since canary startup",
		}),
		UnexpectedPermissionsDeviceOfflineStateErr: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_unexpected_permissions_device_offline_state_err",
			Help: "total count of unexpected permission device offline state errors since canary startup",
		}),
		UnexpectedPermissionsMetadataErr: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_unexpected_permissions_metadata_err",
			Help: "total count of unexpected permission metadata value errors since canary startup",
		}),
		UnexpectedDeviceRepoMetadataErr: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_unexpected_device_repo_metadata_err",
			Help: "total count of unexpected device repo metadata value errors since canary startup",
		}),
		UnexpectedDeviceDataErr: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_unexpected_device_data_err",
			Help: "total count of unexpected device data value errors since canary startup",
		}),
		UnexpectedNotificationStateErr: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_unexpected_notification_state_err",
			Help: "total count of unexpected notification state errors since canary startup",
		}),
		UncategorizedErr: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "canary_uncategorized_err",
			Help: "total count of uncategorized errors since canary startup",
		}),
	}

	reg.MustRegister(m.AuthCount)
	reg.MustRegister(m.AuthLatencyMs)
	reg.MustRegister(m.AuthErr)

	reg.MustRegister(m.DeviceMetaUpdateCount)
	reg.MustRegister(m.DeviceMetaUpdateLatencyMs)
	reg.MustRegister(m.DeviceMetaUpdateErr)

	reg.MustRegister(m.DeviceRepoRequestCount)
	reg.MustRegister(m.DeviceRepoRequestLatencyMs)
	reg.MustRegister(m.DeviceRepoRequestErr)

	reg.MustRegister(m.PermissionsRequestCount)
	reg.MustRegister(m.PermissionsRequestLatencyMs)
	reg.MustRegister(m.PermissionsRequestErr)

	reg.MustRegister(m.DeviceDataRequestCount)
	reg.MustRegister(m.DeviceDataRequestLatencyMs)
	reg.MustRegister(m.DeviceDataRequestErr)

	reg.MustRegister(m.ConnectorLoginCount)
	reg.MustRegister(m.ConnectorLoginLatencyMs)
	reg.MustRegister(m.ConnectorLoginErr)

	reg.MustRegister(m.ConnectorSubscribeCount)
	reg.MustRegister(m.ConnectorSubscribeLatencyMs)
	reg.MustRegister(m.ConnectorSubscribeErr)

	reg.MustRegister(m.ConnectorPublishCount)
	reg.MustRegister(m.ConnectorPublishLatencyMs)
	reg.MustRegister(m.ConnectorPublishErr)

	reg.MustRegister(m.NotificationPublishCount)
	reg.MustRegister(m.NotificationPublishLatencyMs)
	reg.MustRegister(m.NotificationPublishErr)

	reg.MustRegister(m.NotificationReadCount)
	reg.MustRegister(m.NotificationReadLatencyMs)
	reg.MustRegister(m.NotificationReadErr)

	reg.MustRegister(m.NotificationDeleteCount)
	reg.MustRegister(m.NotificationDeleteLatencyMs)
	reg.MustRegister(m.NotificationDeleteErr)

	reg.MustRegister(m.UnexpectedPermissionsDeviceOnlineStateErr)
	reg.MustRegister(m.UnexpectedPermissionsDeviceOfflineStateErr)
	reg.MustRegister(m.UnexpectedPermissionsMetadataErr)
	reg.MustRegister(m.UnexpectedDeviceRepoMetadataErr)
	reg.MustRegister(m.UnexpectedDeviceDataErr)
	reg.MustRegister(m.UnexpectedNotificationStateErr)
	reg.MustRegister(m.UncategorizedErr)

	return m
}
