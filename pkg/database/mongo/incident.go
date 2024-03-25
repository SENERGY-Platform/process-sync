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

package mongo

import (
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/database"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"strings"
)

var incidentIdKey string
var incidentTimeKey string
var incidentNetworkIdKey string
var incidentProcessInstanceIdKey string
var incidentProcessDefinitionIdKey string

func init() {
	prepareCollection(func(config configuration.Config) string {
		return config.MongoIncidentCollection
	},
		model.Incident{},
		[]KeyMapping{
			{
				FieldName: "Incident.ProcessInstanceId",
				Key:       &incidentProcessInstanceIdKey,
			},
			{
				FieldName: "Incident.ProcessDefinitionId",
				Key:       &incidentProcessDefinitionIdKey,
			},
			{
				FieldName: "Incident.Id",
				Key:       &incidentIdKey,
			},
			{
				FieldName: "Incident.Time",
				Key:       &incidentTimeKey,
			},
			{
				FieldName: "SyncInfo.NetworkId",
				Key:       &incidentNetworkIdKey,
			},
		},
		[]IndexDesc{
			{
				Name:   "incidentnetworkindex",
				Unique: false,
				Asc:    true,
				Keys:   []*string{&incidentNetworkIdKey},
			},
			{
				Name:   "incidentbynetworkandinstance",
				Unique: false,
				Asc:    true,
				Keys:   []*string{&incidentNetworkIdKey, &incidentProcessInstanceIdKey},
			},
			{
				Name:   "incidentbynetworkanddefinition",
				Unique: false,
				Asc:    true,
				Keys:   []*string{&incidentNetworkIdKey, &incidentProcessDefinitionIdKey},
			},
			{
				Name:   "incidentcompoundindex",
				Unique: false,
				Asc:    true,
				Keys:   []*string{&incidentIdKey, &incidentNetworkIdKey},
			},
		},
	)
}

func (this *Mongo) incidentCollection() *mongo.Collection {
	return this.client.Database(this.config.MongoTable).Collection(this.config.MongoIncidentCollection)
}

func (this *Mongo) SaveIncident(incident model.Incident) (newDocument bool, err error) {
	ctx, _ := this.getTimeoutContext()
	result, err := this.incidentCollection().ReplaceOne(
		ctx,
		bson.M{
			incidentIdKey:        incident.Id,
			incidentNetworkIdKey: incident.NetworkId,
		},
		incident,
		options.Replace().SetUpsert(true))
	newDocument = result.MatchedCount == 0
	return newDocument, err
}

func (this *Mongo) RemoveIncident(networkId string, incidentId string) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.incidentCollection().DeleteOne(
		ctx,
		bson.M{
			incidentIdKey:        incidentId,
			incidentNetworkIdKey: networkId,
		})
	return err
}

func (this *Mongo) RemoveIncidentOfInstance(networkId string, instanceId string) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.incidentCollection().DeleteMany(
		ctx,
		bson.M{
			incidentNetworkIdKey:         networkId,
			incidentProcessInstanceIdKey: instanceId,
		})
	return err
}

func (this *Mongo) RemoveIncidentOfDefinition(networkId string, definitionId string) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.incidentCollection().DeleteMany(
		ctx,
		bson.M{
			incidentNetworkIdKey:           networkId,
			incidentProcessDefinitionIdKey: definitionId,
		})
	return err
}

func (this *Mongo) RemoveIncidentOfNotInstances(networkId string, notInstanceIds []string) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.incidentCollection().DeleteMany(
		ctx,
		bson.M{
			incidentNetworkIdKey:         networkId,
			incidentProcessInstanceIdKey: bson.M{"$nin": notInstanceIds},
		})
	return err
}

func (this *Mongo) RemoveIncidentOfNotDefinitions(networkId string, notDefinitionIds []string) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.incidentCollection().DeleteMany(
		ctx,
		bson.M{
			incidentNetworkIdKey:           networkId,
			incidentProcessDefinitionIdKey: bson.M{"$nin": notDefinitionIds},
		})
	return err
}

func (this *Mongo) RemoveUnknownIncidents(networkId string, knownIds []string) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.incidentCollection().DeleteMany(
		ctx,
		bson.M{
			incidentIdKey:        bson.M{"$nin": knownIds},
			incidentNetworkIdKey: networkId,
		})
	return err
}

func (this *Mongo) ReadIncident(networkId string, incidentId string) (incident model.Incident, err error) {
	ctx, _ := this.getTimeoutContext()
	result := this.incidentCollection().FindOne(
		ctx,
		bson.M{
			incidentIdKey:        incidentId,
			incidentNetworkIdKey: networkId,
		})
	err = result.Err()
	if err == mongo.ErrNoDocuments {
		return incident, database.ErrNotFound
	}
	if err != nil {
		return
	}
	err = result.Decode(&incident)
	if err == mongo.ErrNoDocuments {
		return incident, database.ErrNotFound
	}
	return incident, err
}

func (this *Mongo) ListIncidents(networkIds []string, processInstanceId string, limit int64, offset int64, sort string) (result []model.Incident, err error) {
	opt := options.Find()
	opt.SetLimit(limit)
	opt.SetSkip(offset)

	parts := strings.Split(sort, ".")
	sortby := definitionIdKey
	switch parts[0] {
	case "id":
		sortby = incidentIdKey
	case "time":
		sortby = incidentTimeKey
	}
	direction := int32(1)
	if len(parts) > 1 && parts[1] == "desc" {
		direction = int32(-1)
	}
	opt.SetSort(bson.D{{sortby, direction}})

	query := bson.M{incidentNetworkIdKey: bson.M{"$in": networkIds}}
	if processInstanceId != "" {
		query[incidentProcessInstanceIdKey] = processInstanceId
	}

	ctx, _ := this.getTimeoutContext()
	cursor, err := this.incidentCollection().Find(ctx, query, opt)
	if err != nil {
		return nil, err
	}
	for cursor.Next(ctx) {
		element := model.Incident{}
		err = cursor.Decode(&element)
		if err != nil {
			return nil, err
		}
		result = append(result, element)
	}
	err = cursor.Err()
	return
}
