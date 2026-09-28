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

package main

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/SENERGY-Platform/process-sync/pkg/model"
	"github.com/swaggest/go-asyncapi/reflector/asyncapi-2.4.0"
	"github.com/swaggest/go-asyncapi/spec-2.4.0"
	"github.com/swaggest/jsonschema-go"
)

type deprecationTestNode struct {
	Id string `json:"id"`
}

type deprecationTestInner struct {
	Old string `json:"old"` //deprecated: please use New
	New string `json:"new"`
}

type deprecationTestMessage struct {
	deprecationTestInner
	// deprecated
	Bare        string                `json:"bare"`
	Node        deprecationTestNode   `json:"node"` //deprecated: use Nodes
	Nodes       []deprecationTestNode `json:"nodes"`
	Named       string                `json:"named"` //deprecatedName is not a deprecation
	Current     string                `json:"current"`
	WithoutName string                `json:",omitempty"` //deprecated
}

func TestDeprecationOfLine(t *testing.T) {
	cases := []struct {
		line         string
		note         string
		isDeprecated bool
	}{
		{line: "deprecated: please use AspectIds", note: "please use AspectIds", isDeprecated: true},
		{line: "  Deprecated - use Nodes", note: "use Nodes", isDeprecated: true},
		{line: "deprecated", note: "", isDeprecated: true},
		{line: "deprecatedName is not a deprecation", isDeprecated: false},
		{line: "this field is deprecated", isDeprecated: false},
	}
	for _, c := range cases {
		note, isDeprecated := deprecationOfLine(c.line)
		if note != c.note || isDeprecated != c.isDeprecated {
			t.Errorf("%#v: got %#v %v, expected %#v %v", c.line, note, isDeprecated, c.note, c.isDeprecated)
		}
	}
}

func TestFieldOfProperty(t *testing.T) {
	message := reflect.TypeOf(deprecationTestMessage{})
	inner := reflect.TypeOf(deprecationTestInner{})
	cases := []struct {
		property  string
		declaring reflect.Type
		fieldName string
	}{
		{property: "old", declaring: inner, fieldName: "Old"},
		{property: "bare", declaring: message, fieldName: "Bare"},
		{property: "WithoutName", declaring: message, fieldName: "WithoutName"},
	}
	for _, c := range cases {
		declaring, fieldName, found := fieldOfProperty(message, c.property)
		if !found || declaring != c.declaring || fieldName != c.fieldName {
			t.Errorf("%v: got %v %v %v", c.property, declaring, fieldName, found)
		}
	}
	if _, _, found := fieldOfProperty(message, "missing"); found {
		t.Error("found a field for an unknown property")
	}
}

func TestMarkDeprecatedProperties(t *testing.T) {
	properties := reflectProperties(t, new(deprecationTestMessage), "AsyncapiGenDeprecationTestMessage")

	expected := map[string]string{
		"old":         "deprecated: please use New",
		"bare":        "deprecated",
		"node":        "deprecated: use Nodes",
		"WithoutName": "deprecated",
	}
	for name, property := range properties {
		description, isDeprecated := expected[name]
		if property["deprecated"] != nil && !isDeprecated {
			t.Errorf("%v: unexpected deprecation %#v", name, property)
			continue
		}
		if !isDeprecated {
			continue
		}
		if property["deprecated"] != true {
			t.Errorf("%v: not marked deprecated %#v", name, property)
		}
		if property["description"] != description {
			t.Errorf("%v: description %#v, expected %#v", name, property["description"], description)
		}
	}
	if _, ok := properties["node"]["$ref"]; !ok {
		t.Errorf("node is expected as a $ref, which is the case a property interceptor misses: %#v", properties["node"])
	}
}

// TestMarkDeprecatedEventDescAspectId covers the message this generator documents and the mgw
// reads: the event descriptions of a fog deployment name their aspects as a list, and the
// single aspect id is kept as a deprecated alias.
func TestMarkDeprecatedEventDescAspectId(t *testing.T) {
	properties := reflectProperties(t, new(model.DeploymentWithEventDesc), "ModelEventDesc")
	if properties["aspect_id"]["deprecated"] != true {
		t.Errorf("aspect_id not marked deprecated: %#v", properties["aspect_id"])
	}
	if properties["aspect_ids"]["deprecated"] != nil {
		t.Errorf("aspect_ids marked deprecated: %#v", properties["aspect_ids"])
	}
}

// reflectProperties reflects sample the way main does and returns the properties of the
// component schema definition.
func reflectProperties(t *testing.T, sample interface{}, definition string) map[string]map[string]interface{} {
	t.Helper()
	reflector := asyncapi.Reflector{Schema: &spec.AsyncAPI{}}
	reflector.DefaultOptions = append(reflector.DefaultOptions, jsonschema.InterceptType(markDeprecatedProperties))
	err := reflector.AddChannel(asyncapi.ChannelInfo{
		Name:    "test",
		Publish: &asyncapi.MessageSample{MessageSample: sample},
	})
	if err != nil {
		t.Fatal(err)
	}
	buff, err := reflector.Schema.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	doc := struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]map[string]interface{} `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}{}
	err = json.Unmarshal(buff, &doc)
	if err != nil {
		t.Fatal(err)
	}
	schema, ok := doc.Components.Schemas[definition]
	if !ok {
		keys := []string{}
		for key := range doc.Components.Schemas {
			keys = append(keys, key)
		}
		t.Fatalf("no schema %v in %v", definition, keys)
	}
	return schema.Properties
}
