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
)

var metadataCamundaDeploymentIdKey string
var metadataNetworkIdKey string

func init() {
	prepareCollection(func(config configuration.Config) string {
		return config.MongoDeploymentMetadataCollection
	},
		model.DeploymentMetadata{},
		[]KeyMapping{
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
	_, err := this.deploymentCollection().DeleteMany(
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
