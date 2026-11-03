# T09 brief — observability: per-request bucket writer + periodic rollup + /stats + /stats-history

## Context
- Change: 2026-10-08-context-orchestrator. Ticket: TASK-097 (spec ticket T09, tasks.md `TICKET-09`). Wave 6, PR4 (observability).
- Branch `release/2`. Module `github.com/devopstales/skillgrid/mnemonic`. Workdir `/Users/paladm/git/ai-test/skillgrid-v2`.
- Repo is SHARED: stage ONLY your files (never `git add -A`). Do NOT re-stage the `mnemonic/mnemonic` build artifact or unrelated working-tree changes.

## Deliverable (scope)
Build `mnemonic/internal/ctx/observability/` + `mnemonic/internal/http/ctx.go`, and register two read-only routes on the existing `mnemonic serve` mux. This ticket delivers the **bucket writer + periodic rollup** and the **`/stats` + `/stats-history`** endpoints.

Out of scope here (separate tickets): `GET /metrics` (T10, TASK-098), `GET /api/session-summaries` + session-end aggregation (T11, TASK-099), the `.skillgrid/ctx/stats_history.json` snapshot + run-mode-in-/stats header (T11/T14). Do not implement those — but structure `/stats` so a `mode` field can be added later (see "Leave a seam").

## Design source
- Plan §17 (`.skillgrid/artifacts/context-orchestrator-plan.md` lines 800–867): 17.1 principle (read-only, no 2nd process/DB, folded into existing UI server), 17.2 endpoints, 17.4 history snapshot.
- tasks.md `TICKET-09` (lines 136–145).
- Acceptance: `observability-endpoints` scenario in `.skillgrid/specs/2026-10-08-context-orchestrator/acceptance.feature` (lines 118–134). This ticket satisfies the `/stats` + `/stats-history` + "rollups survive a server restart" clauses. `/metrics` and `/api/session-summaries` clauses are T10/T11.

## Existing contracts to build on (already merged — read these first)
- **ctx store** `mnemonic/internal/ctx/store/`: `store.New(shared *store.Store) *Store` (store.go:28) — wraps a shared `*store.Store`, opens no 2nd DB (ADR-0033). All accessors are **fail-open** (ADR-0016): a missing `ctx_*` table degrades to empty/zero, never an error that 500s the endpoint. `queryFailOpen` returns `nil, nil` on a missing table → every List/Rollup guards `rows == nil` → return `nil, nil`.
  - Structs (tables.go): `StatsBucket{BucketID, Granularity, BucketStart, TokensSaved, Requests, Sessions, CacheHits, CreatedAt}` (line 69); `RollupRow{Granularity, TokensSaved, Requests, Sessions, CacheHits}` (line 82); `ProxySession{SessionID, Agent, Mode, StartedAt, EndedAt, Requests, TokensBefore, TokensAfter, CacheHits, CacheInvalidations}` (line 12); `ProxySaving` (line 26).
  - Accessors: `PutStatsBucket(b *StatsBucket)` (line 299), `GetStatsBucket(bucketID)` (309), `ListStatsBuckets(granularity string)` (326), `Rollup() ([]RollupRow, error)` (355, sums across all buckets per granularity), `ListProxySessions()` (130), `ListProxySavings(sessionID)` (183), `GetProxySession(sessionID)` (113).
- **The proxy ALREADY writes per-turn rows** to `ctx_proxy_savings` + `ctx_proxy_sessions` (proxy.go `recordTelemetry`, lines 267–316). You do NOT rewrite that. What is missing is the **`ctx_stats_history` bucket write + rollup + the two GET endpoints**.
- **Config** `mnemonic/internal/ctx/config/config.go`: `Observability{Enabled, Stats, StatsHistory, Metrics, MetricsPath, HistoryRollups []string (default ["hourly","daily","weekly","monthly"]), HistorySnapshot, SessionSummary}` (line 93). Defaults at lines 142–149. Use `c.Observability.Stats` / `.StatsHistory` / `.HistoryRollups` to gate + drive the endpoints.
- **HTTP seam** `mnemonic/internal/http/`:
  - Route registration: `registerRoutes()` at server.go:105. Pattern: `s.mux.HandleFunc("GET /stats", s.handleCTXStats)`.
  - Project resolution: `s.withProjectHandle(w, r, func(h *service.ProjectHandle, projectID string){ ... })` (server.go:78) resolves the project, opens a `*service.ProjectHandle`, and on resolve/open error already writes 400/500 — the handler must NOT write a response after calling it for a resolve/open error.
  - Get the ctx store from the handle: `cs := ctxstore.New(h.Store())` — `h.Store()` returns `*store.Store` (service.go:2220). Build `ctxstore` per request (it owns no connection; cheap). `h.Store().Path()` gives the store dir if you need a project-relative path.
  - JSON helper: `writeJSON(w, status, v)` (used throughout, e.g. clm.go). Read-only GET routes are NOT wrapped in `requireWriteAuth`.
  - Reference handler to mirror: `mnemonic/internal/http/clm.go` (withProjectHandle + writeJSON fail-open shape) and `mnemonic/internal/http/eventstats.go:300` (withProjectHandle + compute + writeJSON).
  - **Single-open invariant** (single_open_test.go): a route must open its ProjectHandle exactly once via `withProjectHandle`/`openHandleFor` and must NOT call `store.Open` again inside the handler. `ctxstore.New(h.Store())` reuses the handle's DB — correct.

