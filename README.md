# 32Hybrid
A reinvention and reinnovation of 32 bit architecture from the 70's and 80's

---

## 32HybridHV

**32HybridHV** is the first subproject inside this repository. It is a **compatibility-first** runtime appliance for running existing Win32 (32-bit) binaries unchanged, while ensuring they continue to work correctly past the Year 2038 problem.

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
| docs/hv/vision.md | Goals, non-goals, scope, compatibility-first promise, and what 2038 means |
| docs/hv/architecture.md | Components, network topology, ports, auth model, logging flow, threat model |
| docs/hv/test-plan.md | Post-2038 test strategy, smoke tests, and test matrix |

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

The gRPC service is defined in hv/api/32hybrid_hv.proto. Services:

- **HealthService** — liveness/readiness check
- **RunnerService** — RunExe (launch a Win32 binary) + StreamLogs (server-streaming log delivery)
- **FileService** — PutFile / GetFile (chunked file transfer)

---

## CPU emulator + assembler (Python)

This repo also includes a clean, well-tested Python implementation of a 32-bit CPU emulator and assembler.

### Key design decisions that prevent common bugs

| Classic 32-bit bug | How 32Hybrid avoids it |
|---|---|
| Ambiguous immediate encoding (flag bit collides with value bits) | rb == 0xF in the instruction word is a clean sentinel — the 12-bit immediate occupies its own field without overlap |
| Silent integer overflow | Carry (CF) and overflow (OF) flags are computed independently and correctly for both signed and unsigned arithmetic |
| Wrong sign extension on right-shift | SHR is always logical (zero-fills); SAR is always arithmetic (sign-extends) — no ambiguity |
| Division by zero corruption | Raises ZeroDivisionError before touching any register |
| Runaway programs | cpu.run(max_steps=N) hard-limits execution |
| Out-of-bounds memory access | Every access checks bounds and raises MemoryError |

### Files

| File | Purpose |
|---|---|
| cpu32.py | CPU core — registers, ALU, memory, instruction execution |
| assembler.py | Two-pass assembler — source text → list of 32-bit words |
| tests/test_cpu32.py | Unit + integration tests |

### Running tests

```bash
python -m pytest tests/
```