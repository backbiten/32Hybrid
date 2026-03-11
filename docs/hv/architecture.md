# 32HybridHV — Architecture

## Overview

32HybridHV is a **compatibility appliance** composed of four main components:

```
┌─────────────────────────────────────────────────────────────┐
│  Windows x64 Host (Hyper-V)                                 │
│                                                             │
│  ┌─────────────────┐         ┌──────────────────────────┐  │
│  │  Host CLI        │──gRPC──▶│  Guest Agent (Linux VM)  │  │
│  │  32hybrid-hv     │         │                          │  │
│  └─────────────────┘         │  ┌────────────────────┐  │  │
│                               │  │  Wine 32-bit       │  │  │
│  ┌─────────────────┐         │  │  (Win32 runner)    │  │  │
│  │  LAN Client(s)  │──gRPC──▶│  └────────────────────┘  │  │
│  └─────────────────┘         │                          │  │
│                               │  ┌────────────────────┐  │  │
│                               │  │  Time Shim         │  │  │
│                               │  │  (LD_PRELOAD)      │  │  │
│                               │  └────────────────────┘  │  │
│                               └──────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Components

### Host CLI (`hv/host/`)

- Written in Go.
- Entry point: `32hybrid-hv`
- Responsibilities:
  - Provision and lifecycle-manage the Hyper-V Linux VM (`install`, `start`, `stop`, `status`).
  - Act as a gRPC client to the guest agent.
  - Provide user-facing commands: `run <exe> [args]`, `logs <run-id>`, `put-file`, `get-file`.
  - Handle Hyper-V networking (configure External vSwitch for LAN exposure).

### Guest Agent (`hv/guest/`)

- Written in Go, runs inside the Linux VM.
- Exposes the gRPC service defined in `hv/api/32hybrid_hv.proto`.
- Responsibilities:
  - Receive `RunExe` requests, launch Wine with an isolated prefix, stream stdout/stderr back.
  - Serve `PutFile` / `GetFile` for binary and artifact transfer.
  - Report health via a lightweight REST endpoint.
- Bind addresses:
  - gRPC: `0.0.0.0:50051`
  - REST health: `0.0.0.0:8080`

### Wine Runner (inside guest)

- Wine 32-bit (`wine` + `wine32` packages) installed in the guest image.
- Each application gets its own **Wine prefix** directory (e.g., `/var/32hybrid/prefixes/<run-id>/`) to prevent cross-app state leakage.
- Launched by the guest agent with the time shim preloaded.

### Time Shim (`hv/shim/`)

- A Linux shared library loaded via `LD_PRELOAD` for all Wine processes.
- Intercepts: `time()`, `gettimeofday()`, `clock_gettime()`, `localtime()`, `gmtime()`.
- Delegates to the kernel's 64-bit `clock_gettime(CLOCK_REALTIME)` and re-exposes via compatible structures.
- Goal: ensure that even when Wine or a Win32 app calls legacy libc time functions, the returned values are sourced from a 64-bit clock and are correct past 2038-01-19.

## Network Topology (LAN Mode)

```
LAN
 │
 ├── Windows Host ──[Hyper-V External vSwitch]──┐
 │                                              │
 ├── LAN Client A                           Linux Guest VM
 │                                          :50051 (gRPC)
 └── LAN Client B                           :8080  (REST)
```

- The Hyper-V VM is attached to an **External Virtual Switch** (bridged to the host's physical NIC).
- The guest VM receives a LAN IP address directly (via DHCP or static assignment).
- All LAN clients — including the Windows host CLI — connect to the guest's LAN IP.
- Firewall rules on the guest restrict inbound connections to the configured CIDR allowlist.

### Ports

| Service | Default Port | Protocol |
|---|---|---|
| gRPC control plane | 50051 | TCP (HTTP/2) |
| REST health / status | 8080 | TCP (HTTP/1.1) |

These defaults are set in `hv/guest/config.yaml` and can be overridden via environment variables or CLI flags.

## Auth Model

### gRPC (primary)

- Authentication uses a **bearer token** passed in gRPC metadata:
  ```
  authorization: Bearer <token>
  ```
- The guest agent validates the token against a configured secret on every RPC call.
- Token is generated at provisioning time and stored in the host config file (`~/.32hybrid-hv/config.yaml`).

### REST (health endpoint)

- Unauthenticated by default (only returns `{"status":"ok"}`).
- Can optionally require the same bearer token via the `Authorization` HTTP header if `rest_auth: true` is set in config.

### IP Allowlist

- Guest agent config accepts a list of allowed CIDRs:
  ```yaml
  allowlist:
    - 192.168.1.0/24
    - 10.0.0.1/32
  ```
- Connections from IPs not on the allowlist are rejected before token validation.

### Future: mTLS

mTLS is the target for v0.2. It provides mutual certificate-based authentication and eliminates the need for bearer tokens. The proto and agent are designed to be compatible with a TLS upgrade without breaking the API contract.

## Logging and Artifacts Flow

```
Win32 App (stdout/stderr)
    │
    ▼
Wine process (inside guest)
    │
    ▼
Guest Agent (captures via pipe)
    │
    ├── Streams in real-time via StreamLogs RPC
    │       └──► Host CLI / LAN Client (displays to terminal)
    │
    └── Writes to /var/32hybrid/runs/<run-id>/
            ├── stdout.log
            ├── stderr.log
            └── exit_code
```

- Run artifacts (logs, exit code, any output files) are persisted in the guest under `/var/32hybrid/runs/<run-id>/`.
- The host CLI can retrieve them after the run using `GetFile`.

## Threat Model Basics

| Threat | Mitigation |
|---|---|
| Unauthenticated API access | Bearer token required; IP allowlist |
| Token interception on LAN | mTLS planned for v0.2; use VPN on untrusted networks |
| Malicious Win32 binary | VM isolation (damage stays in guest); per-run Wine prefix; no host filesystem access |
| Time shim bypass by app | Shim covers libc entry points; Wine-level patches for Win32 time APIs as needed |
| Guest VM escape | Hyper-V isolation boundary; guest runs as unprivileged user |
| Log injection | Logs are streamed as structured `LogLine` proto messages; raw bytes are base64-encoded |
| Denial of service (run flood) | Rate limiting and max concurrent runs configurable in guest agent |
