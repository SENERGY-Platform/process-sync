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
	"log/slog"
	"os"
	"reflect"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	struct_logger "github.com/SENERGY-Platform/go-service-base/struct-logger"
)

var LogEnvConfig = true

type Config struct {
	KafkaUrl                   string `json:"kafka_url"`
	KafkaConsumerGroup         string `json:"kafka_consumer_group"`
	ProcessDeploymentDoneTopic string `json:"process_deployment_done_topic"`
	DeviceGroupTopic           string `json:"device_group_topic"`

	AuthExpirationTimeBuffer float64 `json:"auth_expiration_time_buffer"`
	AuthEndpoint             string  `json:"auth_endpoint"`
	AuthClientId             string  `json:"auth_client_id" config:"secret"`
	AuthClientSecret         string  `json:"auth_client_secret" config:"secret"`

	MongoUrl string `json:"mongo_url"`

	ApiPort                           string `json:"api_port"`
	MongoTable                        string `json:"mongo_table"`
	MongoWardenCollection             string `json:"mongo_warden_collection"`
	MongoDeploymentWardenCollection   string `json:"mongo_deployment_warden_collection"`
	MongoProcessDefinitionCollection  string `json:"mongo_process_definition_collection"`
	MongoDeploymentCollection         string `json:"mongo_deployment_collection"`
	MongoDeploymentMetadataCollection string `json:"mongo_deployment_metadata_collection"`
	MongoProcessHistoryCollection     string `json:"mongo_process_history_collection"`
	MongoIncidentCollection           string `json:"mongo_incident_collection"`
	MongoProcessInstanceCollection    string `json:"mongo_process_instance_collection"`
	MongoLastNetworkContactCollection string `json:"mongo_last_network_contact_collection"`
	PermissionsV2Url                  string `json:"permissions_v2_url"`
	DeviceRepoUrl                     string `json:"device_repo_url"`
	AnalyticsEnvelopePrefix           string `json:"analytics_envelope_prefix"`

	CleanupMaxAge   string `json:"cleanup_max_age"`
	CleanupInterval string `json:"cleanup_interval"`

	ApiDocsProviderBaseUrl string `json:"api_docs_provider_base_url"`

	DeveloperNotificationUrl string `json:"developer_notification_url"`

	InitTopics bool `json:"init_topics"`

	Mqtt             []MqttConfig `json:"mqtt"`
	MqttGroupId      string       `json:"mqtt_group_id"` //optional
	MqttCleanSession bool         `json:"mqtt_clean_session"`

	LogLevel             string       `json:"log_level"`
	LoggerTrimFormat     string       `json:"logger_trim_format"`
	LoggerTrimAttributes string       `json:"logger_trim_attributes"`
	logger               *slog.Logger `json:"-"`

	WardenInterval          string `json:"warden_interval"`
	WardenAgeGate           string `json:"warden_age_gate"`
	RunWardenDbLoop         bool   `json:"run_warden_db_loop"`
	RunWardenProcessLoop    bool   `json:"run_warden_process_loop"`
	RunWardenDeploymentLoop bool   `json:"run_warden_deployment_loop"`
}

type MqttConfig struct {
	Broker   string `json:"broker"`
	ClientId string `json:"client_id" config:"secret"`
	User     string `json:"user" config:"secret"`
	Pw       string `json:"pw" config:"secret"`
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
	handleMqttConfig(&config)
	return config, nil
}

func handleMqttConfig(config *Config) {
	m := map[string]MqttConfig{}

	for _, env := range os.Environ() {
		parts := strings.Split(env, "=")
		if len(parts) == 2 {
			switch {
			case parts[0] == "MQTT_BROKER":
				key := ""
				conf := m[key]
				conf.Broker = parts[1]
				m[key] = conf
			case parts[0] == "MQTT_PW":
				key := ""
				conf := m[key]
				conf.Pw = parts[1]
				m[key] = conf
			case parts[0] == "MQTT_USER":
				key := ""
				conf := m[key]
				conf.User = parts[1]
				m[key] = conf
			case parts[0] == "MQTT_CLIENT_ID":
				key := ""
				conf := m[key]
				conf.ClientId = parts[1]
				m[key] = conf
			case strings.HasPrefix(parts[0], "MQTT_BROKER_"):
				key := strings.ToLower(strings.TrimPrefix(parts[0], "MQTT_BROKER_"))
				conf := m[key]
				conf.Broker = parts[1]
				m[key] = conf
			case strings.HasPrefix(parts[0], "MQTT_PW_"):
				key := strings.ToLower(strings.TrimPrefix(parts[0], "MQTT_PW_"))
				conf := m[key]
				conf.Pw = parts[1]
				m[key] = conf
			case strings.HasPrefix(parts[0], "MQTT_USER_"):
				key := strings.ToLower(strings.TrimPrefix(parts[0], "MQTT_USER_"))
				conf := m[key]
				conf.User = parts[1]
				m[key] = conf
			case strings.HasPrefix(parts[0], "MQTT_CLIENT_ID_"):
				key := strings.ToLower(strings.TrimPrefix(parts[0], "MQTT_CLIENT_ID_"))
				conf := m[key]
				conf.ClientId = parts[1]
				m[key] = conf
			}
		}
	}
	if len(m) > 0 {
		config.Mqtt = []MqttConfig{}
		for _, conf := range m {
			config.Mqtt = append(config.Mqtt, conf)
		}
	}
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

func (this *Config) GetLogger() *slog.Logger {
	if this.logger == nil {
		info, ok := debug.ReadBuildInfo()
		project := ""
		org := ""
		if ok {
			if parts := strings.Split(info.Main.Path, "/"); len(parts) > 2 {
				project = strings.Join(parts[2:], "/")
				org = strings.Join(parts[:2], "/")
			}
		}
		this.logger = struct_logger.New(
			struct_logger.Config{
				Handler:        struct_logger.JsonHandlerSelector,
				Level:          this.LogLevel,
				TimeFormat:     time.RFC3339Nano,
				TimeUtc:        true,
				AddMeta:        true,
				TrimFormat:     this.LoggerTrimFormat,
				TrimAttributes: this.LoggerTrimAttributes,
			},
			os.Stdout,
			org,
			project,
		)
	}
	return this.logger
}
