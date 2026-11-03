# T14 Brief — CLI wiring + local-model judgment layer + acceptance sweep (TASK-102)

Final Wave-6 / PR5 ticket. It closes the loop on CTX: wires all ctx subcommands into the
CLI, adds the local-model judgment layer (ADR-0034, the `prefilter → judge → compress`
stage of the proxy pipeline §12.13), prewires the T13 human-gate into the live proxy, and
maps every acceptance scenario to a passing test.

## Context you MUST read first
- Plan: `.skillgrid/artifacts/context-orchestrator-plan.md` §12.13 (proxy pipeline order),
  §18 (traffic/judgment principle), the §17 endpoints. ADRs 0012 (shared store), 0016
  (fail-open), 0033 (ctx = mnemonic subcommand, no own DB handle), 0034 (local Ollama
  judgment, truncate-only, fail-open), 0035/0036 (config.d).
- Acceptance: `.skillgrid/specs/2026-10-08-context-orchestrator/acceptance.feature`
  (23 scenarios).
- Existing proxy: `internal/ctx/proxy/{proxy,savings,compress,memcontext,modes}.go`.
  The transform lives in `savings.go` — `transform(mode, msgs)` → `summarize(m)` is a
  deterministic placeholder (`[ctx-summarized <role>]`). `applyTransform` decodes the body,
  transforms `messages`, re-encodes the full object, fails open on any fault.
- Existing gate (T13): `internal/ctx/traffic/gate.go` (`Apply`/`Revoke`/`ActiveMemories`),
  `Proxy.SetGate(*traffic.Gate)` (proxy.go:52), and `activeMemoryContext` (memcontext.go).
  **`SetGate` is currently called ONLY in tests** — production wiring is this ticket's job.
- Config: `internal/ctx/config/config.go` — `Judgment{Model, Endpoint, Threshold}`
  (defaults `tev1:0.8b`, `http://127.0.0.1:11434/v1/systemone`, `0.22`). `Traffic{...}`.
- CLI: `cmd/mnemonic/ctx.go` — `runCtx` dispatches `search/prime/stats/purge/index/proxy/
  wrap/learn/help`. `ctx_learn.go` exists (T13). Unknown subcommand → usage + exit 2.
