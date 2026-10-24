# Task Brief — T04 (TASK-092): proxy core — token/cache/passthrough modes + savings headers + fail-open

- Ticket: TASK-092 (type: feature, priority: high)
- Change: 2026-10-08-context-orchestrator (CTX v1.11)
- Wave 2 · PR2 (base = PR1/this branch) · `mnemonic/internal/ctx/proxy/` + serve entry
- Reversibility: TWO-WAY (new package + one new CLI subcommand; no schema change)
- Precondition: T02 (ctx_ store) + T03 (ctx config) done.
- SATISFIES: proxy-three-modes, proxy-fail-open-provider-timeout

## Scope
Build the loopback LLM proxy: a `net/http` + `httputil.ReverseProxy` bound to
127.0.0.1:8787. Three modes (token, cache [default], passthrough). Forwards
Anthropic- and OpenAI-compatible requests, computes an ACCURATE `X-CTX-Savings`,
adds the savings headers, and fails open to passthrough on any compression fault
mid-request. Provider retry with exponential backoff; after the final retry returns
the provider error verbatim. Writes per-turn telemetry to `ctx_proxy_savings` +
`ctx_proxy_sessions` via the T02 store.

## SCOPE BOUNDARY (do NOT build these in T04 — later tickets)
- **T05** owns protocol passthrough guarantees (HTTP/2 upgrade, SSE chunked
  streaming, WebSocket round-trip). T04's proxy must NOT buffer a streaming body in a
  way that would break it (forward `Content-Length`/chunked headers + use
  `httputil.ReverseProxy`), but the h2/SSE/WS test fixtures are T05's job.
- **T06** owns `POST /v1/compress` (loopback-only, 404 non-loopback) and the
  RUNTIME mode switch (`ctx proxy mode <m>`, `ctx_proxy_mode_history` row +
  `proxy-mode-changed` event + `/stats` current-mode). T04 defines the Mode
  type + startup `--mode` flag + `CTX_PROXY_MODE` env read, but NOT the runtime
  switch command nor the /v1/compress route. Leave a clean seam (Mode is a field,
  switchable) for T06 to plug in.

## Consumed APIs (already built — do not re-derive)
- Config: `mnemonic/internal/ctx/config` —
  - `cfg, err := config.Load(ctx, startDir)` → `*config.CtxConfig`.
  - `cfg.Proxy` has: `Enabled, Bind, Port, Mode, CompressEndpoint, NonLoopbackStatus,
    SavingsHeaders, FailOpen, ProviderRetries, ProviderRetryBackoffMs, PassthroughProtocols, Upstreams`.
  - Defaults (from `config.DefaultCtxConfig()`): Bind 127.0.0.1, Port 8787, Mode
    "cache", CompressEndpoint "/v1/compress", NonLoopbackStatus 404, SavingsHeaders
    true, FailOpen true, ProviderRetries 3, ProviderRetryBackoffMs 200.
  - `CTX_PROXY_MODE` env is checked AFTER config (per the Proxy config comment);
    a non-empty, valid value overrides `cfg.Proxy.Mode`.
- Store: `mnemonic/internal/ctx/store` —
  - `cs := store.New(shared *store.Store)` (shared handle, ADR-0033).
  - Per-turn: `cs.PutProxySavings(...)` + `cs.PutProxySessions(...)` (typed upsert by PK).
  - Fail-open: a missing table is a no-op write / empty read (ADR-0016) — do not
    hard-fail telemetry.

## Wire protocol
The proxy is agent-agnostic (plan 16.1). It must detect the upstream from the
request and forward with the correct key:
- **Anthropic** — `POST /v1/messages`, header `x-api-key`, body `{model, max_tokens,
  messages:[{role,content}]}`, response carries `usage:{input_tokens,output_tokens}`.
- **OpenAI** — `POST /v1/chat/completions`, header `Authorization: Bearer <key>`,
  body `{model, messages:[...]}`, response carries `usage:{prompt_tokens,completion_tokens}`.
