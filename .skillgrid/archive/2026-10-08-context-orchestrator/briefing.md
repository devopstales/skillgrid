# Context Orchestrator (CTX) — Design Briefing (requirements & intent)

> **STATUS:** `draft` (2026-10-08)
> **Tier:** T2
> **Path:** New Subsystem (3rd Go module `ctx/` + `mnemonic ctx …` subcommand group)
> **Prerequisite plan:** `.skillgrid/artifacts/context-orchestrator-plan.md` v1.11
> **Depends on:** `2026-10-06-context-harness-clm` (CLM, ADR-0025…0028) + `2026-10-07-replace-opencode-hooks-with-plugins` (ADR-0032). CTX execution is gated on both landing; this spec is written now.

## Problem / Intent

The Context Harness (ADR-0025) and the CLM layer (ADR-0027) own the *session*
context lifecycle: capture, query, index, the mirror, the overflow guard. What
they do not own is the *cross-cutting* orchestration plane: composing
multi-source queries into declarative recipes, applying reversible per-type
compression (CCR), shaping both the input and the output of the model,
exposing a **transparent proxy** that any OpenAI/Anthropic-compatible client can
use via a base-URL override, and giving operators a **read-only observability
surface** (live stats, durable history, Prometheus metrics, per-session wide
events) plus a **real-time traffic learner** that mines the session-event stream
into reusable memories.

This change builds that plane as a **thin orchestration layer** — the Context
Orchestrator (CTX). CTX does **not** re-index code, re-implement parsing, own
storage, or re-implement hybrid search. It reads Mnemonic's schema, composes
queries, applies compression, shapes I/O, hosts the proxy, and exposes
observability. v1.11 of the plan is the primary design artifact; this briefing
captures the **delta** over the CLM spec (phases 16/17/18, the 3rd-module
architecture, the new `ctx_*` tables, the `mnemonic ctx …` CLI, and the CLM
absorption amendment) and the locked decisions from the 2026-10-08 interview.

## Purpose & Success Criteria

- **Purpose:** Give CTX one owner above the Context Harness + CLM: a 3rd Go
  module that composes queries into recipes, applies reversible compression,
  shapes model I/O, hosts a loopback transparent proxy (3 modes +
  `/v1/compress`), exposes read-only observability folded into the existing
  `mnemonic serve` UI server, and runs a rule-based real-time traffic learner.
  The agent integrates with zero code change (raw proxy) or one command
  (`mnemonic ctx wrap <agent>`).
- **Success criteria:**
  - `ctx/` is a 3rd module in `go.work` importing `mnemonic` (read-only); it
    never opens its own DB handle and never DDLs non-`ctx_` tables.
  - `mnemonic ctx …` is a subcommand group on the `mnemonic` binary; the
    existing `mnemonic stats` (per-agent rollup) is unchanged and distinct.
  - `mnemonic ctx proxy` binds loopback :8787, forwards Anthropic + OpenAI
    wire traffic (HTTP/1.1, HTTP/2, SSE, WebSocket) intact, applies the pipeline
    in the middle, returns savings headers, and degrades to passthrough on
    compression failure (fail-open).
  - Three modes (`token`/`cache`/`passthrough`) work at startup, at runtime, and
    via `CTX_PROXY_MODE`; a mode change is recorded in `ctx_proxy_mode_history`
    and emitted as a `proxy-mode-changed` event. Default mode is `cache`.
  - `/v1/compress` returns a compressed message list without generating a
    completion; a non-loopback caller receives **404** (not 403).
  - `mnemonic ctx wrap <agent>` (core-4: opencode, claude, codex, cursor) starts
    the proxy, injects config, backs up existing config, and launches the agent;
    it never modifies agent code.
  - The existing `mnemonic serve` UI server gains read-only `GET /stats`,
    `GET /stats-history`, `GET /metrics`, and `GET /api/session-summaries`,
    backed by `ctx_stats_history` and `ctx_session_summaries`, with a JSON
    snapshot under the project directory that survives a re-index.
  - A per-session wide event is aggregated at SessionEnd into one
    `ctx_session_summaries` row within the SessionEnd hook budget (the hook
    enqueues; the worker aggregates).
  - The traffic learner extracts memories in all four categories
    (`error_recovery`, `environment`, `preference`, `architecture`) from the
    session-event stream with **zero LLM calls**, bounded by `max_memories`,
    each memory storing its evidence; a memory is never applied as a rule until
    a human promotes it via `mnemonic ctx learn apply`.
  - The judgment layer (local Ollama, default `tev1:0.8b`) is truncate-only,
    fingerprint-cached, threshold 0.22, and fails open to pre-filter-only.
  - CTX config lives at `config.d/ctx.yaml` (machine-local default + repo-local
    override), reusing the existing `config.d` loader.
  - **CLM absorption amendment:** the unified `mnemonic ctx stats` sub-slice
    reports the CLM row counts (`tool_outputs`, `indexed_files`); CLM's
    unapplied migrations are renumbered 050/051/052 → **053/054/055**, and CTX's
    `ctx_*` migrations start at **056+**.
