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

var deploymentIdKey string
var deploymentNameKey string
var deploymentNetworkIdKey string
var deploymentPlaceholderKey string

func init() {
	prepareCollection(func(config configuration.Config) string {
		return config.MongoDeploymentCollection
	},
		model.Deployment{},
		[]KeyMapping{
			{
				FieldName: "Deployment.Id",
				Key:       &deploymentIdKey,
			},
			{
				FieldName: "Deployment.Name",
				Key:       &deploymentNameKey,
			},
			{
				FieldName: "SyncInfo.NetworkId",
				Key:       &deploymentNetworkIdKey,
			},
			{
				FieldName: "SyncInfo.IsPlaceholder",
				Key:       &deploymentPlaceholderKey,
			},
		},
		[]IndexDesc{
			{
				Name:   "deploymentplaceholderindex",
				Unique: false,
				Asc:    true,
				Keys:   []*string{&deploymentPlaceholderKey},
			},
			{
				Name:   "deploymentnetworkindex",
				Unique: false,
				Asc:    true,
				Keys:   []*string{&deploymentNetworkIdKey},
			},
			{
				Name:   "deploymentcompoundindex",
				Unique: false,
				Asc:    true,
				Keys:   []*string{&deploymentIdKey, &deploymentNetworkIdKey},
			},
		},
	)
}

func (this *Mongo) deploymentCollection() *mongo.Collection {
	return this.client.Database(this.config.MongoTable).Collection(this.config.MongoDeploymentCollection)
}

func (this *Mongo) SaveDeployment(deployment model.Deployment) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.deploymentCollection().ReplaceOne(
		ctx,
		bson.M{
			deploymentIdKey:        deployment.Id,
			deploymentNetworkIdKey: deployment.NetworkId,
		},
		deployment,
		options.Replace().SetUpsert(true))
	return err
}

func (this *Mongo) RemoveDeployment(networkId string, deploymentId string) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.deploymentCollection().DeleteOne(
		ctx,
		bson.M{
			deploymentIdKey:        deploymentId,
			deploymentNetworkIdKey: networkId,
		})
	return err
}

func (this *Mongo) RemovePlaceholderDeployments(networkId string) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.deploymentCollection().DeleteOne(
		ctx,
		bson.M{
			deploymentPlaceholderKey: true,
			deploymentNetworkIdKey:   networkId,
		})
	return err
}

func (this *Mongo) RemoveUnknownDeployments(networkId string, knownIds []string) error {
	ctx, _ := this.getTimeoutContext()
	_, err := this.deploymentCollection().DeleteMany(
		ctx,
		bson.M{
			deploymentIdKey:        bson.M{"$nin": knownIds},
			deploymentNetworkIdKey: networkId,
		})
	return err
}

func (this *Mongo) ReadDeployment(networkId string, deploymentId string) (deployment model.Deployment, err error) {
	ctx, _ := this.getTimeoutContext()
	result := this.deploymentCollection().FindOne(
		ctx,
		bson.M{
			deploymentIdKey:        deploymentId,
			deploymentNetworkIdKey: networkId,
		})
	err = result.Err()
	if err == mongo.ErrNoDocuments {
		return deployment, database.ErrNotFound
	}
	if err != nil {
		return
	}
	err = result.Decode(&deployment)
	if err == mongo.ErrNoDocuments {
		return deployment, database.ErrNotFound
	}
	return deployment, err
}

func (this *Mongo) ListDeployments(networkIds []string, limit int64, offset int64, sort string) (result []model.Deployment, err error) {
	opt := options.Find()
	opt.SetLimit(limit)
	opt.SetSkip(offset)

	parts := strings.Split(sort, ".")
	sortby := definitionIdKey
	switch parts[0] {
	case "id":
		sortby = deploymentIdKey
	case "name":
		sortby = deploymentNameKey
	}
	direction := int32(1)
	if len(parts) > 1 && parts[1] == "desc" {
		direction = int32(-1)
	}
	opt.SetSort(bsonx.Doc{{sortby, bsonx.Int32(direction)}})

	ctx, _ := this.getTimeoutContext()
	cursor, err := this.deploymentCollection().Find(ctx, bson.M{deploymentNetworkIdKey: bson.M{"$in": networkIds}}, opt)
	if err != nil {
		return nil, err
	}
	for cursor.Next(ctx) {
		element := model.Deployment{}
		err = cursor.Decode(&element)
		if err != nil {
			return nil, err
		}
		result = append(result, element)
	}
	err = cursor.Err()
	return
}
