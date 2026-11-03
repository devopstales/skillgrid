# T10 brief — observability endpoints: /metrics + /api/session-summaries + loopback restriction

## Context
- Change: 2026-10-08-context-orchestrator. Ticket: TASK-098 (spec ticket T10). Wave 6, PR4 (observability).
- Branch `release/2`. Module `github.com/devopstales/skillgrid/mnemonic`. Workdir `/Users/paladm/git/ai-test/skillgrid-v2`.
- Repo is SHARED: stage ONLY your files (never `git add -A`). Do NOT re-stage the `mnemonic/mnemonic` build artifact or unrelated working-tree changes.

## Deliverable (scope)
T09 (commit 440ad7aa + d44a5c26) ALREADY built `mnemonic/internal/http/ctx.go` with `GET /stats` + `GET /stats-history` and registered them via `registerCtxRoutes()` in `server.go`. This ticket ADDS the remaining two read-only endpoints and the loopback restriction:
- `GET /metrics` — Prometheus text format, hand-emitted (stdlib only, NO prometheus client dep).
- `GET /api/session-summaries` — paginated per-session wide events from `ctx_session_summaries`.
- **Loopback restriction** on ALL FOUR ctx observability routes (`/stats`, `/stats-history`, `/metrics`, `/api/session-summaries`): a non-loopback origin is rejected with no data leak.

Out of scope (T11/T14): populating `ctx_session_summaries` (the session-end aggregation writer), the `.skillgrid/ctx/stats_history.json` snapshot file, and the run-mode-in-`/stats`-header. The `/api/session-summaries` reader here reads whatever is in the table (may be empty until T11 populates it) — fail-open to empty.

