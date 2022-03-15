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
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

type ValidationKind = bool

const (
	ValidatePublish ValidationKind = true
	ValidateRequest ValidationKind = false
)

//strict for cqrs; else for user
func (this Deployment) Validate(kind ValidationKind) (err error) {
	if this.Version != CurrentVersion {
		return errors.New("unexpected deployment version")
	}
	if this.Id == "" {
		return errors.New("missing deployment id")
	}
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
	if kind == ValidatePublish && this.Diagram.XmlDeployed == "" {
		return errors.New("missing deployment xml")
	}
	for _, element := range this.Elements {
		err = element.Validate(kind)
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

func (this Element) Validate(kind ValidationKind) error {
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
	if this.TimeEvent != nil {
		if this.TimeEvent.Type == "timeDuration" {
			if this.TimeEvent.Time == "" {
				return errors.New("missing time event duration")
			}
			d, err := ParseIsoDuration(this.TimeEvent.Time)
			if err != nil {
				return errors.New("time-event: " + err.Error())
			}
			if d.Seconds() < 5 {
				return errors.New("time-event duration below 5s")
			}
		}
	}
	if this.MessageEvent != nil {
		if this.MessageEvent.Selection.SelectedDeviceGroupId != nil {
			if *this.MessageEvent.Selection.SelectedDeviceGroupId == "" {
				return errors.New("invalid device-group selection in event")
			}
		} else if this.MessageEvent.Selection.SelectedImportId != nil {
			if *this.MessageEvent.Selection.SelectedImportId == "" {
				return errors.New("invalid import selection in event")
			}
			if this.MessageEvent.Selection.SelectedPath == nil || this.MessageEvent.Selection.SelectedPath.Path == "" {
				return errors.New("missing selected_path, but import selected in event")
			}
			if this.MessageEvent.Selection.SelectedPath.CharacteristicId == "" {
				return errors.New("missing selected_path.characteristicId, but import selected in event")
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

func ParseIsoDuration(str string) (result time.Duration, err error) {
	durationRegex := regexp.MustCompile(`P(?P<years>\d+Y)?(?P<months>\d+M)?(?P<days>\d+D)?T?(?P<hours>\d+H)?(?P<minutes>\d+M)?(?P<seconds>\d+S)?`)
	matches := durationRegex.FindStringSubmatch(str)
	if len(matches) < 7 {
		return result, errors.New("invalid iso duration")
	}
	years := ParseIsoDurationInt64(matches[1])
	months := ParseIsoDurationInt64(matches[2])
	days := ParseIsoDurationInt64(matches[3])
	hours := ParseIsoDurationInt64(matches[4])
	minutes := ParseIsoDurationInt64(matches[5])
	seconds := ParseIsoDurationInt64(matches[6])

	hour := int64(time.Hour)
	minute := int64(time.Minute)
	second := int64(time.Second)
	return time.Duration(years*24*365*hour + months*30*24*hour + days*24*hour + hours*hour + minutes*minute + seconds*second), nil
}

func ParseIsoDurationInt64(value string) int64 {
	if len(value) == 0 {
		return 0
	}
	parsed, err := strconv.Atoi(value[:len(value)-1])
	if err != nil {
		return 0
	}
	return int64(parsed)
}
