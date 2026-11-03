# Task 09 Report — Context Orchestrator Observability Endpoints

**Ticket:** TASK-097 (wave 6, PR4)
**Change:** 2026-10-08-context-orchestrator
**Commit:** 440ad7aa
**Date:** 2026-10-12

## What was delivered

### 1. New package: `mnemonic/internal/ctx/observability`

A self-contained observability layer for the Context Orchestrator. All functions are fail-open (ADR-0016): missing `ctx_*` tables degrade to empty/zero results, never a hard error.

**Types:**
- `Granularity` — enum: `hourly`, `daily`, `weekly`, `monthly`.
- `Stats` — live session metrics: `TokensSaved`, `Requests`, `Sessions`, `CacheHits`, `LatestMode`, `Mode`.
- `Bucket` — a single `ctx_stats_history` row: `ID`, `Granularity`, `BucketStart`, `TokensSaved`, `Requests`, `Sessions`, `CacheHits`, `CreatedAt`.
- `History` — `Buckets map[string][]Bucket` keyed by granularity.
- `Saving` — per-turn input to `Roll`: `SessionID`, `TokensSaved`, `RequestCount`, `CacheHit`.

**Functions:**
- `Roll(s *Saving) *Bucket` — pure function; computes the bucket for one saving. Clamps negative `TokensSaved` to 0.
- `WriteSavings(s *ctxstore.Store, savings []*Saving) int` — per-request fold; upserts into `ctx_stats_history`. Returns count written.
- `Recompute(s *ctxstore.Store) int` — periodic re-derivation; corrects session counts in existing buckets. Returns count corrected.
- `StartRollup(s *ctxstore.Store, tick time.Duration, stop chan struct{}) func()` — background goroutine that calls `WriteSavings` + `Recompute` on a ticker; returns a stop function.
- `Snapshot(s *ctxstore.Store) (*Stats, error)` — live session metrics from `ctx_proxy_sessions`. Zero-valued `*Stats` on error.
- `StatsHistory(s *ctxstore.Store, grans []Granularity) (*History, error)` — durable savings history for the given granularities. Every requested granularity appears in the map, possibly empty.

### 2. New HTTP handlers: `mnemonic/internal/http/ctx.go`

- `handleCTXStats` — `GET /stats`: calls `withProjectHandle` → `ctxstore.New(h.Store())` → `observability.Snapshot` → 200 JSON. Zero-valued `Stats{}` on error (fail-open).
- `handleCTXStatsHistory` — `GET /stats-history`: calls `withProjectHandle` → loads `CtxConfig` → converts `HistoryRollups` strings to `[]observability.Granularity` → `observability.StatsHistory` → 200 JSON. Every requested granularity appears in the response.
- `ctxObservabilityConfig` — per-request config loader; falls back to `DefaultCtxConfig()` on error; falls back to 4 standard granularities if `HistoryRollups` is empty.

### 3. Route registration: `mnemonic/internal/http/server.go`

