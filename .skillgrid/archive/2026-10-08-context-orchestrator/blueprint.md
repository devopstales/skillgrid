# Blueprint: Context Orchestrator (CTX) — v1.11 Delta

Status: PROPOSED
Tier: T2
Change: 2026-10-08-context-orchestrator
Spec: .skillgrid/specs/2026-10-08-context-orchestrator/briefing.md
Acceptance: .skillgrid/specs/2026-10-08-context-orchestrator/acceptance.feature
Build Shape: extend (third Go module surface under the `mnemonic` binary; no new binary)
Date: 2026-10-08

## Goal

Deliver the **v1.11 delta** of the Context Orchestrator as a Go subsystem inside the
`mnemonic` binary: an agent-agnostic local proxy (`ctx proxy`) with three compression
modes, a wrap launcher (`ctx wrap`), four read-only observability endpoints folded into
the existing `mnemonic serve` mux, and a zero-LLM rule-based real-time traffic learner.
The proxy applies CTX's existing pipeline (token count → cache check → transform →
summary compression → budget) transparently between an agent and its LLM provider,
emits savings telemetry into new `ctx_*` tables, and renders in the existing Mnemonic
web UI. It is a thin orchestration layer: it reads Mnemonic's store and schema, owns no
second database, and never re-implements search or indexing.

This blueprint is the **how**. The **what** (14 requirements) is the briefing; the
pass/fail contract is `acceptance.feature` (23 scenarios). Every task below is
constrained by the in-force ADRs (0012, 0016, 0021, 0025, 0026, 0033, 0036) and the
v1.11 plan (`.skillgrid/artifacts/context-orchestrator-plan.md`), phases 16–18.

## Architecture

### Build shape and module boundaries

- **Single binary, one Go module.** The proxy, observability, and traffic learner are
  new packages inside the existing `github.com/devopstales/skillgrid/mnemonic` module,
  dispatched through the existing `mnemonic ctx …` subcommand group
  (`mnemonic/cmd/mnemonic/ctx.go`, `runCtx`). ADR-0033: CTX is a `mnemonic` subcommand,
  not a standalone binary. No `go.work` change.
- **New packages** (all under `mnemonic/internal/ctx/`):
  - `internal/ctx/config` — load + merge the `ctx` config block (ADR-0036 `config.d`).
  - `internal/ctx/proxy` — HTTP proxy: modes, upstream forwarding, savings telemetry.
  - `internal/ctx/wrap` — agent launcher (tier-2 setup layer only).
  - `internal/ctx/observability` — read-only metric aggregation for the four endpoints.
  - `internal/ctx/traffic` — rule-based, zero-LLM memory extractor.
  - `internal/ctx/store` — `ctx_*` table accessors (reads/writes the shared SQLite).
- **Read-only seam into CLM.** The proxy and learner read `internal/context_harness`
  (gate decisions, token math, the session-event stream). CTX never writes to
  Mnemonic's event stream and never re-indexes (ADR-0025, 0026, 0033).

### Data layer (ADR-0012: shared store, new `ctx_*` tables)

Six new tables, migrations **056–061** (CLM used 053/054/055; 056 is the next free
number). All in the shared SQLite store opened by the same `openLoop`/service handle the
proxy and serve process already use — no second connection, no second DB file.

| Migration | Table | Purpose |
|-----------|-------|---------|
| 056 | `ctx_proxy_sessions` | per-session proxy metrics (mode, tokens in/out, cache hits) |
| 057 | `ctx_proxy_savings` | per-turn savings rows (before/after, ratio, upstream) |
| 058 | `ctx_proxy_mode_history` | auditable mode-change trail |
| 059 | `ctx_session_summaries` | per-session wide event (one row per session) |
| 060 | `ctx_stats_history` | hourly/daily/weekly/monthly rollups |
| 061 | `ctx_traffic_memories` | rule-extracted memories + evidence JSON |

Column definitions are the plan §9.3 (lines 469–539) verbatim. Fail-open: a missing or
unmigrated `ctx_*` table degrades an endpoint to a 200 empty payload, never a 500
(ADR-0016).

### Config (ADR-0036: `config.d` block)

