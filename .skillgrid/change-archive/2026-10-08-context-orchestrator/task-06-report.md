# T06 Report — /v1/compress endpoint (loopback-only) + mode-switch runtime

**Ticket:** TASK-094 (spec T06 / TICKET-06) · Wave 4 · PR2
**Commit:** `b478a781`
**Status:** DONE — all 4 ACs green, all gates exit 0.

## What I built

### `internal/ctx/proxy/compress.go` (new)
- `compressPath = "/v1/compress"`, `statsPath = "/stats"` route constants.
- `serveCompress` — the loopback-only POST handler:
  - Loopback origin check **first** (`isLoopbackRequest`); non-loopback → `http.NotFound` (404, not 403), method-agnostic.
  - Runs the **same** transform the forward path uses (`applyTransform`, savings.go) against the current mode; a nil/empty result or a transform fault fails open to the untransformed input with **200** (ADR-0016).
  - Never dials the upstream; never emits savings headers.
- `isLoopbackRequest` — parses the host from `r.RemoteAddr` via `net.SplitHostPort`/`net.ParseIP` and asks `net.IP.IsLoopback()` (mirrors T05's hardened `isLoopbackBind`). Missing/unparseable host → non-loopback (fail closed for a loopback-only route).
- `serveStats` — GET handler returning JSON with the **current mode** (`"mode"`) plus live session counters summed from `store.ListProxySessions()` (nil store / missing table → zeros, never 500).

### `internal/ctx/proxy/mode.go` (new)
- `GetMode() Mode` — thread-safe RLock read of the current mode (hot forward path).
- `SetMode(new Mode, reason string) (old Mode, err error)`:
  - Validates `new` with `valid()`; an invalid mode returns an error and leaves the mode untouched (no row, no event).
  - No-ops when `new == current` (no history row, no event).
  - Otherwise flips `p.mode` under the write lock, mints a `ChangeID` (`mode-<unixnano>`), writes a `ctx_proxy_mode_history` row via `store.PutModeChange` (nil store → no-op), records `lastChangeID`, and fires `SetModeEventSink` (nil sink → no-op). History write + sink fire **outside** the lock (no I/O / sink held under the write lock).
- `lastModeChangeID()` — test hook returning the most recent `ChangeID` (the store exposes `GetModeChange` by ID but no list).

### `internal/ctx/proxy/proxy.go` (route wiring only)
- Added `sync` import.
- `Proxy` struct: added `mu sync.RWMutex`, `lastChangeID string`, and `SetModeEventSink func(old, new Mode, reason string)`; moved `mode` under the mutex with a doc comment.
- `ServeHTTP`: short-circuit routes for `/v1/compress` and `/stats` **before** the forward path; the forward path's mode read changed from `p.mode` to `p.GetMode()` (thread-safe).

### Tests
- `compress_test.go` (new) — 7 tests.
- `mode_test.go` (new) — 6 tests.

## AC → test map

| AC | Test(s) | Result |
|----|---------|--------|
| 1. Loopback `/v1/compress` (compressed list, valid JSON, 0 upstream) | `TestCompressLoopbackNoUpstream` (token), `TestCompressLoopbackCacheMode` (cache leaves frozen head intact) | PASS |
| 2. Non-loopback `/v1/compress` → 404 (asserted NOT 403), method-agnostic | `TestCompressNonLoopback404Not403` (POST), `TestCompressNonLoopbackAnyMethod404` (GET) | PASS |
| 3. Mode switch emitted: history row (old=cache,new=token) via `GetModeChange` + `proxy-mode-changed` event + `/stats` reports `token` | `TestSetModeWritesHistoryRow`, `TestSetModeEmitsEventAndFlipsMode`, `TestStatsReportsCurrentMode` | PASS |
| 4. Restart-to-mode: fresh proxy from `CTX_PROXY_MODE=token` starts in `token` | `TestRestartToModeFromEnv` | PASS |

Extra coverage: `TestCompressFailOpenReturnsInput` (ADR-0016 transform fault → 200 + input echo), `TestCompressNilStoreNoop`, `TestSetModeSameModeNoop`, `TestSetModeInvalidModeErrors`, `TestSetModeNilStoreNoop`.

## Gates (all exit 0)

| Gate | Command | Exit |
|------|---------|------|
| G1 | `go build ./...` | 0 |
| G2 | `go test -count=1 ./internal/ctx/proxy/...` | 0 |
| G3 | `go test -count=1 ./internal/ctx/store/... ./internal/ctx/config/...` | 0 |
| G4 | `go vet ./internal/ctx/proxy/... ./cmd/...` | 0 |
| fmt | `gofmt -l internal/ctx/proxy/ cmd/mnemonic/ctx_proxy.go` | empty |

Full `./internal/ctx/proxy/...` suite (T04 forwarding + T05 streaming + new T06) is green — no regressions.

## Staging
Staged only the five owned files (never `git add -A`):
`compress.go`, `mode.go`, `proxy.go`, `compress_test.go`, `mode_test.go`.
The `mnemonic/mnemonic` build artifact and unrelated modified files were NOT staged.

## Concerns / parked items
- **No new store method needed.** `/stats` reuses `store.ListProxySessions()` for live counters; `ListModeChanges` was not added (the brief allows it only if `/stats` needs history it can't get — it doesn't).
- **`SetMode` `ChangeID` uses `time.Now().UnixNano()`**, not a UUID. Two switches in the same nanosecond would collide, which is not realistic at runtime (and the same-mode no-op path mints no ID). A later ticket can swap in a proper ID generator behind the same call site.
- **`/stats` counters are session-aggregated** (sum of `ListProxySessions`), matching the existing telemetry model. A dedicated rollup table would be a T10/T11 concern, not T06.
- **`X-Forwarded-For` not consulted.** The loopback-only proxy has no reverse proxy in the design, so the origin is read from `RemoteAddr`. If a fronting proxy is ever added, `isLoopbackRequest` is the seam to extend.