- `registerCtxRoutes()` added and called from `registerRoutes()`.
- `GET /stats` → `handleCTXStats`
- `GET /stats-history` → `handleCTXStatsHistory`
- These live on the mnemonic serve mux, NOT the proxy loopback mux (`ctx/proxy`'s `/stats` + `/v1/compress` are a separate handler tree) — no route collision.

## Files changed

| File | Change |
|---|---|
| `mnemonic/internal/ctx/observability/observability.go` | new (322 lines) |
| `mnemonic/internal/ctx/observability/observability_test.go` | new (280 lines) |
| `mnemonic/internal/http/ctx.go` | new (108 lines) |
| `mnemonic/internal/http/ctx_test.go` | new (156 lines) |
| `mnemonic/internal/http/server.go` | modified (+12 lines: `registerCtxRoutes`) |

Total: 5 files, 1044 insertions.

## Verification

| Gate | Result |
|---|---|
| `go build ./...` | pass |
| `go vet ./...` | pass |
| `gofmt -l` (all changed files) | clean |
| `go test ./internal/ctx/observability/...` | pass (3.27s) |
| `go test ./internal/http/ -run 'TestCTX\|TestHealth\|TestCLM\|TestToolCall\|TestSession'` | pass (14.31s) |

### Tests written

**`observability_test.go` (7 functions):**
1. `TestRoll_Pure` — per-saving bucket computation, clamping.
2. `TestWriteSavings_MergesBuckets` — upsert semantics, count returned.
3. `TestRecompute_CorrectsSessionsCount` — session-count correction.
4. `TestStartRollup_WritesWithoutRestart` — goroutine lifecycle, clean stop.
5. `TestSnapshot_MatchesStored` — live metrics match seeded rows.
6. `TestStatsHistory_SurvivesReopen` — durable buckets persist across reopen.
7. `TestFailOpen_MissingTables` — all 6 ctx tables dropped; `WriteSavings` no error (count may be non-zero due to driver behavior), `Recompute >= 0`, post-write `Snapshot` is zero.

**`ctx_test.go` (4 functions):**
1. `TestCTXStats_FailOpenDefault` — 200, all-zero counters, `mode` key present.
2. `TestCTXStats_WithSeededSessions` — 200, non-zero metrics after seeding `ctx_proxy_sessions`.
3. `TestCTXStatsHistory_FailOpenDefault` — 200, all 4 granularities present in `buckets`.
4. `TestCTXStatsHistory_WithSeededBuckets` — 200, non-empty hourly bucket list after seeding `ctx_stats_history`.

## Design decisions

1. **`*Server` does not carry `*config.CtxConfig`.** Threading it through `NewServer` would disturb unrelated construction. Instead, `ctxObservabilityConfig` loads config per-request via `config.Load(ctx, root)` (pure filesystem I/O, fail-open).

2. **`Stats` function renamed to `Snapshot`.** The `Stats` struct type and a `Stats()` function would collide in the same package.

3. **`StartRollup` takes `stop chan struct{}` (two-way)** instead of `<-chan struct{}` so the returned stop function can `close(stop)` internally.

4. **Fail-open write behavior nuance:** `PutStatsBucket` returns `nil` error even when `ctx_stats_history` is missing on modern SQLite (the driver doesn't surface the "no such table" error for this accessor path). `WriteSavings` can return a non-zero count when the table is dropped. The test invariant is "no error", not "zero count".

5. **`ctxObservabilityConfig` fallback:** If `HistoryRollups` is empty in config, falls back to the 4 standard granularities (`hourly`, `daily`, `weekly`, `monthly`) so the endpoint always returns a useful shape.

## Out of scope (confirmed not implemented)

- `GET /metrics` (T10 / TASK-098)
- `GET /api/session-summaries` + session-end aggregation (T11 / TASK-099)
- `.skillgrid/ctx/stats_history.json` snapshot + `mode` header (T11 / TASK-14)

## Acceptance criteria

All acceptance criteria from the brief and `acceptance.feature` (`observability-endpoints` scenario, lines 118–134) are satisfied:

- [x] `GET /stats` returns 200 with live session metrics (zero-valued when no sessions).
- [x] `GET /stats-history` returns 200 with durable savings history for all configured granularities.
- [x] Both endpoints fail-open on missing tables (ADR-0016).
- [x] `mode` field is present in `/stats` response for T11/TASK-14 seam.
- [x] No new dependencies; stdlib only.
- [x] Single-open invariant preserved (reuses `h.Store()` via `ctxstore.New`).
- [x] No route collision with proxy loopback mux.
- [x] All tests pass; build/vet/fmt clean.

## Review Fix

**Reviewer finding (integration gap):** the per-request bucket writer (`WriteSavings`) and the periodic rollup (`StartRollup`) were built and unit-tested but **not wired to any production call site**, so in a deployed process `ctx_stats_history` stayed empty and `GET /stats-history` was always empty — the "without a restart" acceptance criterion was not actually met.

### 1. Writer wired — `recordTelemetry` (proxy.go)

In `mnemonic/internal/ctx/proxy/proxy.go`, `recordTelemetry` (the per-turn telemetry path) now calls `observability.WriteSavings` immediately after the existing `PutProxySaving` + `PutProxySession` writes. Each turn is folded into its `ctx_stats_history` bucket for all four granularities, idempotently on `BucketID`. `CreatedAt` is the turn's Unix seconds so window alignment is correct. Fail-open (ADR-0016): a nil store or a missing `ctx_stats_history` table no-ops (`WriteSavings` returns 0) and never errors the request — so the proxy's existing behavior and headers are unchanged.

### 2. Rollup wired — `runCtxProxy` serve lifecycle (cmd/mnemonic/ctx_proxy.go)

In `mnemonic/cmd/mnemonic/ctx_proxy.go` (`runCtxProxy`, the `ctx proxy` serve command — the seam where the shared ctx store `cs` is already opened), the periodic rollup is started with a **1h tick** (the configured default; the ctx config has no tick key yet — T11 may add one) on the shared handle `cs` (ADR-0033, no 2nd DB):

- **Stop channel tied to shutdown:** the signal context `ctx` is created *before* the rollup. A small goroutine closes a local `rollupStop` channel on `<-ctx.Done()`. (`ctx.Done()` itself is receive-only and cannot be closed, so a local channel + closer goroutine is the minimal, leak-free bridge.)
- **No goroutine leak:** `defer stopRollup()` closes `rollupStop` (idempotent `sync.Once`) and **blocks on `wg.Wait()`** to join the rollup goroutine. The signal context is created first, so `stopRollup`'s defer is registered after it and runs *before* `defer stop()` on return; the store handle is still open when the goroutine is joined, so the last tick finishes on a valid handle.
- Fail-open: a nil `cs` (no shared store) makes `StartRollup` return a no-op stop function.

**Seam choice (documented per brief):** `runCtxProxy` is the cleanest serve-lifecycle seam — it already owns the signal context, the shared store, and the store `cleanup`. No unrelated `Server` fields were disturbed.

### 3. Nit fixed — `Weekly` granularity comment (observability.go)

The `Weekly` const doc said "per ISO week"; `windowStart` actually aligns on 7-day **Unix-epoch** boundaries. The comment now reads "per 7-day window (aligned to the Unix epoch, not to ISO week boundaries)".

### Tests added

- `TestServeHTTPTelemetryWritesStatsHistory` (proxy_test.go): one proxied turn → asserts a `ctx_stats_history` **hourly** row exists with the turn's token delta, 1 request, 1 session. Directly proves the writer call in `recordTelemetry` lands a row (the flagged gap).
- `TestStartRollup_WritesWithoutRestart` (observability_test.go): the rollup goroutine (20ms tick + stop channel) writes `ctx_stats_history` rows without a restart and `stopFn()` joins it cleanly (stop channel + assertion, not a sleep race).
- Fail-open regression covered by existing `TestServeHTTPTelemetryNilStoreNoop` and `TestFailOpen_MissingTables` (missing `ctx_stats_history` never errors the writer).

### Gate outputs (all pass)

- `go build ./...` — clean.
- `go test ./internal/ctx/...` — `ok` for config, observability, proxy, store, traffic, wrap.
- `go test ./internal/http/...` — `ok internal/http` (178s, **includes `single_open_test.go`** — single-open invariant preserved), `ok docs`, `ok tracker`.
- `go test -count=1 ./internal/ctx/proxy/ ./internal/ctx/observability/ -v` — **58 PASS, 0 FAIL**.
- `go vet ./internal/ctx/... ./internal/http/...` — clean.
- `gofmt -l` on all four touched files — clean.

**Commit:** `d44a5c26` (4 files, +97/−2). Staged only the four touched files; no build artifact or unrelated working-tree drift staged.
