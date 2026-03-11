# Legacy 32 Integration

## What was imported

The contents of [`backbiten/deadpgp`](https://github.com/backbiten/deadpgp) have been
imported into this repository as a top-level subsystem called **Legacy 32**.

| Source path (deadpgp) | Destination path (32Hybrid) |
|-----------------------|-----------------------------|
| `README.md`           | `legacy32/README.md`        |
| `LICENSE`             | `legacy32/LICENSE`          |
| `tools/`              | `legacy32/tools/`           |

The import is a point-in-time snapshot from the `main` branch of `backbiten/deadpgp`
(commit `3e2ef00`).

## Where it lives

All Legacy 32 code resides under `legacy32/` at the root of this repository.
It is intentionally isolated from the Hyper 32 Go/Buf build system.

- Python tooling: `legacy32/tools/openpgp_import/import.py`
- License: `legacy32/LICENSE` (Eclipse Public License v2.0)

## Current state

- The Python CLI in `legacy32/tools/openpgp_import/import.py` wraps `gpg --decrypt`
  and represents the initial Legacy 32 cryptographic tooling.
- No build-system integration with the Hyper 32 Go modules has been attempted yet.
- No CI pipeline covers `legacy32/` in this PR.

## Next steps (suggested)

1. **Define interfaces** — decide which Legacy 32 operations Hyper 32 needs to call
   (e.g., decrypt, verify signature, audit-log) and specify them as gRPC or Go FFI
   interfaces in `proto/` or `internal/`.
2. **Decide language boundary** — options include: keep Python and invoke via
   subprocess/gRPC sidecar, rewrite critical paths in Go, or expose via a thin Go
   wrapper using `os/exec`.
3. **Wire up CI** — add a separate `legacy32/` job to the CI pipeline (e.g.,
   `pytest`, `flake8`) once the Python tooling matures.
4. **Incremental migration** — as interfaces solidify, move logic from
   `legacy32/tools/` into `internal/legacy32/` (Go) while keeping the Python
   reference implementation for comparison testing.
