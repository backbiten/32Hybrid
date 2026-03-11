// Package runner — stub session enumerator and launcher.
//
// StubEnumerator and StubLauncher are used in development builds (Linux/macOS)
// and unit tests.  On Windows production builds, replace these with real
// implementations that call WTSEnumerateSessions and the Task Scheduler COM API.
package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	commonv1 "github.com/backbiten/32Hybrid/gen/common/v1"
	runnerv1 "github.com/backbiten/32Hybrid/gen/runner/v1"
)

// StubEnumerator returns a single hard-coded ACTIVE session.
// Replace with a Windows WTS implementation in production.
type StubEnumerator struct {
	// SessionID is the session ID to report. Defaults to "1".
	SessionID string
	// Username is the username to report. Defaults to current OS user.
	Username string
}

// ListSessions returns one stub ACTIVE session.
func (e *StubEnumerator) ListSessions(_ context.Context) ([]*runnerv1.Session, error) {
	id := e.SessionID
	if id == "" {
		id = "1"
	}
	username := e.Username
	if username == "" {
		username = currentUsername()
	}
	return []*runnerv1.Session{
		{
			SessionId: id,
			Username:  username,
			State:     commonv1.SessionState_SESSION_STATE_ACTIVE,
		},
	}, nil
}

// StubLauncher simulates launching a process by writing sentinel files.
// Replace with a Windows Task Scheduler implementation in production.
type StubLauncher struct{}

// Launch writes started.txt and exitcode.txt to simulate a zero-exit process.
//
// TODO(v0.2): Implement using the Windows Task Scheduler COM API to create a
// task named "32Hybrid\Run_<run_id>" that runs under the session user.
// The task command should:
//  1. Run the EXE with the provided args/env.
//  2. Write the exit code to exitcode.txt on completion.
func (l *StubLauncher) Launch(_ context.Context, _, runDir, exePath string, _, _ []string) error {
	// Write started.txt sentinel.
	if err := os.WriteFile(filepath.Join(runDir, "started.txt"), []byte("started"), 0o600); err != nil {
		return fmt.Errorf("write started.txt: %w", err)
	}

	// In the stub, immediately write exitcode.txt with exit code 0.
	// The real implementation writes this only after the scheduled task finishes.
	if err := os.WriteFile(filepath.Join(runDir, "exitcode.txt"), []byte("0"), 0o600); err != nil {
		return fmt.Errorf("write exitcode.txt: %w", err)
	}

	_ = exePath // used in real implementation
	return nil
}

func currentUsername() string {
	if u := os.Getenv("USERNAME"); u != "" {
		return u
	}
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return "unknown"
}
