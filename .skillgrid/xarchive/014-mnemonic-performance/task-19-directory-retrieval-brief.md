## 19-directory-retrieval

### Goal

Directory-level recursive retrieval with drill-down + observable trajectory.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-19` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 02, 03

**Files:**
- `internal/mnemonic/memory/retrieval.go` — CREATE: directory-level recursive retrieval + trajectory
- `internal/mnemonic/memory/retrieval_test.go` — CREATE: retrieval tests
- `internal/mnemonic/cmd/skillgrid/mem.go` — MODIFY: add `search --trajectory` flag

**Interfaces:**
- Consumes: FTS trigram from step 02; parallel search from step 03; embeddings
- Produces: `Retrieve` module with intent analysis → hierarchical retrieval → rerank; `retrieval_trails` storage; `--trajectory` CLI flag

### Tasks

- [ ] 19.1 `[RED]` Directory retrieval finds highest-scoring directory first, then drills down
  - [ ] 19.1.a Write failing test (`TestDirectoryRetrievalDrillDown`): create a hierarchical set of observations organized by scope (project/app/core, project/app/api, project/docs); search for a query matching observations in `project/app/core`; verify the retrieval first identifies `project/app/core` as the highest-scoring directory; verify it drills down into that directory; verify results are scoped to the drilled-down directory
  - [ ] 19.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDirectoryRetrievalDrillDown'` — Expected: FAIL
  - [ ] 19.1.c Minimal implementation — create `retrieval.go` with `Retrieve(ctx, query, scope) (Results, Trajectory, error)`; first pass: score all top-level scopes by FTS5 + embedding similarity; second pass: drill down into the highest-scoring scope; repeat until leaf level; depth limit of 5 to prevent infinite recursion
  - [ ] 19.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDirectoryRetrievalDrillDown'` — Expected: PASS
  - [ ] 19.1.e Commit — `feat(mnemonic): directory-level recursive retrieval with drill-down`
- [ ] 19.2 `[RED]` Retrieval trajectory is preserved for debugging
  - [ ] 19.2.a Write failing test (`TestRetrievalTrajectoryPreserved`): run a directory retrieval; verify the trajectory records each directory visited (path, score, depth); verify the trajectory is stored in `retrieval_trails`; verify the trajectory can be retrieved by query ID; verify the trajectory shows the drill-down path
  - [ ] 19.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestRetrievalTrajectoryPreserved'` — Expected: FAIL
  - [ ] 19.2.c Minimal implementation — create a `retrieval_trails` table: `id INTEGER PRIMARY KEY, query TEXT, path TEXT, scores TEXT, depth INTEGER, created_at TIMESTAMP`; record each directory visit during retrieval; store the complete path as a JSON array; add `GetTrajectory(queryID)` to retrieve the trail
  - [ ] 19.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestRetrievalTrajectoryPreserved'` — Expected: PASS
  - [ ] 19.2.e Commit — `feat(mnemonic): preserve retrieval trajectory for debugging`
- [ ] 19.3 `[RED]` Intent analysis classifies query before retrieval
  - [ ] 19.3.a Write failing test (`TestIntentAnalysisClassification`): classify a query like "why does the build fail" as `debugging`; classify "what files are in the auth module" as `exploration`; classify "review the changes to payment.go" as `review`; verify the intent classification affects the retrieval strategy (debugging → wider search, exploration → directory-first)
  - [ ] 19.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestIntentAnalysisClassification'` — Expected: FAIL
  - [ ] 19.3.c Minimal implementation — implement `ClassifyIntent(query string) Intent` using keyword matching and pattern detection; return one of `exploration`, `debugging`, `review`, `refactor`; adjust retrieval parameters based on intent (debugging → wider scope, exploration → directory-first)
  - [ ] 19.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestIntentAnalysisClassification'` — Expected: PASS
  - [ ] 19.3.e Commit — `feat(mnemonic): intent analysis for query classification`
- [ ] 19.4 `[AFK]` Depth limit prevents excessive drill-down on deep hierarchies
  - [ ] 19.4.a Write failing test (`TestDirectoryRetrievalDepthLimit`): create a hierarchy 10 levels deep; run a retrieval; verify it stops at depth 5 (the configured limit); verify it returns the best results found at the limit; verify no stack overflow or infinite loop
  - [ ] 19.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDirectoryRetrievalDepthLimit'` — Expected: FAIL
  - [ ] 19.4.c Minimal implementation — enforce `maxDepth = 5` in the drill-down loop; when the limit is reached, return the best results at that depth; cache directory scores to avoid recomputation on repeated queries
  - [ ] 19.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDirectoryRetrievalDepthLimit'` — Expected: PASS
  - [ ] 19.4.e Commit — `feat(mnemonic): depth limit and caching for directory retrieval`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestDirectoryRetrievalDrillDown\|TestRetrievalTrajectoryPreserved'` | PASS | | |
| Acceptance `@step-19` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/` | PASS | | |
| Rollback boundary | verify default flat search still works when directory retrieval is disabled | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): directory-level recursive retrieval with drill-down and trajectory`

---

