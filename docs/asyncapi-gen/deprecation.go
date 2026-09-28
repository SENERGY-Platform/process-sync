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

// The comment reading below is taken from smart-service-module-worker-lib,
// pkg/middleware/gen/jsdoc/deprecation.go, where the reasons for reading the go comment
// instead of inferring deprecations from field names are recorded in
// docs/deprecation-is-read-from-the-go-comment.md. What differs is how the declaring struct
// of a field is found, because jsonschema-go reports fields differently than that generator.

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/swaggest/jsonschema-go"
)

// deprecationMarker is the word a go comment opens with to mark the field below or beside it
// as deprecated, the convention the platform models are written with:
//
//	AspectId  string   `json:"aspect_id"` //deprecated: please use AspectIds
//	AspectIds []string `json:"aspect_ids,omitempty"`
const deprecationMarker = "deprecated"

// markDeprecatedProperties marks every property of a struct schema whose go field is marked
// deprecated in the source that declares it, as "deprecated": true, the same way jsonschema-go
// renders a `deprecated:"true"` tag. The note becomes the description unless one is set.
//
// It runs as a type interceptor rather than a property interceptor: after the properties of a
// struct are reflected, the struct and its schema are at hand together. A property interceptor
// sees only the field, and for a property that becomes a $ref it does not see the parent.
func markDeprecatedProperties(v reflect.Value, schema *jsonschema.Schema) (bool, error) {
	if len(schema.Properties) == 0 || !v.IsValid() {
		return false, nil
	}
	t := v.Type()
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false, nil
	}
	for name, property := range schema.Properties {
		if property.TypeObject == nil {
			continue
		}
		declaring, fieldName, found := fieldOfProperty(t, name)
		if !found {
			continue
		}
		note, isDeprecated := deprecationOf(declaring, fieldName)
		if !isDeprecated {
			continue
		}
		property.TypeObject.WithExtraPropertiesItem("deprecated", true)
		if property.TypeObject.Description == nil {
			description := deprecationMarker
			if note != "" {
				description += ": " + note
			}
			property.TypeObject.WithDescription(description)
		}
		schema.Properties[name] = property
	}
	return false, nil
}

// fieldOfProperty finds the go field a json property is reflected from, and the struct that
// declares it. The fields of an untagged embedded struct are properties of the outer one, the
// way encoding/json and jsonschema-go both treat them, but they are declared by the embedded
// struct, and that is where their comment is.
func fieldOfProperty(t reflect.Type, property string) (declaring reflect.Type, fieldName string, found bool) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("json")
		if tag == "-" || !field.IsExported() && !field.Anonymous {
			continue
		}
		if tag == "" && field.Anonymous {
			embedded := field.Type
			for embedded.Kind() == reflect.Pointer {
				embedded = embedded.Elem()
			}
			if embedded.Kind() == reflect.Struct {
				if declaring, fieldName, found = fieldOfProperty(embedded, property); found {
					return declaring, fieldName, true
				}
			}
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name == "" {
			name = field.Name
		}
		if name == property {
			return t, field.Name, true
		}
	}
	return nil, "", false
}

// deprecationOf answers whether the go source marks a struct field as deprecated, and with
// what note. The declaring type is needed rather than the type the message is built from,
// because a promoted field is declared by the embedded struct.
func deprecationOf(declaring reflect.Type, fieldName string) (note string, isDeprecated bool) {
	if declaring == nil || declaring.PkgPath() == "" || declaring.Name() == "" {
		return "", false //an anonymous struct has no declaration to look up
	}
	note, isDeprecated = notesOfPackage(declaring.PkgPath())[declaring.Name()+"."+fieldName]
	return note, isDeprecated
}

// notes maps "<type name>.<field name>" to the deprecation note of that field, per package.
// A package is parsed once, however many of its types are asked about.
var notes = struct {
	sync.Mutex
	byPackage map[string]map[string]string
}{byPackage: map[string]map[string]string{}}

func notesOfPackage(pkgPath string) map[string]string {
	notes.Lock()
	defer notes.Unlock()
	if known, ok := notes.byPackage[pkgPath]; ok {
		return known
	}
	result := map[string]string{}
	//also remembers an empty result: a package whose source is not readable stays that way
	notes.byPackage[pkgPath] = result

	pkg, err := build.Import(pkgPath, ".", build.FindOnly)
	if err != nil {
		return result
	}
	parsed, err := parser.ParseDir(token.NewFileSet(), pkg.Dir, nil, parser.ParseComments)
	if err != nil {
		return result
	}
	for _, astPackage := range parsed {
		for _, file := range astPackage.Files {
			collectDeprecations(file, result)
		}
	}
	return result
}

func collectDeprecations(file *ast.File, result map[string]string) {
	ast.Inspect(file, func(node ast.Node) bool {
		typeSpec, isTypeSpec := node.(*ast.TypeSpec)
		if !isTypeSpec {
			return true
		}
		structType, isStruct := typeSpec.Type.(*ast.StructType)
		if !isStruct {
			return true
		}
		for _, field := range structType.Fields.List {
			note, isDeprecated := deprecationComment(field)
			if !isDeprecated {
				continue
			}
			for _, name := range field.Names {
				result[typeSpec.Name.Name+"."+name.Name] = note
			}
		}
		return true
	})
}

// deprecationComment reads the deprecation out of the comment above the field and out of the
// one beside it, the two places a field can carry one.
func deprecationComment(field *ast.Field) (note string, isDeprecated bool) {
	for _, comment := range []string{field.Doc.Text(), field.Comment.Text()} {
		for _, line := range strings.Split(comment, "\n") {
			if note, isDeprecated := deprecationOfLine(line); isDeprecated {
				return note, true
			}
		}
	}
	return "", false
}

// deprecationOfLine reads one comment line. The line marks the field when it opens with the
// word "deprecated" - alone, or followed by a note that the usual ":" or "-" may introduce.
// The word has to end there: a line about a "deprecatedName" says nothing about this field.
func deprecationOfLine(line string) (note string, isDeprecated bool) {
	line = strings.TrimSpace(line)
	if len(line) < len(deprecationMarker) {
		return "", false
	}
	if !strings.EqualFold(line[:len(deprecationMarker)], deprecationMarker) {
		return "", false
	}
	rest := line[len(deprecationMarker):]
	if rest != "" {
		if first, _ := utf8.DecodeRuneInString(rest); unicode.IsLetter(first) || unicode.IsDigit(first) || first == '_' {
			return "", false
		}
	}
	return strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(rest), ":,-")), true
}
