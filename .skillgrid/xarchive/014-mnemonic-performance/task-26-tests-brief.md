## 26-tests

### Goal

Unit + integration coverage for all steps (01-25).

### Out of scope / Non-Goals

- New production features beyond closing RED coverage.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-26` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 02, 03, 04, 05, 06, 07, 08, 09, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25

**Files:**
- `skillgrid-cli/...` — all touched packages: store, memory, service, codeindex, embedder, config, memfs, cmd/skillgrid, mcp, http

**Interfaces:**
- Consumes: all production code from steps 01-25
- Produces: full test coverage; `go test ./skillgrid-cli/...` passes

### Tasks

- [ ] 26.1 `[RED]` Full test suite passes for all touched packages
  - [ ] 26.1.a Write failing test (verify gap): run `go test ./skillgrid-cli/internal/mnemonic/store/ ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/config/ ./skillgrid-cli/internal/mnemonic/embedder/ ./skillgrid-cli/internal/mnemonic/codeindex/` and identify any test gaps or failures from steps 01-25
  - [ ] 26.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/config/ ./skillgrid-cli/internal/mnemonic/embedder/ ./skillgrid-cli/internal/mnemonic/codeindex/` — Expected: FAIL (if gaps exist) or PASS (if all covered)
  - [ ] 26.1.c Minimal implementation — close any remaining RED coverage gaps identified in 26.1.a; add missing edge-case tests for: store pooling (concurrent opens), FTS trigram (empty results fallback), parallel search (50+ stores), TTL (boundary conditions), LLM extraction (error handling), embedder (config-driven selection), triple-store (join correctness), improve() (no regression), session promotion (dedup), temporal edges (boundary timestamps), export (roundtrip), dream (lock + rollback), importance (skew), relations (confidence filtering), provenance (immutability), federated query (dedup), distill lock (timeout), memory types (async commit), directory retrieval (depth limit), snapshots (auto-prune), handoff (staleness), context envelope (size limit), hub (accuracy), skills (hook timeout), memfs (URI resolution with 10k+ observations)
  - [ ] 26.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/store/ ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/config/ ./skillgrid-cli/internal/mnemonic/embedder/ ./skillgrid-cli/internal/mnemonic/codeindex/` — Expected: PASS
  - [ ] 26.1.e Commit — `feat(mnemonic): close RED coverage gaps across all steps`
- [ ] 26.2 `[RED]` Integration tests pass for MCP, HTTP, and CLI layers
  - [ ] 26.2.a Write failing test (verify gap): run `go test ./skillgrid-cli/internal/mnemonic/mcp/ ./skillgrid-cli/internal/mnemonic/http/ ./skillgrid-cli/cmd/skillgrid/` and identify any integration test gaps
  - [ ] 26.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ ./skillgrid-cli/internal/mnemonic/http/ ./skillgrid-cli/cmd/skillgrid/` — Expected: FAIL (if gaps exist) or PASS (if all covered)
  - [ ] 26.2.c Minimal implementation — add integration tests that verify: MCP tools (`mem_search`, `mem_save`, etc.) work with the new features (trigram mode, federated query, memfs); HTTP API endpoints work with new parameters; CLI subcommands (`mem expire`, `mem export`, `mem relations`, `mem provenance`, `mem dream`, `mem snapshot`, `mem handoff`, `mem context`, `mem graph --risk`, `mem skills`, `mem hook`, `mem fs`) produce correct output
  - [ ] 26.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/ ./skillgrid-cli/internal/mnemonic/http/ ./skillgrid-cli/cmd/skillgrid/` — Expected: PASS
  - [ ] 26.2.e Commit — `feat(mnemonic): integration tests for MCP, HTTP, and CLI layers`
- [ ] 26.3 `[AFK]` Full suite `go test ./skillgrid-cli/...` passes
  - [ ] 26.3.a Write failing test (verify gap): run the full suite and identify any failures
  - [ ] 26.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/...` — Expected: FAIL (if gaps exist) or PASS (if all covered)
  - [ ] 26.3.c Minimal implementation — fix any remaining failures across the entire `skillgrid-cli` module; ensure no test regressions from steps 01-25; verify all BDD `@step-NN` scenarios in `acceptance.feature` map to passing unit/integration tests
  - [ ] 26.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/...` — Expected: PASS
  - [ ] 26.3.e Commit — `feat(mnemonic): full test suite green across all packages`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/store/ ./skillgrid-cli/internal/mnemonic/memory/ ./skillgrid-cli/internal/mnemonic/service/ ./skillgrid-cli/internal/mnemonic/config/ ./skillgrid-cli/internal/mnemonic/embedder/ ./skillgrid-cli/internal/mnemonic/codeindex/` | PASS | | |
| Acceptance `@step-26` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/...` | PASS | | |
| Rollback boundary | verify no test regressions from any step | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): full unit + integration coverage for all steps`

---