A new `ctx:` top-level block, loaded the same way `pricing.yaml` is: merge order
lowest→highest = built-in default → `~/.skillgrid/config.d/ctx.yaml` → first
`config.d/ctx.yaml` found walking up from the project root (first-found-wins per key).
Reuse the `findConfigD` helper shape in `internal/config/pricing.go:96`. The `clm:`
block stays in `.skillgrid/config.yaml`; CTX keys do not move there.

Keys (plan §1.2):
- `proxy:` — `enabled`, `bind` (127.0.0.1), `port` (8787), `mode` (token|cache|
  passthrough, default cache), `compress_endpoint`, `non_loopback_status` (404),
  `savings_headers`, `fail_open`, `provider_retries` (3), `provider_retry_backoff_ms`,
  `passthrough_protocols` (http1.1, http2, sse, ws), `upstreams` (anthropic, openai).
- `wrap:` — `enabled`, `agents`, `inject_env`, `backup_config`.
- `observability:` — `enabled`, `stats`, `stats_history`, `metrics`, `metrics_path`
  (/metrics), `history_rollups`, `history_snapshot` (`.skillgrid/ctx/stats_history.json`),
  `session_summary`.
- `traffic_learning:` — `enabled`, `realtime`, `rules_path` (`traffic/rules.yaml`),
  `categories`, `min_confidence` (0.6), `max_memories` (500), `emit_events`.

`proxy.mode` is also settable via `CTX_PROXY_MODE` env (mirrors `HEADROOM_MODE`),
checked after config.

### Proxy surface (plan Phase 16)

- **Bind** 127.0.0.1:8787 by default; non-loopback bind requires explicit opt-in + a
  logged warning (16.6). `/v1/compress` returns **404** (not 403) to non-loopback
  callers so the route stays invisible (16.6).
- **Modes** (16.2): `token` (rewrite prior history to maximize compression), `cache`
  (freeze provider-confirmed cached prefixes, compress only the mutable tail — default),
  `passthrough` (forward verbatim). Settable at startup (`ctx proxy --mode`), runtime
  (`ctx proxy mode <m>`), or env. A change is recorded in `ctx_proxy_mode_history` and
  emitted as a `proxy-mode-changed` event.
- **Endpoints** (16.3): `/health`, `/livez`, `/readyz`, `/stats`, `/stats-history`,
  `/metrics`, `/v1/compress` (compression-only, no upstream call), and `/v1/*` +
  `/v1/messages` proxied upstream.
- **Request lifecycle** (16.4): session lookup/create (id from headers) → mode
  determination → pipeline (token count → cache check → content detect → transform
  select → summary compression → budget → output-shape request) → forward upstream with
  the correct API key → capture response → compute `tokens_before - tokens_after` → emit
  telemetry → return response with savings headers.
- **Savings headers** (16.5): `X-CTX-Savings`, `X-CTX-Original-Tokens`,
  `X-CTX-Compressed-Tokens`, `X-CTX-Mode`, `X-CTX-Cache-Hit`.
- **Fail-open** (16.7): a compression failure degrades to passthrough (request still
  completes); provider timeouts retry up to `provider_retries` with exponential backoff;
  a failed retry returns the provider error verbatim. ADR-0016 + ADR-0021.
- **Agent-agnostic** (16.1, 16.10): the proxy speaks Anthropic + OpenAI wire protocols
  and forwards HTTP/1.1, HTTP/2, SSE, WS verbatim. No per-agent proxy code. No hosted
  service, no control plane, no second UI.
- **Two-tier integration** (16.8): tier-1 = any agent with a base-URL override
  (`ANTHROPIC_BASE_URL`/`OPENAI_BASE_URL`); tier-2 = `ctx wrap <agent>` (start proxy,
  inject config, back up existing config, launch). Supported agents: claude, codex,
  cursor, aider, copilot, opencode, openclaw, cline, continue, goose. The proxy itself
  is tier-1 only; wrap is a setup layer.

### Wrap launcher (plan Phase 16.8)

`ctx wrap <agent>`: start the proxy → inject the correct base-URL/env for the named
agent → back up the existing agent config → `exec` the agent. Idempotent; backs up
before writing. The launcher is the only component that knows about specific agents.

### Observability (plan Phase 17 — folded into the existing `mnemonic serve` mux)

