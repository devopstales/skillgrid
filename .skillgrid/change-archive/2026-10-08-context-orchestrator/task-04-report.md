# T04 (TASK-092) — proxy core: token/cache/passthrough modes + savings headers + fail-open

## Status: COMPLETE

## Deliverables

| File | Role |
|------|------|
| `mnemonic/internal/ctx/proxy/modes.go` | Mode type, parseMode, resolveMode, ResolveMode (exported), messages struct, headSplit |
| `mnemonic/internal/ctx/proxy/savings.go` | EstimateTokens, savings struct, transform (passthrough/token/cache), writeHeaders, summarize |
| `mnemonic/internal/ctx/proxy/retry.go` | Sleep type, callWithRetry, backoffMs, readAll, retryResult |
| `mnemonic/internal/ctx/proxy/proxy.go` | Proxy struct, New, NewWithSleep, Handler, ServeHTTP, forward, detectUpstream, recordTelemetry, sessionID |
| `mnemonic/internal/ctx/proxy/modes_test.go` | Tests: parseMode valid/invalid, resolveMode priority, resolveModeFromEnv, headSplit |
| `mnemonic/internal/ctx/proxy/savings_test.go` | Tests: EstimateTokens stable, transform passthrough/token/cache, savings header accuracy, ratio |
| `mnemonic/internal/ctx/proxy/retry_test.go` | Tests: callWithRetry success/exhaustion/verbatim/transport-error, backoff sequence |
| `mnemonic/internal/ctx/proxy/proxy_test.go` | Integration: passthrough delta-zero, token AC fixture exact savings, fail-open, provider retry verbatim, 502, telemetry write, nil-store noop, session derivation |
| `mnemonic/cmd/mnemonic/ctx_proxy.go` | `ctx proxy` subcommand: flag parsing, config load, store open, proxy New, http.Server with graceful shutdown |
| `mnemonic/cmd/mnemonic/ctx.go` | Added `case "proxy": runCtxProxy(rest)` dispatch line |

## Gate Results

| Gate | Command | Result |
|------|---------|--------|
| G1 | `go build ./...` | PASS |
| G2 | `go test ./internal/ctx/proxy/...` | PASS (1.8s) |
| G3 | `go test ./internal/ctx/store/... ./internal/ctx/config/...` | PASS |
| G4 | `go vet ./internal/ctx/proxy/... ./cmd/...` | PASS |
| fmt | `gofmt -l` on changed files | CLEAN |

## Key Decisions

1. **Mode resolution priority**: CLI `--mode` flag > `CTX_PROXY_MODE` env > config value > default `cache`. Implemented as `resolveMode(override, cfgMode, envValue)` with a thin `ResolveMode` exported wrapper for the CLI.

2. **Transform is pure**: `transform(mode, messages) -> (messages, error)` operates on a normalized `messages` struct (role + content string), not raw JSON. This makes it provider-agnostic and testable. A parse/encode fault fails open to passthrough (ADR-0016).

3. **EstimateTokens = `len(strings.Fields(string(b)))`**: deterministic word count on the exact wire bytes. The "before" is counted on the client's original body; the "after" on the forwarded (possibly compressed) body. Delta = before - after.

4. **Retry budget**: `1 + maxRetries` total attempts. Backoff sequence: base, 2*base, 4*base, ... (exponential). A transport-level error (no upstream status) after the final attempt yields a 502; a non-2xx upstream response is returned verbatim.

5. **Telemetry is fail-open**: a nil ctx store or a missing table is a no-op write. The proxy never 500s because telemetry failed.

6. **Session ID derivation**: `X-CTX-Session` header if present, else `sess-` + first 8 bytes of MD5(RemoteAddr). Deterministic per client address.

7. **Stdlib only**: `net/http`, `net/http/httputil` (not used in T04 — direct forwarding), `crypto/md5`, `encoding/hex`. No new third-party dependencies.

## AC Verification

- **AC a (passthrough delta zero)**: `TestServeHTTPPassthroughDeltaZero` — original body forwarded verbatim, X-CTX-Savings = 0, X-CTX-Mode = passthrough.
- **AC b (token exact savings)**: `TestServeHTTPTokenExactSavings` — the 3-message fixture compresses, X-CTX-Savings = before - after exactly, forwarded body carries `[ctx-summarized` marker.
- **AC c (fail-open)**: `TestServeHTTPFailOpenOnTransformFault` — unparseable body degrades to passthrough, 200 returned, original body forwarded.
- **AC d (provider retry verbatim)**: `TestServeHTTPProviderRetryVerbatim` — 429 retried 1+2=3 times, final 429 + body passed through verbatim.
- **Telemetry**: `TestServeHTTPWritesTelemetry` — per-turn savings row + session row written to a real shared store.
- **Nil-store noop**: `TestServeHTTPTelemetryNilStoreNoop` — nil store, 200, headers still written.

## Scope Exclusions (T05/T06)

- `/v1/compress` endpoint (T06)
- Runtime mode switch via `ctx proxy mode` (T06)
- h2/SSE/WS streaming pass-through fixtures (T05)
- Cache invalidation tracking (T11)

## Fix Round (review findings)

| # | Finding | Fix |
|---|---------|-----|
| Q1 | `parseMessages`/`encodeMessages` dropped non-message fields (`model`, `max_tokens`, etc.) | Replaced with `applyTransform` in `savings.go`: decodes body into `map[string]json.RawMessage`, transforms only the `messages` array in-place, re-encodes the full object. Non-message fields preserved verbatim. |
| Q2 | `forward` buffered the entire upstream body into `forwardResult` | `forward` now accepts `http.ResponseWriter` and streams via `io.Copy`. `retry.go` returns `io.ReadCloser` (with `readAllCloser` for non-2xx final attempts) and captures upstream `ContentType`. |
| Q3 | Proxy could bind to a non-loopback address with no warning | `ctx_proxy.go` logs a warning when the listen address is not loopback. |
| Q5 | `ctxUsage()` missing `ctx proxy` subcommand | Added `ctx proxy` description line. |
| Q6 | `errNoUpstream` sentinel ambiguous with transport errors | Replaced with `noUpstreamStatus` sentinel; transport errors use `errors.Join(noUpstreamStatus, err)`. |
| Q7 | No streaming (SSE) pass-through test | Added `TestServeHTTPStreamResponsePassthrough` (256 KiB SSE body). |
| AC 2b | No OpenAI-compat endpoint test | Added `TestServeHTTPOpenAICompatEndToEnd` for `/v1/chat/completions`. |
| — | No test for non-message-field preservation | Added `TestTransformPreservesNonMessageFields`. |

All gates re-verified after fixes: `go build ./...`, `go vet`, `gofmt -l`, `go test` — PASS.
