/*
 * Copyright 2020 InfAI (CC SES)
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

package deploymentmodel

import (
	"errors"
	"fmt"
	"github.com/beevik/etree"
	"log"
	"runtime/debug"
	"strings"
)

func (this Deployment) Validate() (err error) {
	if this.Name == "" {
		return errors.New("missing deployment name")
	}
	if this.Diagram.XmlRaw == "" {
		return errors.New("missing deployment xml_raw")
	}
	engineAccess, err := xmlContainsEngineAccess(this.Diagram.XmlRaw)
	if err != nil {
		return err
	}
	if engineAccess {
		return errors.New("process tries to access execution engine")
	}
	if this.Diagram.XmlDeployed == "" {
		return errors.New("missing deployment xml")
	}
	for _, element := range this.Elements {
		err = element.Validate()
		if err != nil {
			return err
		}
	}
	return nil
}

func xmlContainsEngineAccess(xml string) (triesAccess bool, err error) {
	defer func() {
		if r := recover(); r != nil && err == nil {
			log.Printf("%s: %s", r, debug.Stack())
			err = errors.New(fmt.Sprint("Recovered Error: ", r))
		}
	}()
	doc := etree.NewDocument()
	err = doc.ReadFromString(xml)
	if err != nil {
		return true, err
	}
	scripts := []string{}
	for _, script := range doc.FindElements("//camunda:script") {
		scripts = append(scripts, script.Text())
	}
	for _, script := range doc.FindElements("//bpmn:script") {
		scripts = append(scripts, script.Text())
	}
	for _, script := range scripts {
		if strings.Contains(script, "execution.") {
			return true, nil
		}
	}
	return false, nil
}

func (this Element) Validate() error {
	if this.BpmnId == "" {
		return errors.New("missing bpmn element id")
	}
	if this.Task != nil {
		if (this.Task.Selection.SelectedDeviceGroupId == nil || *this.Task.Selection.SelectedDeviceGroupId == "") &&
			(this.Task.Selection.SelectedDeviceId == nil || *this.Task.Selection.SelectedDeviceId == "") {
			return errors.New("missing device/device-group selection in task")
		}
	}
	if this.Task != nil &&
		this.Task.Selection.SelectedDeviceId != nil &&
		*this.Task.Selection.SelectedDeviceId == "" &&
		(this.Task.Selection.SelectedServiceId == nil || *this.Task.Selection.SelectedServiceId == "") {
		return errors.New("missing service selection in task")
	}
	if this.MessageEvent != nil {
		if this.MessageEvent.Selection.SelectedDeviceGroupId != nil {
			if *this.MessageEvent.Selection.SelectedDeviceGroupId == "" {
				return errors.New("invalid device-group selection in event")
			}
		} else {
			if this.MessageEvent.Selection.SelectedDeviceId == nil || *this.MessageEvent.Selection.SelectedDeviceId == "" {
				return errors.New("missing device selection in event")
			}
			if this.MessageEvent.Selection.SelectedServiceId == nil || *this.MessageEvent.Selection.SelectedServiceId == "" {
				return errors.New("missing service selection in event")
			}
		}
	}
	return nil
}
