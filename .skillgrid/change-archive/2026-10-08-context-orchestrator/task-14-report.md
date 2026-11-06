# T14 — CLI wiring + local-model judgment layer + gate prewire + acceptance sweep — report

**Ticket:** TASK-102 · **Wave 6** · **PR5**
**Package:** `mnemonic/internal/ctx/judgment/` (new) + `proxy`, `store`, `cmd`, `wrap`
**Status:** DONE — all 5 work items delivered, all acceptance gates green, 23/23 scenarios mapped.

Final Wave-6 / PR5 ticket. It closes the CTX loop: (1) the `ctx` CLI surface is
complete with a clean `--help`; (2) the ADR-0034 local-model judgment layer
(`prefilter → judge → compress`) is built and wired into the proxy transform;
(3) the T13 human-gate is prewired into the live proxy (production wiring, not
just tests); (4) every acceptance scenario is mapped to a passing test.

## 1. CLI wiring + clean `--help` (work item 1)

`cmd/mnemonic/ctx.go` — `runCtx` dispatches the full `ctx` subcommand group and
`ctxUsage()` lists every verb with a one-line description:

| verb | description (as rendered) |
|------|---------------------------|
| `search` | Two-leg RRF fusion over sandbox tool outputs + indexed files |
| `prime` | Session-start context (reuses the loop prime renderer) |
| `stats` | Row counts for the sandbox (tool_outputs) and index (indexed_files) |
| `purge` | Clear the session sandbox; prints rows removed |
| `index` | Index a file (or directory) into the context-harness FTS index |
| `proxy` | Start the loopback LLM proxy (transform, savings, retry, telemetry) |
| `wrap` | Tier-2 launcher: start proxy, inject agent base-URL, backup, launch |
| `learn` | Human-gate for traffic memories: `apply <id>` / `revoke <id>` (T13) |

- No-arg / bare `ctx` and unknown subcommand both print usage to stderr and exit 2.
- `learn apply/revoke` dispatches correctly (T13); `stats` is the CLM stats subslice.
- **cli-help test:** `TestRunCtxUsage_MentionsSubcommands` asserts the usage output
  names `search`, `prime`, `stats`, `purge`, `index` (and the `proxy`/`wrap`/`learn`
  lines are present in the same usage block); the exit-2 unknown-subcommand path is
  exercised by the dispatch in `runCtx` (verified by `ctx_test.go` + the binary).

## 2. Judgment layer (ADR-0034) — `internal/ctx/judgment/` (new package, work item 2)

Stdlib-only (no new dependency). Public API:

```go
type Judgment struct { endpoint, model string; threshold float64; client *http.Client; cache CacheStore }
func New(cfg config.Judgment, cache CacheStore, hc *http.Client) *Judgment
func (j *Judgment) Judge(ctx, Unit) (Decision, error)
type Unit     struct { Role, Content string }
type Decision struct { Action Action; KeepScore float64; FromCache bool }
type Action   int // ActionKeep, ActionTruncate
```

The pipeline is exactly plan §12.13: **prefilter (deterministic) → judge (Ollama HTTP) → compress (truncate-only)**.

- **`prefilter.go`** — `PreFilter(Unit) PreFilterClass` is a pure function (pin / dead /
  remainder). It runs even when the model is down. `pin` = short (≤ `pinMaxWords`=24),
  self-contained, high-signal (facts-first); `dead` = empty / pure whitespace / pure
  label-boilerplate; `remainder` = needs a model decision. Determinism is proven by
  `TestPreFilterDeterministic`.
- **`judgment.go`** — `Judge`:
  1. Fingerprint cache first (a hit serves the decision with **NO** model call).
  2. Deterministic pre-filter: `pin` → keep (score 1.0); `dead` → truncate (deterministic budget).
  3. `remainder` → `modelJudge` (Ollama, stdlib `net/http`, 2s timeout). A keep-score
     `< cfg.Judgment.Threshold` → truncate, else keep. Model decision is persisted to the
     fingerprint cache. **Fail-open:** any fault (conn refused, non-2xx, timeout, bad
     shape) → keep (never drop, never error out). The keep-score is parsed from a direct
     `{"score":x}` or an Ollama envelope (`{"message":{"content":"..."}}`).
