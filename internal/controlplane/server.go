// Package controlplane implements the gRPC ControlPlaneService server.
//
// Responsibilities:
//   - Accept SubmitRun from AVD clients
//   - Upload EXE bytes to Azure Blob storage
//   - Mint SAS URLs (read for exe, write for stdout/stderr/exit.json)
//   - Discover the runner agent host (via Discoverer)
//   - Dial the runner over mTLS and call ListSessions + RunInSession
//   - Persist run records in the in-memory store
package controlplane

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	commonv1 "github.com/backbiten/32Hybrid/gen/common/v1"
	cpv1 "github.com/backbiten/32Hybrid/gen/controlplane/v1"
	runnerv1 "github.com/backbiten/32Hybrid/gen/runner/v1"
	"github.com/backbiten/32Hybrid/internal/config"
	"github.com/backbiten/32Hybrid/internal/discovery"
	"github.com/backbiten/32Hybrid/internal/sas"
	"github.com/backbiten/32Hybrid/internal/store"
)

// Server implements cpv1.ControlPlaneServiceServer.
type Server struct {
	cfg        *config.ControlPlaneConfig
	discoverer discovery.Discoverer
	store      *store.MemStore

	// mu guards concurrent SubmitRun calls (one run dispatched at a time for MVP).
	mu sync.Mutex
}

// NewServer constructs a Server. cfg must be non-nil.
func NewServer(cfg *config.ControlPlaneConfig, disc discovery.Discoverer) *Server {
	return &Server{
		cfg:        cfg,
		discoverer: disc,
		store:      store.New(),
	}
}

