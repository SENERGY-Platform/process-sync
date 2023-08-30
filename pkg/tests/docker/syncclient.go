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
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"log"
	"sync"
)

func MgwProcessSyncClient(ctx context.Context, wg *sync.WaitGroup, camundaDb, camundaUrl, mqttUrl, mqttClientId, networkId, deploymentMetadataStorage string) (err error) {
	log.Println("start mgw-process-sync-client")
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "ghcr.io/senergy-platform/mgw-process-sync-client:dev",
			Env: map[string]string{
				"CAMUNDA_DB":                  camundaDb,
				"CAMUNDA_URL":                 camundaUrl,
				"MQTT_BROKER":                 mqttUrl,
				"MQTT_CLIENT_ID":              mqttClientId,
				"NETWORK_ID":                  networkId,
				"DEBUG":                       "true",
				"DEPLOYMENT_METADATA_STORAGE": deploymentMetadataStorage,
			},
			ExposedPorts:    []string{"8080/tcp"},
			WaitingFor:      wait.ForListeningPort("8080/tcp"),
			AlwaysPullImage: true,
		},
		Started: true,
	})
	if err != nil {
		return err
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		log.Println("DEBUG: remove container mgw-process-sync-client", c.Terminate(context.Background()))
	}()

	return err
}
