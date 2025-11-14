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

package configuration

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestMqttConfig(t *testing.T) {
	t.Run("no env", func(t *testing.T) {
		defaultConfig, err := Load("../../config.json")
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(defaultConfig.Mqtt, []MqttConfig{
			{
				Broker:   "tcp://localhost:1883",
				ClientId: "clientId",
				User:     "",
				Pw:       "",
			},
		}) {
			t.Error("unexpected mqtt config")
			return
		}
	})

	t.Run("old env", func(t *testing.T) {
		t.Setenv("MQTT_BROKER", "tcp://localhost:18832")
		t.Setenv("MQTT_CLIENT_ID", "clientId2")
		t.Setenv("MQTT_USER", "user2")
		t.Setenv("MQTT_PW", "password2")
		defaultConfig, err := Load("../../config.json")
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(defaultConfig.Mqtt, []MqttConfig{
			{
				Broker:   "tcp://localhost:18832",
				ClientId: "clientId2",
				User:     "user2",
				Pw:       "password2",
			},
		}) {
			t.Error("unexpected mqtt config")
			return
		}
	})

	t.Run("old and new env", func(t *testing.T) {
		t.Setenv("MQTT_BROKER", "tcp://localhost:18832")
		t.Setenv("MQTT_CLIENT_ID", "clientId2")
		t.Setenv("MQTT_USER", "user2")
		t.Setenv("MQTT_PW", "password2")

		t.Setenv("MQTT_BROKER_3", "tcp://localhost:18833")
		t.Setenv("MQTT_CLIENT_ID_3", "clientId3")
		t.Setenv("MQTT_USER_3", "user3")
		t.Setenv("MQTT_PW_3", "password3")
		defaultConfig, err := Load("../../config.json")
		if err != nil {
			t.Error(err)
		}
		slices.SortFunc(defaultConfig.Mqtt, func(a, b MqttConfig) int {
			return strings.Compare(a.Broker, b.Broker)
		})
		if !reflect.DeepEqual(defaultConfig.Mqtt, []MqttConfig{
			{
				Broker:   "tcp://localhost:18832",
				ClientId: "clientId2",
				User:     "user2",
				Pw:       "password2",
			},
			{
				Broker:   "tcp://localhost:18833",
				ClientId: "clientId3",
				User:     "user3",
				Pw:       "password3",
			},
		}) {
			t.Error("unexpected mqtt config")
			return
		}
	})

	t.Run("new env", func(t *testing.T) {
		t.Setenv("MQTT_BROKER_1", "tcp://localhost:18831")
		t.Setenv("MQTT_CLIENT_ID_1", "clientId1")
		t.Setenv("MQTT_USER_1", "user1")
		t.Setenv("MQTT_PW_1", "password1")

		t.Setenv("MQTT_BROKER_3", "tcp://localhost:18833")
		t.Setenv("MQTT_CLIENT_ID_3", "clientId3")
		t.Setenv("MQTT_USER_3", "user3")
		t.Setenv("MQTT_PW_3", "password3")
		defaultConfig, err := Load("../../config.json")
		if err != nil {
			t.Error(err)
		}
		slices.SortFunc(defaultConfig.Mqtt, func(a, b MqttConfig) int {
			return strings.Compare(a.Broker, b.Broker)
		})
		if !reflect.DeepEqual(defaultConfig.Mqtt, []MqttConfig{
			{
				Broker:   "tcp://localhost:18831",
				ClientId: "clientId1",
				User:     "user1",
				Pw:       "password1",
			},
			{
				Broker:   "tcp://localhost:18833",
				ClientId: "clientId3",
				User:     "user3",
				Pw:       "password3",
			},
		}) {
			t.Error("unexpected mqtt config")
			return
		}
	})
}
