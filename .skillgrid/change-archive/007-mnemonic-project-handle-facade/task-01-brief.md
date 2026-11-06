## 01-export-project-handle

### Goal

Callers outside `service` can open an exported Project Handle and reach memory/web without needing the wide open-delegate wrapper surface.

### Out of scope / Non-Goals

- MCP/HTTP rewire (steps 02–03)
- Deleting all wrappers yet (step 04)
- Cross-store behavior changes

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-01` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Produces contracts listed under Interfaces are available to dependents
- [ ] No Global Constraint violated

> Depends on: none

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/service/handle.go` (optional split)
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go`
- Test: `skillgrid-cli/internal/mnemonic/service/*_test.go`

**Interfaces:**
- Consumes: none (existing unexported `projectHandle` + `openProject*`)
- Produces: exported `ProjectHandle` type; `OpenForCWD` / `OpenForDirectory` / `Open(projectID)` returning it; `ProjectID()`, `Memory()`, `Web()` (and `Store()` if still required); cross-store methods unchanged on `Service`

### Tasks

- [x] 01.1 `[RED]` Invalid or empty project id aborts open — no partial handle (threat: store open failure)
  - [x] 01.1.a Write failing test — Scenario: Invalid project id aborts open
  - [x] 01.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'InvalidProject|EmptyProject|AbortOpen' -count=1` — Expected: FAIL
  - [x] 01.1.c Minimal implementation (export type; Open validates and aborts)
  - [x] 01.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'InvalidProject|EmptyProject|AbortOpen' -count=1` — Expected: PASS
  - [x] 01.1.e Commit — `feat(mnemonic): abort Project Handle open on bad project id`
- [x] 01.2 `[RED]` Exported handle opens once and exposes Memory/Web for a happy project
  - [x] 01.2.a Write failing test — Scenario: Opened handle exposes memory and web
  - [x] 01.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'ProjectHandle|OpenOnce|Expose' -count=1` — Expected: FAIL
  - [x] 01.2.c Minimal implementation (export `ProjectHandle`; Open* return it)
  - [x] 01.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -run 'ProjectHandle|OpenOnce|Expose' -count=1` — Expected: PASS
  - [x] 01.2.e Commit — `feat(mnemonic): export Project Handle as single-project seam`
- [x] 01.3 `[AFK]` Cross-store methods still compile and ListProjects / resolve smoke — Scenario: Cross-store root ops remain — `Run: go test ./skillgrid-cli/internal/mnemonic/service/ -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/service/ -count=1` | PASS | | |
| Acceptance `@step-01` / `@p0` | map scenarios to unit runs above | PASS | | |
| Runtime harness | N/A — library seam | PASS | | |
| Rollback boundary | revert commit; no migrations | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): export Project Handle seam`

---

