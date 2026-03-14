package legacy32bridge_test

import (
	"strings"
	"testing"

	"github.com/backbiten/32Hybrid/internal/legacy32bridge"
	"github.com/backbiten/32Hybrid/internal/registry"
)

func TestBridge_RegisterAndLookup(t *testing.T) {
	reg := registry.New()
	bridge := legacy32bridge.New(reg)

	err := bridge.RegisterService("network", legacy32bridge.ServiceDescriptor{
		Version:      1,
		Capabilities: legacy32bridge.CapNetwork,
		Description:  "TCP/IP network stack",
	})
	if err != nil {
		t.Fatalf("RegisterService: %v", err)
	}

	svc, ok := bridge.LookupService("network")
	if !ok {
		t.Fatal("LookupService: expected service to exist")
	}
	if svc.Name != "network" {
		t.Errorf("Name: got %q, want %q", svc.Name, "network")
	}
	if svc.Source != "legacy32" {
		t.Errorf("Source: got %q, want %q", svc.Source, "legacy32")
	}
	if svc.State != registry.StatePending {
		t.Errorf("State: got %v, want pending", svc.State)
	}
}

func TestBridge_RegisterEmptyName(t *testing.T) {
	bridge := legacy32bridge.New(registry.New())
	err := bridge.RegisterService("", legacy32bridge.ServiceDescriptor{Version: 1})
	if err == nil {
		t.Error("RegisterService with empty name: expected error, got nil")
	}
}

func TestBridge_RegisterVersionZero(t *testing.T) {
	bridge := legacy32bridge.New(registry.New())
	err := bridge.RegisterService("storage", legacy32bridge.ServiceDescriptor{Version: 0})
	if err == nil {
		t.Error("RegisterService with version 0: expected error, got nil")
	}
}

func TestBridge_LookupMissing(t *testing.T) {
	bridge := legacy32bridge.New(registry.New())
	_, ok := bridge.LookupService("nonexistent")
	if ok {
		t.Error("LookupService: expected false for unregistered service")
	}
}

func TestBridge_DescriptionIncludesVersion(t *testing.T) {
	reg := registry.New()
	bridge := legacy32bridge.New(reg)
	_ = bridge.RegisterService("display", legacy32bridge.ServiceDescriptor{
		Version:      3,
		Capabilities: legacy32bridge.CapDisplay,
	})
	svc, _ := reg.Lookup("display")
	if !strings.Contains(svc.Description, "v3") {
		t.Errorf("Description %q: expected to contain version string", svc.Description)
	}
}

func TestBridge_DescriptionIncludesCapabilities(t *testing.T) {
	reg := registry.New()
	bridge := legacy32bridge.New(reg)
	_ = bridge.RegisterService("io", legacy32bridge.ServiceDescriptor{
		Version:      1,
		Capabilities: legacy32bridge.CapStorage | legacy32bridge.CapInput,
	})
	svc, _ := reg.Lookup("io")
	for _, cap := range []string{"storage", "input"} {
		if !strings.Contains(svc.Description, cap) {
			t.Errorf("Description %q: expected to contain capability %q", svc.Description, cap)
		}
	}
}

func TestBridge_MultipleServices(t *testing.T) {
	reg := registry.New()
	bridge := legacy32bridge.New(reg)

	services := []struct {
		name string
		desc legacy32bridge.ServiceDescriptor
	}{
		{"network", legacy32bridge.ServiceDescriptor{Version: 1, Capabilities: legacy32bridge.CapNetwork}},
		{"storage", legacy32bridge.ServiceDescriptor{Version: 2, Capabilities: legacy32bridge.CapStorage}},
		{"display", legacy32bridge.ServiceDescriptor{Version: 1, Capabilities: legacy32bridge.CapDisplay}},
	}

	for _, s := range services {
		if err := bridge.RegisterService(s.name, s.desc); err != nil {
			t.Fatalf("RegisterService %q: %v", s.name, err)
		}
	}

	if got := len(reg.List()); got != len(services) {
		t.Errorf("registry.List: got %d entries, want %d", got, len(services))
	}
}

// TestBridge_NewNilPanics verifies that passing a nil Registry panics
// immediately rather than producing a silent nil-pointer dereference later.
func TestBridge_NewNilPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("New(nil): expected panic, got none")
		}
	}()
	legacy32bridge.New(nil)
}

// TestEndToEnd_ServiceHandshake exercises the first end-to-end service
// handshake described in the issue: a Legacy32 service registers itself,
// and 32Hybrid userland discovers and starts it via the Root Registry.
func TestEndToEnd_ServiceHandshake(t *testing.T) {
	// Boot phase: Legacy32 nano-kernel registers "network" via the bridge.
	reg := registry.New()
	bridge := legacy32bridge.New(reg)
	err := bridge.RegisterService("network", legacy32bridge.ServiceDescriptor{
		Version:      1,
		Capabilities: legacy32bridge.CapNetwork,
		Description:  "TCP/IP network stack",
	})
	if err != nil {
		t.Fatalf("boot-time registration: %v", err)
	}

	// Userland phase: 32Hybrid app looks up "network" in the Root Registry,
	// starts it (mutates the local copy), then persists the updated state by
	// re-registering it.
	svc, ok := reg.Lookup("network")
	if !ok {
		t.Fatal("userland lookup: service not found")
	}
	if err := svc.Start(); err != nil {
		t.Fatalf("service Start: %v", err)
	}
	// Persist the updated state back to the registry.
	if err := reg.Register(svc); err != nil {
		t.Fatalf("re-register after Start: %v", err)
	}

	// Verify the registry entry now reflects the running state.
	persisted, ok := reg.Lookup("network")
	if !ok {
		t.Fatal("lookup after re-register: service not found")
	}
	if persisted.State != registry.StateRunning {
		t.Errorf("state after re-register: got %v, want running", persisted.State)
	}
}