Four read-only routes registered in `Server.registerRoutes()`
(`internal/http/server.go:105`), implemented as `internal/http/ctx.go` handlers using
the existing `withProjectHandle` + `writeJSON` prologue (pattern: `internal/http/clm.go`):

- `GET /stats` — live session metrics (current mode, tokens saved this session, cache
  hits/invalidations, judgment decisions, pre-filter pins/deads, traffic memories).
- `GET /stats-history` — durable savings history with hourly/daily/weekly/monthly
  rollups from `ctx_stats_history`; also writes the JSON snapshot
  `.skillgrid/ctx/stats_history.json` (17.4).
- `GET /metrics` — Prometheus text format (request counts, token totals, latency
  summaries, per-upstream counts, cache-bust counters, judgment latency, pre-filter
  throughput, prefix-guard blocks, learn-run frequency).
- `GET /api/session-summaries` — paginated per-session wide events from
  `ctx_session_summaries`.

Read-only, no control plane, no second UI, no new auth model — inherits the existing
serve auth. The current proxy mode is exposed in `/stats` and the Live-view header
(17.5); mode changes are `proxy-mode-changed` events.

Per-session wide event (17.3): aggregated at SessionEnd by a `session_summary`
aggregator that reads the session's `ctx_*` rows, captures duration/turns/tokens
saved/judgment keeps+truncs/pre-filter pins+deads/cache invalidations/shaping applied/
effort routed/traffic memories, and stores one row in `ctx_session_summaries` (+ full
JSON in `summary_json`). Runs within the SessionEnd hook budget; never blocks the agent.

### Real-time traffic learner (plan Phase 18)

A pure-function, zero-LLM extractor over Mnemonic's session-event stream:

- **Categories** (18.2): `error_recovery` (failed call → successful retry on same
  target), `environment` (repeated stable facts: commands, paths, versions, env vars),
  `preference` (repeated usage patterns / user corrections), `architecture` (file
  structure / conventions from repeated reads+edits).
- **Rules** (18.3): human-editable `traffic/rules.yaml`; each rule declares category,
  pattern, confidence weight, description. Applied in order; first match wins
  (deterministic). Below `min_confidence` → not stored.
- **Storage + bounds** (18.4): `ctx_traffic_memories`; `max_memories` (500) caps the
  store, evicting the oldest low-confidence first; each memory records its evidence
  events in `evidence_json` for audit.
- **Composition** (18.5): real-time memories are a new input to the offline `ctx learn`
  loop; a high-confidence memory may seed a proposed rule, but promotion still requires
  the v1.10 gate (threshold + human CLI action).
- **Determinism** (18.6): same event stream + same rules → same memories, same order. A
  rule change affects future extractions only; it never rewrites stored memories.

### CLI surface (plan §9.2)

Extend `runCtx` in `mnemonic/cmd/mnemonic/ctx.go` (currently: `search`, `prime`,
`stats`, `purge`, `index`, `help`) with:

- `ctx proxy` — start the proxy (loopback default); `--mode` flag.
- `ctx proxy status` — bind, mode, upstreams, uptime, request count, savings total.
- `ctx proxy mode <mode>` — switch token/cache/passthrough.
- `ctx wrap <agent>` — tier-2 launcher.
- `ctx stats` / `ctx stats --history` — same as `GET /stats` / `/stats-history`.
- `ctx metrics` — Prometheus metrics to stdout.
- `ctx traffic memories` / `ctx traffic report` / `ctx traffic rules reload`.

The existing `ctx stats` verb (which today reports CLM harness stats) is re-pointed to
the observability aggregate so the CLI and `GET /stats` agree; the old harness-stats
behavior is preserved under `ctx stats --harness` for backward compat.

### Terms

- **CTX** — the Context Orchestrator subsystem (this change).
- **Proxy mode** — one of token|cache|passthrough; selects the compression strategy.
- **Savings** — `tokens_before - tokens_after` for a proxied turn.
- **Wide event** — the single per-session summary row aggregated at SessionEnd.
- **Traffic memory** — a rule-extracted, evidence-backed candidate rule.
- **Wrap** — the tier-2 agent launcher (proxy start + config inject + launch).

## Tech Stack

- **Language:** Go 1.25.5 (module `github.com/devopstales/skillgrid/mnemonic`). No new
  third-party dependency for the proxy (stdlib `net/http`, `http/httputil.ReverseProxy`,
  `net/http2`). Prometheus `/metrics` is hand-emitted text (no `client_golang` dep —
  ADR rule: no new dependencies without an ADR).