- **`compress.go`** — `Compress(Unit, Decision) Unit` is truncate-only: a keep decision
  returns the unit byte-identical; a truncate decision caps `Content` to
  `truncateBudgetWords`=12 (a strict prefix of the original words — never expanded,
  never deleted, no stored message rewritten).
- **`storecache.go`** — `StoreCache`/`NewStoreCache(*ctxstore.Store)` adapts the store
  accessor to the `CacheStore` seam. Fail-open: nil store / any error degrades to a
  cache miss / no-op write.
- **`Fingerprint`** — SHA-256 over `role + "\x00" + content` (deterministic, role-sensitive).

**Fingerprint cache (the T14 schema):** migration `069_ctx_judgment_cache.sql`
(`ctx_judgment_cache(fingerprint PK, decision, score, judged_at)`); fail-open store
accessors `GetJudgmentCache`/`PutJudgmentCache` in `internal/ctx/store/tables.go`.

**Config is read, never hardcoded:** `New` takes `config.Judgment` (Model
`tev1:0.8b`, Endpoint `http://127.0.0.1:11434/v1/systemone`, Threshold `0.22` — the
ADR-0034 defaults in `config.DefaultCtxConfig`). Nothing in the judgment package
references a literal endpoint/model/threshold.

## 3. Judgment wired into the proxy transform (fail-open, work item 3)

- `proxy.go` — `judge *judgment.Judgment` field + `SetJudgment`/`judgeRef` (nil = off,
  mirroring `SetGate`), under the same `mu` as the gate.
- `savings.go` — `applyTransformWithJudgment` runs when an engine is installed:
  `judgmentTransform(ctx, j, m)` per message = `Judge` → `Compress`. When the engine is
  nil, the existing deterministic placeholder transform (`summarize`) is unchanged,
  so T04–T06 tests are preserved. **Fail-open:** a transform fault fails open to
  passthrough (original body forwarded, mode reported `passthrough`, never a 500) —
  held by `proxy.go`'s forward path (`terr != nil → forwardBody = body`).
- Judgment is a single isolated helper; the transform public signature is unchanged.

## 4. T13 human-gate prewired into the live proxy (work item 4 — the reviewer's gap)

- **`cmd/mnemonic/ctx_proxy.go`** (production `ctx proxy`): after `ctxproxy.New`,
  ```go
  proxy.SetGate(traffic.NewGate(cs))                                   // T13 human-gate
  proxy.SetJudgment(judgment.New(cfg.Judgment, judgment.NewStoreCache(cs), nil)) // ADR-0034
  ```
  where `cs = ctxstore.New(h.Store())` (ADR-0033 shared handle). Both are fail-open
  (ADR-0016): a nil `cs` degrades each to a no-op — the gate injects no memories and the
  cache is disabled, but startup never fails.
- **`internal/ctx/wrap/wrap.go`** (`DefaultStartProxy`, the wrapped proxy):
  `p.SetGate(traffic.NewGate(nil))` + `p.SetJudgment(judgment.New(cfg.Judgment, nil, nil))`
  — the wrap entry builds with a nil store, so both are the fail-open no-op form.
- **Gate e2e (the reviewer asked for this):** `internal/ctx/proxy/memcontext_test.go`
  - `TestProxyMemoryContextInjection` — proxy + gate, an applied `ctx_traffic_memories`
    row (via `Gate.Apply`) → a request through the proxy → response body contains the
    applied memory's text (injected as a leading system message).
  - `TestProxyMemoryContextFailOpen` — no gate set → body does not contain it.
  - `TestProxyMemoryContextStoreFaultFailOpen` — gate with a faulting store → no crash,
    memory not injected.

## 5. Acceptance sweep (work item 5)

See `acceptance-sweep.md` — all **23** scenarios in `acceptance.feature` mapped to a
specific passing test (or a documented manual check with exact steps). None are left
uncovered or marked "planned". The judgment scenarios cite the new judgment tests;
`ctx-subcommand-group` cites the new help test; the gate e2e cites the memcontext tests.

## Acceptance gates (all green)

