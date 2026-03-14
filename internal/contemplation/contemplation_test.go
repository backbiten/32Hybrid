package contemplation

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestRunShortDuration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	buf := &bytes.Buffer{}
	opts := Options{
		Duration: 80 * time.Millisecond,
		Tick:     20 * time.Millisecond,
		Writer:   buf,
		Label:    "Synchronizing Neural Root...",
		Concepts: []string{"GDT/IDT", "CR0/CR3"},
	}

	if err := Run(ctx, opts); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Synchronizing Neural Root") {
		t.Fatalf("progress output missing label: %q", out)
	}
	if !strings.Contains(out, "GDT/IDT") && !strings.Contains(out, "CR0/CR3") {
		t.Fatalf("progress output missing concepts: %q", out)
	}
	if !strings.Contains(out, "100%") {
		t.Fatalf("progress output missing completion percent: %q", out)
	}
	if !strings.HasSuffix(out, "\n") {
		t.Fatalf("progress output should end with newline, got: %q", out)
	}
}

func TestDurationFromEnv(t *testing.T) {
	t.Setenv("HYPER32_CONTEMPLATION_SECONDS", "45")
	if got := DurationFromEnv("HYPER32_CONTEMPLATION_SECONDS", time.Minute); got != 45*time.Second {
		t.Fatalf("expected 45s from env, got %v", got)
	}

	t.Setenv("HYPER32_CONTEMPLATION_SECONDS", "2m30s")
	if got := DurationFromEnv("HYPER32_CONTEMPLATION_SECONDS", time.Minute); got != 150*time.Second {
		t.Fatalf("expected 2m30s (150s), got %v", got)
	}

	t.Setenv("HYPER32_CONTEMPLATION_SECONDS", "invalid")
	if got := DurationFromEnv("HYPER32_CONTEMPLATION_SECONDS", 30*time.Second); got != 30*time.Second {
		t.Fatalf("invalid env should return fallback, got %v", got)
	}
}