- **Storage:** shared SQLite via the existing store/migrations; `database/sql` rows in
  `internal/ctx/store`.
- **Config:** `gopkg.in/yaml.v3` (already a dependency) via the `config.d` loader.
- **Tests:** `go test ./...`; unit + integration layers. Proxy tested with an
  `httptest` upstream stub; fail-open and 404-invisibility asserted with loopback vs
  non-loopback dialers. Traffic learner tested for determinism (same input → same
  output) and zero-LLM (no network in test).

## Conventions (in force)

- Conventional commits only; `[skillgrid-context]` block names the CTX task ID on the
  commit that closes it.
- Spec-zone artifact (this file) commits before code-zone changes.
- TDD is `false` in config, but the verification ladder still applies: unit tests for
  the config merge, the mode selector, the savings math, the traffic extractor;
  integration test for proxy → stub upstream with the five savings headers.
- Fail-open is a load-bearing invariant (ADR-0016/0021): every CTX code path that can
  touch the agent's request must have a passthrough/200 fallback that is unit-tested.

## Tasks

Granularity: bite-sized, TDD-shaped, 2–5 min each, ordered so each step is independently
verifiable. Waves reflect the plan's parallelizable phases (16 ∥ 17 ∥ 18 share the store
and config, so the store + config land first).

### Wave 0 — foundation (store + config)

- [ ] 0.1 Add `056_ctx_proxy_sessions.sql` (plan §9.3 schema verbatim); run migrations
      against a temp store; assert table + column set.
- [ ] 0.2 Add `057_ctx_proxy_savings.sql`; migration test.
- [ ] 0.3 Add `058_ctx_proxy_mode_history.sql`; migration test.
- [ ] 0.4 Add `059_ctx_session_summaries.sql`; migration test.
- [ ] 0.5 Add `060_ctx_stats_history.sql`; migration test.
- [ ] 0.6 Add `061_ctx_traffic_memories.sql`; migration test.
- [ ] 0.7 `internal/ctx/config`: define the `CtxConfig` struct for all four blocks
      (proxy/wrap/observability/traffic_learning) with the plan defaults.
- [ ] 0.8 `internal/ctx/config`: built-in default YAML (`ctx_default.yaml`, `go:embed`)
      mirroring the plan §1.2 values; unit test defaults match the YAML.
- [ ] 0.9 `internal/ctx/config`: `LoadCtxConfig(startDir)` merge (builtin → home →
      first-found `config.d/ctx.yaml`), reusing the `findConfigD` walk; unit test the
      three-way merge precedence.
- [ ] 0.10 `internal/ctx/config`: `CTX_PROXY_MODE` env override applied after file
      load; unit test env beats config file.
- [ ] 0.11 `internal/ctx/store`: `ProxySessions` upsert/read; fail-open (missing table
      → empty, no error surfaced to caller). Unit test with a temp store.
- [ ] 0.12 `internal/ctx/store`: `Savings` insert/read; unit test.
- [ ] 0.13 `internal/ctx/store`: `ModeHistory` append/read; unit test.
- [ ] 0.14 `internal/ctx/store`: `SessionSummaries` upsert + paginated read; unit test.
- [ ] 0.15 `internal/ctx/store`: `StatsHistory` rollup upsert + read by granularity; unit test.
- [ ] 0.16 `internal/ctx/store`: `TrafficMemories` insert + `max_memories` eviction
      (oldest low-confidence first) + evidence JSON round-trip; unit test eviction order.

### Wave 1 — proxy core (Phase 16)

- [ ] 1.1 `internal/ctx/proxy`: mode type + `SelectMode(cfg, env, perRequestHeader)`
      with precedence env > header > config; unit test all three sources.
- [ ] 1.2 `internal/ctx/proxy`: request-lifecycle skeleton (session lookup/create from
      headers, mode, pipeline call site, forward, capture, savings compute, telemetry)
      with a stub pipeline; unit test lifecycle ordering with an `httptest` upstream.
- [ ] 1.3 `internal/ctx/proxy`: token count → `tokens_before`; integrate the existing
      CLM token math (read-only, `internal/context_harness`); unit test.
