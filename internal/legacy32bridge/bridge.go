// Package legacy32bridge provides adapters that translate Legacy32 nano-kernel
// service descriptors and system-call conventions into the 32Hybrid Root
// Registry model.
//
// The Legacy32 nano-kernel (written in C) exposes services by calling a
// register_service() equivalent that supplies a name and a capability
// descriptor.  This package bridges those registrations into the Go-native
// Registry so that 32Hybrid userland can discover and start Legacy32 services
// through the standard registry.Registry interface.
//
// Example – boot-time bridge registration (mirrors the C pseudocode from the
// issue):
//
//	bridge := legacy32bridge.New(reg)
//	bridge.RegisterService("network", legacy32bridge.ServiceDescriptor{
//	    Version:      1,
//	    Capabilities: legacy32bridge.CapNetwork,
//	})
//
// The call above is the Go equivalent of the Legacy32 C call:
//
//	register_service("network", &net_service_descriptor);
package legacy32bridge

import (
	"fmt"

	"github.com/backbiten/32Hybrid/internal/registry"
)

// Capability is a bitmask describing what a Legacy32 service can do.
// Extend this set as new subsystems are added to the nano-kernel.
type Capability uint32

const (
	// CapNetwork indicates the service provides network I/O primitives.
	CapNetwork Capability = 1 << iota
	// CapStorage indicates the service provides block/file storage access.
	CapStorage
	// CapDisplay indicates the service provides framebuffer or display output.
	CapDisplay
	// CapInput indicates the service handles keyboard/mouse input.
	CapInput
	// CapAudio indicates the service provides audio playback or capture.
	CapAudio
)

// ServiceDescriptor carries the metadata that the Legacy32 nano-kernel
// supplies when it calls register_service().  It is the Go equivalent of
// the C struct passed to that function.
type ServiceDescriptor struct {
	// Version is the ABI version exported by the Legacy32 service (≥1).
	Version uint32

	// Capabilities is the set of capabilities this service offers.
	Capabilities Capability

	// Description is an optional free-form description of the service.
	Description string
}

// Bridge adapts Legacy32 service registrations into the 32Hybrid Root
// Registry.  All methods are safe for concurrent use if the underlying
// Registry is also concurrency-safe (which MemRegistry is).
type Bridge struct {
	reg registry.Registry
}

// New creates a Bridge that writes translated service entries into reg.
// reg must not be nil.
func New(reg registry.Registry) *Bridge {
	if reg == nil {
		panic("legacy32bridge: registry must not be nil")
	}
	return &Bridge{reg: reg}
}

// RegisterService translates a Legacy32 service descriptor into a
// registry.Service and inserts it into the Root Registry.
//
// This is the Go-side counterpart to the Legacy32 C call:
//
//	register_service(name, &descriptor);
func (b *Bridge) RegisterService(name string, desc ServiceDescriptor) error {
	if name == "" {
		return fmt.Errorf("legacy32bridge: service name must not be empty")
	}
	if desc.Version == 0 {
		return fmt.Errorf("legacy32bridge: service %q has invalid version 0", name)
	}

	svc := registry.Service{
		Name:        name,
		Description: buildDescription(name, desc),
		State:       registry.StatePending,
		Source:      "legacy32",
	}
	return b.reg.Register(svc)
}

// LookupService returns the registry.Service entry for a Legacy32 service
// name, reporting whether it was found.
func (b *Bridge) LookupService(name string) (registry.Service, bool) {
	return b.reg.Lookup(name)
}

// buildDescription produces a human-readable description for a bridged
// service, combining any explicit description text with the capability set.
func buildDescription(name string, desc ServiceDescriptor) string {
	base := desc.Description
	if base == "" {
		base = fmt.Sprintf("legacy32 service %q", name)
	}
	caps := capabilityString(desc.Capabilities)
	if caps == "" {
		return fmt.Sprintf("%s (v%d)", base, desc.Version)
	}
	return fmt.Sprintf("%s (v%d, caps: %s)", base, desc.Version, caps)
}

// capabilityString converts a Capability bitmask into a compact human-readable
// string (e.g. "network|storage").  Returns an empty string if no known bits
// are set.
func capabilityString(c Capability) string {
	names := []struct {
		bit  Capability
		name string
	}{
		{CapNetwork, "network"},
		{CapStorage, "storage"},
		{CapDisplay, "display"},
		{CapInput, "input"},
		{CapAudio, "audio"},
	}
	out := ""
	for _, n := range names {
		if c&n.bit != 0 {
			if out != "" {
				out += "|"
			}
			out += n.name
		}
	}
	return out
}
