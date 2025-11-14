/*
 * Copyright 2021 InfAI (CC SES)
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

package docker

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func MgwProcessSync(ctx context.Context, wg *sync.WaitGroup, mongoUrl string, mqttBrokerUrl string) (endpoint string, err error) {
	log.Println("start mgw-process-sync-client")
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "ghcr.io/senergy-platform/process-sync:v0.0.24",
			Env: map[string]string{
				"MONGO_URL":   mongoUrl,
				"MQTT_BROKER": mqttBrokerUrl,
			},
			ExposedPorts:    []string{"8080/tcp"},
			WaitingFor:      wait.ForListeningPort("8080/tcp"),
			AlwaysPullImage: true,
		},
		Started: true,
	})
	if err != nil {
		if c != nil {
			reader, logerr := c.Logs(context.Background())
			if logerr != nil {
				log.Println("ERROR: unable to get container log", logerr)
				return "", err
			}
			buf := new(strings.Builder)
			io.Copy(buf, reader)
			fmt.Println("CLIENT LOGS: ------------------------------------------")
			fmt.Println(buf.String())
			fmt.Println("\n---------------------------------------------------------------")
		}
		return "", err
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			log.Println("DEBUG: remove container mgw-process-sync", c.Terminate(context.Background()))
		}()
		<-ctx.Done()
		/*
			reader, err := c.Logs(context.Background())
			if err != nil {
				log.Println("ERROR: unable to get container log")
				return
			}
			buf := new(strings.Builder)
			io.Copy(buf, reader)
			fmt.Println("CLIENT LOGS: ------------------------------------------")
			fmt.Println(buf.String())
			fmt.Println("\n---------------------------------------------------------------")
		*/
	}()

	port, err := c.MappedPort(ctx, "8080/tcp")
	if err != nil {
		return "", err
	}

	return port.Port(), err
}
