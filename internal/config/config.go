// Package config defines configuration types for the 32Hybrid AVD system.
// All three binaries (control plane, runner, AVD client) read YAML config
// files using this package.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ControlPlaneConfig holds all settings for the control plane binary.
type ControlPlaneConfig struct {
	// ListenAddr is the gRPC listen address for the AVD client API.
	// Default: ":50051"
	ListenAddr string `yaml:"listen_addr"`

	Storage StorageConfig `yaml:"storage"`
	Azure   AzureConfig   `yaml:"azure"`
	Runner  RunnerConfig  `yaml:"runner"`
}

// StorageConfig holds Azure Blob Storage credentials and container names.
type StorageConfig struct {
	// AccountName is the Azure Storage account name (without .blob.core.windows.net).
	AccountName string `yaml:"account_name"`

	// AccountKey is the base64-encoded 512-bit storage account key.
	// Store this value in a secrets manager in production; use a placeholder here.
	AccountKey string `yaml:"account_key"`

	// UploadsContainer is the blob container for uploaded EXE blobs.
	// Default: "uploads"
	UploadsContainer string `yaml:"uploads_container"`

	// RunsContainer is the blob container for run output artifacts.
	// Default: "runs"
	RunsContainer string `yaml:"runs_container"`

	// SASTTLSeconds controls how long minted SAS tokens are valid.
	// Default: 3600 (1 hour)
	SASTTLSeconds int `yaml:"sas_ttl_seconds"`
}

// AzureConfig holds Azure API discovery settings.
type AzureConfig struct {
	// SubscriptionID is the Azure subscription that owns the host pool.
	SubscriptionID string `yaml:"subscription_id"`

	// ResourceGroup is the resource group containing the host pool.
	ResourceGroup string `yaml:"resource_group"`

	// HostPoolName is the AVD host pool name (Personal type for MVP).
	HostPoolName string `yaml:"host_pool_name"`

	// TargetUsername is the configured username to prefer when picking
	// an interactive session (e.g. "DOMAIN\\user" or UPN).
	TargetUsername string `yaml:"target_username"`
}

// RunnerConfig holds the mTLS and addressing settings for the runner agent.
type RunnerConfig struct {
	// Host is the private IP or hostname of the runner agent.
	// For MVP this is a static value from Azure API discovery output.
	// Leave empty to rely solely on AzureConfig discovery at runtime.
	Host string `yaml:"host"`

	// Port is the gRPC port on the runner. Default: 5443
	Port int `yaml:"port"`

	// ClientCertFile is the path to the PEM client certificate (mTLS).
	ClientCertFile string `yaml:"client_cert_file"`

	// ClientKeyFile is the path to the PEM private key for the client cert.
	ClientKeyFile string `yaml:"client_key_file"`

	// CAFile is the path to the PEM CA certificate that signed the runner cert.
	CAFile string `yaml:"ca_file"`
}

// RunnerAgentConfig holds settings for the runner agent binary.
type RunnerAgentConfig struct {
	// ListenAddr is the gRPC listen address. Default: ":5443"
	ListenAddr string `yaml:"listen_addr"`

	// ServerCertFile is the path to the PEM server certificate.
	ServerCertFile string `yaml:"server_cert_file"`

	// ServerKeyFile is the path to the PEM server private key.
	ServerKeyFile string `yaml:"server_key_file"`

	// CAFile is the path to the PEM CA certificate that signed the control-plane cert.
	CAFile string `yaml:"ca_file"`

	// WorkDir is the local directory where run artifacts are staged.
	// Default: C:\32hybrid\runs (Windows)
	WorkDir string `yaml:"work_dir"`

	// DefaultTimeoutSeconds is used when a RunInSession request omits timeout.
	// Default: 300
	DefaultTimeoutSeconds int `yaml:"default_timeout_seconds"`
}

// AVDClientConfig holds settings for the AVD client CLI binary.
type AVDClientConfig struct {
	// ControlPlaneAddr is the address of the control plane gRPC server.
	// e.g. "controlplane.private:50051"
	ControlPlaneAddr string `yaml:"control_plane_addr"`

	// Insecure disables TLS for dev/test use.
	Insecure bool `yaml:"insecure"`
}

// LoadControlPlane reads a YAML file at path into a ControlPlaneConfig.
func LoadControlPlane(path string) (*ControlPlaneConfig, error) {
	cfg := defaultControlPlane()
	if err := loadYAML(path, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// LoadRunnerAgent reads a YAML file at path into a RunnerAgentConfig.
func LoadRunnerAgent(path string) (*RunnerAgentConfig, error) {
	cfg := defaultRunnerAgent()
	if err := loadYAML(path, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// LoadAVDClient reads a YAML file at path into an AVDClientConfig.
func LoadAVDClient(path string) (*AVDClientConfig, error) {
	cfg := defaultAVDClient()
	if err := loadYAML(path, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func defaultControlPlane() *ControlPlaneConfig {
	return &ControlPlaneConfig{
		ListenAddr: ":50051",
		Storage: StorageConfig{
			UploadsContainer: "uploads",
			RunsContainer:    "runs",
			SASTTLSeconds:    3600,
		},
		Runner: RunnerConfig{
			Port: 5443,
		},
	}
}

func defaultRunnerAgent() *RunnerAgentConfig {
	return &RunnerAgentConfig{
		ListenAddr:            ":5443",
		WorkDir:               `C:\32hybrid\runs`,
		DefaultTimeoutSeconds: 300,
	}
}

func defaultAVDClient() *AVDClientConfig {
	return &AVDClientConfig{
		ControlPlaneAddr: "localhost:50051",
		Insecure:         false,
	}
}

func loadYAML(path string, dst any) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("config: open %q: %w", path, err)
	}
	defer f.Close()
	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("config: decode %q: %w", path, err)
	}
	return nil
}
