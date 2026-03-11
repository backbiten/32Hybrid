# 32Hybrid
A reinvention and reinnovation of 32 bit architecture from the 70's and 80's

---

## 32HybridHV

**32HybridHV** is the first subproject inside this repository. It is a **compatibility-first** runtime appliance for running existing Win32 (32-bit) binaries unchanged, while ensuring they continue to work correctly past the [Year 2038 problem](https://en.wikipedia.org/wiki/Year_2038_problem).

### Approach

| Layer | Technology |
|---|---|
| **Host** | Windows x64 + Hyper-V |
| **Guest VM** | Linux (64-bit kernel, 32-bit userland/multiarch) |
| **Win32 runner** | Wine 32-bit (inside the guest VM) |
| **2038 mitigation** | LD_PRELOAD time shim + comprehensive post-2038 test suite |
| **Control plane** | gRPC (port 50051, primary) + REST health (port 8080) |
| **Auth** | Bearer token in gRPC metadata + IP allowlist (v0.1) |

### Key design decisions

- **Binaries are never modified.** The appliance wraps them in a controlled VM boundary and mediates key OS interfaces (especially time) to prevent known failure modes.
- **VM isolation** means crashes, corrupted Wine prefixes, and unexpected behaviour are contained in the guest — the host stays clean.
- **LAN-reachable API** lets multiple machines submit jobs to a shared 32HybridHV instance over gRPC.

### Documentation

| Document | Description |
|---|---|
| [docs/hv/vision.md](docs/hv/vision.md) | Goals, non-goals, scope, compatibility-first promise, and what 2038 means |
| [docs/hv/architecture.md](docs/hv/architecture.md) | Components, network topology, ports, auth model, logging flow, threat model |
| [docs/hv/test-plan.md](docs/hv/test-plan.md) | Post-2038 test strategy, smoke tests, and test matrix |

### Repository structure

```
hv/
  api/          # Protobuf / gRPC service definitions
  host/         # Host CLI (Go) — Hyper-V lifecycle + gRPC client
  guest/        # Guest agent (Go) — gRPC server + Wine launcher
  shim/         # LD_PRELOAD time-mediation shim (C)
  scripts/      # Provisioning and build helper scripts
  tests/        # Integration and post-2038 regression tests
docs/hv/        # Architecture and planning documentation
```

### API skeleton

The gRPC service is defined in [`hv/api/32hybrid_hv.proto`](hv/api/32hybrid_hv.proto). Services:

- **HealthService** — liveness/readiness check
- **RunnerService** — `RunExe` (launch a Win32 binary) + `StreamLogs` (server-streaming log delivery)
- **FileService** — `PutFile` / `GetFile` (chunked file transfer)
