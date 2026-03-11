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

---

## 32Hybrid AVD System

**32Hybrid AVD** is the second subsystem in this repository.  It lets a user
running inside an **Azure Virtual Desktop (AVD)** session submit a Windows GUI
application to run in their own interactive desktop — with artifacts (exit
code, stdout, stderr) uploaded to Azure Blob Storage.

### Architecture

```
 ┌─────────────────────────────┐        ┌────────────────────────────────┐
 │  AVD Session (user's VM)    │        │  Linux Control Plane VM        │
 │                             │ gRPC   │  (private VNet, peered)        │
 │  avdclient submit --exe ... │───────▶│                                │
 │                             │        │  1. Upload EXE → Azure Blob    │
 └─────────────────────────────┘        │  2. Mint SAS URLs              │
                                        │  3. Discover runner (Azure API)│
                                        │  4. ListSessions via mTLS gRPC │
                                        │  5. RunInSession               │
                                        └──────────────────┬─────────────┘
                                                           │ gRPC mTLS :5443
                                        ┌──────────────────▼─────────────┐
                                        │  Runner Agent (Windows host)   │
                                        │                                │
                                        │  1. Download EXE via SAS       │
                                        │  2. Launch via Scheduled Task  │
                                        │  3. Upload exit.json via SAS   │
                                        └────────────────────────────────┘
```

**Components:**

| Component | Binary | Description |
|-----------|--------|-------------|
| AVD Client | `cmd/avdclient` | CLI inside the AVD session; submits and queries runs |
| Control Plane | `cmd/controlplane` | Linux VM; orchestrates blobs, SAS, discovery, runner RPC |
| Runner Agent | `cmd/runner` | Windows session host; downloads EXE, runs it, uploads artifacts |

**Networking:**
- AVD client → Control Plane: gRPC (plaintext dev / TLS prod), port `50051`
- Control Plane → Runner Agent: gRPC over **mTLS**, port `5443`
- VNets are **peered**; NSG must allow TCP 5443 from the control-plane subnet

### Proto definitions

| File | Package | Purpose |
|------|---------|---------|
| [`proto/common/v1/common.proto`](proto/common/v1/common.proto) | `common.v1` | Shared enums (`RunState`, `SessionState`) |
| [`proto/controlplane/v1/controlplane.proto`](proto/controlplane/v1/controlplane.proto) | `controlplane.v1` | AVD client ↔ control plane API |
| [`proto/runner/v1/runner.proto`](proto/runner/v1/runner.proto) | `runner.v1` | Control plane ↔ runner agent API |

Generated Go code lives in `gen/`.

### How to generate protos

**Prerequisites:** `protoc` ≥ 3.21, Go ≥ 1.21.

```sh
make proto          # installs protoc-gen-go + protoc-gen-go-grpc, then generates
```

Or manually:

```sh
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

protoc --proto_path=proto \
       --go_out=gen --go_opt=paths=source_relative \
       proto/common/v1/common.proto

protoc --proto_path=proto \
       --go_out=gen --go_opt=paths=source_relative \
       --go-grpc_out=gen --go-grpc_opt=paths=source_relative \
       --go-grpc_opt=require_unimplemented_servers=false \
       proto/controlplane/v1/controlplane.proto

protoc --proto_path=proto \
       --go_out=gen --go_opt=paths=source_relative \
       --go-grpc_out=gen --go-grpc_opt=paths=source_relative \
       --go-grpc_opt=require_unimplemented_servers=false \
       proto/runner/v1/runner.proto
```

### Build all binaries

```sh
make build
# Produces: bin/controlplane  bin/runner  bin/avdclient
```

### Running locally (development)

**Control plane** (no Azure credentials needed for dev):
```sh
bin/controlplane --config controlplane.yaml
# or without a config file (uses built-in dev defaults on :50051)
bin/controlplane
```

**Runner agent** (no TLS certs needed for dev):
```sh
bin/runner --config runner.yaml
# or without a config file (listens on :5443, stub session enumerator)
bin/runner
```

**AVD client:**
```sh
bin/avdclient --addr localhost:50051 submit --exe notepad.exe -- /A somefile.txt
bin/avdclient --addr localhost:50051 get --run-id <id>
bin/avdclient --addr localhost:50051 list
```

### Sample config files

**controlplane.yaml:**
```yaml
listen_addr: ":50051"
storage:
  account_name: "mystorageaccount"
  account_key: "BASE64_ACCOUNT_KEY_PLACEHOLDER"
  uploads_container: "uploads"
  runs_container: "runs"
  sas_ttl_seconds: 3600
azure:
  subscription_id: "00000000-0000-0000-0000-000000000000"
  resource_group: "rg-avd"
  host_pool_name: "hp-personal"
  target_username: "DOMAIN\\myuser"
runner:
  host: "10.1.2.3"    # static IP for v0.1; leave empty to use Azure discovery
  port: 5443
  client_cert_file: "/etc/32hybrid/certs/cp-client.crt"
  client_key_file:  "/etc/32hybrid/certs/cp-client.key"
  ca_file:          "/etc/32hybrid/certs/ca.crt"
```

**runner.yaml** (on Windows session host):
```yaml
listen_addr: ":5443"
server_cert_file: "C:\\32hybrid\\certs\\runner.crt"
server_key_file:  "C:\\32hybrid\\certs\\runner.key"
ca_file:          "C:\\32hybrid\\certs\\ca.crt"
work_dir:         "C:\\32hybrid\\runs"
default_timeout_seconds: 300
```

### Azure / AVD prerequisites

1. **Storage account** with containers `uploads` and `runs` (Blob service).
   Copy the account key to `controlplane.yaml`.
2. **AVD host pool** (Personal type) in the same or peered VNet.
3. **VNet peering** between the control-plane VNet and the AVD session host VNet
   with bidirectional access enabled.
4. **NSG rule** on the AVD session host subnet: allow inbound TCP 5443 from
   the control-plane VM private IP.
5. **mTLS certs**: generate a CA and issue a cert for both the control plane
   (client role) and the runner agent (server role).  Self-signed is fine for
   MVP.
6. Install `runner.exe` as a Windows service on the session host.

### Repository structure (AVD subsystem)

```
proto/
  common/v1/         # Shared enums
  controlplane/v1/   # AVD client ↔ control plane gRPC
  runner/v1/         # Control plane ↔ runner gRPC
gen/                 # Generated Go code (committed; do not edit manually)
cmd/
  controlplane/      # Control plane binary
  runner/            # Runner agent binary
  avdclient/         # AVD client CLI
internal/
  config/            # YAML config types for all three binaries
  controlplane/      # gRPC server implementation
  runner/            # gRPC server implementation + stub session/launcher
  sas/               # Azure Blob SAS URL generation (HMAC-SHA256)
  discovery/         # Runner host discovery (static + Azure stub)
  store/             # In-memory run record store (MVP)
  runnerutil/        # exit.json schema shared by runner and control plane
Makefile             # proto, build, test targets
buf.yaml             # buf lint/breaking config
buf.gen.yaml         # buf code generation config (uses remote plugins)
```
