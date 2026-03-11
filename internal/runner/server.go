// Package runner implements the gRPC RunnerService server.
//
// The runner agent runs on Windows AVD session hosts.  It:
//   - Lists active Windows logon sessions (MVP: stub returns one placeholder)
//   - Downloads the EXE via Azure Blob SAS URL
//   - Launches the EXE in the target interactive session via a Windows
//     Scheduled Task (the only reliable way to reach a real desktop session)
//   - Uploads exit.json when the process finishes
//
// GUI apps require an ACTIVE session; the runner refuses DISCONNECTED sessions.
// Only one run may execute at a time on a host (enforced by a mutex).
package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonv1 "github.com/backbiten/32Hybrid/gen/common/v1"
	runnerv1 "github.com/backbiten/32Hybrid/gen/runner/v1"
	"github.com/backbiten/32Hybrid/internal/config"
	"github.com/backbiten/32Hybrid/internal/runnerutil"
)

// SessionEnumerator is the interface for listing Windows logon sessions.
// The real implementation uses the Windows WTS API; the stub returns a
// single placeholder session for local development and testing.
type SessionEnumerator interface {
	ListSessions(ctx context.Context) ([]*runnerv1.Session, error)
}

// Launcher is the interface for launching a process in a Windows session.
type Launcher interface {
	// Launch schedules the given command to run in the specified session.
	// It returns when the scheduled task has been created; use the run
	// directory sentinel files to detect completion.
	Launch(ctx context.Context, sessionID, runDir, exePath string, args, env []string) error
}

// Server implements runnerv1.RunnerServiceServer.
type Server struct {
	cfg        *config.RunnerAgentConfig
	enumerator SessionEnumerator
	launcher   Launcher

	// mu enforces one run at a time (MVP constraint).
	mu      sync.Mutex
	running bool

	// runs tracks in-flight and completed run states (keyed by run_id).
	runsMu sync.RWMutex
	runs   map[string]*runState
}

type runState struct {
	RunID      string
	State      commonv1.RunState
	StartedAt  time.Time
	FinishedAt time.Time
	ExitCode   int32
	ErrorMsg   string
}

// NewServer returns a Server using the provided config and stub implementations.
// Swap in real Windows implementations for SessionEnumerator and Launcher on production builds.
func NewServer(cfg *config.RunnerAgentConfig, enum SessionEnumerator, launch Launcher) *Server {
	return &Server{
		cfg:        cfg,
		enumerator: enum,
		launcher:   launch,
		runs:       make(map[string]*runState),
	}
}

// ListSessions implements runnerv1.RunnerServiceServer.
func (s *Server) ListSessions(ctx context.Context, _ *runnerv1.ListSessionsRequest) (*runnerv1.ListSessionsResponse, error) {
	sessions, err := s.enumerator.ListSessions(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "enumerate sessions: %v", err)
	}
	return &runnerv1.ListSessionsResponse{Sessions: sessions}, nil
}

// RunInSession implements runnerv1.RunnerServiceServer.
func (s *Server) RunInSession(ctx context.Context, req *runnerv1.RunInSessionRequest) (*runnerv1.RunInSessionResponse, error) {
	if req.RunId == "" {
		return nil, status.Error(codes.InvalidArgument, "run_id is required")
	}
	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}
	if req.ExeReadSasUrl == "" {
		return nil, status.Error(codes.InvalidArgument, "exe_read_sas_url is required")
	}
	if req.ExitJsonWriteSasUrl == "" {
		return nil, status.Error(codes.InvalidArgument, "exit_json_write_sas_url is required")
	}

	// Enforce one run at a time.
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil, status.Error(codes.ResourceExhausted, "another run is already in progress on this host")
	}
	s.running = true
	s.mu.Unlock()

	// Verify the target session is ACTIVE (GUI apps require an interactive desktop).
	sessions, err := s.enumerator.ListSessions(ctx)
	if err != nil {
		s.clearRunning()
		return nil, status.Errorf(codes.Internal, "list sessions: %v", err)
	}
	if err := assertActiveSession(sessions, req.SessionId); err != nil {
		s.clearRunning()
		return nil, status.Errorf(codes.FailedPrecondition, "%v", err)
	}

	// Prepare run directory.
	workDir := s.cfg.WorkDir
	if workDir == "" {
		workDir = filepath.Join(os.TempDir(), "32hybrid", "runs")
	}
	runDir := filepath.Join(workDir, req.RunId)
	if err := os.MkdirAll(runDir, 0o700); err != nil {
		s.clearRunning()
		return nil, status.Errorf(codes.Internal, "create run dir: %v", err)
	}

	exePath := filepath.Join(runDir, "app.exe")

	rs := &runState{
		RunID:     req.RunId,
		State:     commonv1.RunState_RUN_STATE_PENDING,
		StartedAt: time.Now().UTC(),
	}
	s.setRunState(req.RunId, rs)

	// Execute asynchronously so the RPC returns immediately.
	go s.execRun(context.Background(), req, runDir, exePath, rs)

	return &runnerv1.RunInSessionResponse{RunId: req.RunId}, nil
}

