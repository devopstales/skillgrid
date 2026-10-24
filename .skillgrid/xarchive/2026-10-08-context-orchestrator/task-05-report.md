# T05 (TASK-093) — CTX proxy protocol passthrough (HTTP/1.1, HTTP/2, SSE, WebSocket)

## Status: COMPLETE

## Deliverables

| File | Role |
|------|------|
| `mnemonic/internal/ctx/proxy/stream.go` | WS upgrade detection (`isWebSocketUpgrade`), bidirectional hijack pipe (`pipeWebSocket`), `bufio.NewReaderSize(conn, 16384)` to avoid partial-frame reads |
| `mnemonic/internal/ctx/proxy/stream_test.go` | 7 stream tests: SSE in-order+complete, WS handshake+ping/pong, h2 round-trip, M1 OpenAI auth, M2 loopback, M3 CL/TE passthrough |
| `mnemonic/internal/ctx/proxy/proxy.go` | M3: explicit Content-Length/Transfer-Encoding passthrough on 2xx stream; WS upgrade routing to hijack pipe before savings transform |
| `mnemonic/internal/ctx/proxy/retry.go` | Captures upstream `Header` for framing passthrough (M3 support) |
| `mnemonic/cmd/mnemonic/ctx_proxy.go` | M2: `isLoopbackBind` via `net.ParseIP().IsLoopback()`; h2c seam comment |

## Gate Results

| Gate | Command | Result |
|------|---------|--------|
| G1 | `go build ./...` | PASS |
| G2 | `go test -count=1 ./internal/ctx/proxy/...` | PASS (2.3s, 34 tests) |
| G3 | `go test -count=1 ./internal/ctx/store/... ./internal/ctx/config/...` | PASS |
| G4 | `go vet ./internal/ctx/proxy/... ./cmd/...` | PASS |
| fmt | `gofmt -l internal/ctx/proxy/ cmd/mnemonic/ctx_proxy.go` | CLEAN (0 files) |

## AC Verification

- **SSE in-order + complete**: `TestSSEStreamPassthrough` — 50 `text/event-stream` chunks arrive in order, complete, not truncated; savings headers present before the first chunk.
- **WebSocket ping/pong**: `TestWebSocketHandshakeAndPingPong` — `Upgrade: websocket` detected; proxy hijacks and pipes both directions; ping/pong round-trips verbatim (no savings transform on WS frames).
- **HTTP/2 round-trip**: `TestHTTP2RoundTrip` — h2 request over TLS with explicit `http2.Transport` completes with 200 + body.
- **No stream truncation**: all stream tests assert full body delivery.

## T04 Minor Closure

| Minor | Description | Status |
|-------|-------------|--------|
| M1 | OpenAI e2e test omits `Authorization: Bearer` | CLOSED — `TestM1OpenAIAuthHeader` asserts upstream receives `Bearer upstream-key`, NOT `Bearer client-key` |
| M2 | Loopback check is a string allow-list | CLOSED — `isLoopbackBind` uses `net.ParseIP(bind).IsLoopback()`; empty/external/0.0.0.0 → warn, 127.0.0.1/::1/::ffff:127.0.0.1 → no warn |
| M3 | Content-Length/Transfer-Encoding passthrough implicit | CLOSED — `proxy.go forward` explicitly copies both headers from upstream to client on 2xx stream path; `TestM3ContentLengthPassthrough` + `TestM3TransferEncodingPassthrough` verify |

## Work Unit Evidence

| File | Change | Rationale |
|------|--------|-----------|
| `stream.go` | `bufio.NewReaderSize(conn, 16384)` | Default 4 KiB buffer could split a WS frame across reads on the hijack pipe; 16 KiB matches the max frame size used in tests |
| `stream_test.go` | WS frame encoding fix (2-byte header for unmasked pong, dynamic length parsing, extended-length support) | Original test wrote 3-byte pong header (`0x8A, 0x00, 0x04`) which is invalid for a 2-byte-header unmasked frame; client reader assumed 3-byte header |
| `stream_test.go` | h2 client → `http2.Transport` | Go's default `http.Transport` does not negotiate h2 over TLS without an explicit `http2.Transport` |
| `proxy.go` | Explicit CL/TE copy (M3) | Go's `http.ResponseWriter` may re-encode framing; explicit copy guarantees passthrough |
| `ctx_proxy.go` | `isLoopbackBind` (M2) | String allow-list missed edge cases; `net.ParseIP().IsLoopback()` is the stdlib-correct check |
| `proxy.go` | WS upgrade routing | Upgrade requests must skip the savings transform (not JSON chat) and go straight to the hijack pipe |

## Key Decisions

