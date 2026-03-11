// cmd/controlplane — 32Hybrid control plane gRPC server.
//
// The control plane accepts EXE submissions from AVD clients, uploads blobs
// to Azure Storage, mints SAS URLs, discovers the runner agent, and dispatches
// runs over mTLS gRPC.
//
// Usage:
//
//	controlplane --config /etc/32hybrid/controlplane.yaml
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"

	cpv1 "github.com/backbiten/32Hybrid/gen/controlplane/v1"
	"github.com/backbiten/32Hybrid/internal/config"
	"github.com/backbiten/32Hybrid/internal/controlplane"
	"github.com/backbiten/32Hybrid/internal/discovery"
)

func main() {
	configPath := flag.String("config", "controlplane.yaml", "Path to YAML config file")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	disc := buildDiscoverer(cfg)

	srv := controlplane.NewServer(cfg, disc)

	lis, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		log.Fatalf("listen %s: %v", cfg.ListenAddr, err)
	}

	grpcSrv := grpc.NewServer()
	cpv1.RegisterControlPlaneServiceServer(grpcSrv, srv)

	log.Printf("32Hybrid control plane listening on %s", cfg.ListenAddr)
	if err := grpcSrv.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// loadConfig attempts to load the YAML file at path.  If the file does not
// exist and path is the default, it logs a warning and uses built-in defaults
// so the server can start in dev mode without a config file.
func loadConfig(path string) (*config.ControlPlaneConfig, error) {
	cfg, err := config.LoadControlPlane(path)
	if err != nil {
		if os.IsNotExist(err) && path == "controlplane.yaml" {
			log.Printf("warning: config file %q not found, using defaults (dev mode)", path)
			return config.LoadControlPlane("") // will fail, fall back below
		}
		// If the config file simply does not exist, use defaults.
		if os.IsNotExist(err) {
			log.Printf("warning: config file %q not found, using defaults", path)
			return defaultDevConfig(), nil
		}
		return nil, fmt.Errorf("load config %q: %w", path, err)
	}
	return cfg, nil
}

func defaultDevConfig() *config.ControlPlaneConfig {
	return &config.ControlPlaneConfig{
		ListenAddr: ":50051",
		Storage: config.StorageConfig{
			AccountName:      "devstoreaccount1",
			AccountKey:       "PLACEHOLDER",
			UploadsContainer: "uploads",
			RunsContainer:    "runs",
			SASTTLSeconds:    3600,
		},
		Azure: config.AzureConfig{
			TargetUsername: "",
		},
		Runner: config.RunnerConfig{
			Host: "127.0.0.1",
			Port: 5443,
		},
	}
}

// buildDiscoverer returns a StaticDiscoverer if a host is configured, or an
// AzureDiscoverer otherwise.  In v0.1 the static discoverer is always used.
func buildDiscoverer(cfg *config.ControlPlaneConfig) discovery.Discoverer {
	if cfg.Runner.Host != "" {
		return &discovery.StaticDiscoverer{Host: cfg.Runner.Host}
	}
	return &discovery.AzureDiscoverer{
		SubscriptionID: cfg.Azure.SubscriptionID,
		ResourceGroup:  cfg.Azure.ResourceGroup,
		HostPoolName:   cfg.Azure.HostPoolName,
		TargetUsername: cfg.Azure.TargetUsername,
	}
}
