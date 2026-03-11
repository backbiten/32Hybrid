// Package runnerutil provides helpers shared between the runner agent and
// the control plane's runner-side code paths.
package runnerutil

import (
	"encoding/json"
	"fmt"
	"time"
)

// ExitJSON is the structure written to exit.json and uploaded to Azure Blob
// at the end of every run.  Both the runner (writer) and control plane
// (reader via Blob) use this type.
type ExitJSON struct {
	RunID      string    `json:"run_id"`
	ExitCode   int       `json:"exit_code"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Error      string    `json:"error,omitempty"`
}

// Marshal serialises an ExitJSON value to compact JSON bytes.
func (e *ExitJSON) Marshal() ([]byte, error) {
	b, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("runnerutil: marshal exit.json: %w", err)
	}
	return b, nil
}

// Parse deserialises compact or pretty JSON bytes into an ExitJSON value.
func Parse(b []byte) (*ExitJSON, error) {
	var e ExitJSON
	if err := json.Unmarshal(b, &e); err != nil {
		return nil, fmt.Errorf("runnerutil: parse exit.json: %w", err)
	}
	return &e, nil
}
