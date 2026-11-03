# T11 Brief — session summaries rollup + writer (TASK-099)

## Goal
Populate `ctx_session_summaries` (one wide row per session) so the `/api/session-summaries`
endpoint (already built in T10) returns real data. This is the session-level aggregator the
plan calls the "per-session wide event aggregated at SessionEnd" (§8.8, §9.1, §17.3).

The reader already exists: `GET /api/session-summaries` in `mnemonic/internal/http/ctx.go`
(`handleCTXSessionSummaries`, offset pagination over `store.ListSessionSummaries`). This task
builds the WRITER. It must satisfy acceptance scenarios `session-summary-write` and
`session-summary-read`.

## What the proxy gives us today (do NOT rewrite it)
`recordTelemetry` (proxy.go) already accumulates one row per session in `ctx_proxy_sessions`
via `GetProxySession` + `PutProxySession`, incrementing `Requests`, `TokensBefore`,
`TokensAfter`, `CacheHits`, `CacheInvalidations`, and setting `StartedAt`/`EndedAt`. That
table is the authoritative per-session source. The summary rollup DERIVES from it — it does
not re-parse `ctx_proxy_savings`.

## The seam
There is no dedicated per-turn SessionEnd hook in the loopback proxy (the agent's SessionEnd
is the orchestrator's concern and is out of scope here). So the summary rollup runs on the
SAME cadence as the T09 stats rollup:
- A periodic tick (reusing the `StartRollup`-style goroutine in `runCtxProxy`, `ctx_proxy.go`).
- One FINAL rollup at proxy shutdown (before the store handle is closed), so a just-ended
  session is always summarized.

This satisfies "at session end (or interval)" and "no double-write".

## Implementation

### 1. `mnemonic/internal/ctx/observability/` — session summary rollup (new)
Add a `session_summary.go` (or extend `observability.go`) with:

```go
// SummarizeSessions derives one ctx_session_summaries row per distinct session from
// ctx_proxy_sessions and writes them. Returns the number of rows written.
// Fail-open (ADR-0016): a nil store or missing table returns (0, nil), never an error.
func SummarizeSessions(st *store.Store, now int64) (int, error)
```

For each distinct `session_id` in `ctx_proxy_sessions` (use a new store helper
`ListProxySessions` — it already exists at tables.go:130 — and group by session):
- `session_id` = the session id.
- `agent` = the proxy session's `Agent`.
- `duration_ms` = (EndedAt - StartedAt) in ms, clamped to >= 0.
- `turns` = Requests.
- `tokens_saved` = TokensBefore - TokensAfter, clamped to >= 0.
- `judgment_keeps` / `judgment_truncs` / `prefilter_pins` / `prefilter_deads` = 0 for now
  (T14 judgment layer populates these; leave the columns at 0 — do not fabricate).
- `cache_invalidations` = the proxy session's `CacheInvalidations`.
- `shaping_applied` = NULL for now (no shaping signal yet; leave it nil).
- `effort_routed` = 0.
- `traffic_memories` = 0 (T12/T14 may populate; leave 0).
- `summary_json` = a small deterministic JSON object, e.g.
  `{"mode":"<mode>","requests":N,"tokens_before":B,"tokens_after":A,"tokens_saved":S}` —
  the mode and counters straight from the proxy session row.
- `created_at` = `now`.

Write with `st.PutSessionSummary(&store.SessionSummary{...})` (tables.go:245). The PK is
`session_id` with `INSERT OR REPLACE` semantics (tables.go:259), so re-rolling the same
session is IDEMPOTENT — no double-write. That is how "interval honored (no double-write)"
is met: re-running the rollup updates the same row, it does not append.

### 2. Wire it into the rollup loop
- In `observability.go`, extend the existing `StartRollup` (or add a companion
  `StartSessionRollup`) so each tick ALSO calls `SummarizeSessions(st, now)`. The simplest
  correct approach: make the existing rollup tick call both `Roll`/bucket-write AND
  `SummarizeSessions`. Keep it fail-open and non-blocking.
- In `cmd/mnemonic/ctx_proxy.go` `runCtxProxy`, ensure a FINAL `SummarizeSessions` runs at
  shutdown, AFTER the deferred `stopRollup()` has joined the goroutine but BEFORE `cleanup()`
  closes the handle. (The store is still open at that point.) If the existing deferred chain
  already has a natural place, add a final summarize there.
- Respect the config gate: only run the summary rollup when `cfg.Observability.SessionSummary`
  is true (it defaults true). If the rollup goroutine already receives the config or a flag,
  thread `SessionSummary` through; otherwise guard the final + periodic summarize call.

### 3. Store helper (only if needed)
`ListProxySessions()` already returns all sessions (tables.go:130). If you need to group by
session id in Go, that's fine — fetch all and map. Do not add a new migration.

## Constraints
- ADR-0016 fail-open: every store access degrades to zero/empty, never an error that 500s.
- ADR-0033: reuse the shared handle (`ctxstore.New(h.Store())`), never open a second DB.
- Idempotent: rolling the same session twice must NOT create two rows (PK = session_id,
  INSERT OR REPLACE).
- Stdlib only. No new dependency, no go.work change.
- Do not touch the traffic learner (T12), judgment (T14), or the /api reader (T10) beyond
  what's needed.
- Keep the single-open invariant (`single_open_test.go`) green.

## Acceptance gates (run these; all must pass)
- `go build ./...`
- `go test ./internal/ctx/observability/... -count=1`
- `go test ./internal/http/... -run 'TestCTX|Test.*SessionSumm|TestSingleProjectRoutesUseHandle|Test.*OpenStoreOnce' -count=1`
- `go vet ./internal/ctx/observability/... ./internal/http/... ./cmd/...`
- `gofmt -l` on every file you touched (must be empty)

Required tests to add in `mnemonic/internal/ctx/observability/`:
1. **Write**: seed a real shared store + a couple of `ctx_proxy_sessions` rows (2 sessions,
   distinct counts) → call `SummarizeSessions` → `ListSessionSummaries` returns exactly one
   row per session with correct `turns`, `tokens_saved`, `duration_ms`, `summary_json`
   (mode present), and `created_at` set.
2. **Idempotent (no double-write)**: call `SummarizeSessions` twice → still exactly one row
   per session (PK replace), no duplicates.
3. **Fail-open**: a store missing the 056-061 migrations (or a nil store) → `SummarizeSessions`
   returns `(0, nil)`, no error.
4. **Read-through** (HTTP, in `internal/http`): after seeding sessions + running
   `SummarizeSessions`, `GET /api/session-summaries` (loopback) returns the written rows.

## Report
Write the FULL report to `.skillgrid/sdd/2026-10-08-context-orchestrator/task-11-report.md`:
what changed (files + functions), the idempotency mechanism, the shutdown-final-rollup seam,
config gate, and gate outputs (pass/fail + counts). Also generate
`.skillgrid/sdd/2026-10-08-context-orchestrator/task-11.diff` (git diff of your commits).

## Commit
Conventional subject, e.g. `feat(ctx): session summaries rollup + writer`. EXACTLY this body
block:
```
[skillgrid-context]
change: 2026-10-08-context-orchestrator
ticket: TASK-099
wave: 6
pr: PR4
```
