package contemplation_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/backbiten/32Hybrid/internal/contemplation"
)

// TestConceptForSecond verifies that every second in [0, 900) is covered by a
// named i386 concept and that the correct concept is returned for known
// boundary seconds.
func TestConceptForSecond(t *testing.T) {
	cases := []struct {
		sec     int
		wantIn  string
	}{
		{0, "GDT"},
		{179, "GDT"},
		{180, "IDT"},
		{359, "IDT"},
		{360, "Memory Segments"},
		{539, "Memory Segments"},
		{540, "32-bit Instruction Set"},
		{719, "32-bit Instruction Set"},
		{720, "Open386"},
		{899, "Open386"},
	}
	for _, tc := range cases {
		subject, detail := contemplation.ConceptForSecond(tc.sec, 900)
		if !strings.Contains(subject, tc.wantIn) {
			t.Errorf("ConceptForSecond(%d): subject %q does not contain %q", tc.sec, subject, tc.wantIn)
		}
		if detail == "" {
			t.Errorf("ConceptForSecond(%d): detail is empty", tc.sec)
		}
	}
}

// TestLock_Ready_InitiallyFalse confirms the lock is held at creation.
func TestLock_Ready_InitiallyFalse(t *testing.T) {
	lock := contemplation.New(10*time.Millisecond, &bytes.Buffer{})
	if lock.Ready() {
		t.Error("lock should not be Ready before Run is called")
	}
}

// TestLock_Run_ReleasesLock confirms that Run closes the Done channel and sets
// Ready to true after the configured duration elapses.
func TestLock_Run_ReleasesLock(t *testing.T) {
	var buf bytes.Buffer
	lock := contemplation.New(50*time.Millisecond, &buf)

	go lock.Run()

	select {
	case <-lock.Done():
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("lock did not release within 2 s")
	}

	if !lock.Ready() {
		t.Error("Ready() should return true after lock is released")
	}
}

// TestLock_Run_WritesProgress verifies that progress lines are written to the
// provided writer during the contemplation period.
func TestLock_Run_WritesProgress(t *testing.T) {
	var buf bytes.Buffer
	// Use 1100 ms so totalSec = 1 and at least one progress line is emitted.
	lock := contemplation.New(1100*time.Millisecond, &buf)
	lock.Run()

	out := buf.String()
	if !strings.Contains(out, "Neural Root") {
		t.Errorf("output missing [Neural Root] prefix:\n%s", out)
	}
	if !strings.Contains(out, "Neural Registry lock released") {
		t.Errorf("output missing release message:\n%s", out)
	}
}

// TestLock_Run_Idempotent verifies that calling Run a second time is a no-op
// and does not panic or block.
func TestLock_Run_Idempotent(t *testing.T) {
	var buf bytes.Buffer
	lock := contemplation.New(10*time.Millisecond, &buf)
	lock.Run()
	lock.Run() // second call must be a no-op
	if !lock.Ready() {
		t.Error("lock should be ready after Run completes")
	}
}

// TestLock_RemainingSeconds_ZeroAfterRelease verifies the helper returns 0
// once the lock is released.
func TestLock_RemainingSeconds_ZeroAfterRelease(t *testing.T) {
	lock := contemplation.New(10*time.Millisecond, &bytes.Buffer{})
	lock.Run()
	if r := lock.RemainingSeconds(); r != 0 {
		t.Errorf("RemainingSeconds after release: got %d, want 0", r)
	}
}

// TestLock_RemainingSeconds_NonzeroBeforeRun confirms that before Run is
// called the remaining time equals the configured duration.
func TestLock_RemainingSeconds_NonzeroBeforeRun(t *testing.T) {
	d := 10 * time.Second
	lock := contemplation.New(d, &bytes.Buffer{})
	r := lock.RemainingSeconds()
	want := int(d.Seconds())
	if r != want {
		t.Errorf("RemainingSeconds before Run: got %d, want %d", r, want)
	}
}