## Design source
- Plan §17 (`.skillgrid/artifacts/context-orchestrator-plan.md` lines 800–869): 17.2 endpoints, 17.6 non-goals (read-only, no control plane, no new auth model — inherit the existing UI server's).
- tasks.md `TICKET-10` and the ticket `.backlog/tasks/task-098 ...`.
- Acceptance: `http-stats`, `http-metrics`, `http-loopback-only` scenarios in `.skillgrid/specs/2026-10-08-context-orchestrator/acceptance.feature`.

## Existing contracts to build on (already merged — read these first)
- **`mnemonic/internal/http/ctx.go`** (T09): `handleCTXStats`, `handleCTXStatsHistory`, `ctxObservabilityConfig`. Pattern: `s.withProjectHandle(w, r, func(h *service.ProjectHandle, projectID string){ ... })` → `ctxstore.New(h.Store())` → `writeJSON(w, http.StatusOK, v)`. `*Server` does NOT carry the ctx `*config.CtxConfig`; `ctxObservabilityConfig(ctx, root)` loads it per-request via `config.Load` (fail-open to `config.DefaultCtxConfig()`).
- **Route registration**: `registerCtxRoutes()` in `server.go` (added by T09). The comment at server.go:194 notes these live on the `mnemonic serve` mux, NOT the proxy loopback mux.
- **ctx store** `mnemonic/internal/ctx/store/` (all fail-open, ADR-0016):
  - `SessionSummary{SessionID, Agent, DurationMS, Turns, TokensSaved, JudgmentKeeps, JudgmentTruncs, PrefilterPins, PrefilterDeads, CacheInvalidations, ShapingApplied *bool, EffortRouted, TrafficMemories, SummaryJSON, CreatedAt}` (tables.go:50).
  - `ListSessionSummaries() ([]*SessionSummary, error)` (tables.go:269) — all rows, ordered (check the actual ORDER BY in the file).
  - `ListProxySessions()`, `ListProxySavings(sessionID)`, `Rollup() ([]RollupRow, error)` (355), `ListStatsBuckets(granularity)`.
  - `RollupRow{Granularity, TokensSaved, Requests, Sessions, CacheHits}` (tables.go:82).
  - `ctxstore.New(shared *store.Store) *Store` opens no 2nd DB (ADR-0033).
- **`writeJSON(w, status, v)`** helper exists in the http package (used by clm.go / T09). For `/metrics` you will hand-write the text body (Content-Type `text/plain; version=0.0.4`) — do NOT use writeJSON.
- **Single-open invariant**: `mnemonic/internal/http/single_open_test.go` — a route must open its ProjectHandle exactly once via `withProjectHandle` and must NOT call `store.Open` again inside the handler. `ctxstore.New(h.Store())` reuses the handle's DB — correct.

## Implementation
1. **`mnemonic/internal/http/ctx.go`** — add:
   - **`handleCTXMetrics`** (GET /metrics): `withProjectHandle` → `ctxstore.New(h.Store())` → gather counters from the stored ctx data and hand-emit Prometheus text. Metrics to emit (each with `# HELP` + `# TYPE` lines):
     - `ctx_requests_total` (counter) — total requests, summed from `ctx_proxy_sessions` (`Requests` column) or per-savings rows.
     - `ctx_tokens_saved_total` (counter) — total tokens saved (sum `TokensBefore - TokensAfter` clamped ≥0 across savings, or `TokensSaved` across session summaries; document which source).
     - `ctx_cache_hits_total` / `ctx_cache_invalidations_total` (counters) — from `ctx_proxy_sessions` (`CacheHits`, `CacheInvalidations`).
     - `ctx_sessions_total` (counter) — count of sessions.
     - Per-upstream counts `ctx_upstream_requests_total{upstream="..."}` if the stored data carries an upstream field; otherwise emit a zero/default upstream label and note it.
     Keep it to what the stored ctx tables actually support (request counts, token totals, cache counters, session counts, per-upstream if present). Latency/judgment/pre-filter/learn-run metrics are OUT of scope (no stored source yet) — do not fabricate them; emit a `# HELP ... (not yet instrumented)` comment line or omit. Fail-open: a missing table → the counter emits `0`, never a 500.
   - **`handleCTXSessionSummaries`** (GET /api/session-summaries): `withProjectHandle` → `ctxstore.New(h.Store())` → `ListSessionSummaries()` → paginate. Read `?limit=` (default 50, max 100) and `?cursor=` (offset-based or keyset on `CreatedAt`/`SessionID`; document the choice) query params. Return a JSON envelope like `{"items": [...], "next": "<cursor-or-empty>"}`. Fail-open: missing table → `{"items": [], "next": ""}` 200.
   - **Loopback restriction** for ALL FOUR routes: add a guard (e.g. a `ctxLoopbackOK(w, r) bool` helper or a small middleware wrapper) that rejects a request whose origin is not loopback. Determine loopback from the request: `r.RemoteAddr` host part is `127.0.0.1`/`::1`/`localhost` (parse with `net.SplitHostPort`, handle the no-port case). On non-loopback, write `403` with a short JSON body (`{"error":"loopback only"}`) and return NO ctx data (no leak). Apply this guard to all four handlers (the two T09 handlers + the two new ones). Keep the guard itself simple and testable.
2. **`registerCtxRoutes()`** in `server.go`: register the two new routes (`GET /metrics`, `GET /api/session-summaries`). Keep the existing two.
3. Keep the `/stats` `Mode` seam from T09 (do not add the header — T11/T14).

## Acceptance (gates)
- `go build ./...` (module) exit 0.
- `go test ./internal/http/...` green. Add/adjust tests (mirror `ctx_test.go` T09 setup: `store.Open` a temp store, real `*Server`):
  - The 4 routes are registered and answer **200 on loopback** (seed a couple of `ctx_proxy_sessions` / `ctx_session_summaries` rows via the store, then assert each route returns 200 + non-empty/consistent body).
  - `/metrics` returns valid Prometheus text (assert the `# HELP`/`# TYPE` lines + at least the `ctx_requests_total` and `ctx_tokens_saved_total` lines with correct values matching the seeded data).
  - `/api/session-summaries` returns JSON consistent with stored `ctx_session_summaries` rows; pagination works (seed >limit rows, assert `items` capped at limit + a non-empty `next` cursor, and that the next page returns the remainder).
  - **Non-loopback origin rejected** with 403 + no data leak (force a non-loopback `RemoteAddr` in the test request — e.g. `httptest` `Recorder` with a crafted `RemoteAddr`, or set `r.RemoteAddr` directly) for all four routes.
  - fail-open: a store missing the ctx tables → all four routes 200 with empty/zero bodies (no 500).
- `go vet ./internal/http/...` clean; `gofmt -l` clean on touched files; `single_open_test.go` still green.
- No new 3rd-party dependency (stdlib only — Prometheus text is hand-emitted). No `go.work` change.

## Constraints
- ADR-0016 fail-open on every ctx_* read (missing table ≠ error, never 500 the endpoint).
- ADR-0033: reuse the shared store handle (`ctxstore.New(h.Store())`); open NO second DB. Respect the single-open invariant.
- Read-only endpoints; no `requireWriteAuth` on a GET; inherit the existing UI server's auth model (plan §17.6).
- No new dependencies without an ADR (stdlib only).
- Conventional commit; do NOT add AI-attribution trailers. Commit body block:
  ```
  [skillgrid-context]
  change: 2026-10-08-context-orchestrator
  ticket: TASK-098
  wave: 6
  pr: PR4
  ```

## Report
Write the FULL report to `.skillgrid/sdd/2026-10-08-context-orchestrator/task-10-report.md` (metrics emitted + source table for each, pagination cursor choice, loopback-guard approach + how the test forces a non-loopback RemoteAddr, gate outputs, file list). Your final chat reply must be a SHORT status (<15 lines): what you built, gate results (pass/fail + counts), commit SHA, and any parked items. Do not paste the report into chat.
