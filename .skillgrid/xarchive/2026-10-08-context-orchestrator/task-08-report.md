# T08 Report — Wrap E2E: turn-through-proxy with savings

## Status: DONE

## Commit
`6035d03a` — `test(ctx): add wrap E2E turn-through-proxy with savings` (1 file, 238 insertions)

## What was built

`mnemonic/internal/ctx/wrap/e2e_test.go` (package `wrap`, test-only):

- **`TestWrapE2ETurnThroughProxy`** — the plan-16.10 acceptance proof:
  1. Opens a real shared store (`store.Open(t.TempDir(), "ctxwrap-e2e")`) + `ctxstore.New`.
  2. Stands up a real `httptest` upstream returning a valid 200 chat-completion body.
  3. Builds the real T04 proxy (`ctxproxy.New(cfg, cs, log, Mode("token"))`) on an
     ephemeral `127.0.0.1:0` listener via `http.Server`.
  4. Runs the real T07 `Wrap` launcher with a stub claude agent (`/bin/bash -c` +
     `curl`) that reads the injected `ANTHROPIC_BASE_URL`, POSTs one `/v1/messages`
     turn tagged `x-ctx-session: e2e-sess`, and prints `<status> <savings>`.
  5. Asserts: agent saw `200` + non-zero `X-CTX-Savings`; the upstream received the
     `[ctx-summarized ...]`-transformed body; a telemetry row landed on the real
     store with `TokensAfter < TokensBefore` and `session.Requests == 1`; and the
     agent source tree is byte-identical before/after.

- **Helpers:** `stubAgentEnvAgent(baseVar)` (bash+curl one-shot agent; dumps
  response headers via `curl -D` to a temp file and greps `X-CTX-Savings` —
  macOS curl 8.7 lacks the `header_*` write-out variable),
  `realProxyStoreStartProxy(t, cfg, cs, upstream)` (real proxy + loopback listener,
  returns the `Options.StartProxy` seam closure), `waitForStore` (polls up to 5 s
  for the telemetry row, which is written after the response is flushed).

## Gating (all exit 0)

| Gate | Command | Result |
|------|---------|--------|
| G1 | `go build ./...` | ok |
| G2 | `go test -count=1 ./internal/ctx/wrap/...` | ok (E2E PASS, 0.72 s) |
| G3 | `go test -count=1 ./internal/ctx/proxy/... ./internal/ctx/store/... ./internal/ctx/config/...` | ok |
| G4 | `go vet ./internal/ctx/wrap/...` | ok |
| fmt | `gofmt -l internal/ctx/wrap/` | empty |

## Production change / concern

Test-only — no production code changed. One environment note: the stub agent
pins `/bin/bash` (not `sh`) and uses `curl -D` header dump because the macOS
system curl (8.7.1) predates the `header_*` write-out variable; this is test
infrastructure, not a product dependency. The E2E manipulates the process-global
`ANTHROPIC_BASE_URL` env while the real proxy builds (restored in `t.Cleanup`);
other `wrap` tests read the base URL only from the per-call env slice, so there
is no cross-test interference under the default serial test binary.

## Fix round (review round 1 — F1, F4)

Two Important findings from the spec-compliance review, both closed in
`mnemonic/internal/ctx/wrap/e2e_test.go` (test-only, no production change).

**F1 — process-global env mutation vs future `t.Parallel()`.**
`realProxyStoreStartProxy` did `os.Setenv("ANTHROPIC_BASE_URL", upstream.URL)`
and restored it via `t.Cleanup`. Safe only because no wrap test calls
`t.Parallel()`, but the Setenv→build→restore window was an unguarded
process-global mutation. Closed by adding a package-level
`proxyEnvMu sync.Mutex` and `Lock()`-ing the full Setenv→build→restore window
(`proxyEnvMu.Lock()` at the top of the helper; `proxyEnvMu.Unlock()` inside the
`t.Cleanup` closure that restores the env). Added a one-line `// NOTE:` comment
in the helper stating wrap tests must not call `t.Parallel()` while this
process-env mutation exists. No production test seam added on the proxy (brief
forbids production changes).

**F4 — `upstreamBodies` data race under `go test -race`.**
The httptest upstream handler appended to `upstreamBodies` from the proxy's
goroutine; the test read it after `res.Command.Wait()` with no modeled sync
point. Closed deterministically with a mutex: `upstreamBodies` is now guarded by
`upstreamMu sync.Mutex` — the handler `Lock()`s around its `append`, and the test
`Lock()`s, copies the slice out (`append([]string(nil), upstreamBodies...)`),
`Unlock()`s, and asserts against the copy. `go test -race -count=1
./internal/ctx/wrap/...` is clean.

### Gates (all exit 0, run in `mnemonic/`)

| Gate | Command | Exit |
|------|---------|------|
| G1 | `go build ./...` | 0 |
| G2 | `go test -count=1 ./internal/ctx/wrap/...` | 0 |
| G2r | `go test -race -count=1 ./internal/ctx/wrap/...` | 0 (clean — F4 point) |
| G3 | `go test -count=1 ./internal/ctx/proxy/... ./internal/ctx/store/... ./internal/ctx/config/...` | 0 |
| G4 | `go vet ./internal/ctx/wrap/...` | 0 |
| fmt | `gofmt -l internal/ctx/wrap/` | 0 (empty) |

### Fix commit

`5a5c38fb` — `test(ctx): harden wrap E2E env + race guards` (1 file changed, 29 insertions, 6 deletions).
