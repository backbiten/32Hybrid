# 32HybridHV — Vision

## Goals

- Run **existing Win32 (32-bit) binaries unchanged** inside a strict, reproducible VM boundary.
- Provide a **compatibility-first** runtime appliance so legacy software continues to function correctly past the **2038-01-19** Unix time rollover and related time-representation hazards.
- Keep the operational surface simple: a single gRPC/REST control-plane API, a host CLI, and a guest agent — no modifications to the binaries being executed.
- Support LAN-reachable deployment so multiple hosts on a local network can submit jobs to a shared 32HybridHV instance.

## Non-Goals

- Rewriting or patching the Win32 applications themselves.
- Providing a universal fix for every possible 2038-related failure mode inside every binary (some failures are inherent to closed-source code and cannot be mediated externally).
- Supporting architectures other than **Windows x64 host + Hyper-V** in v0.1 (dual-boot, bare-metal Linux, cloud VMs are deferred).
- Implementing a full .NET / CIL runtime — the control plane is REST/gRPC only.

## Scope

| In scope (v0.1) | Out of scope (v0.1) |
|---|---|
| Hyper-V VM provisioning (Windows x64 host) | Dual-boot support |
| Linux guest (64-bit kernel, 32-bit userland) | Cloud-hosted guest (Azure remote execution) |
| Wine 32-bit runner inside guest | DOS emulation (future lane) |
| LD_PRELOAD time-mediation shim for Wine | Full Wine fork / patching |
| gRPC control plane + REST health endpoint | .NET / CIL host app |
| Token-based auth + IP allowlist | mTLS (planned for v0.2) |
| Post-2038 test harness | End-user GUI |

## Compatibility-First Promise

The core guarantee of 32HybridHV is **non-interference**: the binaries you submit are executed without modification. The appliance wraps them in a controlled environment that:

1. Presents a **64-bit-safe time model** to the Wine layer via a userland shim, so common time API calls (`time()`, `gettimeofday()`, `clock_gettime()`) return values that are correct past 2038-01-19.
2. Isolates each application in its own Wine prefix so cross-app state corruption is prevented.
3. Provides deterministic, reproducible execution by pinning the guest OS image and Wine version.

## What 2038 Means for This Project

The **Year 2038 Problem** (Y2K38) occurs when a 32-bit signed integer used to store Unix timestamps overflows at **03:14:07 UTC on 2038-01-19**. After that point, systems relying on a 32-bit `time_t` will interpret times as negative numbers or wrap to 1901, causing:

- Incorrect timestamps in logs, databases, and file metadata
- Authentication and certificate validation failures (token expiry, TLS cert checks)
- Scheduling and cron-style job misfires
- Silent data corruption in any application that computes time deltas

**What 32HybridHV does about it:**

- The guest Linux kernel is built with 64-bit `time_t` (standard on 64-bit kernels with 32-bit userland via multiarch).
- The Wine layer is configured and shimmed so that Win32 time APIs (`GetSystemTime`, `GetLocalTime`, `time()` via msvcrt) return values computed from a 64-bit source.
- A test suite runs the appliance with a virtualized clock set to post-2038 dates, verifying that known-bad patterns are caught and handled.
- Applications whose internal logic hard-codes 32-bit time will still fail internally — but the *runtime infrastructure* will not amplify those failures.

## How the VM Boundary Helps

Isolation is the key architectural benefit:

- The **guest VM is expendable**: if an app crashes, corrupts its prefix, or behaves unexpectedly around a time boundary, the damage is contained to that VM instance.
- The **host never runs the Win32 binary directly**: the host CLI only talks to the guest agent over a versioned API. This means the host stays clean regardless of what the guest app does.
- **Reproducibility**: you can snapshot the VM state before a test run, execute, observe, and revert — making 2038/2042 regression testing practical.
- **Upgrade path**: the guest image can be replaced (newer Wine, patched libc) without changing the host CLI or the API contract.