Upstream base URL + API key come from the proxy's own config/env (e.g.
`ANTHROPIC_BASE_URL`/`ANTHROPIC_API_KEY`, `OPENAI_BASE_URL`/`OPENAI_API_KEY`) — the
client's `Authorization` header is replaced with the configured upstream key before
forwarding. Token count for the proxy's OWN savings math: for the compression
delta use a deterministic estimate (see savings below); for the `usage` echoed
back to the client, pass through the upstream response `usage` unchanged.

## Savings headers (16.5) — add to EVERY proxied response
- `X-CTX-Savings` — `tokens_before - tokens_after` (the proxy's compression delta).
- `X-CTX-Original-Tokens` — `tokens_before`.
- `X-CTX-Compressed-Tokens` — `tokens_after`.
- `X-CTX-Mode` — the mode used (token|cache|passthrough).
- `X-CTX-Cache-Hit` — `true`/`false` (cache mode only; passthrough/token → `false`).
The AC fixture asserts these are ACCURATE (see Acceptance). `X-CTX-Savings` must
equal the fixture's expected `tokens_before - tokens_after` exactly.

## The three modes (16.2)
- **passthrough** — no transform; forward the body verbatim. `tokens_after ==
  tokens_before`, `X-CTX-Savings: 0`.
- **token** — maximize compression: the transform may rewrite prior history. (T04:
  the transform is a hook; a simple deterministic reducer is fine so the delta is
  measurable and non-zero on the fixture.)
- **cache** (default) — freeze the stable prefix, compress only the mutable tail.
  T04 implements it as: split the messages into a frozen head + mutable tail; apply
  the transform to the tail only. `X-CTX-Cache-Hit` reflects whether a cached
  prefix was detected (T04: a simple stable-prefix detector; T06/T11 refine).
The transform is the SAME code path in all modes; the mode selects how much is
transformed. Keep the transform a pure function `transform(mode, messages)
(messages, error)` so a later ticket can swap in real compression without touching
the transport.

## Request lifecycle (16.4)
1. Session lookup/create — session id from a header (define a header, e.g.
   `X-CTX-Session`; if absent, derive a stable id from the client).
2. Mode determination — config → `CTX_PROXY_MODE` env → (T06: per-request).
3. Pipeline — token count → transform select (by mode) → summary compression.
4. Forward upstream with the correct API key.
5. Capture response; compute `tokens_before - tokens_after`.
6. Emit telemetry (ctx_proxy_savings row + ctx_proxy_sessions counters); return the
   response with the savings headers.

## Fail-open (16.7)
- A transform/compression failure → degrade to passthrough (forward the ORIGINAL
  body verbatim); the request STILL completes (200 with upstream body). Never 500
  the client because compression failed.
- Provider (upstream) timeout/error → retry up to `cfg.Proxy.ProviderRetries` with
  exponential backoff from `ProviderRetryBackoffMs`. After the final retry, return
  the provider's error/status VERBATIM (pass the upstream status + body through).

## Loopback-only (16.6)
- Bind 127.0.0.1 by default. If `Bind` is non-loopback, log a warning (the proxy
  still starts — opt-in is the non-loopback address itself). (The 404-non-loopback
  behavior for /v1/compress is T06.)

## Serve entry (CLI)
Extend the EXISTING `cmd/mnemonic/ctx.go` `ctx` subcommand group (it currently has
search/prime/stats/purge/index). Add:
- `mnemonic ctx proxy [--mode token|cache|passthrough] [--bind ...] [--port ...]`
  → loads config (`config.Load`), applies `--mode` override + `CTX_PROXY_MODE`,
  builds the proxy with a shared `*store.Store` (via `openLoop`/`newMnemonicService`
  like the other ctx subcommands), and starts serving on the loopback bind until
  SIGINT/SIGTERM. Keep it minimal — the proxy's `http.Server` + graceful shutdown.
- A testable constructor: the package exposes `proxy.New(cfg, store, logger)
  *Proxy` and `(*Proxy).Handler() http.Handler` so tests can drive it via
  `httptest` without a real port. The CLI wiring is thin.

## Files (own these only)
- `mnemonic/internal/ctx/proxy/{proxy,modes,savings,retry}.go`
- `mnemonic/internal/ctx/proxy/*_test.go`
- `mnemonic/cmd/mnemonic/ctx_proxy.go` (serve entry) — and a one-line `case "proxy":`
  dispatch in `cmd/mnemonic/ctx.go` (minimal, additive).
Do NOT edit the store, config, or non-ctx files.

## Acceptance Criteria
1. `go build ./...` green.
2. `go test ./internal/ctx/proxy/...` green, proving:
   a. An **Anthropic-compat** request through the proxy (real `httptest` upstream)
      completes with accurate `X-CTX-Savings` + the 5 savings headers.
   b. An **OpenAI-compat** request through the proxy completes with accurate
      `X-CTX-Savings` + headers.
   c. A **fault mid-compression** (transform returns an error) still completes in
      passthrough — client gets the upstream body, `X-CTX-Mode` reflects the fallback,
      no 500.
   d. **Upstream timeout** retries with the configured backoff, then returns the
      provider error verbatim (assert retry count == `ProviderRetries`, then the
      upstream status/body passes through).
3. Savings math: the test seeds a fixture where `tokens_before` and `tokens_after`
   are known, and asserts `X-CTX-Savings == tokens_before - tokens_after` exactly.

## Definition of Done
- Tests pass for touched packages; `gofmt`/`go vet` clean; no new warnings.
- No new 3rd-party dep (stdlib `net/http` + `net/http/httputil` only).
- The transform + retry + fail-open are pure/isolated so T05/T06 plug in cleanly.

## Implementation Plan
1. `modes.go` — `Mode` type + parse/validate (token|cache|passthrough), env
   override read, the stable-prefix split for cache mode.
2. `savings.go` — the pure `transform(mode, messages)` + a deterministic token
   estimate (count of non-whitespace words, or bytes/4 — document it; it must be
   STABLE so the fixture delta is exact) + the 5 savings-header writers.
3. `retry.go` — provider call with exponential backoff (start `ProviderRetryBackoffMs`,
   ×2 each retry, cap at `ProviderRetries`), verbatim error passthrough on final
   failure; injectable clock/sleeper for the test.
4. `proxy.go` — `Proxy` struct, `New(cfg, store, logger)`, `Handler()` wiring the
   request lifecycle: session id → mode → transform (fail-open to passthrough) →
   forward via `httputil.ReverseProxy` with upstream key injection → capture →
   telemetry (ctx store) → savings headers.
5. `*_test.go` — the 4 AC fixtures (anthropic, openai, mid-compression fault,
   upstream timeout) using `httptest.NewServer` as the upstream; assert headers +
   savings + retry count.
6. `cmd/mnemonic/ctx_proxy.go` + the `ctx.go` dispatch line.

## Gates
- G1: `cd mnemonic && go build ./...` → exit 0
- G2: `cd mnemonic && go test ./internal/ctx/proxy/...` → exit 0
- G3: `cd mnemonic && go test ./internal/ctx/store/... ./internal/ctx/config/...` →
  exit 0 (regression: T02/T03 packages still green after the proxy consumes them)
- G4: `cd mnemonic && go vet ./internal/ctx/proxy/... ./cmd/...` → exit 0
- Fails-when: `go test ./internal/ctx/proxy/...` exit != 0 OR `X-CTX-Savings`
  inaccurate on the fixture OR a mid-compression fault 500s the client.

## ADR Constraints (do not re-litigate)
- ADR-0016: fail-open — compression fault → passthrough; provider fault → verbatim
  after retries; a missing ctx_ table → no-op telemetry.
- ADR-0033: shared store handle (no 2nd DB) for telemetry.
- ADR-0013: commit carries a `[skillgrid-context]` block naming TASK-092.

## Work Unit Evidence (fill in report)
| Unit | Focused test | Runtime harness | Rollback boundary |
|------|--------------|-----------------|-------------------|
| T04 proxy core | `go test ./internal/ctx/proxy/...` | real `httptest` upstream + proxy via `httptest` Handler (loopback, ephemeral) | `git revert <sha>` (new internal/ctx/proxy package + cmd/mnemonic/ctx_proxy.go + one dispatch line; two-way) |
