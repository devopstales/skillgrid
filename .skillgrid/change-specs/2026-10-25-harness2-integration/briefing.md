---
type: Spec
title: "2026-10-25-harness2-integration — briefing"
archive: "2026-10-25-harness2-integration"
status: draft

---

# Change: Harness² First-Class Host Integration

> **STATUS:** `ready-for-tasks`
>
> **Classification:** standard (fast-track ineligible) — no new Go module dependency, but it adds a subprocess trust boundary (Go spawns an external Python CLI) and new write REST routes, so it clears the `fast_track.blocked_when` bar.
>
> **Tier:** T2 (project default; full blueprint + two-axis review).
>
> **Build shape:** Tracer thread — the thinnest end-to-end path (CLI spawn → parse one run's `judged.json` → HTTP route → SPA table) is built first, then thickens.

## Goal

Integrate [Harness²](https://github.com/google-research/harness2) — a Python 3.12/3.13 agent-benchmarking + harness-optimization engine (arXiv:2610.14224, Apache-2.0) — into skillgrid-v2 as a **first-class host**. skillgrid does *not* optimize itself: it spawns the external Python CLI, manages its lifecycle, and surfaces runs, progress, and judged results through the existing CLI, REST, and embedded SPA.

Harness² runs a contrastive **Propose–Probe–Compose** loop that optimizes the *harness around a frozen model* (prompts, tools, memory, skills, subagents, guardrails, scripts, plugins) over professional-work benchmarks (JobBench, Harvey Labs, WorkBuddy-Bench). skillgrid's job is to be the **runner/host**, not the subject.

## Problem

A developer running Harness² today has to: (1) install a Python 3.12 + `uv` + Node 20 + `gcloud` stack by hand; (2) export a wall of `HARNESS2_*` env vars; (3) run `harness2` / `harness2-judge` / `harness2-report` from a shell; (4) dig through a `runs/` directory of JSON to understand what happened. None of this is visible in the tool they already use daily.

skillgrid already:
- ships an **external-CLI install flow** (`skillgrid-cli/internal/install/`, the `SecTool`/`installSecurityTools` pattern — trivy, nuclei, wapiti3, akca are all `uv`/`go`-managed CLIs shell-ed out to, non-fatally);
- exposes a **local-first HTTP/REST + embedded SPA** (`mnemonic serve`, `internal/http/`, write routes gated by `SKILLGRID_HTTP_TOKEN`);
- has a **doctor/capability** pattern for external-tool preflight;
- persists **per-project state under `~/.skillgrid/mnemonic/`**.

Harness² fits the same slot as the security scanners: an external, `uv`-managed CLI that skillgrid *hosts*, with a stable file contract (`runs/`) to surface. The difference is it is long-running and produces a results tree, so it also earns first-class REST + SPA presence.

## Design Decisions

1. **skillgrid is host, not subject.** Out of scope: using Harness² to benchmark/optimize the skillgrid harness itself, and implementing a Python `SubstrateLike`/`BenchmarkAdapter` against skillgrid memory + code index. Those are future changes. This change makes Harness² *runnable and observable from skillgrid*.

2. **Go spawns the Python CLI; no Go↔Python import boundary.** skillgrid's cgo-free single-binary invariant (pure-Go `modernc.org/sqlite`) is preserved because Harness² is an external process, not a `go.mod` dependency. No new Go module dependency is added — the change uses only `os/exec`, `encoding/json`, stdlib. The Python runtime is a *machine-level* dependency installed by the installer (decision 3), recorded as an ADR per the "no new dependency without an ADR" rule (ADR-0008 precedent).

3. **Harness² is a first-class `ExternalTool` in the install flow**, alongside trivy/nuclei/wapiti3/akca. `harness2` is a `uv`-managed package (`uv pip install harness2` / `uv tool install harness2`), non-fatal: skip if on PATH, warn if `uv`/Node missing, respect `--dry-run`. The installer documents the external binaries Harness² needs (`uv`, Node 20+, `gcloud` for real runs; Docker for WorkBuddy; `harness2-demo` runs offline with none).

4. **`runs/` lives under the skillgrid data dir** — `~/.skillgrid/mnemonic/harness2/runs/` — set via `HARNESS2_RUNS_DIR`. Rationale: local-first, survives re-clones, consistent with the existing store layout, and a single place `doctor`, the CLI, and the SPA all read. (This supersedes the per-project-location option.)

5. **New `mnemonic harness …` subcommand group** (mirrors the `mnemonic ctx …` subcommand pattern, ADR-0034): `setup`, `check`, `run`, `judge`, `report`, `list`, `status`. One file each under `cmd/mnemonic/`, stdlib `flag` only. Each command shells out to the corresponding `harness2-*` CLI entry point and, for `list`/`status`, reads the `runs/` file contract directly (no Python round-trip).

6. **New read-heavy REST routes** under `internal/http` (write routes token-gated like all other write routes): `GET /harness/runs`, `GET /harness/runs/{id}`, `POST /harness/runs` (launch a sweep), `POST /harness/runs/{id}/judge`. Read routes stay open, matching the existing surface.

7. **New SPA view** (Vite + React, `skillgrid-ui/`): a Harness runs list, per-run task table with judge scores, base-vs-harnessed delta, and adopted-harness diff. Read-heavy; reuses existing table/SP patterns.

8. **Threat matrix is non-trivial** (subprocess + file contract + REST writes). The Go side is *defense in depth*: it never executes a shell string, it parses `runs/` defensively, and it validates run identity before resume. See the Threat Matrix in the blueprint.

## Out of Scope (explicit)

- skillgrid memory/skills as the *optimized* harness (the "evolve skillgrid" variant) — deferred.
- A Python `SubstrateLike`/`BenchmarkAdapter` implementation that plugs skillgrid into Harness² as the c3 "memory"/"skills" components.
- Packaging Harness² into the skillgrid binary (it stays external via `uv`).
- Vertex AI credential management (skillgrid surfaces `gcloud` availability in `doctor` but does not manage the credential itself).

## Research Sources

| Source | URL | Key takeaway |
|---|---|---|
| Harness² README | `github.com/google-research/harness2` | Python 3.12/3.13, POSIX/Linux, Apache-2.0. Propose–Probe–Compose loop optimizing the harness around a frozen model. 8 editable components (system prompt, guardrails, memory, tool guidance, subagents, skills, scripts, plugins). Benchmarks: JobBench, Harvey Labs (LAB), WorkBuddy-Bench. |
| Harness² pyproject | `github.com/google-research/harness2/blob/main/pyproject.toml` | CLI entry points: `harness2` (run_sweep), `harness2-judge`, `harness2-report`, `harness2-setup`, `harness2-check`, `harness2-demo`, `harness2-bridge`. Core deps: zero third-party. Extras: `.[render]` (openpyxl, python-pptx), `.[bridge]` (litellm, fastapi, google-auth). |
| Harness² core/spec.py | `github.com/google-research/harness2/blob/main/harness2/core/spec.py` | `BenchSpec` (frozen, one (benchmark, substrate) cell), `BenchmarkAdapter` protocol (`tasks`, `execute`, `score`, `trajectory`, …), `ChannelSpec` (claude_cli/opencode). |
| Harness² substrate/base.py | `github.com/google-research/harness2/blob/main/harness2/substrate/base.py` | `Substrate`/`SubstrateLike` protocol (`surface`, `materialise`, `execute`, `read_trajectory`, `preflight`) — the agent-framework seam. The Python-side extension point (out of scope here). |
| Harness² SETUP | `github.com/google-research/harness2/blob/main/docs/SETUP.md` | External binaries: Node 20+, uv, gcloud CLI, Docker (WorkBuddy), Pandoc/Poppler (LAB). `harness2-demo` runs offline with no API keys. |
| skillgrid install flow | `skillgrid-cli/internal/install/install.go` | `SecTool`/`installSecurityTools` pattern: `Manager` (uv/go) + `InstallArgs` + `Bin` + `Hint`; non-fatal, `--dry-run`, captures output. `uv` already a prerequisite. |
| skillgrid HTTP surface | `mnemonic/internal/http/` | `serve` binds 127.0.0.1:7438; write routes via `requireWriteAuth` (`SKILLGRID_HTTP_TOKEN`); OpenAPI + Swagger UI; embedded SPA via `//go:embed`. |
| skillgrid CLI pattern | `mnemonic/cmd/mnemonic/` | One file per subcommand, stdlib `flag`; `mnemonic ctx …` is the precedent subcommand group (ADR-0034). |

## Trust Boundaries

- **Subprocess trust boundary (new).** Go spawns `harness2-*` as a child process. Mitigations: fixed-argv exec (no shell string interpolation), env allow-list (only `HARNESS2_*` + PATH inherited, no secret passthrough beyond the documented `HARNESS2_*` model vars), stdout/stderr captured into the existing session/traffic logging, and a hard per-run timeout.
- **File-contract trust boundary (new).** skillgrid reads the external `runs/` tree (JSON). Parsed defensively: unknown/extra fields ignored, malformed JSON → the run is reported as `unparsed` rather than crashing the server, and run identity (`config.json`) is validated before resume.
- **REST write routes (new).** `POST /harness/runs` and `POST /harness/runs/{id}/judge` spawn a process — gated by `SKILLGRID_HTTP_TOKEN` like every other write route.

## Verification Floor

L2 (full suite + build) for the standard change; the file-contract parser and run-identity validation are the risk surface, so they carry L3 unit coverage (fixture `runs/` trees). The end-to-end spawn path is verified by an env-gated integration test (`harness2-demo`, offline) that is skipped when the CLI is absent.

## Rollback

- **Code:** revert the Go changes (`internal/harness/`, `cmd/mnemonic/harness*.go`, `internal/http` routes, SPA view). Additive — no migration, no existing route touched.
- **Config:** remove `harness2` from the install tool list; `HARNESS2_RUNS_DIR` reverts to Harness² default.
- **Machine:** `uv tool uninstall harness2` removes the CLI; `~/.skillgrid/mnemonic/harness2/` can be deleted.

## Open Questions

None. All decisions resolved during planning (runs location = data dir, host-not-subject scope, Go-spawns-Python process model).
