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

package drainevent

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-node-termination-handler/pkg/ec2metadata"
)

const (
	// ACPIPowerBTNKind is a const to define an ACPI Power BTN kind of drainable event
	ACPIPowerBTNKind = "ACPI_POWER_BTN"
	acpiSocketPath   = "/var/run/acpid.socket"
	maxInt           = int(^uint(0) >> 1)
)

// MonitorForACPIEvents monitors acpid socket for power button events to trigger a drain event
func MonitorForACPIEvents(drainChan chan<- DrainEvent, _ chan<- DrainEvent, _ *ec2metadata.EC2MetadataService) error {
	if _, err := os.Stat(acpiSocketPath); err != nil {
		return fmt.Errorf("ACPID Socket does not exist: %w", err)
	}
	conn, err := net.Dial("unix", acpiSocketPath)
	if err != nil {
		return fmt.Errorf("Could not open socket to acpid: %w", err)
	}
	defer conn.Close()
	log.Println("Listening to acpid socket for possible shutdown events")
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf[:])
		if err == io.EOF {
			continue
		}
		if err != nil {
			return fmt.Errorf("ACPI socket read encountered an error: %w", err)
		}
		log.Printf("Read socket: %s\n", string(buf[0:n]))
		if strings.Contains(string(buf[0:n]), "PBTN") {
			log.Printf("ACPI event PBTN triggered drain and cordon!\n")
			randomID := rand.Intn(maxInt)
			drainChan <- DrainEvent{
				EventID:     fmt.Sprintf("acpi-pbtn-%d", randomID),
				Kind:        ACPIPowerBTNKind,
				StartTime:   time.Now().UTC(),
				Description: "ACPI Power Button event received.",
			}
		}
	}
}