- [ ] 1.4 `internal/ctx/proxy`: `/v1/*` + `/v1/messages` upstream forwarding via
      `httputil.ReverseProxy` with the correct per-upstream API key injection;
      integration test against an `httptest` Anthropic-shape stub.
- [ ] 1.5 `internal/ctx/proxy`: `cache` mode — freeze confirmed-cached prefix, compress
      only the mutable tail; unit test prefix is byte-identical, tail is transformed.
- [ ] 1.6 `internal/ctx/proxy`: `token` mode — rewrite prior history when it improves
      compression; unit test.
- [ ] 1.7 `internal/ctx/proxy`: `passthrough` mode — forward verbatim, no compression;
      integration test (body identical).
- [ ] 1.8 `internal/ctx/proxy`: `compress` mode selects the strategy per 16.2; record the
      active mode into `ctx_proxy_sessions`; unit test.
- [ ] 1.9 `internal/ctx/proxy`: savings headers (`X-CTX-Savings`,
      `X-CTX-Original-Tokens`, `X-CTX-Compressed-Tokens`, `X-CTX-Mode`,
      `X-CTX-Cache-Hit`) on the response; integration test asserts all five.
- [ ] 1.10 `internal/ctx/proxy`: `/v1/compress` — compression-only, returns compressed
      message list, does NOT call upstream; unit test (no upstream hit recorded).
- [ ] 1.11 `internal/ctx/proxy`: loopback-only invisibility — `/v1/compress` returns
      **404** to a non-loopback caller; unit test with a non-loopback peer address.
- [ ] 1.12 `internal/ctx/proxy`: non-loopback bind requires explicit opt-in + logs a
      warning; unit test the warning + refusal without opt-in.
- [ ] 1.13 `internal/ctx/proxy`: fail-open — compression error degrades to passthrough,
      request still completes 200; unit test injects a pipeline error.
- [ ] 1.14 `internal/ctx/proxy`: provider retry with exponential backoff
      (`provider_retries`, backoff_ms); a failed retry returns the provider error
      verbatim; unit test with a flaky stub.
- [ ] 1.15 `internal/ctx/proxy`: telemetry — per-turn `ctx_proxy_savings` insert +
      `ctx_proxy_sessions` counters; integration test asserts rows after N turns.
- [ ] 1.16 `internal/ctx/proxy`: `/health`, `/livez`, `/readyz` handlers; unit test each.
- [ ] 1.17 `internal/ctx/proxy`: `Server.Start(cfg)` binds 127.0.0.1:port and serves the
      proxy mux; integration test dials loopback and gets 200 on `/health`.

### Wave 2 — wrap launcher (Phase 16.8)

- [ ] 2.1 `internal/ctx/wrap`: agent registry (claude, codex, cursor, aider, copilot,
      opencode, openclaw, cline, continue, goose) → base-URL/env mapping; unit test the
      table.
- [ ] 2.2 `internal/ctx/wrap`: `wrap <agent>` starts the proxy, injects the agent's
      base-URL env, backs up the existing agent config, and execs the agent; unit test
      the backup-before-write with a temp config.
- [ ] 2.3 `internal/ctx/wrap`: idempotency — a second wrap with the same agent does not
      re-backup over the original; unit test.

### Wave 3 — observability (Phase 17, folded into serve mux)

- [ ] 3.1 `internal/ctx/observability`: live `/stats` aggregate (mode, tokens saved,
      cache hits/invalidations, judgment decisions, pre-filter pins/deads, traffic
      memories) read from `ctx_*` + the CLM handle; unit test with seeded rows.
- [ ] 3.2 `internal/http/ctx.go`: `handleCtxStats` via `withProjectHandle`; register
      `GET /stats` in `registerRoutes`; fail-open 200 on empty/missing table (ADR-0016).
- [ ] 3.3 `internal/ctx/observability`: `/stats-history` rollups (hourly/daily/weekly/
      monthly) from `ctx_stats_history`; unit test each granularity.
- [ ] 3.4 `internal/http/ctx.go`: `handleCtxStatsHistory`; register `GET /stats-history`;
      also write the `.skillgrid/ctx/stats_history.json` snapshot (17.4); unit test file
      is written and mirrors the rows.