// SubmitRun implements cpv1.ControlPlaneServiceServer.
func (s *Server) SubmitRun(ctx context.Context, req *cpv1.SubmitRunRequest) (*cpv1.SubmitRunResponse, error) {
	if len(req.ExeBytes) == 0 {
		return nil, status.Error(codes.InvalidArgument, "exe_bytes is required")
	}
	exeName := req.ExeName
	if exeName == "" {
		exeName = "app.exe"
	}

	// Verify client-supplied SHA-256 if provided.
	digest := sha256.Sum256(req.ExeBytes)
	hexDigest := hex.EncodeToString(digest[:])
	if req.ExeSha256 != "" && !strings.EqualFold(req.ExeSha256, hexDigest) {
		return nil, status.Errorf(codes.InvalidArgument,
			"exe_sha256 mismatch: got %s, computed %s", req.ExeSha256, hexDigest)
	}

	runID := uuid.New().String()
	now := time.Now().UTC()
	rec := &store.RunRecord{
		RunID:       runID,
		State:       commonv1.RunState_RUN_STATE_PENDING,
		SubmittedAt: now,
	}
	s.store.Put(rec)

	// Upload EXE to blob: uploads/<sha256>/<exe_name>
	exeBlobName := fmt.Sprintf("uploads/%s/%s", hexDigest, exeName)
	if err := s.uploadBlob(ctx, s.cfg.Storage.UploadsContainer, exeBlobName, req.ExeBytes); err != nil {
		s.updateState(runID, commonv1.RunState_RUN_STATE_ERROR, err.Error())
		return nil, status.Errorf(codes.Internal, "upload exe: %v", err)
	}

	ttl := time.Duration(s.cfg.Storage.SASTTLSeconds) * time.Second
	if ttl == 0 {
		ttl = time.Hour
	}
	expiry := now.Add(ttl)

	baseParams := sas.Params{
		AccountName: s.cfg.Storage.AccountName,
		AccountKey:  s.cfg.Storage.AccountKey,
		Expiry:      expiry,
	}

	exeSAS, err := s.mintSAS(baseParams, s.cfg.Storage.UploadsContainer, exeBlobName, "r")
	if err != nil {
		s.updateState(runID, commonv1.RunState_RUN_STATE_ERROR, err.Error())
		return nil, status.Errorf(codes.Internal, "mint exe SAS: %v", err)
	}

	runsContainer := s.cfg.Storage.RunsContainer
	stdoutBlob := fmt.Sprintf("%s/stdout.txt", runID)
	stderrBlob := fmt.Sprintf("%s/stderr.txt", runID)
	exitJSONBlob := fmt.Sprintf("%s/exit.json", runID)

	stdoutSAS, err := s.mintSAS(baseParams, runsContainer, stdoutBlob, "w")
	if err != nil {
		s.updateState(runID, commonv1.RunState_RUN_STATE_ERROR, err.Error())
		return nil, status.Errorf(codes.Internal, "mint stdout SAS: %v", err)
	}
	stderrSAS, err := s.mintSAS(baseParams, runsContainer, stderrBlob, "w")
	if err != nil {
		s.updateState(runID, commonv1.RunState_RUN_STATE_ERROR, err.Error())
		return nil, status.Errorf(codes.Internal, "mint stderr SAS: %v", err)
	}
	exitJSONSAS, err := s.mintSAS(baseParams, runsContainer, exitJSONBlob, "w")
	if err != nil {
		s.updateState(runID, commonv1.RunState_RUN_STATE_ERROR, err.Error())
		return nil, status.Errorf(codes.Internal, "mint exit.json SAS: %v", err)
	}

	// Record blob paths before dispatching.
	rec.StdoutBlobPath = stdoutBlob
	rec.StderrBlobPath = stderrBlob
	rec.ExitJSONBlobPath = exitJSONBlob
	s.store.Put(rec)

	// Dispatch to runner (one at a time per host for MVP).
	s.mu.Lock()
	defer s.mu.Unlock()

	host, err := s.discoverer.Discover(ctx)
	if err != nil {
		s.updateState(runID, commonv1.RunState_RUN_STATE_ERROR, err.Error())
		return nil, status.Errorf(codes.Unavailable, "discover runner: %v", err)
	}

	runnerAddr := fmt.Sprintf("%s:%d", host.PrivateIP, s.cfg.Runner.Port)
	conn, err := s.dialRunner(runnerAddr)
	if err != nil {
		s.updateState(runID, commonv1.RunState_RUN_STATE_ERROR, err.Error())
		return nil, status.Errorf(codes.Unavailable, "dial runner %s: %v", runnerAddr, err)
	}
	defer conn.Close()

	rc := runnerv1.NewRunnerServiceClient(conn)

	listResp, err := rc.ListSessions(ctx, &runnerv1.ListSessionsRequest{})
	if err != nil {
		s.updateState(runID, commonv1.RunState_RUN_STATE_ERROR, err.Error())
		return nil, status.Errorf(codes.Unavailable, "list sessions: %v", err)
	}

	sessionID, err := pickSession(listResp.Sessions, s.cfg.Azure.TargetUsername)
	if err != nil {
		s.updateState(runID, commonv1.RunState_RUN_STATE_ERROR, err.Error())
		return nil, status.Errorf(codes.FailedPrecondition, "pick session: %v", err)
	}

	timeout := req.TimeoutSeconds
	if timeout == 0 {
		timeout = 300
	}

	_, err = rc.RunInSession(ctx, &runnerv1.RunInSessionRequest{
		RunId:               runID,
		SessionId:           sessionID,
		ExeReadSasUrl:       exeSAS,
		StdoutWriteSasUrl:   stdoutSAS,
		StderrWriteSasUrl:   stderrSAS,
		ExitJsonWriteSasUrl: exitJSONSAS,
		Args:                req.Args,
		Env:                 req.Env,
		TimeoutSeconds:      timeout,
	})
	if err != nil {
		s.updateState(runID, commonv1.RunState_RUN_STATE_ERROR, err.Error())
		return nil, status.Errorf(codes.Internal, "RunInSession: %v", err)
	}

	rec.SessionID = sessionID
	rec.State = commonv1.RunState_RUN_STATE_RUNNING
	s.store.Put(rec)

	return &cpv1.SubmitRunResponse{RunId: runID}, nil
}

// GetRun implements cpv1.ControlPlaneServiceServer.
func (s *Server) GetRun(_ context.Context, req *cpv1.GetRunRequest) (*cpv1.GetRunResponse, error) {
	if req.RunId == "" {
		return nil, status.Error(codes.InvalidArgument, "run_id is required")
	}
	rec, err := s.store.Get(req.RunId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "%v", err)
	}
	return store.ToProto(rec), nil
}

