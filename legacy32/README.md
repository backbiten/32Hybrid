# deadpgp

Forking old forgotten PGP protocols used in the 1980s and 1990s—reframed for modern, safety-focused workflows.

## Current status (as of 2026-03-11)

- This repo is early-stage.
- Current tooling includes a small Python CLI that wraps `gpg --decrypt` (see `tools/openpgp_import/import.py`).

## Direction / Roadmap

This project is evolving toward an orchestration + policy layer over GnuPG/OpenPGP to support a *sequenced* "reveal" workflow with:

- **Electoral college approvals** (configured as a list of GPG key fingerprints + quorum rules)
- **Local AI-veto policy checks** (no network calls)
- **Auditable decisions** (JSONL audit log)

See `docs/SPEC.md` for the proposed protocol/state machine and configuration format.
