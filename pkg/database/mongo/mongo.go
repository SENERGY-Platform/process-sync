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
	"context"
	"errors"
	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"log"
	"reflect"
	"runtime/debug"
	"strings"
	"time"
)

type Mongo struct {
	config configuration.Config
	client *mongo.Client
}

var CreateCollections = []func(db *Mongo) error{}

func New(conf configuration.Config) (*Mongo, error) {
	db := &Mongo{config: conf}
	ctx, _ := db.getTimeoutContext()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(conf.MongoUrl))
	if err != nil {
		return nil, err
	}
	db.client = client
	for _, creators := range CreateCollections {
		err = creators(db)
		if err != nil {
			client.Disconnect(context.Background())
			return nil, err
		}
	}
	return db, nil
}

func (this *Mongo) ensureIndex(collection *mongo.Collection, indexname string, indexKey string, asc bool, unique bool) error {
	ctx, _ := this.getTimeoutContext()
	var direction int32 = -1
	if asc {
		direction = 1
	}
	_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bsonx.Doc{{indexKey, bsonx.Int32(direction)}},
		Options: options.Index().SetName(indexname).SetUnique(unique),
	})
	return err
}

func (this *Mongo) ensureTextIndex(collection *mongo.Collection, indexname string, indexKey string) error {
	ctx, _ := this.getTimeoutContext()
	_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bsonx.Doc{{indexKey, bsonx.String("text")}},
		Options: options.Index().SetName(indexname),
	})
	return err
}

func (this *Mongo) ensureCompoundIndex(collection *mongo.Collection, indexname string, asc bool, unique bool, indexKeys ...string) error {
	ctx, _ := this.getTimeoutContext()
	var direction int32 = -1
	if asc {
		direction = 1
	}
	keys := []bsonx.Elem{}
	for _, key := range indexKeys {
		keys = append(keys, bsonx.Elem{Key: key, Value: bsonx.Int32(direction)})
	}
	_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bsonx.Doc(keys),
		Options: options.Index().SetName(indexname).SetUnique(unique),
	})
	return err
}

func (this *Mongo) Disconnect() {
	log.Println(this.client.Disconnect(context.Background()))
}

func (this *Mongo) getTimeoutContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

func getBsonFieldName(obj interface{}, fieldName string) (bsonName string, err error) {
	field, found := reflect.TypeOf(obj).FieldByName(fieldName)
	if !found {
		return "", errors.New("field '" + fieldName + "' not found")
	}
	tags, err := bsoncodec.DefaultStructTagParser.ParseStructTags(field)
	return tags.Name, err
}

func getBsonFieldPath(obj interface{}, path string) (bsonPath string, err error) {
	t := reflect.TypeOf(obj)
	pathParts := strings.Split(path, ".")
	bsonPathParts := []string{}
	for _, name := range pathParts {
		field, found := t.FieldByName(name)
		if !found {
			return "", errors.New("field path '" + path + "' not found at '" + name + "'")
		}
		tags, err := bsoncodec.DefaultStructTagParser.ParseStructTags(field)
		if err != nil {
			return bsonPath, err
		}
		bsonPathParts = append(bsonPathParts, tags.Name)
		t = field.Type
	}
	bsonPath = strings.Join(bsonPathParts, ".")
	return
}

func getFiledList(t reflect.Type) (result []reflect.StructField) {
	num := t.NumField()
	for i := 0; i < num; i++ {
		result = append(result, t.Field(i))
	}
	return
}

type KeyMapping struct {
	FieldName string
	Key       *string
}

type IndexDesc struct {
	Name        string
	Unique      bool
	Asc         bool
	Keys        []*string
	IsTextIndex bool
}

func prepareCollection(mongoCollectionName func(config configuration.Config) string, structure interface{}, keyMappings []KeyMapping, indexes []IndexDesc) {
	for _, mapping := range keyMappings {
		var err error
		*(mapping.Key), err = getBsonFieldPath(structure, mapping.FieldName)
		if err != nil {
			debug.PrintStack()
			log.Fatal(err)
		}
	}
	CreateCollections = append(CreateCollections, func(db *Mongo) error {
		collectionName := mongoCollectionName(db.config)
		collection := db.client.Database(db.config.MongoTable).Collection(collectionName)
		for _, index := range indexes {
			keys := []string{}
			for _, key := range index.Keys {
				if key == nil {
					return errors.New("key is nil pointer")
				}
				keys = append(keys, *key)
			}
			log.Println("ensure index for", collectionName, index.Name, keys)
			if len(keys) == 1 {
				var err error
				if index.IsTextIndex {
					err = db.ensureTextIndex(collection, index.Name, keys[0])
				} else {
					err = db.ensureIndex(collection, index.Name, keys[0], index.Asc, index.Unique)
				}
				if err != nil {
					return err
				}
			}
			if len(keys) > 1 {
				err := db.ensureCompoundIndex(collection, index.Name, index.Asc, index.Unique, keys...)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}