- Store: `internal/ctx/store/` — `ctxstore.New(h.Store())` (ADR-0033). No
  `ctx_judgment_cache` table exists yet (it's a T14 addition).

## Scope (5 work items)

### 1. CLI wiring + clean --help (`cmd/mnemonic/ctx.go`)
- Ensure `ctx --help` (and no-arg) lists EVERY subcommand with a one-line description:
  `search, prime, stats, purge, index, proxy, wrap, learn`. (Most already present — verify
  the list is complete and accurate; the `learn` line must mention `apply`/`revoke`.)
- Unknown subcommand → clear error to stderr + exit 2 (already the case; keep it).
- Verify `ctx proxy`, `ctx wrap`, `ctx learn apply|revoke`, `ctx stats` all dispatch
  correctly. No new subcommands are needed — `stats` is the observability read path
  (the four HTTP endpoints are T09/T10; `ctx stats` is the CLM stats subslice).
- Add a focused test (e.g. `ctx_help_test.go`) asserting the help output names all the
  required subcommands and that an unknown subcommand exits non-zero with a clear message.

### 2. Judgment layer (ADR-0034) — `internal/ctx/judgment/` (new package)
Build a small, **stdlib-only** judgment engine that makes per-unit truncate decisions:
- **Deterministic pre-filter** (no model): a pure function over a context unit that
  classifies it. At minimum implement the plan's facts-first / structural-dead /
  label-remainder logic:
  - `pin`  — keep as-is (facts-first: a short, self-contained, high-signal unit).
  - `dead` — structural dead weight (e.g. empty, pure whitespace, or a pure
              boilerplate/label unit).
  - `remainder` — needs a model decision.
  The pre-filter must be DETERMINISTIC and run even when the model is down.
- **Model judge** (Ollama, truncate-only): for `remainder` units, call the local Ollama
  endpoint (`cfg.Judgment.Endpoint` + model `cfg.Judgment.Model`) using stdlib `net/http`.
  The model returns a keep-score in [0,1]; `score < cfg.Judgment.Threshold` → TRUNCATE,
  else KEEP. **Truncate-only**: a truncated unit is shortened (e.g. capped to a budget),
  NEVER deleted and NEVER rewritten (no stored message is rewritten).
- **Fingerprint cache** (`ctx_judgment_cache`): add migration
  `069_ctx_judgment_cache.sql`:
  ```sql
  CREATE TABLE IF NOT EXISTS ctx_judgment_cache (
      fingerprint  TEXT PRIMARY KEY,   -- sha256 over the unit (role + content)
      decision     TEXT,               -- 'keep' | 'trunc'
      score        REAL,
      judged_at    INTEGER
  );
  ```
  Add store accessors (`GetJudgmentCache(fp)`, `PutJudgmentCache(...)`, fail-open, ADR-0016)
  in `internal/ctx/store/`. On a cache hit, serve the decision and make NO model call.
- **Fail-open (the critical AC, ADR-0016 + 0034)**: if the Ollama endpoint is down (conn
  refused, non-2xx, timeout), do NOT error and do NOT drop the unit — the deterministic
  pre-filter result stands and the unit is KEPT (or truncated by the deterministic budget),
  so nothing blocks the request and no unit is dropped. Wrap the HTTP call with a short
  timeout (e.g. 2s) so a hanging endpoint doesn't stall the request.
- Public API (your design, but testable):
  ```go
  type Judgment struct { /* endpoint, model, threshold, client *http.Client, cache */ }
  func New(cfg config.Judgment, cache CacheStore, hc *http.Client) *Judgment
  func (j *Judgment) Judge(ctx context.Context, unit Unit) (Decision, error)
  type Unit struct { Role, Content string }
  type Decision struct { Action Action; KeepScore float64; FromCache bool }
  type Action int // ActionKeep, ActionTruncate
  ```
  Keep the Ollama HTTP client injectable so tests can point it at `httptest.Server`.

### 3. Wire judgment into the proxy transform (fail-open, minimal)
- Inject the judgment engine into the proxy (e.g. `Proxy.SetJudgment(*judgment.Judgment)`;
  nil = off, mirroring `SetGate`). Default OFF in tests unless a test installs it.
- In `savings.go`, change `transform` to route through `prefilter → judge → compress`
  (plan §12.13) WITHOUT changing `transform`'s public signature if you can avoid a large
  refactor. The cleanest minimal approach: when a judgment engine is installed, before
  `summarize`, run the deterministic pre-filter on each unit; for units the pre-filter
  flags `dead`, keep the existing `summarize` (truncation) — for `pin`, keep as-is; for
  `remainder`, consult the judge (truncate-only). When NO engine is installed, the
  existing deterministic placeholder behavior is unchanged (preserves T04–T06 tests).
- **FAIL-OPEN**: any judgment fault (model down, cache error, unexpected shape) leaves the
  unit in its pre-judgment state — NEVER drop it, NEVER 500 the request. The existing
  `applyTransform` fail-open-to-passthrough must still hold.
- Keep it isolated: the judgment call is a single helper in savings.go; do not re-architect
  the transform. Update/keep `summarize` as the truncate implementation.

### 4. Prewire the T13 human-gate into the live proxy (the reviewer's gap)
- In `cmd/mnemonic/ctx_proxy.go` (where the proxy is built for `ctx proxy`), construct
  `traffic.NewGate(ctxstore.New(h.Store()))` and call `proxy.SetGate(gate)` AFTER
  construction. Fail-open: if the store is unavailable, skip `SetGate` (proxy runs with no
  gate — applied memories simply don't inject; never fail startup).
- Do the same in `internal/ctx/wrap/wrap.go` if the wrapped proxy should carry applied
  memories into the agent's context (it builds the same proxy). If wrap currently passes a
  nil store, wire the gate only when a store is available.
- Add an END-TO-END test (the reviewer specifically asked for this): a proxy WITH a gate,
  a `ctx_traffic_memories` row applied via `Gate.Apply`, a request through the proxy →
  assert the response body contains the applied memory's `memory_text`. And a negative:
  with no applied memory, the body does not.

### 5. Acceptance sweep (all 23 scenarios covered)
- Create `.skillgrid/sdd/2026-10-08-context-orchestrator/acceptance-sweep.md` mapping EACH
  of the 23 scenarios in acceptance.feature to (a) a specific passing test
  (`pkg::TestName`), or (b) a documented manual check with exact steps. NONE may be left
  uncovered or marked "planned". For scenarios already covered by T01–T13 tests, cite the
  existing test. For the judgment scenarios (judgment-truncate-only,
  judgment-fingerprint-cached, judgment-fail-open), cite your new tests. For
  cli-help, cite your new help test.
- Run the full suite and confirm green.

## Constraints
- ADR-0016 fail-open everywhere (model down, cache missing, store missing → degrade, never
  block, never drop a unit, never 500).
- ADR-0033: reuse the shared handle (`ctxstore.New(h.Store())`), no second Open.
- **No new 3rd-party dependency.** AC#5 is explicit: stdlib `net/http` + the existing
  `httputil.ReverseProxy` only. Ollama is called over plain HTTP. Do NOT touch go.mod
  (the pre-existing bubbletea go.mod/go.sum edits are NOT yours — leave them unstaged).
- Keep `single_open_test.go` green. Go 1.22+. gofmt clean.
- Migrations: add `069_ctx_judgment_cache.sql` only (068 is T13's, already applied).

## Acceptance gates (ALL must pass)
- `go build ./...`
- `go test ./... -count=1`  (REPO ROOT — the full suite, per AC#1)
- `go vet ./...`
- `gofmt -l` on every file you touched (empty)
- `go list -m all | grep -v github.com/devopstales` — confirm NO new non-stdlib module
  beyond what was already there (i.e. go.mod dependency set unchanged).

Required new tests:
1. **judgment-truncate-only**: a unit above threshold → `ActionTruncate`; unit is
   truncated (shorter) but NOT deleted, and no stored message content is rewritten
   (role + non-truncated units byte-identical).
2. **judgment-fingerprint-cached**: same unit judged twice → second is `FromCache=true`
   and NO model call (assert via a counting httptest server).
3. **judgment-fail-open**: Ollama endpoint down (closed port) → `Judge` returns a
   keep (or deterministic truncate), NOT an error, and the unit is present in the
   transformed output (not dropped).
4. **pre-filter determinism**: same input → same classification, no model, no randomness.
5. **cli-help**: `ctx --help` names proxy/wrap/learn/stats/search; unknown subcommand →
   exit 2 + clear stderr.
6. **gate e2e**: proxy + gate, applied memory appears in response body; without applied,
   it does not.
7. **acceptance-sweep.md**: all 23 scenarios mapped.

## Report
Write the FULL report to `.skillgrid/sdd/2026-10-08-context-orchestrator/task-14-report.md`:
the pipeline wiring (where judgment inserts), the fail-open paths, the gate prewire
(ctx_proxy.go + wrap.go), the judgment cache + fingerprint, the acceptance sweep summary
(23/23), and gate outputs (pass/fail + counts). Generate
`.skillgrid/sdd/2026-10-08-context-orchestrator/task-14.diff` (git diff of your commits).

## Commit
Conventional subject, e.g. `feat(ctx): wire CLI + local judgment layer + gate; acceptance sweep`.
EXACTLY this body block:
```
[skillgrid-context]
change: 2026-10-08-context-orchestrator
ticket: TASK-102
wave: 6
pr: PR5
```
You may make more than one commit if the work is large, but EVERY commit must carry the
exact body block above.