## Implementation
1. **`mnemonic/internal/ctx/observability/`** — new package.
   - **Bucket writer**: from the existing per-turn data (a `store.ProxySaving` row: mode, upstream, tokens before/after, cache hit) compute the tokens saved this turn (`TokensBefore - TokensAfter`, clamp ≥ 0) and write/merge a `ctx_stats_history` bucket. Bucket keying: `Granularity` ∈ {hourly, daily, weekly, monthly}; `BucketStart` = the bucket-aligned start unix seconds for the saving's `CreatedAt`; `BucketID` = a deterministic `<granularity>-<bucketStart>` string so `INSERT OR REPLACE` (PutStatsBucket) merges repeated writes for the same bucket. Accumulate `TokensSaved`, `Requests`, `CacheHits`; `Sessions` = count of distinct sessions seen in that bucket (track in-memory per-bucket or re-derive; document the approach). Provide a pure, testable `Roll(saving *store.ProxySaving) (buckets []*store.StatsBucket)` (or equivalent) that takes one saving and returns the bucket rows to upsert — no store inside.
   - **Rollup runner**: a goroutine started explicitly (e.g. `StartRollup(store, cfg, tick, stopCh)` returning a stop func) that, every tick, re-derives bucket rows from `store.ListProxySavings`/`ListProxySessions` (or reads a short window) and `PutStatsBucket`s them, so `ctx_stats_history` is kept current **without a proxy restart**. Keep it idempotent (re-run must not double-count — PutStatsBucket is INSERT OR REPLACE on the deterministic BucketID). Must be stoppable in tests (stopCh / done channel). Do NOT start it implicitly inside the package on import.
   - **Aggregate read**: `Stats(store) (*Stats, error)` and `StatsHistory(store, granularities []string) (*History, error)` returning plain structs the handlers marshal. `StatsHistory` reads `store.ListStatsBuckets(g)` per requested granularity (from `cfg.Observability.HistoryRollups`) — fail-open (missing table → empty). `Stats` returns live session metrics: current mode (latest of sessions), tokens saved (sum TokensBefore-TokensAfter across savings or sessions), cache hits, cache invalidations, requests, sessions — read from `ctx_proxy_sessions`/`ctx_proxy_savings` (fail-open).
   - **Leave a seam**: `Stats` struct has a `Mode` field populated from the latest session's mode; do NOT wire the Live-view header (that's T11/T14).
2. **`mnemonic/internal/http/ctx.go`** — new file, package `http`.
   - `handleCTXStats` (GET /stats): `withProjectHandle` → `ctxstore.New(h.Store())` → `observability.Stats` → `writeJSON(200, ...)`. Empty/zero store → 200 with zero-valued JSON (never 500).
   - `handleCTXStatsHistory` (GET /stats-history): `withProjectHandle` → read `cfg` rollup list (the Server must have access to the ctx config; if not trivially available, default to `["hourly","daily","weekly","monthly"]` — see "Config access" below) → `observability.StatsHistory` → `writeJSON(200, ...)`.
3. **Route registration** in `server.go` `registerRoutes()`: add the two GET routes. Read-only, no `requireWriteAuth`.
4. **Config access for rollup list**: check whether `*Server` already carries the ctx `*config.CtxConfig`. If not, either thread it through the Server constructor minimally, or default to the four standard granularities with a comment. Prefer the smallest seam that does not disturb unrelated Server fields. Document what you chose in the report.

## Acceptance (gates)
- `go build ./...` (module) exit 0.
- `go test ./internal/ctx/observability/...` green:
  - after N simulated turn-savings (call the bucket writer with N `store.ProxySaving` rows across modes), the store holds `ctx_stats_history` buckets with correct per-granularity `TokensSaved`/`Requests`/`CacheHits` sums.
  - the periodic rollup, started with a short tick + stop channel, writes `ctx_stats_history` rows **without a restart** (assert rows appear after a tick, then stop cleanly — no goroutine leak).
  - `Stats` JSON matches the stored aggregate (tokens saved / cache hits / requests).
  - `StatsHistory` returns all four granularities from `ctx_stats_history` (persisted → survives a re-open of the store, i.e. restart).
  - fail-open: a store opened WITHOUT the 056–061 migrations (or a fresh non-ctx store) → endpoints return 200 with zero-valued JSON, no error.
- `go test ./internal/http/...` green: GET /stats and GET /stats-history over `httptest` against a real `*Server` (mirror clm_test.go / eventstats.go test setup: `store.Open` a temp store, `withProjectHandle` path) return 200 JSON with the expected shape.
- `go vet ./internal/ctx/... ./internal/http/...` clean; `gofmt -l` clean on touched files.
- No new 3rd-party dependency (stdlib only). No `go.work` change.

## Constraints
- ADR-0016 fail-open on every ctx_* read/write (missing table ≠ error).
- ADR-0033: reuse the shared store handle (`ctxstore.New(h.Store())`); open NO second DB. Respect the single-open invariant (single_open_test.go).
- No new dependencies without an ADR (stdlib only).
- Conventional commit; do NOT add AI-attribution trailers. Commit body block:
  ```
  [skillgrid-context]
  change: 2026-10-08-context-orchestrator
  ticket: TASK-097
  wave: 6
  pr: PR4
  ```
- Table names are authoritative: `ctx_stats_history` (buckets), `ctx_proxy_sessions`, `ctx_proxy_savings`. The plan's "ctx_proxy_stats" is a stale alias — the real per-turn table is `ctx_proxy_savings`.

## Report
Write the FULL report to `.skillgrid/sdd/2026-10-08-context-orchestrator/task-09-report.md` (design notes, bucket-keying + sessions-count approach, rollup idempotency, config-access decision, gate outputs, file list). Your final chat reply must be a SHORT status (<15 lines): what you built, gate results (pass/fail + counts), commit SHA(s), and any parked items. Do not paste the report into chat.