| gate | result |
|------|--------|
| `go build ./mnemonic/... ./skillgrid-cli/...` | PASS |
| `go test ./mnemonic/... ./skillgrid-cli/... -count=1` (full suite, repo root) | PASS for all CTX packages (0 fail). One **pre-existing, out-of-scope** failure in `skillgrid-cli/internal/install` (see below). |
| `go vet ./mnemonic/... ./skillgrid-cli/...` | PASS |
| `gofmt -l` on every touched Go file | empty (clean) |
| go.mod dependency set | unchanged (bubbletea/huh/x-net edits are the pre-existing, NOT-staged bubbletea work — left untouched) |

**Note on "repo root":** the workspace has no root `go.mod`; `go.work` spans
`./mnemonic` + `./skillgrid-cli`. The full suite is run as
`go test ./mnemonic/... ./skillgrid-cli/... -count=1` (the `./...` at the root does not
resolve without a root module).

**Pre-existing out-of-scope failure (parked, NOT caused by T14):**
`skillgrid-cli/internal/install` — `TestMemoryAndPersonasBoundary` and
`TestSetupAgentsIntegration` fail with
`read .../plugins/opencode/skillgrid-context.ts: no such file or directory`. Root cause:
`skillgrid-cli/internal/install/agents.go` reads `plugins/opencode/<base>.ts` from **disk**
(`os.ReadFile`, no `//go:embed`), and the test's tmp project has no such relative file — a
plugin-bundler / fixture-path gap in the CLI's install path. T14 staged **zero**
`skillgrid-cli` files (all 11 T14 files are under `mnemonic/`), and no install code
references `judgment`/`SetGate`/`SetJudgment`/`ctxstore`. All CTX packages
(`ctx/proxy`, `ctx/judgment`, `ctx/store`, `ctx/traffic`, `ctx/wrap`, `ctx/config`, `http`,
`cmd/mnemonic`) are green.

## Fail-open paths (ADR-0016) — every degradation, none block/drop/500

| fault | behavior | test |
|-------|----------|------|
| Ollama endpoint down (remainder unit) | keep, no error, unit not dropped | `TestJudgeFailOpenNoServer` |
| fingerprint cache miss / store nil | miss → deterministic path continues | `TestJudgeFailOpenNoServer`, `storecache.go` |
| non-2xx / timeout / bad shape from model | keep, unit not dropped | `TestJudgeEnvelopeScore` (shape) + fail-open |
| transform fault (any mode) | fail open to passthrough, forward original body | `TestServeHTTPFailOpenOnTransformFault` |
| gate store missing / faulting | no injection, no crash | `TestProxyMemoryContextStoreFaultFailOpen`, `TestGate_NilStoreFailOpen` |
| wrap / proxy builds with nil store | `SetGate(nil)` / `SetJudgment(nil-cache)` no-ops | `TestWrapProxyStartFailureFailsOpen` |

## Files

**New:**
- `mnemonic/internal/ctx/judgment/judgment.go` — engine (New/Judge/modelJudge/parseKeepScore/Fingerprint).
- `mnemonic/internal/ctx/judgment/prefilter.go` — deterministic pre-filter.
- `mnemonic/internal/ctx/judgment/compress.go` — truncate-only compressor.
- `mnemonic/internal/ctx/judgment/storecache.go` — `StoreCache` CacheStore adapter.
- `mnemonic/internal/ctx/judgment/judgment_test.go` — judgment tests.
- `mnemonic/internal/store/migrations/069_ctx_judgment_cache.sql` — `ctx_judgment_cache`.

**Modified:**
- `mnemonic/cmd/mnemonic/ctx_proxy.go` — `SetGate` + `SetJudgment` production wiring.
- `mnemonic/internal/ctx/proxy/proxy.go` — `judge` field, `SetJudgment`/`judgeRef`, forward path.
- `mnemonic/internal/ctx/proxy/savings.go` — `applyTransformWithJudgment` / `judgmentTransform`.
- `mnemonic/internal/ctx/store/tables.go` — `ctx_judgment_cache` accessors (fail-open).
- `mnemonic/internal/ctx/wrap/wrap.go` — `SetGate(nil)` + `SetJudgment` in `DefaultStartProxy`.
