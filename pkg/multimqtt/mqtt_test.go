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

package multimqtt

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/tests/docker"
	paho "github.com/eclipse/paho.mqtt.golang"
)

func TestMultiMqttClient(t *testing.T) {
	wg := &sync.WaitGroup{}
	defer wg.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, mqtt1Ip, err := docker.Mqtt(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	mqtt1Url := "tcp://" + mqtt1Ip + ":1883"

	_, mqtt2Ip, err := docker.Mqtt(ctx, wg)
	if err != nil {
		t.Error(err)
		return
	}
	mqtt2Url := "tcp://" + mqtt2Ip + ":1883"

	config := configuration.Config{
		MqttCleanSession: true,
		MqttGroupId:      "testgroup",
		Mqtt: []configuration.MqttConfig{
			{
				Broker:   mqtt1Url,
				Pw:       "",
				User:     "",
				ClientId: "client1",
			},
			{
				Broker:   mqtt2Url,
				Pw:       "",
				User:     "",
				ClientId: "client2",
			},
		},
	}

	subscribe := func(client paho.Client) {
		token := client.Subscribe("test", 2, func(client paho.Client, message paho.Message) {
			t.Log("sub", string(message.Payload()))
		})
		if token.Wait() && token.Error() != nil {
			t.Error(token.Error())
			return
		}
	}

	client := NewClient(config.Mqtt, func(options *paho.ClientOptions) {
		options.SetCleanSession(config.MqttCleanSession)
		options.SetResumeSubs(true)
		options.SetConnectionLostHandler(func(c paho.Client, err error) {
			o := c.OptionsReader()
			config.GetLogger().Error("connection to mqtt broker lost", "error", err, "client", o.ClientID())
		})
		options.SetOnConnectHandler(func(c paho.Client) {
			o := c.OptionsReader()
			config.GetLogger().Info("connected to mqtt broker", "client", o.ClientID())
			subscribe(c)
		})
	})

	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		t.Error(token.Error())
		return
	}

	broker1 := paho.NewClient(paho.NewClientOptions().AddBroker(mqtt1Url).SetAutoReconnect(true))
	if token := broker1.Connect(); token.Wait() && token.Error() != nil {
		t.Error(token.Error())
		return
	}

	broker1ReadPubTopic := false
	token = broker1.Subscribe("pubtopic", 2, func(client paho.Client, message paho.Message) {
		broker1ReadPubTopic = true
		if string(message.Payload()) != "testmessage" {
			t.Error("unexpected payload")
		}
	})
	if token.Wait() && token.Error() != nil {
		t.Error(token.Error())
		return
	}
	defer func() {
		if !broker1ReadPubTopic {
			t.Error("broker1 did not receive pubtopic")
		}
	}()

	broker2 := paho.NewClient(paho.NewClientOptions().AddBroker(mqtt2Url).SetAutoReconnect(true))
	if token = broker2.Connect(); token.Wait() && token.Error() != nil {
		t.Error(token.Error())
		return
	}
	broker2ReadPubTopic := false
	token = broker2.Subscribe("pubtopic", 2, func(client paho.Client, message paho.Message) {
		broker2ReadPubTopic = true
		if string(message.Payload()) != "testmessage" {
			t.Error("unexpected payload")
		}
	})
	if token.Wait() && token.Error() != nil {
		t.Error(token.Error())
		return
	}
	defer func() {
		if !broker2ReadPubTopic {
			t.Error("broker1 did not receive pubtopic")
		}
	}()

	token = client.Publish("pubtopic", 2, false, "testmessage")
	if token.Wait() && token.Error() != nil {
		t.Error(token.Error())
		return
	}

	receivedBroker1Msg := false
	receivedBroker2Msg := false
	token = client.Subscribe("subtopic", 2, func(client paho.Client, message paho.Message) {
		switch string(message.Payload()) {
		case "broker1":
			receivedBroker1Msg = true
		case "broker2":
			receivedBroker2Msg = true
		default:
			t.Error("unexpected payload")
		}
	})
	if token.Wait() && token.Error() != nil {
		t.Error(token.Error())
		return
	}
	defer func() {
		if !receivedBroker1Msg {
			t.Error("client did not receive broker1 msg")
		}
		if !receivedBroker2Msg {
			t.Error("client did not receive broker2 msg")
		}
	}()

	token = broker1.Publish("subtopic", 2, false, "broker1")
	if token.Wait() && token.Error() != nil {
		t.Error(token.Error())
		return
	}
	token = broker2.Publish("subtopic", 2, false, "broker2")
	if token.Wait() && token.Error() != nil {
		t.Error(token.Error())
		return
	}
	time.Sleep(time.Second)
}
