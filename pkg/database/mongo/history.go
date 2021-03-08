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
	"go.mongodb.org/mongo-driver/x/bsonx"
	"strings"
)

var historyIdKey string
var historyNetworkIdKey string
var historyPlaceholderKey string

func init() {
	prepareCollection(func(config configuration.Config) string {
		return config.MongoProcessHistoryCollection
	},
		model.HistoricProcessInstance{},
		[]KeyMapping{
			{
				FieldName: "SyncInfo.IsPlaceholder",
				Key:       &historyPlaceholderKey,
			},
			{
				FieldName: "HistoricProcessInstance.Id",
				Key:       &historyIdKey,
			},
			{
				FieldName: "SyncInfo.NetworkId",
				Key:       &historyNetworkIdKey,
			},
		},
		[]IndexDesc{
			{
				Name:   "historyplaceholderindex",
				Unique: false,
				Asc:    true,
				Keys:   []*string{&instancePlaceholderKey},
			},
			{
				Name:   "historynetworkindex",
				Unique: false,
				Asc:    true,
				Keys:   []*string{&historyNetworkIdKey},
			},
			{
				Name:   "historycompoundindex",
				Unique: false,
				Asc:    true,
				Keys:   []*string{&historyIdKey, &historyNetworkIdKey},
			},
		},
	)
}

func (this *Mongo) processHistoryCollection() *mongo.Collection {
	return this.client.Database(this.config.MongoTable).Collection(this.config.MongoProcessHistoryCollection)
}

func (this *Mongo) SaveHistoricProcessInstance(historicProcessInstance model.HistoricProcessInstance) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.processHistoryCollection().ReplaceOne(
		ctx,
		bson.M{
			historyIdKey:        historicProcessInstance.Id,
			historyNetworkIdKey: historicProcessInstance.NetworkId,
		},
		historicProcessInstance,
		options.Replace().SetUpsert(true))
	return err
}

func (this *Mongo) RemoveHistoricProcessInstance(networkId string, historicProcessInstanceId string) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.processHistoryCollection().DeleteOne(
		ctx,
		bson.M{
			historyIdKey:        historicProcessInstanceId,
			historyNetworkIdKey: networkId,
		})
	return err
}

func (this *Mongo) RemoveUnknownHistoricProcessInstances(networkId string, knownIds []string) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.processHistoryCollection().DeleteMany(
		ctx,
		bson.M{
			historyIdKey:        bson.M{"$nin": knownIds},
			historyNetworkIdKey: networkId,
		})
	return err
}

func (this *Mongo) ReadHistoricProcessInstance(networkId string, historicProcessInstanceId string) (historicProcessInstance model.HistoricProcessInstance, err error) {
	ctx, _ := this.getTimeoutContext()
	result := this.processHistoryCollection().FindOne(
		ctx,
		bson.M{
			historyIdKey:        historicProcessInstanceId,
			historyNetworkIdKey: networkId,
		})
	err = result.Err()
	if err == mongo.ErrNoDocuments {
		return historicProcessInstance, database.ErrNotFound
	}
	if err != nil {
		return
	}
	err = result.Decode(&historicProcessInstance)
	if err == mongo.ErrNoDocuments {
		return historicProcessInstance, database.ErrNotFound
	}
	return historicProcessInstance, err
}

func (this *Mongo) ListHistoricProcessInstances(networkIds []string, limit int64, offset int64, sort string) (result []model.HistoricProcessInstance, err error) {
	opt := options.Find()
	opt.SetLimit(limit)
	opt.SetSkip(offset)

	parts := strings.Split(sort, ".")
	sortby := historyIdKey
	switch parts[0] {
	case "id":
		sortby = historyIdKey
	}
	direction := int32(1)
	if len(parts) > 1 && parts[1] == "desc" {
		direction = int32(-1)
	}
	opt.SetSort(bsonx.Doc{{sortby, bsonx.Int32(direction)}})

	ctx, _ := this.getTimeoutContext()
	cursor, err := this.processHistoryCollection().Find(ctx, bson.M{historyNetworkIdKey: bson.M{"$in": networkIds}}, opt)
	if err != nil {
		return nil, err
	}
	for cursor.Next(ctx) {
		element := model.HistoricProcessInstance{}
		err = cursor.Decode(&element)
		if err != nil {
			return nil, err
		}
		result = append(result, element)
	}
	err = cursor.Err()
	return
}

func (this *Mongo) RemovePlaceholderHistoricProcessInstances(networkId string) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.processHistoryCollection().DeleteMany(
		ctx,
		bson.M{
			historyPlaceholderKey: true,
			historyNetworkIdKey:   networkId,
		})
	return err
}