// GetStatus implements runnerv1.RunnerServiceServer.
func (s *Server) GetStatus(_ context.Context, req *runnerv1.GetStatusRequest) (*runnerv1.GetStatusResponse, error) {
	if req.RunId == "" {
		return nil, status.Error(codes.InvalidArgument, "run_id is required")
	}
	rs := s.getRunState(req.RunId)
	if rs == nil {
		return nil, status.Errorf(codes.NotFound, "run %q not found", req.RunId)
	}
	resp := &runnerv1.GetStatusResponse{
		RunId:        rs.RunID,
		State:        rs.State,
		ErrorMessage: rs.ErrorMsg,
		ExitCode:     rs.ExitCode,
	}
	if !rs.StartedAt.IsZero() {
		resp.StartedAt = rs.StartedAt.Unix()
	}
	if !rs.FinishedAt.IsZero() {
		resp.FinishedAt = rs.FinishedAt.Unix()
	}
	return resp, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal execution pipeline
// ─────────────────────────────────────────────────────────────────────────────

func (s *Server) execRun(ctx context.Context, req *runnerv1.RunInSessionRequest, runDir, exePath string, rs *runState) {
	defer s.clearRunning()

	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	if timeout == 0 {
		d := s.cfg.DefaultTimeoutSeconds
		if d == 0 {
			d = 300
		}
		timeout = time.Duration(d) * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Step 1: Download EXE.
	if err := downloadSAS(ctx, req.ExeReadSasUrl, exePath); err != nil {
		s.finishRun(ctx, req, rs, 0, fmt.Sprintf("download exe: %v", err))
		return
	}

	// Step 2: Launch via Scheduled Task in the target session.
	rs.State = commonv1.RunState_RUN_STATE_RUNNING
	rs.StartedAt = time.Now().UTC()
	s.setRunState(req.RunId, rs)

	if err := s.launcher.Launch(ctx, req.SessionId, runDir, exePath, req.Args, req.Env); err != nil {
		s.finishRun(ctx, req, rs, 0, fmt.Sprintf("launch: %v", err))
		return
	}

	// Step 3: Wait for completion by polling the exitcode sentinel file.
	exitCode, err := waitForExit(ctx, runDir)
	if err != nil {
		s.finishRun(ctx, req, rs, 0, fmt.Sprintf("wait for exit: %v", err))
		return
	}

	s.finishRun(ctx, req, rs, int32(exitCode), "")
}

func (s *Server) finishRun(ctx context.Context, req *runnerv1.RunInSessionRequest, rs *runState, exitCode int32, errMsg string) {
	rs.FinishedAt = time.Now().UTC()
	rs.ExitCode = exitCode
	if errMsg != "" {
		rs.State = commonv1.RunState_RUN_STATE_ERROR
		rs.ErrorMsg = errMsg
	} else if exitCode == 0 {
		rs.State = commonv1.RunState_RUN_STATE_SUCCEEDED
	} else {
		rs.State = commonv1.RunState_RUN_STATE_FAILED
	}
	s.setRunState(req.RunId, rs)

	// Upload exit.json.
	ej := &runnerutil.ExitJSON{
		RunID:      req.RunId,
		ExitCode:   int(exitCode),
		StartedAt:  rs.StartedAt,
		FinishedAt: rs.FinishedAt,
		Error:      errMsg,
	}
	ejBytes, err := ej.Marshal()
	if err == nil {
		_ = uploadSAS(ctx, req.ExitJsonWriteSasUrl, ejBytes, "application/json")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func (s *Server) clearRunning() {
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()
}

func (s *Server) setRunState(runID string, rs *runState) {
	s.runsMu.Lock()
	s.runs[runID] = rs
	s.runsMu.Unlock()
}

func (s *Server) getRunState(runID string) *runState {
	s.runsMu.RLock()
	defer s.runsMu.RUnlock()
	return s.runs[runID]
}

// assertActiveSession returns an error if the requested session is not ACTIVE.
func assertActiveSession(sessions []*runnerv1.Session, sessionID string) error {
	for _, sess := range sessions {
		if sess.SessionId != sessionID {
			continue
		}
		if sess.State != commonv1.SessionState_SESSION_STATE_ACTIVE {
			return fmt.Errorf("session %q is %v, not ACTIVE — GUI apps require an interactive desktop",
				sessionID, sess.State)
		}
		return nil
	}
	return fmt.Errorf("session %q not found on this host", sessionID)
}

// downloadSAS downloads the resource at sasURL and writes it to destPath.
func downloadSAS(ctx context.Context, sasURL, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sasURL, nil)
	if err != nil {
		return fmt.Errorf("build GET request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", sasURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("GET returned %d: %s", resp.StatusCode, body)
	}
	f, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o700)
	if err != nil {
		return fmt.Errorf("create dest file: %w", err)
	}
	defer f.Close()
	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write dest file: %w", err)
	}
	return nil
}

// uploadSAS uploads data to sasURL as a BlockBlob using HTTP PUT.
func uploadSAS(ctx context.Context, sasURL string, data []byte, contentType string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, sasURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("build PUT request: %w", err)
	}
	req.Header.Set("x-ms-blob-type", "BlockBlob")
	req.Header.Set("Content-Type", contentType)
	req.ContentLength = int64(len(data))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("PUT %s: %w", sasURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("PUT returned %d: %s", resp.StatusCode, body)
	}
	return nil
}

// waitForExit polls for an exitcode.txt sentinel file written by the launched
// process wrapper (launch.cmd).  It blocks until the file appears or the
// context deadline fires.
//
// TODO(v0.2): Replace with a proper IPC mechanism (e.g. named pipe or event).
func waitForExit(ctx context.Context, runDir string) (int, error) {
	exitFile := filepath.Join(runDir, "exitcode.txt")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return 0, fmt.Errorf("timeout waiting for process to finish")
		case <-ticker.C:
			data, err := os.ReadFile(exitFile)
			if err != nil {
				continue // file not yet written
			}
			var code int
			if err := json.Unmarshal(bytes.TrimSpace(data), &code); err != nil {
				// Try plain integer text (e.g. "0\n")
				_, _ = fmt.Sscanf(string(bytes.TrimSpace(data)), "%d", &code)
			}
			return code, nil
		}
	}
}
