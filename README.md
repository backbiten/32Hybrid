# 32Hybrid

A 32-bit compatibility appliance system combining gRPC services, Azure Blob storage, and Wine virtualization to run Win32 binaries on modern systems.

## Features

- **AI Contemplation Period**: Mandatory 15-minute synchronization phase ensuring perfect i386 architecture empathy
- **Legacy32 Support**: Kernel-level 32-bit compatibility layer
- **WinStratch UI**: Windows 2000-style interface for system interaction
- **Distributed Architecture**: Control plane and runner agents for scalable execution
- **Azure Integration**: Blob storage for binary distribution and result collection

## Quick Start

### Build

```bash
# Build all components
make

# Build C legacy components
cd legacy32
make
```

### Run Tests

```bash
# Run all tests
make test

# Test contemplation period
./scripts/test-contemplation.sh
```

### Demo

```bash
# Quick demo (skips 15-minute wait)
./contemplation-demo --skip-wait

# Full demo with contemplation period
./contemplation-demo
```

## Components

### Control Plane
gRPC server that accepts and manages run submissions.
- Port: 50051
- Config: YAML-based configuration

### Runner Agent
Windows-hosted gRPC service that executes Win32 binaries.
- Port: 5443
- TLS: Mutual TLS authentication

### AVD Client
CLI tool for submitting and managing runs.

### Legacy32
Kernel-level 32-bit compatibility layer with contemplation period enforcement.

### WinStratch
Windows 2000-style UI components for system interaction.

### AI Teacher
AI component that ensures perfect i386 architecture understanding through contemplation.

## Contemplation Period

The AI Contemplation Period is a mandatory 15-minute synchronization phase that ensures the AI Teacher operates with perfect empathy for the underlying i386 architecture.

For detailed information, see [CONTEMPLATION.md](CONTEMPLATION.md).

### Quick Info

- **Duration**: 15 minutes (900 seconds)
- **Phases**: 8 phases covering GDT, IDT, Memory Segmentation, Paging, ISA, Protected Mode, I/O, and Open386
- **Location**: `legacy32/contemplation.c`, `internal/winstratch/contemplation.go`, `internal/teacher/teacher.go`
- **Files**: `/tmp/contemplation_progress`, `/tmp/neural_registry_unlocked`

## Documentation

- [Contemplation Period Specification](CONTEMPLATION.md)
- [Detailed Contemplation Docs](docs/contemplation-period.md)
- [HV Architecture](docs/hv/architecture.md)
- [Test Plan](docs/hv/test-plan.md)

## Third-Party Components

| Component | URL | Description |
|-----------|-----|-------------|
| `third_party/kali` | https://www.kali.org/ | security auditing and penetration testing tools |
| `third_party/ipfire` | https://www.ipfire.org/ | versatile and state-of-the-art Open Source firewall |

## License

See LICENSE file for details.
