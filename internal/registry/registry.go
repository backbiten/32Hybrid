// Package registry implements the Root Registry for 32Hybrid/Legacy32 service
// discovery and orchestration.
//
// The Root Registry is the central authority for service lookup and
// registration across both the 32Hybrid modern runtime and the Legacy32
// nano-kernel subsystems. Services (drivers, network stacks, applications)
// register themselves at boot time or on demand; other subsystems discover
// and start them via the Lookup and Register methods.
//
// Example – registering a service:
//
//	reg := registry.New()
//	reg.Register(registry.Service{
//	    Name:        "network",
//	    Description: "TCP/IP network stack",
//	})
//
// Example – looking up a service (mirrors the Go pseudocode in the issue):
//
//	if svc, ok := reg.Lookup("network"); ok {
//	    if err := svc.Start(); err != nil { /* handle */ }
//	}
package registry

import (
	"fmt"
	"sync"
)

// State represents the lifecycle state of a registered service.
type State int

const (
	// StatePending means the service is registered but has not been started.
	StatePending State = iota
	// StateRunning means the service is active and ready to receive requests.
	StateRunning
	// StateStopped means the service has been stopped gracefully.
	StateStopped
	// StateError means the service encountered a fatal error.
	StateError
)

// String returns a human-readable representation of a State value.
func (s State) String() string {
	switch s {
	case StatePending:
		return "pending"
	case StateRunning:
		return "running"
	case StateStopped:
		return "stopped"
	case StateError:
		return "error"
	default:
		return fmt.Sprintf("unknown(%d)", int(s))
	}
}

// Service describes a single entry in the Root Registry.
type Service struct {
	// Name is the unique, stable identifier for this service (e.g. "network",
	// "storage", "display"). Names are case-sensitive and must not be empty.
	Name string

	// Description is an optional human-readable summary of the service.
	Description string

	// State reflects the current lifecycle of the service.
	State State

	// Source identifies the subsystem that registered the service (e.g.
	// "32hybrid", "legacy32"). Informational only.
	Source string
}

// Start transitions the service to StateRunning.
// It mutates the local copy of the Service; callers should call
// Registry.Register again with the updated value to persist the new state.
// Returns an error if the service has no name.
func (s *Service) Start() error {
	if s.Name == "" {
		return fmt.Errorf("registry: cannot start service with empty name")
	}
	s.State = StateRunning
	return nil
}

// Stop transitions the service to StateStopped.
func (s *Service) Stop() {
	s.State = StateStopped
}

// Registry is the interface for the Root Registry.
// All methods must be safe for concurrent use.
type Registry interface {
	// Register adds or replaces a service entry.
	// Returns an error if Name is empty.
	Register(svc Service) error

	// Lookup returns the service registered under name and true, or a zero
	// Service and false if no such service exists.
	Lookup(name string) (Service, bool)

	// List returns a snapshot of all registered services.
	List() []Service

	// Deregister removes the service with the given name.
	// It is not an error to deregister a service that is not registered.
	Deregister(name string)
}

// MemRegistry is an in-memory, thread-safe implementation of Registry.
// It is suitable for single-process use (MVP); replace with a persistent or
// distributed backend for multi-host deployments.
type MemRegistry struct {
	mu       sync.RWMutex
	services map[string]Service
}

// New returns an empty MemRegistry ready for use.
func New() *MemRegistry {
	return &MemRegistry{services: make(map[string]Service)}
}

// Register implements Registry.
func (r *MemRegistry) Register(svc Service) error {
	if svc.Name == "" {
		return fmt.Errorf("registry: service name must not be empty")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.services[svc.Name] = svc
	return nil
}

// Lookup implements Registry.
func (r *MemRegistry) Lookup(name string) (Service, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	svc, ok := r.services[name]
	return svc, ok
}

// List implements Registry.
func (r *MemRegistry) List() []Service {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Service, 0, len(r.services))
	for _, svc := range r.services {
		out = append(out, svc)
	}
	return out
}

// Deregister implements Registry.
func (r *MemRegistry) Deregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.services, name)
}
