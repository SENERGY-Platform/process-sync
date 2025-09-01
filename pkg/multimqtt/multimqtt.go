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
	"errors"
	"sync"

	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	paho "github.com/eclipse/paho.mqtt.golang"
)

func NewClient(configs []configuration.MqttConfig, setOptions func(*paho.ClientOptions)) *MultiClient {
	result := &MultiClient{}
	for _, config := range configs {
		options := paho.NewClientOptions().
			SetPassword(config.Pw).
			SetUsername(config.User).
			SetClientID(config.ClientId).
			AddBroker(config.Broker).
			SetAutoReconnect(true)
		setOptions(options)
		result.clients = append(result.clients, paho.NewClient(options))
	}
	return result
}

type MultiClient struct {
	clients []paho.Client
}

func do(clients []paho.Client, f func(client paho.Client) paho.Token) paho.Token {
	token, done := NewToken()
	wg := sync.WaitGroup{}
	var err error
	var mux sync.Mutex
	for _, client := range clients {
		wg.Add(1)
		go func(client paho.Client) {
			subT := f(client)
			subT.Wait()
			mux.Lock()
			defer mux.Unlock()
			err = errors.Join(err, subT.Error())
			wg.Done()
		}(client)
	}
	go func() {
		wg.Wait()
		mux.Lock()
		defer mux.Unlock()
		done(err)
	}()
	return token
}

func (this *MultiClient) IsConnected() bool {
	connected := false
	for _, client := range this.clients {
		if !client.IsConnected() {
			return false
		} else {
			connected = true
		}
	}
	return connected
}

func (this *MultiClient) IsConnectionOpen() bool {
	connected := false
	for _, client := range this.clients {
		if !client.IsConnectionOpen() {
			return false
		} else {
			connected = true
		}
	}
	return connected
}

func (this *MultiClient) Connect() paho.Token {
	return do(this.clients, func(client paho.Client) paho.Token {
		return client.Connect()
	})
}

func (this *MultiClient) Disconnect(quiesce uint) {
	wg := sync.WaitGroup{}
	for _, client := range this.clients {
		wg.Add(1)
		go func(client paho.Client) {
			client.Disconnect(quiesce)
			wg.Done()
		}(client)
	}
	wg.Wait()
}

func (this *MultiClient) Publish(topic string, qos byte, retained bool, payload interface{}) paho.Token {
	return do(this.clients, func(client paho.Client) paho.Token {
		return client.Publish(topic, qos, retained, payload)
	})
}

func (this *MultiClient) Subscribe(topic string, qos byte, callback paho.MessageHandler) paho.Token {
	return do(this.clients, func(client paho.Client) paho.Token {
		return client.Subscribe(topic, qos, callback)
	})
}

func (this *MultiClient) SubscribeMultiple(filters map[string]byte, callback paho.MessageHandler) paho.Token {
	return do(this.clients, func(client paho.Client) paho.Token {
		return client.SubscribeMultiple(filters, callback)
	})
}

func (this *MultiClient) Unsubscribe(topics ...string) paho.Token {
	return do(this.clients, func(client paho.Client) paho.Token {
		return client.Unsubscribe(topics...)
	})
}

func (this *MultiClient) AddRoute(topic string, callback paho.MessageHandler) {
	for _, client := range this.clients {
		client.AddRoute(topic, callback)
	}
}

func (this *MultiClient) OptionsReader() paho.ClientOptionsReader {
	//TODO implement me
	panic("implement me")
}
