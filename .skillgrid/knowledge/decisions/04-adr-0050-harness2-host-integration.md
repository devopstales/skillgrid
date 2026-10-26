---
type: Decision
title: "Harness² host integration: external subprocess, no Go dependency"
status: accepted
date: 2026-10-25
supersedes: []
superseded_by: []
---

# ADR-0050: Harness² is hosted as an external subprocess, not a Go dependency

## Status

Accepted (2026-10-25).

## Context

skillgrid-v2 integrates [Harness²](https://github.com/google-research/harness2) (Python 3.12/3.13, `uv`-managed, Apache-2.0) so a developer can launch, monitor, and read the judged results of an agent-harness optimization run from the tool they already use. The integration is **host, not subject**: skillgrid spawns and observes Harness²; it does not feed its own memory/skills into Harness² as the optimized harness, and it does not implement a Python `SubstrateLike`/`BenchmarkAdapter` (both deferred).

Constraints that shaped the decision:

- The repo's **cgo-free single-binary invariant** (pure-Go `modernc.org/sqlite`). A Go↔Python bridge (cffi, pybind11, cgo) would break it.
- The **"no new dependency without an ADR"** rule (ADR-0008 precedent).
- The **Distribution Surface** owns transports and no data ownership (ADR-0004); the **Installer** owns external-CLI lifecycle.
- Harness² has **no Go API and no MCP server**; its stable surface is a CLI (`harness2`, `harness2-judge`, `harness2-report`, `harness2-setup`, `harness2-check`, `harness2-demo`) and a file contract (`runs/` with `config.json`, `judged.json`, `base_judged.json`, task summaries, adopted harnesses).

## Decision

1. **Process model — Go spawns the Python CLI.** skillgrid invokes `harness2-*` as a child process via `os/exec`. There is no Go↔Python import boundary and no new Go module dependency: the change uses only `os/exec`, `encoding/json`, and stdlib. Harness² is a **machine-level** dependency installed by the installer (`uv`-managed), not a `go.mod` entry.

2. **`runs/` location.** The runs tree lives under the skillgrid data dir — `~/.skillgrid/mnemonic/harness2/runs/` — set via `HARNESS2_RUNS_DIR`. Rationale: local-first, survives re-clones, one place `doctor` / the CLI / the SPA all read, consistent with the existing store layout. The per-project-location option was rejected (runs are machine-level artifacts, not per-repo state, and would fragment across worktrees).

3. **CLI surface — `mnemonic harness …` subcommand group** (mirrors the `mnemonic ctx …` pattern, ADR-0034). Subcommands: `setup`, `check`, `run`, `judge`, `report`, `list`, `status`. `list`/`status` read the `runs/` file contract directly (no Python round-trip); the rest shell out to the matching `harness2-*` entry point.

4. **Distribution Surface.** New read-heavy REST routes under `internal/http`: `GET /harness/runs`, `GET /harness/runs/{id}`, `POST /harness/runs` (launch), `POST /harness/runs/{id}/judge`. Write routes token-gated by `SKILLGRID_HTTP_TOKEN` like all other write routes. A new SPA view renders the run list, per-run task table with judge scores, base-vs-harnessed delta, and adopted-harness diff.

5. **Subprocess + file-contract defense.** Fixed-argv exec (no shell string), env allow-list (only `HARNESS2_*` + `PATH` inherited), stdout/stderr captured into existing session/traffic logging, hard per-run timeout. The `runs/` parser is defensive: unknown fields ignored, malformed JSON → run reported `unparsed` (never a server crash), and run identity (`config.json`) validated before resume.

## Consequences

**Positive**
- cgo-free invariant preserved; no `go.mod` change; no CGo bridge, no GIL.
- The integration is additive and roll-back-able: deleting the Go files + the SPA view + the tool list entry removes it, with no migration.
- One `runs/` location means `doctor`, the CLI, and the SPA agree by construction.

**Negative / costs**
- Harness² is Linux/Python-3.12-only and needs `gcloud` (Vertex AI) for real runs; on macOS dev boxes the offline `harness2-demo` is the only path, so the end-to-end spawn integration test is env-gated and skipped when the CLI is absent.
- A new subprocess trust boundary is introduced (ADR-0005 is extended, not changed): a fixed-argv, env-allow-listed, timeout-bounded child, read-only against skillgrid state.

**Follow-ups (explicitly deferred to future changes)**
- Using Harness² to benchmark/optimize the skillgrid harness itself (skillgrid memory + code index as the c3 "memory"/"skills" components) — requires a Python `SubstrateLike`/`BenchmarkAdapter`.
- Vertex AI credential management (skillgrid reports `gcloud` availability; it does not manage the credential).