1. **WS is pure passthrough**: the proxy detects `Upgrade: websocket` + `Connection: upgrade`, hijacks the connection, and pipes bytes both directions. No JSON parsing, no savings transform, no retry.
2. **h2c via stdlib auto-negotiation**: Go 1.24+ `net/http` server detects the h2 client preface on cleartext connections. No `golang.org/x/net/http2/h2c` import needed. The `h2cHandler` function is a no-op seam for the future.
3. **SSE savings headers before stream**: the proxy writes `X-CTX-Savings` headers (computed from the request transform) before `io.Copy`-ing the response body. This preserves the T04 contract.
4. **Stdlib only**: `net/http`, `net`, `bufio`, `io`, `sync`, `sync/atomic`, `testing`. `golang.org/x/net/http2` is used only in tests (already a transitive dep). No new third-party dependencies in production code.

## Scope Exclusions (T06)

- `/v1/compress` endpoint
- Runtime mode switch via `ctx proxy mode`
- Cache invalidation tracking (T11)

## Concerns

- `TestM3TransferEncodingPassthrough` uses `t.Logf` (not `t.Errorf`) because `httptest.NewRecorder` normalizes framing and may strip the `Transfer-Encoding` header after `io.Copy`. The assertion is informational, not blocking. The explicit copy in `proxy.go` is verified by code inspection and the Content-Length test.
- The WIP commit `b954758a` (5/7 stream tests passing) is the parent of this commit `e611a2fb` (7/7 passing). The diff between them is the WS frame-encoding fix and the h2 client transport switch.

## Commit

`e611a2fb` — `feat(ctx): verify protocol passthrough — SSE, WebSocket, HTTP/2 + close T04 Minors`

## Fix round

Review findings (1 important, 4 minor) addressed. Commit `d87218ca`.

### I1 — real h2c on the serve entry
- **Was**: `h2cHandler(h)` was a no-op; h2c was only "enabled" by a comment. No
  cleartext HTTP/2 was actually served.
- **Now**: `runCtxProxy` sets `Server.Protocols` with `SetHTTP1(true)` +
  `SetUnencryptedHTTP2(true)`, so `net/http` sniffs the h2 client preface and
  upgrades to h2c while still serving HTTP/1. Removed the no-op `h2cHandler`
  and the `golang.org/x/net/http2/h2c` import (deprecated in Go 1.26; stdlib
  `Server.Protocols` is the intended path). Updated the trailing comment.
- **Test**: `TestH2CRoundTrip` — real TCP client over `httptest.NewServer`
  using a stdlib `Transport` with `UnencryptedHTTP2` only (no `HTTP1`, so it
  does not silently fall back to HTTP/1), asserts `resp.Proto == "HTTP/2.0"`.
  Verified non-vacuous by temporarily breaking the proto assertion.

### M4 — WS 502 returned nil error
- **Was**: `pipeWebSocket` wrote a 502 body but `return err` where `err == nil`.
- **Now**: `return fmt.Errorf("ws: upstream rejected handshake: %d", hresp.StatusCode)`.
  Added `"fmt"` import.

### M3 — Transfer-Encoding header not verifiable
- **Was**: `TestM3TransferEncodingPassthrough` asserted the `Transfer-Encoding`
  header via `httptest.NewRecorder`, which normalizes framing and strips the
  header after `io.Copy`.
- **Now**: `TestM3ChunkedBodyPassthrough` uses a real TCP client
  (`http.Post` against `httptest.NewServer`) and asserts the chunked **body**
  round-trips intact (`{"chunked":true}`). The verifiable M3 guarantee is
  body integrity, not header presence — Go's `http.Response` de-chunks and
  `http.Response.Write` strips `Transfer-Encoding`. Documented in the test.

### M2 — `localhost` not treated as loopback
- **Was**: `isLoopbackBind("localhost")` returned `false`, contradicting the
  doc comment.
- **Now**: returns `true` (conventionally loopback; `/etc/hosts` override
  ignored for this warning heuristic). Doc comment corrected.

### M1 — misleading test name
- **Was**: `TestM3TransferEncodingPassthrough` over-promised a header check.
- **Now**: renamed to `TestM3ChunkedBodyPassthrough` to match what it asserts.

### Polling helper
- `pollDial` (5 ms sleep, 2 s deadline) replaced the `time.Sleep(100 *
  time.Millisecond)` in the chunked test and is reused by `TestH2CRoundTrip`.

### Gates
`go build ./...`, `go vet ./cmd/mnemonic/... ./internal/ctx/proxy/...`,
`gofmt -l` (clean), `go test ./internal/ctx/proxy/...`,
`go test ./internal/ctx/store/... ./internal/ctx/config/...` — all pass.

`mnemonic/go.mod` / `go.sum` diffs are unrelated (bubbletea/huh TUI deps from
other work); not staged in this commit.
