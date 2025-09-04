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
	"errors"
	"strings"

	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/database"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var definitionIdKey string
var definitionNameKey string
var definitionNetworkIdKey string
var definitionDeploymentKey string

func init() {
	prepareCollection(func(config configuration.Config) string {
		return config.MongoProcessDefinitionCollection
	},
		model.ProcessDefinition{},
		[]KeyMapping{
			{
				FieldName: "ProcessDefinition.Id",
				Key:       &definitionIdKey,
			},
			{
				FieldName: "ProcessDefinition.Name",
				Key:       &definitionNameKey,
			},
			{
				FieldName: "ProcessDefinition.DeploymentId",
				Key:       &definitionDeploymentKey,
			},
			{
				FieldName: "SyncInfo.NetworkId",
				Key:       &definitionNetworkIdKey,
			},
		},
		[]IndexDesc{
			{
				Name:   "definitionnetworkindex",
				Unique: false,
				Asc:    true,
				Keys:   []*string{&definitionNetworkIdKey},
			},
			{
				Name:   "definitiondeploymentindex",
				Unique: false,
				Asc:    true,
				Keys:   []*string{&definitionDeploymentKey, &definitionNetworkIdKey},
			},
			{
				Name:   "definitioncompoundindex",
				Unique: false,
				Asc:    true,
				Keys:   []*string{&definitionIdKey, &definitionNetworkIdKey},
			},
		},
	)
}

func (this *Mongo) processDefinitionCollection() *mongo.Collection {
	return this.client.Database(this.config.MongoTable).Collection(this.config.MongoProcessDefinitionCollection)
}

func (this *Mongo) SaveProcessDefinition(processDefinition model.ProcessDefinition) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.processDefinitionCollection().ReplaceOne(
		ctx,
		bson.M{
			definitionIdKey:        processDefinition.Id,
			definitionNetworkIdKey: processDefinition.NetworkId,
		},
		processDefinition,
		options.Replace().SetUpsert(true))
	return err
}

func (this *Mongo) RemoveProcessDefinition(networkId string, processDefinitionId string) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.processDefinitionCollection().DeleteOne(
		ctx,
		bson.M{
			definitionIdKey:        processDefinitionId,
			definitionNetworkIdKey: networkId,
		})
	return err
}

func (this *Mongo) RemoveUnknownProcessDefinitions(networkId string, knownIds []string) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.processDefinitionCollection().DeleteMany(
		ctx,
		bson.M{
			definitionIdKey:        bson.M{"$nin": knownIds},
			definitionNetworkIdKey: networkId,
		})
	return err
}

func (this *Mongo) ReadProcessDefinition(networkId string, processDefinitionId string) (processDefinition model.ProcessDefinition, err error) {
	ctx, _ := this.getTimeoutContext()
	result := this.processDefinitionCollection().FindOne(
		ctx,
		bson.M{
			definitionIdKey:        processDefinitionId,
			definitionNetworkIdKey: networkId,
		})
	err = result.Err()
	if err == mongo.ErrNoDocuments {
		return processDefinition, database.ErrNotFound
	}
	if err != nil {
		return
	}
	err = result.Decode(&processDefinition)
	if err == mongo.ErrNoDocuments {
		return processDefinition, database.ErrNotFound
	}
	return processDefinition, err
}

func (this *Mongo) ListProcessDefinitions(networkIds []string, limit int64, offset int64, sort string) (result []model.ProcessDefinition, err error) {
	opt := options.Find()
	opt.SetLimit(limit)
	opt.SetSkip(offset)

	parts := strings.Split(sort, ".")
	sortby := definitionIdKey
	switch parts[0] {
	case "id":
		sortby = definitionIdKey
	case "name":
		sortby = definitionNameKey
	}
	direction := int32(1)
	if len(parts) > 1 && parts[1] == "desc" {
		direction = int32(-1)
	}
	opt.SetSort(bson.D{{sortby, direction}})

	ctx, _ := this.getTimeoutContext()
	cursor, err := this.processDefinitionCollection().Find(ctx, bson.M{definitionNetworkIdKey: bson.M{"$in": networkIds}}, opt)
	if err != nil {
		return nil, err
	}
	for cursor.Next(ctx) {
		element := model.ProcessDefinition{}
		err = cursor.Decode(&element)
		if err != nil {
			return nil, err
		}
		result = append(result, element)
	}
	err = cursor.Err()
	return
}

func (this *Mongo) GetDefinitionByDeploymentId(networkId string, deploymentId string) (processDefinition model.ProcessDefinition, err error) {
	ctx, _ := this.getTimeoutContext()
	result := this.processDefinitionCollection().FindOne(
		ctx,
		bson.M{
			definitionDeploymentKey: deploymentId,
			definitionNetworkIdKey:  networkId,
		})
	err = result.Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return processDefinition, database.ErrNotFound
	}
	if err != nil {
		return
	}
	err = result.Decode(&processDefinition)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return processDefinition, database.ErrNotFound
	}
	return processDefinition, err
}

func (this *Mongo) GetDefinitionsOfDeploymentIdList(networkId string, deploymentIds []string) (result map[string]model.ProcessDefinition, err error) {
	result = map[string]model.ProcessDefinition{}
	ctx, _ := this.getTimeoutContext()
	cursor, err := this.processDefinitionCollection().Find(ctx, bson.M{
		definitionNetworkIdKey:  networkId,
		definitionDeploymentKey: bson.M{"$in": deploymentIds}})
	if err != nil {
		return nil, err
	}
	for cursor.Next(ctx) {
		element := model.ProcessDefinition{}
		err = cursor.Decode(&element)
		if err != nil {
			return nil, err
		}
		result[element.DeploymentId] = element
	}
	err = cursor.Err()
	return
}
