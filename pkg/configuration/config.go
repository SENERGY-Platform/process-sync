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

package configuration

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var LogEnvConfig = true

type Config struct {
	Debug bool `json:"debug"`

	KafkaUrl                   string `json:"kafka_url"`
	KafkaConsumerGroup         string `json:"kafka_consumer_group"`
	ProcessDeploymentDoneTopic string `json:"process_deployment_done_topic"`
	DeviceGroupTopic           string `json:"device_group_topic"`

	AuthExpirationTimeBuffer float64 `json:"auth_expiration_time_buffer"`
	AuthEndpoint             string  `json:"auth_endpoint"`
	AuthClientId             string  `json:"auth_client_id"`
	AuthClientSecret         string  `json:"auth_client_secret"`

	MongoUrl string `json:"mongo_url"`

	MqttPw                            string `json:"mqtt_pw" config:"secret"`
	MqttUser                          string `json:"mqtt_user" config:"secret"`
	MqttClientId                      string `json:"mqtt_client_id" config:"secret"`
	MqttBroker                        string `json:"mqtt_broker"`
	MqttCleanSession                  bool   `json:"mqtt_clean_session"`
	MqttGroupId                       string `json:"mqtt_group_id"` //optional
	ApiPort                           string `json:"api_port"`
	MongoTable                        string `json:"mongo_table"`
	MongoProcessDefinitionCollection  string `json:"mongo_process_definition_collection"`
	MongoDeploymentCollection         string `json:"mongo_deployment_collection"`
	MongoDeploymentMetadataCollection string `json:"mongo_deployment_metadata_collection"`
	MongoProcessHistoryCollection     string `json:"mongo_process_history_collection"`
	MongoIncidentCollection           string `json:"mongo_incident_collection"`
	MongoProcessInstanceCollection    string `json:"mongo_process_instance_collection"`
	MongoLastNetworkContactCollection string `json:"mongo_last_network_contact_collection"`
	PermissionsUrl                    string `json:"permissions_url"`
	DeviceRepoUrl                     string `json:"device_repo_url"`
	AnalyticsEnvelopePrefix           string `json:"analytics_envelope_prefix"`

	CleanupMaxAge   string `json:"cleanup_max_age"`
	CleanupInterval string `json:"cleanup_interval"`
}

// loads config from json in location and used environment variables (e.g ZookeeperUrl --> ZOOKEEPER_URL)
func Load(location string) (config Config, err error) {
	file, err := os.Open(location)
	if err != nil {
		return config, err
	}
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		return config, err
	}
	handleEnvironmentVars(&config)
	return config, nil
}

var camel = regexp.MustCompile("(^[^A-Z]*|[A-Z]*)([A-Z][^A-Z]+|$)")

func fieldNameToEnvName(s string) string {
	var a []string
	for _, sub := range camel.FindAllStringSubmatch(s, -1) {
		if sub[1] != "" {
			a = append(a, sub[1])
		}
		if sub[2] != "" {
			a = append(a, sub[2])
		}
	}
	return strings.ToUpper(strings.Join(a, "_"))
}

// preparations for docker
func handleEnvironmentVars(config *Config) {
	configValue := reflect.Indirect(reflect.ValueOf(config))
	configType := configValue.Type()
	for index := 0; index < configType.NumField(); index++ {
		fieldName := configType.Field(index).Name
		fieldConfig := configType.Field(index).Tag.Get("config")
		envName := fieldNameToEnvName(fieldName)
		envValue := os.Getenv(envName)
		if envValue != "" {
			if !strings.Contains(fieldConfig, "secret") {
				fmt.Println("use environment variable: ", envName, " = ", envValue)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Int64 {
				i, _ := strconv.ParseInt(envValue, 10, 64)
				configValue.FieldByName(fieldName).SetInt(i)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.String {
				configValue.FieldByName(fieldName).SetString(envValue)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Bool {
				b, _ := strconv.ParseBool(envValue)
				configValue.FieldByName(fieldName).SetBool(b)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Float64 {
				f, _ := strconv.ParseFloat(envValue, 64)
				configValue.FieldByName(fieldName).SetFloat(f)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Slice {
				val := []string{}
				for _, element := range strings.Split(envValue, ",") {
					val = append(val, strings.TrimSpace(element))
				}
				configValue.FieldByName(fieldName).Set(reflect.ValueOf(val))
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Map {
				value := map[string]string{}
				for _, element := range strings.Split(envValue, ",") {
					keyVal := strings.Split(element, ":")
					key := strings.TrimSpace(keyVal[0])
					val := strings.TrimSpace(keyVal[1])
					value[key] = val
				}
				configValue.FieldByName(fieldName).Set(reflect.ValueOf(value))
			}
		}
	}
}
