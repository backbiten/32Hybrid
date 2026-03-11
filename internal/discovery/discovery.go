// Package discovery defines the interface for locating a runner agent host and
// provides a placeholder implementation suitable for v0.1.
//
// In v0.1 the control plane targets a single personal-pool session host.
// A real implementation would call the Azure Virtual Desktop REST API to
// enumerate session hosts and resolve their private IPs.
package discovery

import (
	"context"
	"fmt"
)

// SessionHost describes a discovered AVD session host.
type SessionHost struct {
	// Name is the session host resource name (e.g. "vm-avd-0.contoso.local").
	Name string

	// PrivateIP is the private IPv4 address reachable over the peered VNet.
	PrivateIP string
}

// Discoverer resolves the target runner agent host.
type Discoverer interface {
	// Discover returns the session host that should receive the next run.
	// Implementations may contact Azure APIs or use static configuration.
	Discover(ctx context.Context) (*SessionHost, error)
}

// StaticDiscoverer returns a fixed host and IP from configuration.
// It is the v0.1 implementation — replace with AzureDiscoverer when ready.
type StaticDiscoverer struct {
	// Host is the static hostname or IP of the runner agent.
	Host string
}

// Discover returns the statically configured host.
func (s *StaticDiscoverer) Discover(_ context.Context) (*SessionHost, error) {
	if s.Host == "" {
		return nil, fmt.Errorf("discovery: static host is not configured")
	}
	return &SessionHost{
		Name:      s.Host,
		PrivateIP: s.Host,
	}, nil
}

// AzureDiscoverer contacts the Azure Virtual Desktop REST API to discover
// the private IP of the configured personal-pool session host.
//
// TODO(v0.2): Implement using github.com/Azure/azure-sdk-for-go.
// Required Azure RBAC: "Desktop Virtualization Reader" on the host pool.
// Required API call:
//
//	GET /subscriptions/{sub}/resourceGroups/{rg}/providers/
//	    Microsoft.DesktopVirtualization/hostPools/{pool}/sessionHosts
//
// Then resolve the private IP via the NIC resource or Azure Instance Metadata.
type AzureDiscoverer struct {
	SubscriptionID string
	ResourceGroup  string
	HostPoolName   string
	TargetUsername string
}

// Discover is not yet implemented and always returns an error.
// Replace the body of this method in v0.2 once Azure SDK integration is added.
func (a *AzureDiscoverer) Discover(_ context.Context) (*SessionHost, error) {
	// TODO(v0.2): call Azure VDesktop API; resolve private IP via NIC resource.
	return nil, fmt.Errorf(
		"discovery: AzureDiscoverer is not yet implemented "+
			"(subscription=%s, rg=%s, pool=%s) — set runner.host in config for v0.1",
		a.SubscriptionID, a.ResourceGroup, a.HostPoolName,
	)
}
