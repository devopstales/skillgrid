## 02-process-flows

### Goal

Precomputed process layer — trace execution flows from 010's entry points (routes/handlers/CLI mains) through 005 call edges into named `processes`/`process_steps` with a cross-community flag + LLM labels cached by content-hash, exposed via `code_processes` / `code_process` and surfaced in 005's `code_explain_symbol`.

### Out of scope / Non-Goals

- Community detection (step 01)
- Knowledge nodes (step 03)
- PDG/taint (011)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-02` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 01-community-detection (communities + cross-community flag), 010 (entry points)

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/process/trace.go`
- Create: `skillgrid-cli/internal/mnemonic/process/labels.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_code_process.go`
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go` (process facade + process participation in `code_explain_symbol`)
- Test: `skillgrid-cli/internal/mnemonic/process/...`, `skillgrid-cli/internal/mnemonic/mcp/...`, `skillgrid-cli/internal/mnemonic/service/...`

**Interfaces:**
- Consumes: 010's entry points (route/handler/CLI-main symbols), 005 call edges, 01's communities + `code_explain_symbol`
- Produces: `processes` / `process_steps` tables (cross-community flag + content-hash key); deterministic entry-point → call-chain trace (depth-capped, graph-stops); LLM labels cached by content-hash; `code_processes`, `code_process <name>` MCP tools; `code_explain_symbol` now surfaces process participation (step N/M)

### Tasks

- [ ] 02.1 `[RED]` Mnemonic tool surface — `code_explain_symbol` surfaces process participation + 005 tools stable (Scenario: Symbol explanation surfaces process participation and existing search tools stay stable) — threat: Mnemonic tool surface
  - [ ] 02.1.a Write failing test — after process pass, `code_explain_symbol <sym>` (005) includes which processes the symbol participates in (step N/M); 005 tool names + required params unchanged; process tools registered with distinct names
  - [ ] 02.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -run ExplainSymbolProcess -count=1` — Expected: FAIL
  - [ ] 02.1.c Minimal implementation — wire process participation into `code_explain_symbol` + register `code_processes` / `code_process`
  - [ ] 02.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -run ExplainSymbolProcess -count=1` — Expected: PASS
  - [ ] 02.1.e Commit — `feat(mnemonic): surface process participation in code_explain_symbol`
- [ ] 02.2 `[RED]` Entry-point → call-chain trace builds precomputed flows (Scenario: Process list returns precomputed flows from entry points)
  - [ ] 02.2.a Write failing test — seeded from 010 entry points (routes/handlers/CLI mains), trace through 005 call edges into `processes` + `process_steps`; `code_processes` returns complete flows in one call (no per-query traversal); each process has named steps + a cross-community flag + an LLM label
  - [ ] 02.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/process/... ./skillgrid-cli/internal/mnemonic/mcp/... -run ProcessTrace -count=1` — Expected: FAIL
  - [ ] 02.2.c Minimal implementation — `process/trace.go` (entry-point seeding, depth-capped BFS over call edges, cross-community flag from 01, persist `processes`/`process_steps`) + `tools_code_process.go` `code_processes` + service facade
  - [ ] 02.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/process/... ./skillgrid-cli/internal/mnemonic/mcp/... -run ProcessTrace -count=1` — Expected: PASS
  - [ ] 02.2.e Commit — `feat(mnemonic): precomputed process flow tracing`
- [ ] 02.3 `[RED]` LLM labels cached by content-hash (Scenario: Process labels are cached by content-hash and re-labeled only on change)
  - [ ] 02.3.a Write failing test — a flow is LLM-labeled once and the label is cached keyed to the process content-hash; an unchanged re-index does NOT re-call the LLM (label reused); a changed flow (new content-hash) re-labels
  - [ ] 02.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/process/... -run ProcessLabels -count=1` — Expected: FAIL
  - [ ] 02.3.c Minimal implementation — `process/labels.go` (content-hash key, LLM call, cache store)
  - [ ] 02.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/process/... -run ProcessLabels -count=1` — Expected: PASS
  - [ ] 02.3.e Commit — `feat(mnemonic): llm process labels cached by content-hash`
- [ ] 02.4 `[RED]` Dispatch-boundary truncation with "stops at" note + per-hop confidence (Scenario: Trace stops at a dispatch boundary with a note)
  - [ ] 02.4.a Write failing test — a trace that hits a dispatch boundary (interface→impl, message bus, callback) is truncated with a "stops at <symbol> (<reason>)" note (reusing 005 graph-stops), not silently cut; `code_process <name>` shows each hop's Confidence Label + the stop note
  - [ ] 02.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/process/... -run DispatchStop -count=1` — Expected: FAIL
  - [ ] 02.4.c Minimal implementation — depth/step cap + dispatch-boundary detection + per-hop confidence in `process/trace.go`
  - [ ] 02.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/process/... -run DispatchStop -count=1` — Expected: PASS
  - [ ] 02.4.e Commit — `feat(mnemonic): dispatch-boundary truncation in process trace`
- [ ] 02.5 `[AFK]` Process detail returns the full trace with confidence (Scenario: Process detail returns the full step-by-step trace) — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/process/... -run ProcessDetail -count=1` — Expected: PASS
- [ ] 02.6 `[AFK]` LLM down → flow cached unlabeled, never fabricated (Scenario: LLM down caches the flow unlabeled) — `Run: go test ./skillgrid-cli/internal/mnemonic/process/... -run LLMDownUnlabeled -count=1` — Expected: PASS
- [ ] 02.7 `[AFK]` Entry point with no traceable chain → single-step or skipped (Scenario: Untraceable entry point yields single-step or is skipped) — `Run: go test ./skillgrid-cli/internal/mnemonic/process/... -run UntraceableEntry -count=1` — Expected: PASS
- [ ] 02.8 `[AFK]` Cross-community process is flagged (Scenario: Cross-community process is flagged) — `Run: go test ./skillgrid-cli/internal/mnemonic/process/... -run CrossCommunityFlag -count=1` — Expected: PASS
- [ ] 02.9 `[AFK]` Process tools reject bad args clearly (Scenario: Process tools reject bad args clearly) — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run ProcessArgs -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/process/... ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -count=1` | PASS | | |
| Acceptance `@step-02` / `@p0` | BDD / mapped unit scenarios (incl. `ProcessTrace`, `ExplainSymbolProcess`, `ProcessLabels`) | PASS | | |
| Runtime harness | `code_processes` / `code_process` on a fixture with 010 entry points | PASS | | |
| Rollback boundary | Drop `process/` + process tools + `code_explain_symbol` participation | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): precomputed process flows with cached llm labels`

---

