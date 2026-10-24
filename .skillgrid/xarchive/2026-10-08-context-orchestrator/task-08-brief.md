# T08 (TASK-096) — wrap E2E: agent turn through proxy with savings

One ticket, one work unit. This is a **test-only** ticket: it adds an
end-to-end test that proves the launcher (T07) + proxy (T04/T05) + store (T02)
path work together — a wrapped agent completes a full turn through the proxy and
the savings headers are present on the response, with the agent's source tree
byte-identical before/after. No new production code is expected (the E2E may
reveal a small wiring gap — if so, fix it minimally and say so in the report).

## What you're building
- A new file: `mnemonic/internal/ctx/wrap/e2e_test.go` (package `wrap`, so it
  can use the unexported `Wrap`, `Options`, `StartProxy` seam, `newTestCfg`,
  `envWith`, `agentSourceTree` helpers already in `wrap_test.go`).
- An optional tiny shared test helper if you need one that doesn't already
  exist (e.g. an `openSharedStore` for the wrap package). Keep it test-only.

## The E2E contract (this is the whole point)
1. **Open a real shared store** (migrations applied) — the same pattern the
   proxy tests use, but the wrap package can't see the proxy's package-private
   helper, so do it here:
   - `import store "github.com/devopstales/skillgrid/mnemonic/internal/store"`
   - `shared, err := store.Open(t.TempDir(), "ctxwrap-e2e")`
   - `cs := ctxstore.New(shared)` where `ctxstore "github.com/devopstales/skillgrid/mnemonic/internal/ctx/store"`.
2. **Build a REAL proxy with that store** (NOT nil) and a REAL loopback
   listener, so telemetry is recorded and savings headers are emitted:
   - `up := httptest.NewServer(...)` — an upstream that returns a valid
     OpenAI/Anthropic-style chat-completion JSON body (enough for the proxy to
     parse `messages` and compute a token delta; reuse the shape the T04
     `newUpstream`/savings tests use — read `internal/ctx/proxy/proxy_test.go`
     for the exact upstream JSON).
   - `cfg := newTestCfg(t, nil)`; set `cfg.Proxy.BaseURL = up.URL`,
     `cfg.Proxy.APIKey = "upstream-key"`, `cfg.Proxy.Mode` to a mode that
     produces a non-zero delta (e.g. `"token"`), and `cfg.Proxy.SavingsHeaders
     = true` (so `X-CTX-Savings` is emitted).
   - `p := ctxproxy.New(cfg, cs, log, ctxproxy.Mode(cfg.Proxy.Mode))`.
   - Listen on `127.0.0.1:0` (ephemeral loopback), `srv := &http.Server{Handler: p.Handler()}`, serve in a goroutine, base URL = `http://<addr>`.
3. **Run the launcher through `Wrap`** with a `StartProxy` seam that returns
   YOUR base URL + addr + a stop func (so the launched stub points at the real
   proxy with the real store):
   - `Options{ Cfg: cfg, AgentName: "claude", Binary: <stub>, Args: [...],
     Env: envWith("PATH=/usr/bin:/bin"), StateDir: t.TempDir(),
     BackupDir: t.TempDir(), Stdout/Stderr: buffers }`.
   - **Stub agent**: a `sh -c` (or a tiny Go test helper) that issues ONE real
     HTTP request to the proxy for `claude` — i.e. POST to
     `<ANTHROPIC_BASE_URL>/v1/messages` (Anthropic path) with a
     `{"model":..., "messages":[{"role":"user","content":"hi"}]}` body and the
     client auth header, and prints the response status + the
     `X-CTX-Savings` header value to its stdout (which you capture via
     `Options.Stdout`). This is the "agent runs a turn through the proxy".
   - The stub MUST use the injected `ANTHROPIC_BASE_URL` (proves injection
     reached the subprocess) — read it from the env the stub was launched with.
4. **Assert the turn completed with savings headers present:**
   - The captured stub stdout contains the `X-CTX-Savings` header (a non-empty
     numeric value) and an upstream 2xx — i.e. the request actually round-tripped
     through the proxy to the upstream and back.
   - The savings headers present end-to-end (not just unit-level): `X-CTX-Savings`
     is the required one; you may also assert `X-CTX-Original-Tokens` /
     `X-CTX-Compressed-Tokens` / `X-CTX-Mode` are present.
5. **Assert the agent's source tree is byte-identical** before/after the launch
   using the existing `agentSourceTree(t)` checksum helper (call it before and
   after `Wrap` + `Command.Wait()`; assert equal). This is AC #6 / DoD #6.
6. **Telemetry landed in the shared store (stronger, optional-but-recommended):**
   after the turn, `cs.ListProxySavings(sessionID)` (or query by the session the
   proxy derived) is non-empty and `cs.GetProxySession(sessionID)` shows
   `Requests >= 1`. If asserting this, use the `X-CTX-Session` header you set on
   the stub request so the session id is deterministic. (If this adds friction,
   the savings-header assertion is the mandatory gate; the store assertion is a
   bonus.)

## Acceptance (the test that MUST exist and pass)
- `TestWrapE2ETurnSavingsHeaders` (or similar): the wrapped claude turn reaches
  the real proxy (real store), completes with `X-CTX-Savings` present on the
  response, and the agent source tree is byte-identical.
- Fails-when: E2E test exit != 0 OR the savings header is absent on the turn response.

## Constraints
- OWN ONLY: `mnemonic/internal/ctx/wrap/e2e_test.go` (+ any test-only helper in
  the same package). Do NOT change production code unless the E2E reveals a real
  wiring bug — if you do, keep it minimal, list it in the report, and make sure
  existing tests still pass.
- Do NOT require a real claude/codex install — the stub issues the HTTP turn.
- stdlib only (net/http, net/http/httptest, os, os/exec, strings, testing, and
  the two internal packages `internal/store` + `internal/ctx/store` +
  `internal/ctx/proxy`). No new 3rd-party deps.
- Reuse existing helpers in `wrap_test.go` (`newTestCfg`, `envWith`,
  `agentSourceTree`, `envGet`) — do not redefine them.

## Gates — run in `mnemonic/`, ALL must exit 0
- G1: `go build ./...`
- G2: `go test -count=1 ./internal/ctx/wrap/...` (E2E + existing wrap tests)
- G3: `go test -count=1 ./internal/ctx/proxy/... ./internal/ctx/store/... ./internal/ctx/config/...` (no regressions)
- G4: `go vet ./internal/ctx/wrap/...`
- fmt: `gofmt -l internal/ctx/wrap/` (must be empty)

## Commit
Conventional subject `test(ctx): add wrap E2E turn-through-proxy with savings`
(or `fix(ctx):` if you had to touch production code) and this exact body block:
  [skillgrid-context]
  change: 2026-10-08-context-orchestrator
  ticket: TASK-096
  wave: 5
  pr: PR3

## Report (required)
Write the FULL report to
`.skillgrid/sdd/2026-10-08-context-orchestrator/task-08-report.md`: the E2E
harness design (shared store + real proxy + stub agent), exactly how the turn
is issued and how the savings header is asserted, whether you also asserted the
store telemetry, any production wiring change (if any) + why, gate commands with
exit codes, commit SHA, concerns/parked items.

## Reply
Keep your reply to the orchestrator SHORT (<15 lines): DONE or BLOCKED, commit
SHA, gate exit codes (G1-G4 + fmt), and one line on any production wiring change
or concern. Full detail goes in the report file, not the reply.
