// cmd/runner — 32Hybrid runner agent gRPC server.
//
// The runner agent runs on Windows AVD session hosts.  It listens for
// RunInSession requests from the control plane, downloads the EXE via SAS
// URL, launches it in the target interactive desktop session via a Scheduled
// Task, and uploads exit.json when the process finishes.
//
// Usage:
//
//	runner --config C:\32hybrid\runner.yaml
package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	runnerv1 "github.com/backbiten/32Hybrid/gen/runner/v1"
	"github.com/backbiten/32Hybrid/internal/config"
	"github.com/backbiten/32Hybrid/internal/runner"
)

func main() {
	configPath := flag.String("config", "runner.yaml", "Path to YAML config file")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	srv := runner.NewServer(cfg,
		&runner.StubEnumerator{},
		&runner.StubLauncher{},
	)

	creds, err := buildServerCreds(cfg)
	if err != nil {
		log.Fatalf("TLS config: %v", err)
	}

	lis, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		log.Fatalf("listen %s: %v", cfg.ListenAddr, err)
	}

	var grpcOpts []grpc.ServerOption
	if creds != nil {
		grpcOpts = append(grpcOpts, grpc.Creds(creds))
	}
	grpcSrv := grpc.NewServer(grpcOpts...)
	runnerv1.RegisterRunnerServiceServer(grpcSrv, srv)

	log.Printf("32Hybrid runner agent listening on %s", cfg.ListenAddr)
	if err := grpcSrv.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// loadConfig loads config from the YAML file at path, or returns dev defaults.
func loadConfig(path string) (*config.RunnerAgentConfig, error) {
	cfg, err := config.LoadRunnerAgent(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("warning: config file %q not found, using defaults (dev mode)", path)
			return defaultDevConfig(), nil
		}
		return nil, fmt.Errorf("load config %q: %w", path, err)
	}
	return cfg, nil
}

func defaultDevConfig() *config.RunnerAgentConfig {
	return &config.RunnerAgentConfig{
		ListenAddr:            ":5443",
		WorkDir:               os.TempDir(),
		DefaultTimeoutSeconds: 300,
	}
}

// buildServerCreds returns mTLS server credentials when cert/key/CA are
// configured, or plaintext credentials for dev (with a warning).
func buildServerCreds(cfg *config.RunnerAgentConfig) (credentials.TransportCredentials, error) {
	if cfg.ServerCertFile == "" || cfg.ServerKeyFile == "" || cfg.CAFile == "" {
		log.Printf("warning: TLS cert/key/CA not configured — using plaintext gRPC (dev mode only)")
		return insecure.NewCredentials(), nil
	}

	cert, err := tls.LoadX509KeyPair(cfg.ServerCertFile, cfg.ServerKeyFile)
	if err != nil {
		return nil, fmt.Errorf("load server cert: %w", err)
	}
	caPEM, err := os.ReadFile(cfg.CAFile)
	if err != nil {
		return nil, fmt.Errorf("read CA: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("parse CA cert from %s", cfg.CAFile)
	}
	tlsCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pool,
		MinVersion:   tls.VersionTLS13,
	}
	return credentials.NewTLS(tlsCfg), nil
}
