# T06 — /v1/compress endpoint (loopback-only) + mode-switch runtime

**Ticket:** TASK-094 (spec T06 / TICKET-06) · Wave 4 · PR2 · Blocks: T08, T13 · Blocked by: T04 (done)
**SATISFIES:** `compress-endpoint-loopback`, `mode-switch-emitted` (acceptance.feature @step-06, @step-07)

## Goal
Add two runtime surfaces to the existing proxy (T04) and wire the mode into `/stats`:
1. `POST /v1/compress` — compresses a message list with NO upstream completion. **Loopback-only**: a non-loopback origin gets **404 (not 403)**.
2. Runtime mode switch — flips the active `Mode`, writes a `ctx_proxy_mode_history` row (old+new), emits a `proxy-mode-changed` event, and exposes the current mode in `GET /stats`. Honors `CTX_PROXY_MODE` env + restart-to-mode.

This is the DISTINCT trust boundary the plan calls out (loopback-only `/v1/compress`) + the mode runtime. It does NOT touch the forwarding/savings/telemetry core (T04) or streaming (T05).

## What already exists (DO NOT rebuild)
- `internal/ctx/proxy/proxy.go` — `*Proxy{ cfg, store *store.Store, log, mode Mode }`; `mode` is a plain field (comment already says "switchable by T06"). `Handler()` has a route switch on `r.URL.Path`; the forward path calls `applyTransform(mode, body)`, `savings`, `recordTelemetry`.
- `internal/ctx/proxy/modes.go` — `Mode` type + `ModeToken/ModeCache/ModePassthrough`, `valid()`, `String()`, `parseMode`, `resolveMode`, `ResolveMode` (CLI entry: CLI `--mode` > `CTX_PROXY_MODE` > config, default `cache`).
- `internal/ctx/proxy/savings.go` — the compression transform (`applyTransform`) + `savings` math. REUSE for `/v1/compress`; do not reimplement.
- `internal/ctx/store/tables.go` — ALREADY has `ModeChange` struct + `PutModeChange` + `GetModeChange` (table `ctx_proxy_mode_history`, migration 058), plus `ProxySession`/`ProxySaving`/`StatsBucket`/`Rollup`. **The store is done (T02) — you only need to CALL it, not add methods** (unless `/stats` needs a rollup that doesn't exist — check first).
- `recordTelemetry` in proxy.go already writes `ProxySaving` + `ProxySession` per request (nil store / missing table → no-op, ADR-0016).

## Acceptance (must be green)
`go test ./internal/ctx/proxy/...` exits 0, with tests proving:
1. **Loopback `/v1/compress`**: a loopback `POST /v1/compress` with a message-list body returns a **compressed message list** (valid JSON, `messages` transformed per the cache/passthrough transform) and **no upstream call** is made (assert the upstream httptest server saw 0 requests).
2. **Non-loopback `/v1/compress`**: a `POST /v1/compress` from a non-loopback origin returns **404** and the test explicitly asserts it is **NOT 403**.
3. **Mode switch emitted**: switching cache→token at runtime (a) writes a `ctx_proxy_mode_history` row with `old_mode=cache`, `new_mode=token` (assert via `store.GetModeChange`), (b) emits a `proxy-mode-changed` event, and (c) `GET /stats` then reports the current mode as `token`.
4. **Restart-to-mode**: setting `CTX_PROXY_MODE=token` + constructing a fresh proxy starts in `token` mode (assert `GetMode()=="token"`).

## Design / constraints
- **Files (Owns):** `internal/ctx/proxy/compress.go` (the `/v1/compress` handler + loopback origin check), `internal/ctx/proxy/mode.go` (runtime mode switch + `GetMode` + the mode field's read/mutex), `internal/ctx/proxy/proxy.go` (route `/v1/compress` + `/stats` in the `Handler()` switch; wire the switch into the forward path), `internal/ctx/proxy/{compress,mode}_test.go`. You MAY add a `ListModeChanges` store method ONLY if `/stats` needs history it can't get — otherwise use what exists.
- **Loopback-only 404 (not 403):** check the request origin the same way T05 hardened `isLoopbackBind` — `net.ParseIP(host).IsLoopback()`. A non-loopback origin to `/v1/compress` returns `404 Not Found` (route "invisible", plan 11.18/R67) — deliberately 404, not 403, so a non-loopback scanner can't distinguish the route from a missing path. Return 404 for `/v1/compress` regardless of method when non-loopback.
- **No upstream on compress:** `/v1/compress` runs the SAME transform pipeline (savings.go `applyTransform` + redaction boundary) but returns the compressed list directly — it must NOT dial the upstream and must NOT emit savings headers (there's no before/after vs. an upstream response). Reuse the transform; do not duplicate the math.
- **Mode switch runtime:**
  - `GetMode() Mode` returns the current mode (thread-safe).
  - `SetMode(new Mode, reason string) (old Mode, err error)`: validate with `parseMode`/`valid()`; if same as current, no-op (no history row). Otherwise update the field, write `store.PutModeChange(&ModeChange{ChangeID:<uuid or ts>, SessionID:<current session or "">, OldMode, NewMode, Reason, ChangedAt: now})` (nil store → no-op), and emit a `proxy-mode-changed` event.
  - **Event emission:** keep it simple + testable — a `SetModeEventSink func(old, new Mode, reason string)` field on `*Proxy` (set by the CLI/serve entry to log + record); the test sets a capturing sink and asserts it fired with cache→token. Do NOT invent an event-bus package.
  - **Concurrency:** `mode` is read on the hot forward path and written by `SetMode`. Guard with a `sync.RWMutex` (or `atomic.Value`); the forward path takes `RLock` to read the mode. Keep it cheap — no lock held across I/O.
- **`CTX_PROXY_MODE` + restart-to-mode:** `resolveMode` already reads the env (T04). The AC is that a FRESH proxy built with `CTX_PROXY_MODE=token` starts in `token`. Ensure `New`/`resolveMode` honors it (it does — verify the test proves it). No new env plumbing needed.
- **`GET /stats`:** returns live session metrics as JSON and MUST include the current mode (`"mode": "token"`). Reuse the store rollup/`ListProxySessions` for the live counters if available; the key T06 requirement is that the current mode is exposed. Keep it loopback-only consistent with the proxy (it's served on the same loopback listener).
- **ADR-0016 fail-open:** a compression error on `/v1/compress` degrades gracefully (return the untransformed list with 200, or a clear JSON error — do NOT 500 the client for a transform fault; pick one and document it, prefer returning the input list so the caller still completes). Nil store / missing table → no-op (no 500).
- **ADR-0033:** shared `*store.Store` handle — reuse `p.store`, never open a 2nd DB.
- **No new 3rd-party deps** (stdlib only).
- **Do NOT change** the savings math/transform, the forward path's existing behavior, or the T05 streaming path. `/v1/compress` and `/stats` are NEW routes that short-circuit before the forward path.
- **Do NOT build** `ctx wrap` (T07), the mode CLI subcommand (`ctx proxy mode token` is T14's CLI wiring — you expose the runtime method + env, T14 calls it), or the observability endpoints `/stats-history`/`/metrics` (T10/T11).

## Gates (ALL exit 0)
- G1: `cd /Users/paladm/git/ai-test/skillgrid-v2/mnemonic && go build ./...`
- G2: `cd /Users/paladm/git/ai-test/skillgrid-v2/mnemonic && go test -count=1 ./internal/ctx/proxy/...`
- G3: `cd /Users/paladm/git/ai-test/skillgrid-v2/mnemonic && go test -count=1 ./internal/ctx/store/... ./internal/ctx/config/...`
- G4: `cd /Users/paladm/git/ai-test/skillgrid-v2/mnemonic && go vet ./internal/ctx/proxy/... ./cmd/...`
- fmt: `gofmt -l internal/ctx/proxy/ cmd/mnemonic/ctx_proxy.go` empty.

## Commit
Conventional subject (feat(ctx): ...). Body block exactly:
```
[skillgrid-context]
change: 2026-10-08-context-orchestrator
ticket: TASK-094
wave: 4
pr: PR2
```
Stage ONLY your files. NEVER `git add -A`. Do NOT re-stage the `mnemonic/mnemonic` build artifact.

## Report
Write the FULL report to `/Users/paladm/git/ai-test/skillgrid-v2/.skillgrid/sdd/2026-10-08-context-orchestrator/task-06-report.md`: what you built (files + responsibilities), each AC mapped to its test, gates with exit codes, commit SHA, and any concerns/parked items.
