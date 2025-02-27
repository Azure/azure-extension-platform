// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package status

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// extension needn't write status files for other operations
	EnableStatus  = "Enable"
	UpdateStatus  = "Update"
	DisableStatus = "Disable"
)

const (
	ErrorClarificationSubStatusName = "ErrorClarification"
)

type CmdFunc func() (message string, err error)

type Cmd struct {
	f                  CmdFunc // associated function
	name               string  // human readable string
	shouldReportStatus bool    // determines if running this should log to a .status file
	failExitCode       int     // exitCode to use when commands fail
}

// StatusReport contains one or more status items and is the parent object
type StatusReport []StatusItem

// StatusItem is used to serialize an individual part of the status read by the server
type StatusItem struct {
	Version      int    `json:"version"`
	TimestampUTC string `json:"timestampUTC"`
	Status       Status `json:"status"`
}

type ErrorClarification struct {
	Code    int
	Message string
}

type StatusType string

const (
	// StatusTransitioning indicates the operation has begun but not yet completed
	StatusTransitioning StatusType = "transitioning"

	// StatusError indicates the operation failed
	StatusError StatusType = "error"

	// StatusSuccess indicates the operation succeeded
	StatusSuccess StatusType = "success"
)

// Status is used for serializing status in a manner the server understands
type Status struct {
	Operation        string           `json:"operation"`
	Status           StatusType       `json:"status"`
	FormattedMessage FormattedMessage `json:"formattedMessage"`
	Substatuses      []Substatus      `json:"substatus"`
}

type Substatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Code   int    `json:"code"`
}

// FormattedMessage is a struct used for serializing status
type FormattedMessage struct {
	Lang    string `json:"lang"`
	Message string `json:"message"`
}

// New creates a new Status instance
func New(t StatusType, operation string, message string) StatusReport {
	return []StatusItem{
		{
			Version:      1, // this is the protocol version do not change unless you are sure
			TimestampUTC: time.Now().UTC().Format(time.RFC3339),
			Status: Status{
				Operation: operation,
				Status:    t,
				FormattedMessage: FormattedMessage{
					Lang:    "en",
					Message: message},
			},
		},
	}
}

func NewError(operation string, ec ErrorClarification) StatusReport {
	return []StatusItem{
		{
			Version:      1, // this is the protocol version do not change unless you are sure
			TimestampUTC: time.Now().UTC().Format(time.RFC3339),
			Status: Status{
				Operation: operation,
				Status:    StatusError,
				FormattedMessage: FormattedMessage{
					Lang:    "en",
					Message: ec.Message},
				Substatuses: []Substatus{
					{
						Name:   ErrorClarificationSubStatusName,
						Status: string(StatusError),
						Code:   ec.Code,
					},
				},
			},
		},
	}
}

func (r StatusReport) marshal() ([]byte, error) {
	return json.MarshalIndent(r, "", "\t")
}

// Save persists the status message to the specified status folder using the
// sequence number. The operation consists of writing to a temporary file in the
// same folder and moving it to the final destination for atomicity.
func (r StatusReport) Save(statusFolder string, seqNo uint) error {
	fn := fmt.Sprintf("%d.status", seqNo)
	path := filepath.Join(statusFolder, fn)
	tmpFile, err := os.CreateTemp(statusFolder, fn)
	if err != nil {
		return fmt.Errorf("status: failed to create temporary file: %v", err)
	}
	tmpFile.Close()

	b, err := r.marshal()
	if err != nil {
		return fmt.Errorf("status: failed to marshal into json: %v", err)
	}

	if err := os.WriteFile(tmpFile.Name(), b, 0644); err != nil {
		return fmt.Errorf("status: failed to path=%s error=%v", tmpFile.Name(), err)
	}

	if err := os.Rename(tmpFile.Name(), path); err != nil {
		return fmt.Errorf("status: failed to move to path=%s error=%v", path, err)
	}

	return nil
}

type StatusMessageFormatter func(operationName string, t StatusType, msg string) string

// StatusMsg creates the reported status message based on the provided operation
// type and the given message string.
//
// A message will be generated for empty string. For error status, pass the
// error message.
func StatusMsg(operationName string, t StatusType, msg string) string {
	s := operationName
	switch t {
	case StatusSuccess:
		s += " succeeded"
	case StatusTransitioning:
		s += " in progress"
	case StatusError:
		s += " failed"
	}

	if msg != "" {
		// append the original
		s += ": " + msg
	}

	return s
}
