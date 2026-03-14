package registry_test

import (
	"testing"

	"github.com/backbiten/32Hybrid/internal/registry"
)

func TestMemRegistry_RegisterAndLookup(t *testing.T) {
	reg := registry.New()

	err := reg.Register(registry.Service{
		Name:        "network",
		Description: "TCP/IP network stack",
		Source:      "legacy32",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	svc, ok := reg.Lookup("network")
	if !ok {
		t.Fatal("Lookup: expected service to exist")
	}
	if svc.Name != "network" {
		t.Errorf("Name: got %q, want %q", svc.Name, "network")
	}
	if svc.Description != "TCP/IP network stack" {
		t.Errorf("Description: got %q, want %q", svc.Description, "TCP/IP network stack")
	}
	if svc.Source != "legacy32" {
		t.Errorf("Source: got %q, want %q", svc.Source, "legacy32")
	}
}

func TestMemRegistry_LookupMissing(t *testing.T) {
	reg := registry.New()
	_, ok := reg.Lookup("nonexistent")
	if ok {
		t.Error("Lookup: expected false for unregistered service")
	}
}

func TestMemRegistry_RegisterEmptyName(t *testing.T) {
	reg := registry.New()
	err := reg.Register(registry.Service{Name: ""})
	if err == nil {
		t.Error("Register: expected error for empty name, got nil")
	}
}

func TestMemRegistry_List(t *testing.T) {
	reg := registry.New()
	for _, name := range []string{"display", "storage", "network"} {
		if err := reg.Register(registry.Service{Name: name}); err != nil {
			t.Fatalf("Register %q: %v", name, err)
		}
	}
	list := reg.List()
	if len(list) != 3 {
		t.Errorf("List: got %d items, want 3", len(list))
	}
}

func TestMemRegistry_Deregister(t *testing.T) {
	reg := registry.New()
	_ = reg.Register(registry.Service{Name: "network"})
	reg.Deregister("network")
	_, ok := reg.Lookup("network")
	if ok {
		t.Error("Lookup after Deregister: expected service to be absent")
	}
	// Deregistering a service that doesn't exist must not panic.
	reg.Deregister("network")
}

func TestMemRegistry_Replace(t *testing.T) {
	reg := registry.New()
	_ = reg.Register(registry.Service{Name: "network", Description: "old"})
	_ = reg.Register(registry.Service{Name: "network", Description: "new"})

	svc, ok := reg.Lookup("network")
	if !ok {
		t.Fatal("Lookup: expected service to exist after replacement")
	}
	if svc.Description != "new" {
		t.Errorf("Description: got %q, want %q", svc.Description, "new")
	}
}

func TestService_StartStop(t *testing.T) {
	svc := &registry.Service{Name: "display", State: registry.StatePending}

	if err := svc.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if svc.State != registry.StateRunning {
		t.Errorf("State after Start: got %v, want running", svc.State)
	}

	svc.Stop()
	if svc.State != registry.StateStopped {
		t.Errorf("State after Stop: got %v, want stopped", svc.State)
	}
}

func TestService_StartEmptyName(t *testing.T) {
	svc := &registry.Service{}
	if err := svc.Start(); err == nil {
		t.Error("Start: expected error for empty name, got nil")
	}
}

func TestState_String(t *testing.T) {
	cases := []struct {
		s    registry.State
		want string
	}{
		{registry.StatePending, "pending"},
		{registry.StateRunning, "running"},
		{registry.StateStopped, "stopped"},
		{registry.StateError, "error"},
		{registry.State(99), "unknown(99)"},
	}
	for _, tc := range cases {
		if got := tc.s.String(); got != tc.want {
			t.Errorf("State(%v).String() = %q, want %q", int(tc.s), got, tc.want)
		}
	}
}