- [ ] 3.5 `internal/ctx/observability`: Prometheus text emitter (request counts, token
      totals, latency summaries, per-upstream counts, cache-bust counters, judgment
      latency, pre-filter throughput, prefix-guard blocks, learn-run frequency); unit
      test the text format.
- [ ] 3.6 `internal/http/ctx.go`: `handleCtxMetrics` at `metrics_path` (/metrics);
      register the route; unit test the `text/plain; version=0.0.4` content type.
- [ ] 3.7 `internal/ctx/observability`: SessionEnd wide-event aggregator (duration,
      turns, tokens saved, judgment keeps/truncs, pre-filter pins/deads, cache
      invalidations, shaping applied, effort routed, traffic memories) → one
      `ctx_session_summaries` row + `summary_json`; unit test the aggregation.
- [ ] 3.8 Wire the aggregator into the existing SessionEnd path (run within the hook
      budget, never block the agent); integration test: end a session, assert the row.
- [ ] 3.9 `internal/http/ctx.go`: `handleCtxSessionSummaries` (paginated); register
      `GET /api/session-summaries`; unit test pagination + empty page.
- [ ] 3.10 Re-point `ctx stats` CLI to the observability aggregate; preserve the old
      harness stats under `ctx stats --harness`; unit test both branches.

### Wave 4 — traffic learner (Phase 18)

- [ ] 4.1 `internal/ctx/traffic`: rule schema (category, pattern, confidence weight,
      description) + load `traffic/rules.yaml`; unit test load + validation.
- [ ] 4.2 `internal/ctx/traffic`: extractor — pure function over the session-event
      stream applying rules in order, first match wins; unit test determinism (same
      input → same memories, same order).
- [ ] 4.3 `internal/ctx/traffic`: `error_recovery` category (failed call → successful
      retry on same target); unit test.
- [ ] 4.4 `internal/ctx/traffic`: `environment` category (repeated stable facts); unit test.
- [ ] 4.5 `internal/ctx/traffic`: `preference` category (repeated usage / corrections); unit test.
- [ ] 4.6 `internal/ctx/traffic`: `architecture` category (structure / conventions from
      repeated reads+edits); unit test.
- [ ] 4.7 `internal/ctx/traffic`: `min_confidence` gate (below → not stored); unit test.
- [ ] 4.8 `internal/ctx/traffic`: evidence capture — store the producing events in
      `evidence_json`; unit test round-trip.
- [ ] 4.9 `internal/ctx/traffic`: real-time hook — run the extractor during the session
      on the event stream (zero LLM calls; no network in test); integration test.
- [ ] 4.10 `internal/ctx/traffic`: feed high-confidence memories as a proposed-rule input
      to `ctx learn` (promotion still gated by the v1.10 threshold + human CLI); unit test
      the hand-off without auto-promotion.
- [ ] 4.11 `ctx traffic memories` / `ctx traffic report` (precision + categories) /
      `ctx traffic rules reload` CLI verbs; unit test each.

### Wave 5 — CLI wiring + integration

- [ ] 5.1 `mnemonic/cmd/mnemonic/ctx.go`: `ctx proxy` (`--mode`), `ctx proxy status`,
      `ctx proxy mode <mode>`; unit test dispatch.
- [ ] 5.2 `mnemonic/cmd/mnemonic/ctx.go`: `ctx wrap <agent>`; unit test dispatch.
- [ ] 5.3 `mnemonic/cmd/mnemonic/ctx.go`: `ctx metrics`, `ctx stats --history`; unit test
      dispatch to the observability package.
- [ ] 5.4 End-to-end integration: start proxy against an `httptest` upstream, drive N
      requests in `cache` mode, assert the five savings headers, the `ctx_proxy_savings`
      + `ctx_proxy_sessions` rows, a `/stats` reading, and a `/metrics` scrape.
- [ ] 5.5 Acceptance sweep: walk all 23 `acceptance.feature` scenarios against the
      running proxy + serve; record pass/fail per scenario tag.

## Verification

- `go build ./...` clean; `go vet ./...` clean.
- `go test ./...` green (unit + integration layers).
- Migration test confirms 056–061 apply idempotently on a fresh store.
- Fail-open spot-checks: disable CTX config → proxy passthrough + observability 200s.
- Acceptance: all 23 scenarios in `acceptance.feature` pass (Wave 5.5 sweep).
