# ADR Review Manifest

- **Change:** `2026-10-08-context-orchestrator`
- **Status:** in review
- **Review date:** 2026-10-08
- **Prerequisite plan:** `.skillgrid/artifacts/context-orchestrator-plan.md` v1.11 (phases 1–15 = v1.10 body, referenced as prerequisite)

## In-Force ADRs Reviewed

- `.skillgrid/artifacts/04-adr-0012-sqlite-as-second-brain.md` — local SQLite stays the per-machine source of truth. CTX adds `ctx_*` tables to the **same** shared store; it never opens its own DB handle (reads via the mnemonic module). No storage swap.
- `.skillgrid/artifacts/04-adr-0016-second-brain-capability-layer.md` — fail-open floors hold: the judgment layer degrades to deterministic pre-filter on model unavailability; the proxy fails open to passthrough on compression faults; traffic learning never blocks the session. No CTX component blocks the agent.
- `.skillgrid/artifacts/04-adr-0021-pre-tool-policy-fail-open.md` — the CTX pre-filter is deterministic and runs on actual content (not a prediction); `CTX_PROXY_MODE` is the only runtime forcing mechanism, consistent with the fail-open policy posture.
- `.skillgrid/artifacts/04-adr-0025-context-harness-owner.md` — Context Harness owns the session context lifecycle (capture / query / index / routing). CTX sits **above** this owner: it orchestrates the already-captured context (compression, judgment, retrieval ladder, proxy) without re-implementing capture or storage.
- `.skillgrid/artifacts/04-adr-0026-indexed-files-reuses-observations-schema.md` — `indexed_files` (a CLM table) is a project-scoped table CTX reads for its retrieval ladder. CTX renumbered CLM's unapplied migrations 050/051/052 → 053/054/055; CTX `ctx_*` tables start at 056.
- `.skillgrid/artifacts/04-adr-0024-central-pg-shared-memory-state.md` — the new `ctx_*` tables are per-machine derived state (session-scoped `ctx_proxy_sessions`/`ctx_session_summaries`/`ctx_traffic_memories`; project-scoped `ctx_stats_history`). No sync change required; they stay local.
- Locked constraint "No new dependencies without an ADR" — satisfied: the `ctx/` module re-uses `github.com/devopstales/skillgrid/mnemonic` (store access) and the existing local Ollama HTTP endpoint; no new third-party Go or Node dependencies.
- Locked constraint "Go 1.22+ minimum to build" — satisfied: the `ctx/` module is a Go 1.22 module in the workspace.

## New Durable ADRs Created

- `.skillgrid/artifacts/04-adr-0033-three-module-workspace-ctx.md` — CTX is a 3rd Go module (`ctx/`) in `go.work`, a library that imports the mnemonic module for store access (read-only); it opens no DB handle of its own and creates only `ctx_`-prefixed tables. Supersedes the two-module framing from the `2026-10-06-mnemonic-standalone` change.
- `.skillgrid/artifacts/04-adr-0034-ctx-cli-mnemonic-subcommand.md` — `ctx` is a `mnemonic` subcommand group (`mnemonic ctx …`), not a standalone binary; the mnemonic binary wires the `runCtx` package. `mnemonic ctx stats` is the unified CTX stats surface (proxy/session metrics + CLM `tool_outputs`/`indexed_files` row counts as a sub-slice) and is distinct from the unchanged per-agent `mnemonic stats` rollup.
- `.skillgrid/artifacts/04-adr-0035-local-ollama-judgment-model.md` — the judgment layer uses a local Ollama model (default `tev1:0.8b` at `http://127.0.0.1:11434/v1/systemone`; `clef-flash` opt-in) for per-unit truncate-only decisions, threshold 0.22, fingerprint-cached; it fails open to deterministic pre-filter when the endpoint is unavailable.
- `.skillgrid/artifacts/04-adr-0036-ctx-config-configd.md` — CTX config lives at `config.d/ctx.yaml` (machine-local `~/.skillgrid/config.d/ctx.yaml` + repo-local `.skillgrid/config.d/ctx.yaml` override, first-found-wins per key), reusing the existing `config.d` loader; the `clm:` block stays in `.skillgrid/config.yaml`.

## Supersessions

- **Two-module workspace framing** (from the `2026-10-06-mnemonic-standalone` change) — superseded by ADR-0033: the workspace now has three modules (`mnemonic/`, `skillgrid-cli/`, `ctx/`).
- **CLM migration numbering 050/051/052** — superseded by this change's renumber to 053/054/055 (ADR-0034 / absorption amendment), clearing 050–052 for CTX `ctx_*` tables starting at 056.

## Prerequisite Changes (execution-gated, not re-spec'd)

- `2026-10-06-context-harness-clm` (ADR-0025…0028) — `sliced`, 0 tickets executed. CTX builds on its `context_harness` owner, `tool_outputs`/`indexed_files` tables, and `ctx` CLI surface. CTX spec lands now; execution waits on this change landing.
- `2026-10-07-replace-opencode-hooks-with-plugins` (ADR-0032) — prerequisite for the CTX OpenCode plugin host. Execution waits on this change landing.
