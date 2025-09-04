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

var instanceIdKey string
var instanceNetworkIdKey string
var instancePlaceholderKey string
var instanceBusinessKeyKey string

func init() {
	prepareCollection(func(config configuration.Config) string {
		return config.MongoProcessInstanceCollection
	},
		model.ProcessInstance{},
		[]KeyMapping{
			{
				FieldName: "ProcessInstance.Id",
				Key:       &instanceIdKey,
			},
			{
				FieldName: "ProcessInstance.BusinessKey",
				Key:       &instanceBusinessKeyKey,
			},
			{
				FieldName: "SyncInfo.NetworkId",
				Key:       &instanceNetworkIdKey,
			},
			{
				FieldName: "SyncInfo.IsPlaceholder",
				Key:       &instancePlaceholderKey,
			},
		},
		[]IndexDesc{
			{
				Name:   "instanceplaceholderindex",
				Unique: false,
				Asc:    true,
				Keys:   []*string{&instancePlaceholderKey},
			},
			{
				Name:   "instancenetworkindex",
				Unique: false,
				Asc:    true,
				Keys:   []*string{&instanceNetworkIdKey},
			},
			{
				Name:   "instancecompoundindex",
				Unique: false,
				Asc:    true,
				Keys:   []*string{&instanceIdKey, &instanceNetworkIdKey},
			},
		},
	)
}

func (this *Mongo) processInstanceCollection() *mongo.Collection {
	return this.client.Database(this.config.MongoTable).Collection(this.config.MongoProcessInstanceCollection)
}

func (this *Mongo) SaveProcessInstance(processInstance model.ProcessInstance) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.processInstanceCollection().ReplaceOne(
		ctx,
		bson.M{
			instanceIdKey:        processInstance.Id,
			instanceNetworkIdKey: processInstance.NetworkId,
		},
		processInstance,
		options.Replace().SetUpsert(true))
	return err
}

func (this *Mongo) RemoveProcessInstance(networkId string, processInstanceId string) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.processInstanceCollection().DeleteOne(
		ctx,
		bson.M{
			instanceIdKey:        processInstanceId,
			instanceNetworkIdKey: networkId,
		})
	return err
}

func (this *Mongo) RemovePlaceholderProcessInstances(networkId string) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.processInstanceCollection().DeleteMany(
		ctx,
		bson.M{
			instancePlaceholderKey: true,
			instanceNetworkIdKey:   networkId,
		})
	return err
}

func (this *Mongo) RemoveUnknownProcessInstances(networkId string, knownIds []string) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.processInstanceCollection().DeleteMany(
		ctx,
		bson.M{
			instanceIdKey:        bson.M{"$nin": knownIds},
			instanceNetworkIdKey: networkId,
		})
	return err
}

func (this *Mongo) ReadProcessInstance(networkId string, processInstanceId string) (processInstance model.ProcessInstance, err error) {
	ctx, _ := this.getTimeoutContext()
	result := this.processInstanceCollection().FindOne(
		ctx,
		bson.M{
			instanceIdKey:        processInstanceId,
			instanceNetworkIdKey: networkId,
		})
	err = result.Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return processInstance, database.ErrNotFound
	}
	if err != nil {
		return
	}
	err = result.Decode(&processInstance)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return processInstance, database.ErrNotFound
	}
	return processInstance, err
}

func (this *Mongo) FindProcessInstances(query model.InstanceQuery) (result []model.ProcessInstance, err error) {
	opt := options.Find()
	opt.SetLimit(query.Limit)
	opt.SetSkip(query.Offset)

	if query.Sort == "" {
		query.Sort = "id"
	}
	parts := strings.Split(query.Sort, ".")
	sortby := instanceIdKey
	switch parts[0] {
	case "id":
		sortby = instanceIdKey
	}
	direction := int32(1)
	if len(parts) > 1 && parts[1] == "desc" {
		direction = int32(-1)
	}
	opt.SetSort(bson.D{{sortby, direction}})

	filter := bson.M{}
	if query.NetworkIds != nil {
		filter[instanceNetworkIdKey] = bson.M{"$in": query.NetworkIds}
	}
	if query.BusinessKeys != nil {
		filter[instanceBusinessKeyKey] = bson.M{"$in": query.BusinessKeys}
	}

	ctx, _ := this.getTimeoutContext()
	cursor, err := this.processInstanceCollection().Find(ctx, filter, opt)
	if err != nil {
		return nil, err
	}
	for cursor.Next(ctx) {
		element := model.ProcessInstance{}
		err = cursor.Decode(&element)
		if err != nil {
			return nil, err
		}
		result = append(result, element)
	}
	err = cursor.Err()
	return
}

func (this *Mongo) ListProcessInstances(networkIds []string, limit int64, offset int64, sort string) (result []model.ProcessInstance, err error) {
	opt := options.Find()
	opt.SetLimit(limit)
	opt.SetSkip(offset)

	parts := strings.Split(sort, ".")
	sortby := instanceIdKey
	switch parts[0] {
	case "id":
		sortby = instanceIdKey
	}
	direction := int32(1)
	if len(parts) > 1 && parts[1] == "desc" {
		direction = int32(-1)
	}
	opt.SetSort(bson.D{{sortby, direction}})

	ctx, _ := this.getTimeoutContext()
	cursor, err := this.processInstanceCollection().Find(ctx, bson.M{instanceNetworkIdKey: bson.M{"$in": networkIds}}, opt)
	if err != nil {
		return nil, err
	}
	for cursor.Next(ctx) {
		element := model.ProcessInstance{}
		err = cursor.Decode(&element)
		if err != nil {
			return nil, err
		}
		result = append(result, element)
	}
	err = cursor.Err()
	return
}