// ListRuns implements cpv1.ControlPlaneServiceServer.
func (s *Server) ListRuns(_ context.Context, _ *cpv1.ListRunsRequest) (*cpv1.ListRunsResponse, error) {
	records := s.store.List()
	resp := &cpv1.ListRunsResponse{
		Runs: make([]*cpv1.GetRunResponse, 0, len(records)),
	}
	for _, r := range records {
		resp.Runs = append(resp.Runs, store.ToProto(r))
	}
	return resp, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func (s *Server) updateState(runID string, st commonv1.RunState, errMsg string) {
	rec, err := s.store.Get(runID)
	if err != nil {
		return
	}
	rec.State = st
	rec.ErrorMsg = errMsg
	s.store.Put(rec)
}

func (s *Server) mintSAS(base sas.Params, container, blobName, perms string) (string, error) {
	p := base
	p.Container = container
	p.BlobName = blobName
	p.Permissions = perms
	return sas.BlobSASURL(p)
}

// uploadBlob uploads data to Azure Blob Storage using the REST API with a
// SharedKey-authenticated PUT request.
//
// This avoids requiring the full Azure SDK as a dependency for MVP.
func (s *Server) uploadBlob(ctx context.Context, container, blobName string, data []byte) error {
	accountName := s.cfg.Storage.AccountName
	if accountName == "" {
		return fmt.Errorf("storage account name is not configured")
	}

	blobURL := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s",
		accountName, container, blobName)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, blobURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("build upload request: %w", err)
	}

	dateStr := time.Now().UTC().Format(http.TimeFormat)
	req.Header.Set("x-ms-blob-type", "BlockBlob")
	req.Header.Set("x-ms-date", dateStr)
	req.Header.Set("x-ms-version", "2020-12-06")
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = int64(len(data))

	authHeader, err := sharedKeyAuth(accountName, s.cfg.Storage.AccountKey, dateStr, req.ContentLength, container, blobName)
	if err != nil {
		return fmt.Errorf("compute SharedKey auth: %w", err)
	}
	req.Header.Set("Authorization", authHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("PUT blob: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("PUT blob returned %d: %s", resp.StatusCode, body)
	}
	return nil
}

// sharedKeyAuth computes the Azure SharedKey Authorization header for a blob PUT.
// https://learn.microsoft.com/en-us/rest/api/storageservices/authorize-with-shared-key
func sharedKeyAuth(accountName, accountKeyB64, dateStr string, contentLength int64, container, blobName string) (string, error) {
	canonicalHeaders := fmt.Sprintf(
		"x-ms-blob-type:BlockBlob\nx-ms-date:%s\nx-ms-version:2020-12-06",
		dateStr,
	)
	canonicalResource := fmt.Sprintf("/%s/%s/%s", accountName, container, blobName)

	clStr := ""
	if contentLength > 0 {
		clStr = fmt.Sprintf("%d", contentLength)
	}

	stringToSign := strings.Join([]string{
		"PUT",
		"",    // Content-Encoding
		"",    // Content-Language
		clStr, // Content-Length
		"",    // Content-MD5
		"application/octet-stream", // Content-Type
		"",   // Date (x-ms-date used instead)
		"",   // If-Modified-Since
		"",   // If-Match
		"",   // If-None-Match
		"",   // If-Unmodified-Since
		"",   // Range
		canonicalHeaders + "\n",
		canonicalResource,
	}, "\n")

	sig, err := blobHMACSHA256(accountKeyB64, stringToSign)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("SharedKey %s:%s", accountName, sig), nil
}

// blobHMACSHA256 signs a string with the storage account key.
func blobHMACSHA256(accountKeyB64, message string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(accountKeyB64)
	if err != nil {
		return "", fmt.Errorf("base64-decode account key: %w", err)
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

// dialRunner opens a gRPC connection to the runner agent with mTLS if
// certificates are configured, or plaintext if none are set (dev only).
func (s *Server) dialRunner(addr string) (*grpc.ClientConn, error) {
	rc := s.cfg.Runner
	if rc.ClientCertFile == "" || rc.ClientKeyFile == "" || rc.CAFile == "" {
		// No certs configured — use plaintext (dev / test only).
		return grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	cert, err := tls.LoadX509KeyPair(rc.ClientCertFile, rc.ClientKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load client cert: %w", err)
	}
	caPEM, err := os.ReadFile(rc.CAFile)
	if err != nil {
		return nil, fmt.Errorf("read CA file: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("parse CA cert from %s", rc.CAFile)
	}
	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		MinVersion:   tls.VersionTLS13,
	}
	return grpc.NewClient(addr, grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
}

// pickSession chooses the best interactive session from the list.
// Rules (in priority order):
//  1. Prefer a session whose username matches targetUsername (case-insensitive).
//  2. Fall back to any ACTIVE session.
//  3. Fail if no ACTIVE session is found.
func pickSession(sessions []*runnerv1.Session, targetUsername string) (string, error) {
	var fallback string
	for _, sess := range sessions {
		if sess.State != commonv1.SessionState_SESSION_STATE_ACTIVE {
			continue
		}
		if strings.EqualFold(sess.Username, targetUsername) {
			return sess.SessionId, nil
		}
		if fallback == "" {
			fallback = sess.SessionId
		}
	}
	if fallback != "" {
		return fallback, nil
	}
	return "", fmt.Errorf("no ACTIVE session available on the runner host")
}
