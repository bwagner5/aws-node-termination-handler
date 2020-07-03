// Copyright 2016-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package monitor

import (
	"time"

	"github.com/aws/aws-node-termination-handler/pkg/node"
)

<<<<<<< HEAD:pkg/monitor/types.go
type DrainTask func(InterruptionEvent, node.Node) error
=======
type drainTask func(InterruptionEvent, node.Node) error
>>>>>>> this works to drain nodes from a central pods or group of pods:pkg/interruptionevent/interruption-event.go

// InterruptionEvent gives more context of the interruption event
type InterruptionEvent struct {
	EventID       string
	Kind          string
	Description   string
	State         string
	NodeName      string
	StartTime     time.Time
	EndTime       time.Time
	Drained       bool
<<<<<<< HEAD:pkg/monitor/types.go
	PreDrainTask  DrainTask `json:"-"`
	PostDrainTask DrainTask `json:"-"`
=======
	PreDrainTask  drainTask `json:"-"`
	PostDrainTask drainTask `json:"-"`
>>>>>>> this works to drain nodes from a central pods or group of pods:pkg/interruptionevent/interruption-event.go
}

// TimeUntilEvent returns the duration until the event start time
func (e *InterruptionEvent) TimeUntilEvent() time.Duration {
	return e.StartTime.Sub(time.Now())
}

// Monitor is an interface which can be implemented for various sources of interruption events
type Monitor interface {
	Monitor() error
	Kind() string
}
