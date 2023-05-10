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

package security

import (
	"encoding/json"
	"github.com/SENERGY-Platform/permission-search/lib/client"
	"runtime/debug"
	"strconv"
)

type ListElement struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func (this *Security) List(token string, resource string, limit string, offset string, rights string) (result []ListElement, err error) {
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		debug.PrintStack()
		return result, err
	}
	offsetInt, err := strconv.Atoi(offset)
	if err != nil {
		debug.PrintStack()
		return result, err
	}
	return client.List[[]ListElement](this.permissionsearch, token, resource, client.ListOptions{
		QueryListCommons: client.QueryListCommons{
			Limit:  limitInt,
			Offset: offsetInt,
			Rights: rights,
		},
	})
}

func (this *Security) ListElements(token string, resource string, limit string, offset string, rights string, result interface{}) (err error) {
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		debug.PrintStack()
		return err
	}
	offsetInt, err := strconv.Atoi(offset)
	if err != nil {
		debug.PrintStack()
		return err
	}
	temp, err := client.List[[]ListElement](this.permissionsearch, token, resource, client.ListOptions{
		QueryListCommons: client.QueryListCommons{
			Limit:  limitInt,
			Offset: offsetInt,
			Rights: rights,
		},
	})
	if err != nil {
		return err
	}
	b, err := json.Marshal(temp)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, result)
	if err != nil {
		return err
	}
	return nil
}
