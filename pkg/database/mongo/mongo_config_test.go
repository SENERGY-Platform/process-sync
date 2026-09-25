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
	"testing"

	"github.com/SENERGY-Platform/process-sync/pkg/configuration"
)

func TestValidateConfig(t *testing.T) {
	cases := []struct {
		name    string
		conf    configuration.Config
		wantErr error
	}{
		{"database set, no user", configuration.Config{MongoDatabase: "sync"}, nil},
		{"empty database", configuration.Config{}, errEmptyDatabase},
		{"empty database with user", configuration.Config{MongoUser: "svc", MongoPassword: "pw"}, errEmptyDatabase},
		{"user without password", configuration.Config{MongoDatabase: "sync", MongoUser: "svc"}, errMissingPassword},
		{"user with password", configuration.Config{MongoDatabase: "sync", MongoUser: "svc", MongoPassword: "pw"}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := validateConfig(c.conf); err != c.wantErr {
				t.Errorf("validateConfig() = %v, want %v", err, c.wantErr)
			}
		})
	}
}

// clientOptions must not apply credentials when MongoUser is empty, since SetAuth on an
// unauthenticated deployment would make the driver attempt authentication anyway.
func TestClientOptionsAppliesCredentialsOnlyWhenUserSet(t *testing.T) {
	opts := clientOptions(configuration.Config{MongoUrl: "mongodb://localhost:27017"})
	if opts.Auth != nil {
		t.Errorf("expected no auth when MongoUser is empty, got %+v", opts.Auth)
	}

	opts = clientOptions(configuration.Config{
		MongoUrl:        "mongodb://localhost:27017",
		MongoUser:       "svc",
		MongoPassword:   "pw",
		MongoAuthSource: "admin",
	})
	if opts.Auth == nil {
		t.Fatal("expected auth to be set")
	}
	if opts.Auth.Username != "svc" || opts.Auth.Password != "pw" || opts.Auth.AuthSource != "admin" {
		t.Errorf("unexpected auth: %+v", opts.Auth)
	}
}
