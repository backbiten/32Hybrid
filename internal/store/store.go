// Package store provides an in-memory run record store for the control plane.
// For MVP a simple mutex-protected map is sufficient; replace with a database
// or durable cache (e.g. Redis) in production.
package store

import (
	"fmt"
	"sync"
	"time"

	cpv1 "github.com/backbiten/32Hybrid/gen/controlplane/v1"
	commonv1 "github.com/backbiten/32Hybrid/gen/common/v1"
)

// RunRecord holds all state for a single submitted run.
type RunRecord struct {
	RunID       string
	State       commonv1.RunState
	SubmittedAt time.Time
	StartedAt   time.Time
	FinishedAt  time.Time
	ExitCode    int32
	ErrorMsg    string
	SessionID   string

	// Blob paths (relative to the runs container root).
	ExitJSONBlobPath string
	StdoutBlobPath   string
	StderrBlobPath   string
}

// MemStore is an in-memory, thread-safe store for run records.
type MemStore struct {
	mu   sync.RWMutex
	runs map[string]*RunRecord
}

// New returns an empty MemStore.
func New() *MemStore {
	return &MemStore{runs: make(map[string]*RunRecord)}
}

// Put inserts or replaces a run record.
func (s *MemStore) Put(r *RunRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[r.RunID] = r
}

// Get retrieves a run record by ID.  Returns an error if not found.
func (s *MemStore) Get(runID string) (*RunRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.runs[runID]
	if !ok {
		return nil, fmt.Errorf("store: run %q not found", runID)
	}
	return r, nil
}

// List returns all run records in insertion order (approximate; map iteration
// is random).  For MVP ordering is not guaranteed.
func (s *MemStore) List() []*RunRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*RunRecord, 0, len(s.runs))
	for _, r := range s.runs {
		out = append(out, r)
	}
	return out
}

// ToProto converts a RunRecord to a GetRunResponse protobuf message.
func ToProto(r *RunRecord) *cpv1.GetRunResponse {
	resp := &cpv1.GetRunResponse{
		RunId:            r.RunID,
		State:            r.State,
		SubmittedAt:      r.SubmittedAt.Unix(),
		ExitCode:         r.ExitCode,
		ErrorMessage:     r.ErrorMsg,
		SessionId:        r.SessionID,
		ExitJsonBlobPath: r.ExitJSONBlobPath,
		StdoutBlobPath:   r.StdoutBlobPath,
		StderrBlobPath:   r.StderrBlobPath,
	}
	if !r.StartedAt.IsZero() {
		resp.StartedAt = r.StartedAt.Unix()
	}
	if !r.FinishedAt.IsZero() {
		resp.FinishedAt = r.FinishedAt.Unix()
	}
	return resp
}
