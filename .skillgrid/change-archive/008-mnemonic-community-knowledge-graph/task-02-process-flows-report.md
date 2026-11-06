# 008 Step 02 — Precomputed Process Flows

Status: **DONE** — all `@step-02` scenarios implemented, full relevant suite + 005/008/010 baseline green.

## What was implemented per sub-task

### 02.1 `code_explain_symbol` surfaces process participation + tools stable
- `service/service.go`: added a process facade (`CodeProcessTrace`, `CodeProcesses`, `CodeProcess`, `CodeProcessParticipations`) + type aliases. Added an **additive** `Processes []process.Participation` field to `OrientResult` (the symbol-explain answer) — 005's existing fields unchanged.
- `mcp/tools_code_orient.go`: `handleCodeOrient` (005's symbol-explain tool) now resolves the symbol and, when it is found, attaches `process.Participations(db, id)` so the response gains a `processes: [{process, step, total}]` field (step N/M).
- `mcp/tools_code_process.go`: new `code_processes` (list precomputed flows, one call, no per-query traversal) + `code_process <name>` (full step-by-step trace with per-hop confidence + stop note). Both distinct `code_*` names.
- Registered in `server.go` (`registerProcessTools`) + added to `menuCodeTools` (unlisted by default, re-enabled via `CodeToolsEnvVar`, like the 008 community tools).
- `mcp/server_test.go`: `TestAllToolsRegistered` `want` list grew 65 → 67 (the two new tools) — the tool-surface baseline grows additively; every 005 tool keeps name + required params (`TestToolSurfaceBaseline`, `TestCodeSearchSchemaStable` unchanged and passing).

### 02.2 Entry-point → call-chain trace builds precomputed flows
- `store/migrations/014_process_flows.sql`: additive `processes` / `process_steps` / `process_meta_cache` tables (content-hash key + cross-community flag + label columns). 005 symbols/edges + 012 community tables untouched.
- `process/trace.go`: `Run` consults `process_meta_cache` (content-hash) first (cache hit → `FromCache=true`, no re-trace). `traceOne` is a **depth-capped BFS** over the 005 call/references/imports edges with **stable ordering** (neighbors by symbol id, a symbol visited once). Seeds come from 010's entry points (route/handler/CLI-main symbols); the cross-community flag is computed from **01's `communities` table**. Persists `processes` + `process_steps`. `List`/`Get` serve the flows without per-query traversal.
- `process/hash.go`: FNV-1a content-hash helpers (pure Go, CGo-free).

### 02.3 LLM labels cached by content-hash
- `process/labels.go`: `labelFlows` keys the label cache to each process's `content_hash`. An unchanged re-run is a cache hit (`FromCache=true`) and the LLM is **not** re-called; the stored label is reused. A changed flow (new content hash) re-labels (a new LLM call). Per-pass call budget (`RunOptions.LLMLimit`). The label is a pure function of the flow content (deterministic `flowSummary`), so caching is sound.

### 02.4 Dispatch-boundary truncation with "stops at" note + per-hop confidence
- `process/trace.go`: `traceOne` detects a dispatch boundary when a callee's symbol kind is `interface` / `bus` / `callback` (or an unresolved name-only callee). The boundary symbol **is recorded as the last step** (reached) and a `StopNote{Symbol, Reason, Line}` is emitted — "stops at `<symbol>` (`<reason>`)", reusing the 005 graph-stops idea — so the trace is not silently cut. Every hop carries its edge `confidence` label.

## RED → GREEN evidence

The whole `process` package is new, so the RED state is a **build failure** (the tests reference the not-yet-written API). Captured by removing `process/trace.go` (the source of `Process`/`Entry`/`Run`/`RunOptions`/`StopNote`/`Participation`):

```
$ go test ./internal/mnemonic/process/... -run 'ProcessTrace|ProcessLabels|DispatchStop' -count=1
# github.com/.../mnemonic/process [github.com/.../mnemonic/process.test]
internal/mnemonic/process/labels.go:17:58: undefined: Process
internal/mnemonic/process/labels.go:17:71: undefined: LLM
internal/mnemonic/process/labels_test.go:38:15: undefined: Entry
internal/mnemonic/process/labels_test.go:43:16: undefined: Run
internal/mnemonic/process/labels_test.go:43:59: undefined: RunOptions
FAIL  (build failed)
```
Expected FAIL: the tests require the process API that 02.2/02.3/02.4 add. Restoring the implementation → GREEN (below).

| Sub-task | Test | Result |
|----------|------|--------|
| 02.1 | `TestExplainSymbolProcess` (mcp) | PASS |
| 02.2 | `TestProcessTrace`, `TestProcessDeterministic` (process) | PASS |
| 02.3 | `TestProcessLabels` (process) | PASS |
| 02.4 | `TestDispatchStop` (process) | PASS |
| 02.5 AFK | `TestProcessDetail` (mcp) | PASS |
| 02.6 AFK | `TestLLMDownUnlabeled` (process) | PASS |
| 02.7 AFK | `TestUntraceableEntry` (process) | PASS |
| 02.8 AFK | `TestCrossCommunityFlag` (+ negative control `TestCrossCommunityNotFlaggedWhenSingleCommunity`) | PASS |
| 02.9 AFK | `TestProcessArgs` (mcp) | PASS |

Per-test (process package):
```
--- PASS: TestProcessLabels
--- PASS: TestLLMDownUnlabeled
--- PASS: TestDispatchStop
--- PASS: TestUntraceableEntry
--- PASS: TestCrossCommunityFlag
--- PASS: TestCrossCommunityNotFlaggedWhenSingleCommunity
--- PASS: TestProcessTrace
--- PASS: TestProcessDeterministic
ok  .../process
```

## How the LLM is stubbed/injected

`process.LLM` is a **pure-Go interface** (`Label(ctx, summary) (string, error)`); there is no CGo LLM client in the package (CGo-free constraint held). `Run` takes `llm process.LLM` as a parameter, so tests inject:
- `stubLLM{label, err}` — value-receiver stub returning a fixed label; `err=errLLMDown` exercises the LLM-down path (02.6).
- `callCounter{label, calls}` — pointer-receiver stub that records call counts, proving an unchanged re-index makes **0** LLM calls while a changed flow makes **1** (02.3).

The content-hash cache makes label caching testable without a live LLM: `TestProcessLabels` asserts `c1.calls==1` (first label), `c2.calls==0` (cache hit, label reused), `c3.calls==1` (changed flow re-labels).

## Full relevant suite

```
$ go test ./internal/mnemonic/process/... ./internal/mnemonic/mcp/... ./internal/mnemonic/service/... -count=1
ok  .../process    0.749s
ok  .../mcp       25.962s
ok  .../service   17.615s
```
(plus `ok .../store 3.905s` for the 014 schema test `TestStoreOpenAppliesProcessSchema`.)

## 005/008/010 baseline still passes

```
$ go test ./internal/mnemonic/community/... ./internal/mnemonic/route/... ./internal/mnemonic/graph/... -count=1
ok  .../community   1.849s
ok  .../route        0.512s
ok  .../graph        1.401s
```
`TestToolSurfaceBaseline` + `TestCodeSearchSchemaStable` + `TestAllToolsRegistered` (now 67) all pass — 005 tools keep name + required params; the surface grew additively 65 → 67.

## Step-03 follow-up (query-time gap)

**`CodeProcessTrace` has 0 non-test callers and there is no non-test `LLM`
implementation**, so `code_processes` returns empty in production until step 03
wires `Indexer.Run` to call the process pass (and provides a real `LLM` impl).
This is a **deferral, not a step-02 defect** — the Global Constraint says
processes are "precomputed at index time", and the indexer hook is explicitly
this change's step-03 work.

Step-03 tasks **do** cover the process-pass indexer hook — it is not lost:
- `## 03-knowledge-graph-nodes` Files: `codeindex/indexer.go` — "hook
  community + **process** + knowledge passes after 005 extraction, same tx"
  (tasks.md:396).
- Interfaces Produces: "indexer hook running community + **process** +
  knowledge passes in the same incremental tx" (tasks.md:403).
- **Task 03.8** `[AFK]` "Indexer hook runs all three passes in the same
  transaction" (`IndexerHook`) is the exact task that closes this gap — it is
  the task to implement the `Indexer.Run` hook that calls `CodeProcessTrace`
  with a real `LLM` impl.

Until 03.8 lands, `code_processes`/`code_process` are CWD-scoped (the query
time resolves entry points and runs the pass on demand, as the step-02 tests
exercise); the query-time path is the only one available today. A code comment
on `CodeProcessTrace` (service.go) now points at this deferral and at task 03.8.

## Commits
- `ed140a3` feat(mnemonic): process flow schema, tracing, labels and dispatch-stop
- `25e56de` feat(mnemonic): process tools and code_explain process participation

## Concerns / notes
- **Entry-point seeding is query-time, not indexer-time.** 010's entry points are `route` symbols (created during indexing), so the process pass seeds from them at call time (`handleCodeProcesses`/the service facade resolves route→handler and runs `Run`). The acceptance "Indexer hook runs community, process, and knowledge in one transaction" scenario is **step 03** scope (it names knowledge), so I did not hook the indexer here — the trace is advisory and lazy. If a step wants the flows precomputed at index time, the service facade `CodeProcessTrace` is ready to call from `Indexer.Run`.
- **`code_explain_symbol` == 005's `code_orient`.** The repo's symbol-explain tool is `code_orient` (not literally named `code_explain_symbol`); participation was added there, which is the 005 symbol-explain surface.
- **Dispatch kinds** are a small fixed set (`interface`/`bus`/`callback`) plus unresolved name-only callees. If the 005 extractors emit other dispatch markers, the set in `dispatchKinds` (process/trace.go) is the single place to extend.
- The LLM label is non-deterministic by nature; it is **cached by content-hash** so an unchanged re-index never re-calls it, and when the LLM is down the flow is cached **unlabeled** (empty label, `label_status="unlabeled"`) — never fabricated.
