/*
 * Copyright 2026 InfAI (CC SES)
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
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TestStartAuthenticates needs a throwaway server with access control; MONGO_AUTH_TEST_USER
// and MONGO_AUTH_TEST_PASSWORD are root credentials, used to create and remove the test users.
func TestStartAuthenticates(t *testing.T) {
	url, rootUser, rootPassword := os.Getenv("MONGO_AUTH_TEST_URL"), os.Getenv("MONGO_AUTH_TEST_USER"), os.Getenv("MONGO_AUTH_TEST_PASSWORD")
	if testing.Short() || url == "" || rootUser == "" || rootPassword == "" {
		t.Skip("needs MONGO_AUTH_TEST_URL, MONGO_AUTH_TEST_USER and MONGO_AUTH_TEST_PASSWORD, not in -short")
	}
	ctx := context.Background()
	root, err := mongo.Connect(ctx, options.Client().ApplyURI(url).SetAuth(options.Credential{Username: rootUser, Password: rootPassword, AuthSource: "admin"}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = root.Disconnect(ctx) })

	suffix := randomHex(t)
	testDB, otherDB := "process_sync_auth_test_"+suffix, "process_sync_auth_other_"+suffix
	svcUser, svcPassword := "process-sync-test-"+suffix, randomHex(t)
	otherUser, otherPassword := "process-sync-other-"+suffix, randomHex(t)
	createUser(t, root, svcUser, svcPassword, testDB)
	createUser(t, root, otherUser, otherPassword, otherDB)
	passwords := []string{svcPassword, otherPassword, rootPassword}

	config := func(user, password string) configuration.Config {
		return configuration.Config{
			MongoUrl:                          url,
			MongoUser:                         user,
			MongoPassword:                     password,
			MongoAuthSource:                   "admin",
			MongoDatabase:                     testDB,
			MongoProcessDefinitionCollection:  "process_definition",
			MongoDeploymentCollection:         "deployments",
			MongoProcessHistoryCollection:     "histories",
			MongoIncidentCollection:           "incidents",
			MongoProcessInstanceCollection:    "instances",
			MongoDeploymentMetadataCollection: "deployment_metadata",
			MongoLastNetworkContactCollection: "last_network_collection",
			MongoWardenCollection:             "warden",
			MongoDeploymentWardenCollection:   "deployment_warden",
		}
	}

	cases := []struct {
		name, user, password string
		wantErr              string
	}{
		{"correct credentials", svcUser, svcPassword, ""},
		{"no credentials", "", "", "mongo startup check failed: "},
		{"user without rights on the database", otherUser, otherPassword, "mongo startup check failed: "},
		{"wrong password", svcUser, svcPassword + "-wrong", "mongo startup check failed: "},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			db, err := New(config(c.user, c.password))
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				db.Disconnect()
				return
			}
			if err == nil {
				db.Disconnect()
				t.Fatalf("expected an error containing %q", c.wantErr)
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("unexpected error: %v", err)
			}
			for _, pw := range passwords {
				if strings.Contains(err.Error(), pw) {
					t.Error("error text contains a password")
				}
			}
		})
	}
}

// createUser registers the cleanup first, so a partly failed creation is removed as well.
func createUser(t *testing.T, root *mongo.Client, user, password, db string) {
	t.Helper()
	admin := root.Database("admin")
	t.Cleanup(func() {
		ctx := context.Background()
		_ = admin.RunCommand(ctx, bson.D{{Key: "dropUser", Value: user}}).Err()
		_ = root.Database(db).Drop(ctx)
	})
	cmd := bson.D{
		{Key: "createUser", Value: user},
		{Key: "pwd", Value: password},
		{Key: "roles", Value: bson.A{bson.D{{Key: "role", Value: "readWrite"}, {Key: "db", Value: db}}}},
	}
	if err := admin.RunCommand(context.Background(), cmd).Err(); err != nil {
		t.Fatalf("create user: %v", err)
	}
}

func randomHex(t *testing.T) string {
	t.Helper()
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(b)
}
