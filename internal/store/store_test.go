package store_test

import (
	"fmt"
	"testing"
	"time"

	commonv1 "github.com/backbiten/32Hybrid/gen/common/v1"
	"github.com/backbiten/32Hybrid/internal/store"
)

func TestMemStore_PutGet(t *testing.T) {
	s := store.New()
	rec := &store.RunRecord{
		RunID:       "run-1",
		State:       commonv1.RunState_RUN_STATE_PENDING,
		SubmittedAt: time.Now(),
	}
	s.Put(rec)

	got, err := s.Get("run-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.RunID != rec.RunID {
		t.Errorf("got RunID %q, want %q", got.RunID, rec.RunID)
	}
}

func TestMemStore_GetNotFound(t *testing.T) {
	s := store.New()
	_, err := s.Get("nonexistent")
	if err == nil {
		t.Error("expected error for missing run, got nil")
	}
}

func TestMemStore_List(t *testing.T) {
	s := store.New()
	for i := range 3 {
		s.Put(&store.RunRecord{RunID: fmt.Sprintf("%c", 'a'+i)})
	}
	if got := len(s.List()); got != 3 {
		t.Errorf("List: got %d items, want 3", got)
	}
}

func TestToProto(t *testing.T) {
	now := time.Now()
	rec := &store.RunRecord{
		RunID:            "r1",
		State:            commonv1.RunState_RUN_STATE_RUNNING,
		SubmittedAt:      now,
		StartedAt:        now,
		FinishedAt:       now,
		ExitCode:         0,
		ExitJSONBlobPath: "r1/exit.json",
	}
	proto := store.ToProto(rec)
	if proto.RunId != "r1" {
		t.Errorf("RunId: got %q, want %q", proto.RunId, "r1")
	}
	if proto.ExitJsonBlobPath != "r1/exit.json" {
		t.Errorf("ExitJsonBlobPath mismatch")
	}
}
