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

package mongo

import (
	"errors"
	"strings"

	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var wardenDeploymentIdKey string
var wardenNetworkIdKey string
var wardenBusinessKeyKey string

var deploymentWardenDeploymentIdKey string
var deploymentWardenNetworkIdKey string

func init() {
	prepareCollection(func(config configuration.Config) string {
		return config.MongoWardenCollection
	},
		model.WardenInfo{},
		[]KeyMapping{
			{
				FieldName: "ProcessDeploymentId",
				Key:       &wardenDeploymentIdKey,
			},
			{
				FieldName: "NetworkId",
				Key:       &wardenNetworkIdKey,
			},
			{
				FieldName: "BusinessKey",
				Key:       &wardenBusinessKeyKey,
			},
		},
		[]IndexDesc{
			{
				Name:   "warden_deploymentid_index",
				Unique: false,
				Asc:    true,
				Keys:   []*string{&wardenDeploymentIdKey},
			},
			{
				Name:   "warden_networkid_businesskey_index",
				Unique: false,
				Asc:    true,
				Keys:   []*string{&wardenNetworkIdKey, &wardenBusinessKeyKey},
			},
		},
	)

	prepareCollection(func(config configuration.Config) string {
		return config.MongoDeploymentWardenCollection
	},
		model.DeploymentWardenInfo{},
		[]KeyMapping{
			{
				FieldName: "DeploymentId",
				Key:       &deploymentWardenDeploymentIdKey,
			},
			{
				FieldName: "NetworkId",
				Key:       &deploymentWardenNetworkIdKey,
			},
		},
		[]IndexDesc{
			{
				Name:   "deploymentwarden_networkid_businesskey_index",
				Unique: false,
				Asc:    true,
				Keys:   []*string{&deploymentWardenNetworkIdKey, &deploymentWardenDeploymentIdKey},
			},
		},
	)
}

func (this *Mongo) deploymentWardenCollection() *mongo.Collection {
	return this.client.Database(this.config.MongoTable).Collection(this.config.MongoDeploymentWardenCollection)
}

func (this *Mongo) wardenCollection() *mongo.Collection {
	return this.client.Database(this.config.MongoTable).Collection(this.config.MongoWardenCollection)
}

func (this *Mongo) SetDeploymentWardenInfo(info model.DeploymentWardenInfo) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.deploymentWardenCollection().ReplaceOne(
		ctx,
		bson.M{
			deploymentWardenNetworkIdKey:    info.NetworkId,
			deploymentWardenDeploymentIdKey: info.DeploymentId,
		},
		info,
		options.Replace().SetUpsert(true))
	return err
}

func (this *Mongo) RemoveDeploymentWardenInfo(networkId string, deploymentId string) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.deploymentWardenCollection().DeleteMany(
		ctx,
		bson.M{
			deploymentWardenNetworkIdKey:    networkId,
			deploymentWardenDeploymentIdKey: deploymentId,
		})
	return err
}

func (this *Mongo) GetDeploymentWardenInfoByDeploymentId(networkId string, deploymentId string) (info model.DeploymentWardenInfo, exists bool, err error) {
	ctx, _ := this.getTimeoutContext()
	result := this.deploymentWardenCollection().FindOne(
		ctx,
		bson.M{
			deploymentWardenNetworkIdKey:    networkId,
			deploymentWardenDeploymentIdKey: deploymentId,
		})
	err = result.Err()
	if errors.Is(err, mongo.ErrNoDocuments) {
		return info, false, nil
	}
	if err != nil {
		return info, exists, err
	}
	err = result.Decode(&info)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return info, false, nil
	}
	return info, true, nil
}

func (this *Mongo) FindDeploymentWardenInfo(query model.DeploymentWardenInfoQuery) (result []model.DeploymentWardenInfo, err error) {
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
		filter[deploymentWardenNetworkIdKey] = bson.M{"$in": query.NetworkIds}
	}
	if query.ProcessDeploymentIds != nil {
		filter[deploymentWardenDeploymentIdKey] = bson.M{"$in": query.ProcessDeploymentIds}
	}

	ctx, _ := this.getTimeoutContext()
	cursor, err := this.deploymentWardenCollection().Find(ctx, filter, opt)
	if err != nil {
		return nil, err
	}
	for cursor.Next(ctx) {
		element := model.DeploymentWardenInfo{}
		err = cursor.Decode(&element)
		if err != nil {
			return nil, err
		}
		result = append(result, element)
	}
	err = cursor.Err()
	return
}

func (this *Mongo) SetWardenInfo(info model.WardenInfo) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.wardenCollection().ReplaceOne(
		ctx,
		bson.M{
			wardenNetworkIdKey:   info.NetworkId,
			wardenBusinessKeyKey: info.BusinessKey,
		},
		info,
		options.Replace().SetUpsert(true))
	return err
}

func (this *Mongo) RemoveWardenInfo(networkId string, businessKey string) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.wardenCollection().DeleteMany(
		ctx,
		bson.M{
			wardenNetworkIdKey:   networkId,
			wardenBusinessKeyKey: businessKey,
		})
	return err
}

func (this *Mongo) FindWardenInfo(query model.WardenInfoQuery) (result []model.WardenInfo, err error) {
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
		filter[wardenNetworkIdKey] = bson.M{"$in": query.NetworkIds}
	}
	if query.ProcessDeploymentIds != nil {
		filter[wardenDeploymentIdKey] = bson.M{"$in": query.ProcessDeploymentIds}
	}
	if query.BusinessKeys != nil {
		filter[wardenBusinessKeyKey] = bson.M{"$in": query.BusinessKeys}
	}

	ctx, _ := this.getTimeoutContext()
	cursor, err := this.wardenCollection().Find(ctx, filter, opt)
	if err != nil {
		return nil, err
	}
	for cursor.Next(ctx) {
		element := model.WardenInfo{}
		err = cursor.Decode(&element)
		if err != nil {
			return nil, err
		}
		result = append(result, element)
	}
	err = cursor.Err()
	return
}
