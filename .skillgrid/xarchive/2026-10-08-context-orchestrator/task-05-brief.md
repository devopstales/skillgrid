# T05 (TASK-093) — CTX proxy protocol passthrough (HTTP/1.1, HTTP/2, SSE, WebSocket)

## Context
T04 (commit fb2364d0 + fix e9956886) built the proxy core with a streaming-safe 2xx path:
`forward()` now does `io.Copy(w, res.Body)` (headers-before-body, no full buffering), and
`retry.go` returns the 2xx `resp.Body` still open with its `ContentType`. The non-2xx path is
buffered (acceptable — errors are rare + small). T04's review parked 3 Minors that are now in
scope here: M2 (loopback check is a string allow-list — harden with `net.ParseIP().IsLoopback()`),
M3 (Content-Length/Transfer-Encoding passthrough is implicit — make it explicit on the 2xx stream),
M1 (OpenAI e2e test omits `Authorization: Bearer` — add the header + assert the client key is not
forwarded).

T05 ensures the proxy passes HTTP/1.1, HTTP/2, SSE (chunked streaming), and WebSocket frames
INTACT — no body buffering that breaks streaming — and wires the serve entry to enable h2 +
upgrade. This is the streaming seam T04 set up; verify it holds under real protocol fixtures and
close the 3 parked Minors.

## Scope
1. **SSE passthrough (streaming):** a `text/event-stream` response from the upstream must arrive
   chunk-by-chunk IN ORDER and COMPLETE — not buffered/truncated. The savings headers (computed
   from the request transform) are written BEFORE the stream; the stream then `io.Copy`s.
2. **WebSocket passthrough:** a WS handshake `Upgrade` request is detected; the proxy upgrades and
   pipes the hijacked connection both directions (bidirectional pipe) so ping/pong round-trips.
   WS frames must NOT go through the savings-header transform (they're not JSON chat requests) —
   passthrough verbatim.
3. **HTTP/2 passthrough:** an h2 request through the proxy completes (the serve entry must enable
   h2 — `TLSConfig` with NextProtos `h2`, or rely on http/2 auto-negotiation for the loopback
   serve). Verify a single h2 round-trip completes.
4. **Close T04 Minors:**
   - M2: replace the string allow-list loopback check in `ctx_proxy.go` with
     `net.ParseIP(ip).IsLoopback()` (parse the host part of the bind; `0.0.0.0`/`::`/`""`/external
     → warn; `127.0.0.1`/`::1`/`::ffff:127.0.0.1` → no warn).
   - M3: on the 2xx stream path, explicitly copy `Content-Length` and `Transfer-Encoding` from the
     upstream response header to the client (where present) instead of relying on Go re-encoding.
   - M1: add `Authorization: Bearer client-key` to the OpenAI e2e test and assert the upstream did
     NOT receive the client key (auth replacement works on the OpenAI shape).

## Files (Owns)
- `mnemonic/internal/ctx/proxy/stream.go` — WS upgrade detection + bidirectional hijack pipe; any
  streaming helpers. (New file.)
- `mnemonic/internal/ctx/proxy/ws.go` — OR fold WS into stream.go; pick the cleaner split. (Optional new file.)
- `mnemonic/internal/ctx/proxy/{proxy,stream}_test.go`, `ws_test.go` — fixtures: SSE in-order+complete,
  WS handshake + ping/pong, h2 round-trip. (New test files.)
- `mnemonic/cmd/mnemonic/ctx_proxy.go` — serve entry: enable h2 + WS upgrade; apply M2 loopback hardening.
- `mnemonic/internal/ctx/proxy/proxy.go` — M3 explicit Content-Length/Transfer-Encoding passthrough on
  the 2xx stream; route WS upgrades to the hijack pipe before the savings transform.
- `mnemonic/internal/ctx/proxy/savings.go` — (M1) only if the test lives here; otherwise no change.

## Constraints
- ADR-0016 fail-open: a non-upgrading/non-JSON request still gets a 200 passthrough (never 500 on a
  protocol the proxy can't transform).
- ADR-0033: shared store handle (no 2nd DB) — telemetry unchanged.
- No new 3rd-party deps. WebSocket upgrade detection + hijack pipe are stdlib
  (`r.Header.Get("Upgrade")`, `r.Header.Get("Connection")` contains `upgrade`,
  `hijack := w.(http.Hijacker)`, `io.Copy` both directions, `net.Conn`).
- Do NOT change the savings MATH or the transform (T04 closed it). Do NOT add `/v1/compress` or the
  runtime `ctx proxy mode` switch (T06).
- T04's 28 proxy tests must stay green.

## Acceptance (go test ./internal/ctx/proxy/...)
- SSE: a `text/event-stream` upstream with N chunks → all N arrive at the client IN ORDER and
  COMPLETE; savings headers present BEFORE the first chunk; the stream is not truncated.
- WebSocket: a handshake `Upgrade: websocket` request → the proxy upgrades; a ping/pong
  round-trips across the hijacked connection; frames are not buffered or mangled.
- HTTP/2: an h2 request through the serve entry completes with a 200 + body.
- No stream truncated by the proxy (no early close, no half-buffered SSE).
- M1/M2/M3 verified by their respective tests (OpenAI auth-replacement assert; loopback warning
  matrix; Content-Length/Transfer-Encoding present on a fixed-length 2xx).

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
ticket: TASK-093
wave: 4
pr: PR2
```
Stage ONLY your files. NEVER `git add -A` (the repo is shared; `mnemonic/mnemonic` is a tracked
build artifact — do NOT re-stage it).

## Report
Write to `/Users/paladm/git/ai-test/skillgrid-v2/.skillgrid/sdd/tasks/task-05-report.md`:
Status, per-file what-did, gates with exit codes, Work Unit Evidence table, the T04-Minor closure
note (M1/M2/M3), any concerns, commit SHA.

## Reply
Short status (<15 lines): status, commit SHA, gate results (one line each), and whether
SSE+WS+h2 fixtures are all green.
