## 03-layered-retrieval-budgets

### Goal

Retrieval bootstraps from L2/L3 and falls back to L1/L0 (RRF) for a fact — and every `mem_*` read is budgeted (item + char + timeout) so memory can never overwhelm the context window; `mem_get_observation` stays the only full-content path; CLI parity.

### Out of scope / Non-Goals

- Distillation (step 02); governance (step 01); changing `mem_*` tool names/required params

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-03` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 02-layered-distill

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/memory/budget.go`
- Create: `skillgrid-cli/internal/mnemonic/memory/retrieve.go`
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go`
- Modify: `skillgrid-cli/cmd/skillgrid/main.go`
- Test: `skillgrid-cli/internal/mnemonic/memory/budget_test.go`, `skillgrid-cli/internal/mnemonic/memory/retrieve_test.go`, `skillgrid-cli/internal/mnemonic/service/retrieval_test.go`, `skillgrid-cli/cmd/skillgrid/main_test.go`

**Interfaces:**
- Consumes: 02 layered store (`observation_layers`/`personas` as L2/L3 bootstrap sources) + 01 governance fields; 005 RRF read path
- Produces: layered retrieval (L2/L3-first, L1/L0 RRF fallback) + a uniform budget wrapper (item + char + timeout) on `mem_search`/`mem_context`/`mem_timeline`; `mem_get_observation` as the only full-content path; tunable budget config; CLI parity for `mem_layers`/`mem_governance`/`mem_share`

### Tasks

<!-- [RED] items are the applicable threat-matrix rows (Mnemonic tool surface + Context-window flood), ordered BEFORE the production [AFK] tasks. Each uses the TDD micro-cycle a–e. Scenario names are the referenceable strings from acceptance.feature @step-03. -->

- [ ] 03.1 `[RED]` Threat "Context-window flood" — 20-hit search budgeted (truncated in-list + `mem_get_observation` id present + timeout honored with `truncated: true` partial) (Scenarios: `twenty-hit-search-is-budgeted`, `char-budget-truncates-with-explicit-omitted-count`, `context-timeout-returns-truncated-partial`)
  - [ ] 03.1.a Write failing test
  - [ ] 03.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/... -run Budget -count=1` — Expected: FAIL
  - [ ] 03.1.c Minimal implementation
  - [ ] 03.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/... -run Budget -count=1` — Expected: PASS
  - [ ] 03.1.e Commit — `feat(memory): item+char+timeout budget wrapper on mem_* reads`
- [ ] 03.2 `[RED]` Threat "Mnemonic tool surface" — layered retrieval + `mem_get_observation` only-full-content + 005 tools still stable + bad args rejected (Scenarios: `layered-retrieval-l2-l3-first-with-l1-l0-rrf-fallback`, `mem-get-observation-is-only-full-content-path`, `budgeted-reads-005-stable`)
  - [ ] 03.2.a Write failing test
  - [ ] 03.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/... ./skillgrid-cli/internal/mnemonic/mcp/... -run Retrieve -count=1` — Expected: FAIL
  - [ ] 03.2.c Minimal implementation
  - [ ] 03.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/... ./skillgrid-cli/internal/mnemonic/mcp/... -run Retrieve -count=1` — Expected: PASS
  - [ ] 03.2.e Commit — `feat(memory): layered L2/L3-first retrieval + mem_get_observation full-content path`
- [ ] 03.3 `[AFK]` `mem_context` / `mem_search` return L2/L3 first; specific-fact query falls back to L1/L0 via the existing RRF (Scenario: `layered-retrieval-l2-l3-first-with-l1-l0-rrf-fallback`) — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/retrieve_test.go -run Layered -count=1` — Expected: PASS
- [ ] 03.4 `[AFK]` Every `mem_*` read enforces the item-count cap, char budget (explicit "N chars omitted"), and context timeout (`truncated: true` partial + reason, never hangs) (Scenarios: `twenty-hit-search-is-budgeted`, `char-budget-truncates-with-explicit-omitted-count`, `context-timeout-returns-truncated-partial`) — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/... -run Budget -count=1` — Expected: PASS
- [ ] 03.5 `[AFK]` A 20-hit search returns 20 budgeted snippets (not 20× full JSON); every in-list result carries its `mem_get_observation` id (Scenarios: `twenty-hit-search-is-budgeted`, `every-inlist-result-carries-get-observation-id`) — `Run: go test ./skillgrid-cli/internal/mnemonic/service/... -run BudgetedSearch -count=1` — Expected: PASS
- [ ] 03.6 `[AFK]` `mem_get_observation` remains the only full-content path; the budget is tunable (config); truncation never silent (Scenarios: `mem-get-observation-is-only-full-content-path`, `budget-is-tunable-via-config`) — `Run: go test ./skillgrid-cli/internal/mnemonic/service/... -run FullContent -count=1` — Expected: PASS
- [ ] 03.7 `[AFK]` Route `mem_context`/`mem_search`/`mem_timeline` through the layered + budgeted path; `mem_get_observation` stays the only full-content path (Scenario: `mem-context-search-timeline-budgeted`) — `Run: go test ./skillgrid-cli/internal/mnemonic/service/... -run RouteReads -count=1` — Expected: PASS
- [ ] 03.8 `[AFK]` CLI parity for `mem_layers`/`mem_governance`/`mem_share` (Scenario: `cli-parity-for-layer-governance-share`) — `Run: go test ./skillgrid-cli/cmd/skillgrid/... -count=1` — Expected: PASS
- [ ] 03.9 `[AFK]` 005 `mem_*` names + required params unchanged; bad args rejected clearly (Scenarios: `budgeted-reads-005-stable`, `bad-retrieval-args-rejected`) — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/... ./skillgrid-cli/internal/mnemonic/mcp/... -count=1` | PASS | | |
| Acceptance `@step-03` / `@p0` | `go test ./skillgrid-cli/internal/mnemonic/service/... ./skillgrid-cli/cmd/skillgrid/... -count=1` + BDD `@step-03 @p0` | PASS | | |
| Runtime harness | `go test ./... -count=1` (from `skillgrid-cli` / repo root per module layout) | PASS | | |
| Rollback boundary | drop `memory/budget` + `memory/retrieve` + budget wiring + CLI additions; 005 read path + 01/02 additive fields intact (unused) | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(memory): layered retrieval (L2/L3-first) + item/char/timeout budgets on every mem_* read + CLI parity`

---