- **Out of scope:** per-agent proxy code (the proxy is agent-agnostic); a
  hosted proxy service; a control plane on the proxy; a second UI; a second
  database; LLM calls inside the traffic learner; automatic promotion of
  traffic memories; cross-project memory federation; rewriting stored messages
  (PrefixGuard); a separate observability process; re-designing CTX phases 1–15
  (referenced as prerequisite via the plan, not re-spec'd here).

## Context

- **Prerequisite plan (v1.11):** `.skillgrid/artifacts/context-orchestrator-plan.md`
  — 18 phases, config blocks, migration table, interface definitions, R1–R75,
  M1–M28. This spec owns the **v1.11 delta** (phases 16/17/18 + the 3rd-module
  architecture + the new tables/CLI + the CLM absorption amendment). Phases 1–15
  (recipe engine, folders/compression, CCR, hooks, redaction, judgment, plugin,
  output shaping, failure mining) are carried forward from the plan and are
  **prerequisite work**, not re-designed here.
- **Module layout:** `go.work` currently holds two modules — `mnemonic/`
  (module `github.com/devopstales/skillgrid/mnemonic`, binary at
  `mnemonic/cmd/mnemonic/`) and `skillgrid-cli/` (installer). CTX adds a 3rd:
  `ctx/` (module `github.com/devopstales/skillgrid/ctx`). (ADR-0033.)
- **CLI dispatcher:** `mnemonic/cmd/mnemonic/main.go` is a hand-rolled
  `flag`-based switch (no cobra); `runServe` is in `mcp.go`. CTX adds
  `case "ctx"` → `runCtx`. The binary already ships `stats` (per-agent
  rollup) — `ctx stats` is distinct under the `ctx` parent. (ADR-0034.)
- **UI server (observability host):** `mnemonic serve` binds
  `127.0.0.1:7438` (mcp.go:168, `SKILLGRID_MNEMONIC_PORT`), read routes open,
  write routes gated by `SKILLGRID_HTTP_TOKEN`. CTX's `/stats`,
  `/stats-history`, `/metrics`, `/api/session-summaries` fold into **this**
  server — same bind, same auth, no second process, no second port. (ADR-0033;
  Q12.)
- **Config.d loader:** `mnemonic/internal/config/pricing.go:96` (`findConfigD`)
  implements repo-local `.skillgrid/config.d/<name>.yaml` overriding machine-local
  `~/.skillgrid/config.d/<name>.yaml`, first-found-wins per key. Used by
  `indexing.yaml`, `mcp.yaml`, `tools.yaml`. CTX reuses it for `ctx.yaml`.
  (ADR-0036.)
- **Judgment model:** local Ollama catalog (verified) offers `tev1:0.8b`
  (decision, 797MB), `clef-flash` (9B), `llama3.2:1b`, `embeddinggemma:300m`.
  CTX's judgment layer defaults to `tev1:0.8b` at
  `http://127.0.0.1:11434/v1/systemone`. (ADR-0035.)
- **CLM spec (prerequisite + absorption):** `.skillgrid/specs/2026-10-06-context-harness-clm/`
  — ADR-0025…0028 in force; `sliced` but 0 tickets executed. Its migrations are
  `050_tool_outputs.sql`, `051_indexed_files.sql`, `052_context_revisions.sql`
  (unapplied). This change **absorbs** the CLM row counts into the unified
  `ctx stats` sub-slice and **renumbers** those unapplied migrations to
  053/054/055 so CTX's `ctx_*` tables can start at 056+. (Amendment below.)
- **replace-hooks (prerequisite):** `.skillgrid/specs/2026-10-07-replace-opencode-hooks-with-plugins/`
  (ADR-0032) — the 5 native TS plugins. CTX's OpenCode plugin adapter composes
  with them; CTX execution waits on this landing.
- **In-force ADRs reviewed:** ADR-0004 (component map), ADR-0012 (SQLite
  store), ADR-0013 (repo is source of truth), ADR-0016 (second-brain fail-open
  floors), ADR-0018 (additive search signals), ADR-0021 (pre-tool fail-open),
  ADR-0022 (checkpoint), ADR-0024 (central PG shared state), ADR-0025 (Context
  Harness owner), ADR-0026 (`indexed_files` reuses observations shape),
  ADR-0027 (CLM), ADR-0028 (`context_revisions` session-scoped), ADR-0030
  (scan findings structured store), ADR-0032 (opencode/kilo native plugins).
  Locked constraint "No new dependencies without an ADR" — satisfied: CTX uses
  only existing deps (stdlib HTTP for the proxy; no new Go module dependency
  beyond the workspace-internal `mnemonic` import).

## Approaches Considered

- **Chosen — 3rd module + mnemonic subcommand:** CTX is a 3rd Go module
  (`ctx/`) importing `mnemonic` read-only; its CLI is the `mnemonic ctx …`
  group; the proxy is `mnemonic ctx proxy` (loopback :8787); observability
  folds into `mnemonic serve`. Rationale: clean dependency direction
  (ctx → mnemonic), no second process, one binary the operator already runs,
  CTX is independently testable, and the DB ownership stays with Mnemonic.
- **Rejected — standalone `ctx` binary:** a separate binary duplicates the
  dispatcher, adds a 2nd artifact to install/version, and the proxy would need
  its own DB handle (violating "Mnemonic owns storage"). The subcommand keeps
  one binary and one DB owner.
- **Rejected — package inside `mnemonic/`:** folding CTX into `mnemonic/`
  blurs the boundary the plan draws (Mnemonic owns schema/storage/search; CTX
  owns orchestration/proxy/observability) and forces CTX's large surface
  (proxy, traffic learner, judgment) into the mnemonic module's build.
- **Rejected — 2nd observability process/port:** a separate stats/metrics
  server is a second process to run and a second port to firewall; folding
  into the existing UI server keeps one bind + one auth model and is the
  plan's stated non-goal ("no second UI server, no second database").

## Requirements

1. **3rd module, read-only DB access.** `ctx/` is a module in `go.work`
   importing `mnemonic`; it reads Mnemonic's schema and never opens its own
   handle nor DDLs non-`ctx_` tables.
   - **Current:** `go.work` holds `mnemonic/` + `skillgrid-cli/`; no `ctx/`.
   - **Target:** `ctx/go.mod` (module `github.com/devopstales/skillgrid/ctx`),
     added to `go.work`; `ctx/` imports
     `github.com/devopstales/skillgrid/mnemonic` for store access.
   - **Acceptance:** `go build ./...` in the workspace passes with 3 modules;
     a `grep` of `ctx/` for a direct `sql.Open`/`*sql.DB` handle on the shared
     store returns none (all DB access goes through the mnemonic import).
   - **Acceptance scenario:** `happy path ctx-module-builds` → `acceptance.feature`

2. **`mnemonic ctx` subcommand group.** The CTX CLI surface is a subcommand
   group on the `mnemonic` binary; the existing `mnemonic stats` is unchanged
   and distinct.
   - **Current:** `main.go` dispatcher has no `ctx` case; `mnemonic stats`
     (per-agent rollup) exists.
   - **Target:** `case "ctx"` → `runCtx` binding the verbs: `proxy [status|mode
     <m>]`, `wrap <agent>`, `stats [--history]`, `search <query>`, `index
     <path>`, `purge`, `metrics`, `traffic memories|report|rules reload`,
     `learn [report|apply <id>|revoke <id>]`, `judge`, `checkpoint
     list|read <id>`. `mnemonic stats` (root) still returns the per-agent
     rollup.
   - **Acceptance:** `mnemonic ctx stats` returns the unified CTX surface;
     `mnemonic stats` (no `ctx`) returns the per-agent rollup unchanged;
     `mnemonic ctx --help` lists the CTX verbs.
   - **Acceptance scenario:** `happy path ctx-subcommand-group` → `acceptance.feature`

3. **Proxy surface, 3 modes, loopback, savings headers, fail-open.** The
   transparent proxy forwards Anthropic + OpenAI wire traffic intact, applies
   the pipeline in the middle, and is loopback-only by default.
   - **Current:** no proxy.
   - **Target:** `mnemonic ctx proxy --port 8787` (default bind `127.0.0.1`);
     modes `token`/`cache`/`passthrough` (default `cache`); forwards
     HTTP/1.1, HTTP/2, SSE, WebSocket; returns `X-CTX-Savings`,
     `X-CTX-Original-Tokens`, `X-CTX-Compressed-Tokens`, `X-CTX-Mode`,
     `X-CTX-Cache-Hit`; compression error → passthrough (request completes);
     provider timeout → retry with `provider_retry_backoff_ms`.
   - **Acceptance:** a proxied request through an Anthropic-style and an
     OpenAI-style client completes with accurate savings headers; a
     compression fault still completes the request (passthrough); a provider
     timeout retries with the configured backoff.
   - **Acceptance scenario:** `happy path proxy-three-modes` → `acceptance.feature`

4. **`/v1/compress` + loopback invisibility.** The compression-only endpoint
   returns a compressed message list without generating a completion;
   non-loopback callers get 404.
   - **Current:** no `/v1/compress`.
   - **Target:** `POST /v1/compress` accepts a message list, runs the pipeline
     (no upstream call), returns the compressed list; a non-loopback caller
     receives **404** (not 403); binding non-loopback requires explicit
     opt-in + a logged warning.
   - **Acceptance:** `POST /v1/compress` from loopback returns a compressed
     list and no upstream completion is made; the same call from a
     non-loopback origin returns 404.
   - **Acceptance scenario:** `happy path compress-endpoint-loopback` → `acceptance.feature`

5. **Mode switch + run mode as a signal.** Mode is settable at startup, at
   runtime, and via `CTX_PROXY_MODE`; changes are recorded and emitted; the
   current mode is exposed in `/stats` and the Live view header.
   - **Current:** no mode state.
   - **Target:** `mnemonic ctx proxy mode <mode>` (runtime) + `CTX_PROXY_MODE`;
     each change writes a `ctx_proxy_mode_history` row and emits a
     `proxy-mode-changed` event; `/stats` and the Live view header show the
     current mode.
   - **Acceptance:** switching mode at runtime changes the next request's
     compression strategy, records a history row, emits the event, and the new
     mode is visible in `/stats`.
   - **Acceptance scenario:** `happy path mode-switch-emitted` → `acceptance.feature`

6. **`ctx wrap` two-tier launcher (core-4).** Tier 2 injects config and
   launches a named agent; core-4 in v1, the other 6 are opt-in config.
   - **Current:** no `wrap`.
   - **Target:** `mnemonic ctx wrap <agent>` for core-4 (opencode, claude,
     codex, cursor) starts the proxy, injects the correct base-URL/config,
     backs up existing config (`backup_config`), and launches the agent;
     aider/copilot/openclaw/cline/continue/goose are opt-in via the `wrap:`
     config. It never modifies agent code.
   - **Acceptance:** `mnemonic ctx wrap claude` starts the proxy, writes the
     injected config (with a backup), and launches claude pointing at the
     proxy; a core-4 agent runs a turn through the proxy.
   - **Acceptance scenario:** `happy path wrap-core4-launches` → `acceptance.feature`

7. **Observability endpoints folded into `mnemonic serve`.** Read-only
   `/stats`, `/stats-history`, `/metrics`, `/api/session-summaries` on the
   existing UI server (same bind + auth, no 2nd process/port).
   - **Current:** `mnemonic serve` has the memory/code/sessions/web routes; no
     CTX observability routes.
   - **Target:** the UI server gains the four read-only routes backed by
     `ctx_stats_history` + `ctx_session_summaries`; `/metrics` is
     Prometheus-parseable; `/stats-history` returns hourly/daily/weekly/monthly
     rollups and survives a restart.
   - **Acceptance:** `GET /stats` returns live session metrics; `GET
     /stats-history` returns rollups and survives a server restart; `GET
     /metrics` parses as Prometheus; `GET /api/session-summaries` returns
     paginated per-session rows.
   - **Acceptance scenario:** `happy path observability-endpoints` → `acceptance.feature`

8. **Per-session wide event at SessionEnd.** One `ctx_session_summaries` row
   per session, aggregated in the worker within the hook budget.
   - **Current:** session events are a many-row stream; no wide event.
   - **Target:** at SessionEnd the hook enqueues `session_summary_aggregate`;
     the worker aggregates duration, turns, tokens saved, judgment keeps/truncs,
     pre-filter pins/deads, cache invalidations, shaping, effort routed, traffic
     memories into one row (+ full JSON); the hook does not block the agent.
   - **Acceptance:** a finished session produces exactly one
     `ctx_session_summaries` row whose counts match the session's events; the
     SessionEnd hook completes within budget (the agent is not blocked).
   - **Acceptance scenario:** `happy path session-summary-aggregated` → `acceptance.feature`

9. **History snapshot JSON under the project.** A durable JSON snapshot
   mirroring `ctx_stats_history`, surviving a re-index.
   - **Current:** no snapshot.
   - **Target:** `.skillgrid/ctx/stats_history.json` written atomically, kept in
     sync with `ctx_stats_history`, part of the project (survives re-index).
   - **Acceptance:** after rollups run, the JSON file matches the DB rollups; a
     re-index leaves the file intact.
   - **Acceptance scenario:** `happy path history-snapshot-durable` → `acceptance.feature`

10. **Real-time traffic learner (4 categories, zero LLM).** Rule-based
    extraction from the session-event stream, bounded, evidence-carrying.
    - **Current:** no traffic learner.
    - **Target:** `traffic/rules.yaml` (human-editable) drives a pure
      extractor over the session-event stream; categories
      `error_recovery`/`environment`/`preference`/`architecture`; first matching
      rule wins (deterministic); below `min_confidence` not stored;
      `max_memories` caps `ctx_traffic_memories` (oldest low-confidence
      evicted first); each memory stores `evidence_json`.
    - **Acceptance:** a session with a failed-then-retried tool call extracts
      an `error_recovery` memory with evidence; the store stays ≤ `max_memories`;
      no LLM call is made during extraction.
    - **Acceptance scenario:** `happy path traffic-learner-extracts` → `acceptance.feature`

11. **Traffic → learn composition, human promotion gate.** Real-time
    memories feed the offline `ctx learn` loop; promotion is a human CLI
    action.
    - **Current:** no `ctx learn` (prerequisite, v1.10 Phase 15).
    - **Target:** a high-confidence real-time memory may seed a **proposed**
      rule; promotion requires the v1.10 gate (`promotion_threshold_sessions` +
      `mnemonic ctx learn apply <id>`); the two layers do not contradict
      (real-time per-session vs offline cross-session).
    - **Acceptance:** a traffic memory is never applied as a rule until a human
      runs `mnemonic ctx learn apply <id>`; after apply, the rule is active;
      `revoke` removes it.
    - **Acceptance scenario:** `happy path traffic-promotion-human-gate` → `acceptance.feature`

12. **Judgment layer (local Ollama, truncate-only, fail-open).** Per-unit
    keep/truncate/compress-tier decisions, fingerprint-cached, threshold 0.22.
    - **Current:** no judgment layer.
    - **Target:** `judgment.model` (default `tev1:0.8b`), `judgment.endpoint`
      (default `http://127.0.0.1:11434/v1/systemone`); a fast per-unit
      structured-output call; truncate-only (never deletes, never rewrites
      prose); fingerprint-cached; threshold 0.22 (0.20–0.25); endpoint
      unavailable → fail open to pre-filter-only.
    - **Acceptance:** a context unit above threshold is truncated (not
      deleted); a repeat unit is served from the fingerprint cache (no model
      call); with Ollama down, the pre-filter still runs and nothing blocks.
    - **Acceptance scenario:** `happy path judgment-truncate-only-fail-open` → `acceptance.feature`

13. **CTX config at `config.d/ctx.yaml`.** Machine-local default + repo-local
    override, reusing the `config.d` loader.
    - **Current:** no `ctx.yaml`; the `clm:` block is in
      `.skillgrid/config.yaml`.
    - **Target:** `~/.skillgrid/config.d/ctx.yaml` (machine-local: `proxy`,
      `wrap`, `observability`, `traffic_learning`, `judgment`) + optional
      repo-local `.skillgrid/config.d/ctx.yaml` override; repo-local wins per
      key; absent file = all defaults. The `clm:` block stays in
      `.skillgrid/config.yaml`.
    - **Acceptance:** a value in repo-local `ctx.yaml` overrides the same key in
      machine-local; an absent `ctx.yaml` yields all defaults; the `clm:` block
      is still read from `.skillgrid/config.yaml`.
    - **Acceptance scenario:** `happy path ctx-config-precedence` → `acceptance.feature`

14. **CLM absorption amendment.** The unified `ctx stats` sub-slice reports
    CLM row counts; CLM's unapplied migrations renumber 050/051/052 →
    053/054/055; CTX `ctx_*` migrations start at 056+.
    - **Current:** CLM spec names migrations `050_tool_outputs`,
      `051_indexed_files`, `052_context_revisions` (unapplied); `ctx stats` did
      not exist.
    - **Target:** CLM's three unapplied migrations are renumbered to
      `053_tool_outputs`, `054_indexed_files`, `055_context_revisions` (the CLM
      spec's `briefing.md`/`blueprint.md`/`tasks.md` migration numbers are
      amended; no applied migration is touched); CTX's `ctx_*` migrations
      (`ctx_proxy_sessions`, `ctx_proxy_savings`, `ctx_proxy_mode_history`,
      `ctx_session_summaries`, `ctx_stats_history`, `ctx_traffic_memories`)
      start at `056_…`; `mnemonic ctx stats` reports the CLM row counts
      (`tool_outputs`, `indexed_files`) as a sub-slice.
    - **Acceptance:** after the amendment, no CLM migration is numbered 050/051/
      052; CTX's first migration is 056; `mnemonic ctx stats` includes the
      `tool_outputs` + `indexed_files` counts.
    - **Acceptance scenario:** `happy path clm-renumber-ctx-stats-subslice` → `acceptance.feature`

## Implementation Decisions

- **Modules to build/modify:**
  - NEW `ctx/go.mod` (module `github.com/devopstales/skillgrid/ctx`) + `ctx/`
    package tree: `internal/proxy` (proxy + wrap launcher), `internal/observability`
    (stats/history/metrics/session-summary), `internal/traffic` (rule
    extractor), `internal/judgment` (curator), `internal/recipe` (composition),
    `internal/store` (+ `migrations/` for `ctx_*` tables), `internal/config`
    (ctx.yaml via the shared `config.d` loader).
  - CHANGED `go.work` — add `./ctx`.
  - CHANGED `mnemonic/cmd/mnemonic/main.go` — add `case "ctx"` → `runCtx`;
    `runCtx` binds verbs to the `ctx/` library.
  - CHANGED `mnemonic/internal/ui` (UI server) — register the four read-only
    observability routes (`/stats`, `/stats-history`, `/metrics`,
    `/api/session-summaries`) backed by the `ctx/` observability package.
  - CHANGED CLM spec (`.skillgrid/specs/2026-10-06-context-harness-clm/`) —
    amend migration numbers 050/051/052 → 053/054/055 (briefing, blueprint,
    tasks); no applied migration changes.
  - NEW `config.d/ctx.yaml` templates (machine-local `~/.skillgrid/config.d/`
    default + repo-local optional).
- **Interfaces:**
  - `mnemonic ctx proxy` — loopback :8787; Anthropic + OpenAI wire; modes
    token/cache/passthrough; savings headers; `/v1/compress` (404 non-loopback).
  - `mnemonic ctx wrap <agent>` — start proxy + inject config + launch (core-4).
  - `mnemonic ctx stats [--history]` — unified CTX surface (proxy/session
    metrics + CLM row-count sub-slice).
  - `GET /stats`, `/stats-history`, `/metrics`, `/api/session-summaries` —
    read-only, on the existing UI server.
  - `ctx.yaml` blocks: `proxy`, `wrap`, `observability`, `traffic_learning`,
    `judgment` (per the plan §1.2).
- **Data flow:**
  - Proxy: agent → (base-URL override) → `ctx proxy` :8787 → pipeline
    (token count → semantic cache → type detect → transform → summary compress
    → budget → shape) → forward upstream with API key → capture response →
    emit `proxy-request`/`proxy-savings` → return with savings headers.
  - Observability: `mnemonic serve` UI server → `/stats*`/`/metrics`/
    `/api/session-summaries` → `ctx_*` tables → JSON.
  - Session summary: SessionEnd hook → enqueue `session_summary_aggregate` →
    worker aggregates → one `ctx_session_summaries` row + `session-summary`
    event.
  - Traffic: session-event stream → rule extractor (pure) → `ctx_traffic_memories`
    (+ evidence) → `traffic-memory-extracted` event → `ctx learn` input.
  - Judgment: context unit → pre-filter (deterministic) → judgment model
    (fingerprint-cached, truncate-only) → decision → compress/shape.
- **Error handling:** Fail-open floors hold. Proxy compression error →
  passthrough (request completes); provider timeout → retry with backoff, then
  the provider error verbatim. Judgment endpoint down → pre-filter-only.
  Session-summary aggregation is worker-side; the hook only enqueues. Traffic
  extraction is a pure function (deterministic). `ctx purge` on a missing
  table is a no-op.
- **Dependencies:** CTX uses only existing deps — stdlib `net/http` + `httputil`
  for the proxy, `go.work`-internal `mnemonic` import for the store. No new
  third-party Go module (satisfies the "no new dependencies without an ADR"
  constraint; the `mnemonic` import is workspace-internal, not a new
  dependency).
- **Migrations (final numbering):**
  - 053 `tool_outputs` (CLM, renumbered from 050)
  - 054 `indexed_files` (CLM, renumbered from 051)
  - 055 `context_revisions` (CLM, renumbered from 052)
  - 056+ CTX `ctx_*` tables: `ctx_proxy_sessions`, `ctx_proxy_savings`,
    `ctx_proxy_mode_history`, `ctx_session_summaries`, `ctx_stats_history`,
    `ctx_traffic_memories` (exact sub-numbering assigned during execution).

## Testing Decisions

- **What makes a good test:** Assert external behavior — the savings headers,
  the `/v1/compress` 404 vs 200, the mode-history row, the session-summary
  row counts, the traffic memory + evidence, the config precedence, the CLM
  renumber — not internal helpers. The traffic extractor, judgment
  truncate-only logic, and the proxy pipeline stages are pure functions over a
  message list / event stream: unit-test them directly without an agent or an
  Ollama server (use a fake model endpoint).
- **Modules to test:**
  - `ctx/internal/proxy` — 3-mode behavior, savings-header accuracy,
    `/v1/compress` loopback vs non-loopback (404), fail-open on compression
    error, provider-timeout retry.
  - `ctx/internal/observability` — `/stats`, `/stats-history` (rollups +
    restart survival), `/metrics` (Prometheus parse), `/api/session-summaries`
    pagination; history snapshot matches DB + survives re-index.
  - `ctx/internal/traffic` — the four categories, first-match-wins,
    confidence floor, `max_memories` bound + eviction, evidence present, zero
    LLM calls.
  - `ctx/internal/judgment` — truncate-only (never delete/rewrite),
    fingerprint-cache hit (no model call), threshold 0.22, fail-open to
    pre-filter when the endpoint is down.
  - `ctx/internal/config` — `config.d` precedence (repo-local > machine-local),
    absent-file defaults, `clm:` still from `.skillgrid/config.yaml`.
  - `mnemonic/cmd/mnemonic` — `case "ctx"` routing; `mnemonic ctx stats` vs
    `mnemonic stats` distinctness.
  - migrations — 053/054/055 (CLM renumber) + 056+ (`ctx_*`) apply cleanly to a
    fresh and a pre-053 store.
- **Prior art:** `mnemonic/internal/ui` route tests (httptest round-trip),
  `hybrid/rank.go` tests, `memory` integration seed tests, the CLM spec's
  `context_harness` pure-function tests.
- **Edge cases:** mode switch at runtime vs startup vs env; `/v1/compress`
  from non-loopback (404) vs loopback (200); compression fault mid-request
  (passthrough, request completes); provider timeout (retry then verbatim
  error); session with zero events (empty summary row); traffic memory at
  exactly `min_confidence` (stored) vs just below (not stored); `max_memories`
  boundary (oldest low-confidence evicted); Ollama down (pre-filter-only,
  nothing blocks); repo-local `ctx.yaml` key overriding machine-local; a CLM
  store that already has 053/054/055 applied (CTX 056+ applies on top).

## Impact on Global Docs

- `.skillgrid/artifacts/00-prd.md`: None (scope/metric unchanged; new subsystem).
- `.skillgrid/ASSUMPTIONS.md`: In-force set carries ADR-0033…0036 (added during
  the interview).
- `.skillgrid/ARCHITECTURE.md`: Update §5 (packages: add `ctx/` module + the
  `runCtx` command package), §10 (transport: proxy loopback :8787 + the four
  observability routes on the existing UI server), §9 (storage: migration
  table 053–055 renumber + 056+ `ctx_*`), §14 (config: `config.d/ctx.yaml`),
  the module map (2 → 3 modules).
- `.skillgrid/specs/2026-10-06-context-harness-clm/`: migration numbers
  050/051/052 → 053/054/055 (briefing, blueprint, tasks amended).

## Clarity Report

| Dimension | Score | Min | Status | Notes |
|-----------|-------|-----|--------|-------|
| Goal Clarity | 0.95 | 0.75 | PASS | 14 falsifiable requirements over the v1.11 delta |
| Boundary Clarity | 0.90 | 0.70 | PASS | 3rd module + mnemonic subcommand + UI-server fold; Mnemonic owns storage |
| Constraint Clarity | 0.90 | 0.65 | PASS | Fail-open floors; loopback-only proxy; no new deps; Go 1.22+; config.d reuse |
| Acceptance Criteria | 0.90 | 0.70 | PASS | 14 requirements each with Current/Target/Acceptance + scenario |
| **Clarity** | **0.88** | ≤0.20 | PASS | 1 − mean(0.95,0.90,0.90,0.90) = 0.88 (gap 0.12 ≤ 0.20) |

**Interview log (2026-10-08):**

| Round | Question | Decision locked |
|-------|----------|-----------------|
| 1 (Q1) | Where does CTX code live? | 3rd Go module `ctx/` in `go.work`, imports `mnemonic` read-only, no own DB handle. (ADR-0033) |
| 1 (Q2) | Where do `ctx_*` migrations live? | `ctx/internal/store/migrations/`, applied by the mnemonic binary to the shared DB. |
| 1 (Q3) | Two surfaces: proxy + observability host? | Proxy = `mnemonic ctx proxy` (loopback :8787); observability folds into the existing `mnemonic serve` UI server. (ADR-0033) |
| 1 (Q4) | `ctx stats` collision? | Unified `mnemonic ctx stats` (proxy/session metrics + CLM row-count sub-slice); the pre-existing `mnemonic stats` (per-agent rollup) is unchanged and distinct. (ADR-0034) |
| 1 (Q5) | `ctx wrap` agent set? | Core-4 in v1: opencode, claude, codex, cursor; the plan's other 6 are opt-in config. |
| 1 (Q6) | Judgment model? | Local Ollama, default `tev1:0.8b` via `/v1/systemone`; `clef-flash` opt-in; fail-open to pre-filter. (ADR-0035) |
| 1 (Q7) | Spec scope? | This spec owns the v1.11 delta (phases 16/17/18 + 3rd-module architecture + tables/CLI + CLM absorption); phases 1–15 are prerequisite, referenced not re-spec'd. |
| 1 (Q8) | Timing? | Spec written now (planning); execution gated on replace-hooks + CLM landing. |
| 2 (Q9) | `ctx` CLI location? | `mnemonic` subcommand group (`mnemonic ctx …`), not a standalone binary. (ADR-0034) |
| 2 (Q10) | Core-4 wrap set? | opencode, claude, codex, cursor. |
| 2 (Q11) | Judgment default? | `tev1:0.8b` (local Ollama `/v1/systemone`); `judgment.model`/`judgment.endpoint` config. (ADR-0035) |
| 2 (Q12) | Observability bind/auth? | Inherit the existing UI server's loopback bind + auth; no separate port. |
| 2 (Q13) | CTX config location? | `~/.skillgrid/config.d/ctx.yaml` (machine-local) + optional repo-local override; reuse the `config.d` loader. (ADR-0036) |
| 2 (Q14) | Phase 15 (`ctx learn`) scope? | In scope for this spec (spec'd, not re-designed; v1.10). |
| 2 (Q15) | `ctx stats` vs `mnemonic stats` naming? | `mnemonic ctx stats` = unified CTX stats; `mnemonic stats` (per-agent rollup) unchanged and distinct. (ADR-0034) |

## Open Questions & Assumptions

- **Assumption:** The OpenCode `context` plugin hook (ADR-0032's 5 plugins) is
  the integration seam CTX's plugin adapter composes with; CTX execution waits
  on replace-hooks landing. Known prerequisite dependency, not a blocker for
  this spec.
- **Assumption:** The local Ollama endpoint `http://127.0.0.1:11434/v1/systemone`
  is available for the judgment model. If not, the judgment layer fails open to
  pre-filter-only (no block) — the deterministic floor holds.
- **Assumption:** The CLM spec's migrations (053/054/055 after renumber) are
  unapplied at CTX execution time. If CLM lands first and applies them, CTX's
  056+ migrations apply on top; if not, the renumber is applied as part of CTX
  execution (no already-applied migration is touched).
- **Known limitation (per the plan, carried forward):** the proxy is a
  wire-level interceptor; "the win is headroom, not money" — token mode can
  invalidate the provider prefix cache, so the default is cache mode.

## Decisions (ADR)

- `.skillgrid/artifacts/04-adr-0033-three-module-workspace-ctx.md` — CTX is a
  3rd Go module importing `mnemonic` read-only; CLI is `mnemonic ctx …`;
  proxy + observability are subcommands/routes, not separate processes.
- `.skillgrid/artifacts/04-adr-0034-ctx-cli-mnemonic-subcommand.md` — `ctx` is
  a `mnemonic` subcommand group; the existing `mnemonic stats` is unchanged and
  distinct; the unified `ctx stats` unifies within the CTX surface.
- `.skillgrid/artifacts/04-adr-0035-local-ollama-judgment-model.md` — judgment
  model is local Ollama (default `tev1:0.8b`), truncate-only,
  fingerprint-cached, threshold 0.22, fail-open to pre-filter.
- `.skillgrid/artifacts/04-adr-0036-ctx-config-configd.md` — CTX config is
  `config.d/ctx.yaml` (machine-local + repo-local override), reusing the
  `config.d` loader; the `clm:` block stays in `.skillgrid/config.yaml`.

## Terms

- Context Orchestrator (CTX) — `.skillgrid/artifacts/02-technical-terms.md`
- CTX Module — same file
- Proxy Surface — same file
- ctx wrap — same file
- Judgment Layer — same file
- Traffic Learner — same file
- Session Summary — same file
- CTX Stats — same file
- CTX Config — same file
