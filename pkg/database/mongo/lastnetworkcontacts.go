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
	"time"

	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var networkIdKey string
var networkTimeKey string

func init() {
	prepareCollection(func(config configuration.Config) string {
		return config.MongoLastNetworkContactCollection
	},
		model.LastNetworkContact{},
		[]KeyMapping{
			{
				FieldName: "NetworkId",
				Key:       &networkIdKey,
			},
			{
				FieldName: "Time",
				Key:       &networkTimeKey,
			},
		},
		[]IndexDesc{
			{
				Name:        "network_id_index",
				Keys:        []*string{&networkIdKey},
				Asc:         true,
				Unique:      false,
				IsTextIndex: true,
			},
			{
				Name:   "last_contact_time_index",
				Unique: false,
				Asc:    true,
				Keys:   []*string{&networkTimeKey},
			},
		},
	)
}

func (this *Mongo) lastNetworkContactCollection() *mongo.Collection {
	return this.client.Database(this.config.MongoTable).Collection(this.config.MongoLastNetworkContactCollection)
}

func (this *Mongo) SaveLastContact(lastContact model.LastNetworkContact) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.lastNetworkContactCollection().ReplaceOne(
		ctx,
		bson.M{
			networkIdKey: lastContact.NetworkId,
		},
		lastContact,
		options.Replace().SetUpsert(true))
	return err
}

func (this *Mongo) FilterNetworkIds(networkIds []string) (result []string, err error) {
	result = []string{}
	ctx, _ := this.getTimeoutContext()
	cursor, err := this.lastNetworkContactCollection().Find(ctx, bson.M{networkIdKey: bson.M{"$in": networkIds}})
	if err != nil {
		return nil, err
	}
	for cursor.Next(ctx) {
		element := model.LastNetworkContact{}
		err = cursor.Decode(&element)
		if err != nil {
			return nil, err
		}
		result = append(result, element.NetworkId)
	}
	err = cursor.Err()
	return result, err
}

func (this *Mongo) GetOldNetworkIds(maxAge time.Duration) (result []string, err error) {
	ctx, _ := this.getTimeoutContext()
	before := time.Now().Add(-maxAge)
	cursor, err := this.lastNetworkContactCollection().Find(ctx, bson.M{networkTimeKey: bson.M{"$lt": before}})
	if err != nil {
		return nil, err
	}
	for cursor.Next(ctx) {
		element := model.LastNetworkContact{}
		err = cursor.Decode(&element)
		if err != nil {
			return nil, err
		}
		result = append(result, element.NetworkId)
	}
	err = cursor.Err()
	return result, err
}

func (this *Mongo) ListKnownNetworkIds() (result []string, err error) {
	ctx, _ := this.getTimeoutContext()
	cursor, err := this.lastNetworkContactCollection().Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	for cursor.Next(ctx) {
		element := model.LastNetworkContact{}
		err = cursor.Decode(&element)
		if err != nil {
			return nil, err
		}
		result = append(result, element.NetworkId)
	}
	err = cursor.Err()
	return result, err
}

func (this *Mongo) RemoveOldElements(maxAge time.Duration) (err error) {
	networkIds, err := this.GetOldNetworkIds(maxAge)
	if err != nil {
		return err
	}
	this.config.GetLogger().Info("remove old elements", "network_ids", networkIds)
	if len(networkIds) == 0 {
		return nil
	}
	ctx, _ := this.getTimeoutContext()
	_, err = this.processDefinitionCollection().DeleteMany(ctx, bson.M{definitionNetworkIdKey: bson.M{"$in": networkIds}})
	if err != nil {
		return err
	}
	_, err = this.deploymentCollection().DeleteMany(ctx, bson.M{deploymentNetworkIdKey: bson.M{"$in": networkIds}})
	if err != nil {
		return err
	}
	_, err = this.deploymentMetadataCollection().DeleteMany(ctx, bson.M{metadataNetworkIdKey: bson.M{"$in": networkIds}})
	if err != nil {
		return err
	}
	_, err = this.processHistoryCollection().DeleteMany(ctx, bson.M{historyNetworkIdKey: bson.M{"$in": networkIds}})
	if err != nil {
		return err
	}
	_, err = this.incidentCollection().DeleteMany(ctx, bson.M{incidentNetworkIdKey: bson.M{"$in": networkIds}})
	if err != nil {
		return err
	}
	_, err = this.processInstanceCollection().DeleteMany(ctx, bson.M{instanceNetworkIdKey: bson.M{"$in": networkIds}})
	if err != nil {
		return err
	}
	_, err = this.lastNetworkContactCollection().DeleteMany(ctx, bson.M{networkIdKey: bson.M{"$in": networkIds}})
	if err != nil {
		return err
	}
	return nil
}
