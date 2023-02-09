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
	"log"
	"runtime/debug"
)

var metadataDeploymentIdKey string
var metadataCamundaDeploymentIdKey string
var metadataNetworkIdKey string
var metadataEventGroupIdKey string

func init() {
	var err error
	metadataEventGroupIdKey, err = getBsonFieldPath(model.DeploymentMetadata{}, "Metadata.DeploymentModel.EventDescriptions")
	if err != nil {
		debug.PrintStack()
		log.Fatal(err)
	}
	metadataEventGroupIdKey = metadataEventGroupIdKey + ".device_group_id"

	prepareCollection(func(config configuration.Config) string {
		return config.MongoDeploymentMetadataCollection
	},
		model.DeploymentMetadata{},
		[]KeyMapping{
			{
				FieldName: "Metadata.DeploymentModel.Id",
				Key:       &metadataDeploymentIdKey,
			},
			{
				FieldName: "Metadata.CamundaDeploymentId",
				Key:       &metadataCamundaDeploymentIdKey,
			},
			{
				FieldName: "SyncInfo.NetworkId",
				Key:       &metadataNetworkIdKey,
			},
		},
		[]IndexDesc{
			{
				Name:   "deploymen_metadata_idindex",
				Unique: false,
				Asc:    true,
				Keys:   []*string{&metadataCamundaDeploymentIdKey, &metadataNetworkIdKey},
			},
			{
				Name:   "deploymen_metadata_deployment_network_id_index",
				Unique: false,
				Asc:    true,
				Keys:   []*string{&metadataDeploymentIdKey, &metadataNetworkIdKey},
			},
			{
				Name:   "deploymen_metadata_deployment_id_index",
				Unique: false,
				Asc:    true,
				Keys:   []*string{&metadataDeploymentIdKey},
			},
			{
				Name:   "deploymen_metadata_event_device_group_id_index",
				Unique: false,
				Asc:    true,
				Keys:   []*string{&metadataEventGroupIdKey},
			},
		},
	)
}

func (this *Mongo) deploymentMetadataCollection() *mongo.Collection {
	return this.client.Database(this.config.MongoTable).Collection(this.config.MongoDeploymentMetadataCollection)
}

func (this *Mongo) SaveDeploymentMetadata(metadata model.DeploymentMetadata) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.deploymentMetadataCollection().ReplaceOne(
		ctx,
		bson.M{
			metadataCamundaDeploymentIdKey: metadata.CamundaDeploymentId,
			metadataNetworkIdKey:           metadata.NetworkId,
		},
		metadata,
		options.Replace().SetUpsert(true))
	return err
}

func (this *Mongo) RemoveUnknownDeploymentMetadata(networkId string, knownIds []string) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.deploymentMetadataCollection().DeleteMany(
		ctx,
		bson.M{
			metadataCamundaDeploymentIdKey: bson.M{"$nin": knownIds},
			metadataNetworkIdKey:           networkId,
		})
	return err
}

func (this *Mongo) ReadDeploymentMetadata(networkId string, deploymentId string) (metadata model.DeploymentMetadata, err error) {
	ctx, _ := this.getTimeoutContext()
	result := this.deploymentMetadataCollection().FindOne(
		ctx,
		bson.M{
			metadataCamundaDeploymentIdKey: deploymentId,
			metadataNetworkIdKey:           networkId,
		})
	err = result.Err()
	if err == mongo.ErrNoDocuments {
		return metadata, database.ErrNotFound
	}
	if err != nil {
		return
	}
	err = result.Decode(&metadata)
	if err == mongo.ErrNoDocuments {
		return metadata, database.ErrNotFound
	}
	return metadata, err
}

func (this *Mongo) ListDeploymentMetadata(query model.MetadataQuery) (result []model.DeploymentMetadata, err error) {
	ctx, _ := this.getTimeoutContext()
	filter := bson.M{}
	if query.DeploymentId != nil {
		filter[metadataDeploymentIdKey] = *query.DeploymentId
	}
	if query.CamundaDeploymentId != nil {
		filter[metadataCamundaDeploymentIdKey] = *query.CamundaDeploymentId
	}
	if query.NetworkId != nil {
		filter[metadataNetworkIdKey] = *query.NetworkId
	}
	cursor, err := this.deploymentMetadataCollection().Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	for cursor.Next(ctx) {
		element := model.DeploymentMetadata{}
		err = cursor.Decode(&element)
		if err != nil {
			return nil, err
		}
		result = append(result, element)
	}
	err = cursor.Err()
	log.Printf("DEBUG: result = %#v\n %v\n", result, err)
	return
}

func (this *Mongo) ListDeploymentMetadataByEventDeviceGroupId(deviceGroupId string) (result []model.DeploymentMetadata, err error) {
	ctx, _ := this.getTimeoutContext()
	filter := bson.M{metadataEventGroupIdKey: deviceGroupId}
	cursor, err := this.deploymentMetadataCollection().Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	for cursor.Next(ctx) {
		element := model.DeploymentMetadata{}
		err = cursor.Decode(&element)
		if err != nil {
			return nil, err
		}
		result = append(result, element)
	}
	err = cursor.Err()
	return
}

func (this *Mongo) RemoveDeploymentMetadata(networkId string, deploymentId string) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.deploymentMetadataCollection().DeleteMany(
		ctx,
		bson.M{
			metadataCamundaDeploymentIdKey: deploymentId,
			metadataNetworkIdKey:           networkId,
		})
	return err
}

func (this *Mongo) GetDeploymentMetadataOfDeploymentIdList(networkId string, deploymentIds []string) (result map[string]model.DeploymentMetadata, err error) {
	result = map[string]model.DeploymentMetadata{}
	ctx, _ := this.getTimeoutContext()
	cursor, err := this.deploymentMetadataCollection().Find(ctx, bson.M{
		metadataNetworkIdKey:           networkId,
		metadataCamundaDeploymentIdKey: bson.M{"$in": deploymentIds}})
	if err != nil {
		return nil, err
	}
	for cursor.Next(ctx) {
		element := model.DeploymentMetadata{}
		err = cursor.Decode(&element)
		if err != nil {
			return nil, err
		}
		result[element.CamundaDeploymentId] = element
	}
	err = cursor.Err()
	return
}
