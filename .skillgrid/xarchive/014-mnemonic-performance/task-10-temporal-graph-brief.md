## 10-temporal-graph

### Goal

Temporal knowledge graph edges with valid_from/to.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-10` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 07

**Files:**
- `internal/mnemonic/codeindex/graph.go` — MODIFY: add valid_from/valid_to to edges
- `internal/mnemonic/codeindex/graph_test.go` — CREATE: temporal edge tests

**Interfaces:**
- Consumes: codeindex `edges` table; triple-store linkage from step 07
- Produces: temporal bounds on graph edges; filtered queries hiding expired edges; `mem graph` CLI temporal status

### Tasks

- [ ] 10.1 `[RED]` Edges have valid_from set to current time when relationship is observed
  - [ ] 10.1.a Write failing test (`TestEdgeValidFromSetOnCreate`): create a symbol relationship (edge); verify the edge has `valid_from` set to approximately the current UNIX timestamp; verify `valid_to` is NULL (active); verify the edge is visible in normal queries
  - [ ] 10.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/ -run 'TestEdgeValidFromSetOnCreate'` — Expected: FAIL
  - [ ] 10.1.c Minimal implementation — add `valid_from` (INTEGER, UNIX timestamp) and `valid_to` (INTEGER, nullable, NULL = active) columns to the `edges` table via migration; set `valid_from` to `time.Now().Unix()` when a new edge is created; default `valid_to` to NULL
  - [ ] 10.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/ -run 'TestEdgeValidFromSetOnCreate'` — Expected: PASS
  - [ ] 10.1.e Commit — `feat(mnemonic): add valid_from/valid_to to graph edges`
- [ ] 10.2 `[RED]` Expired edges are hidden from queries but preserved for history
  - [ ] 10.2.a Write failing test (`TestExpiredEdgesHiddenFromQueries`): create an edge with `valid_to` in the past (expired); create another edge with `valid_to` as NULL (active); run a normal edge query; verify only the active edge is returned; run a history query; verify both edges are returned
  - [ ] 10.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/ -run 'TestExpiredEdgesHiddenFromQueries'` — Expected: FAIL
  - [ ] 10.2.c Minimal implementation — add a query filter: `WHERE valid_from <= ? AND (valid_to IS NULL OR valid_to > ?)` with `? = time.Now().Unix()`; add a separate `QueryEdgesWithHistory` method that returns all edges including expired; verify the logic handles `valid_from <= now` correctly
  - [ ] 10.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/ -run 'TestExpiredEdgesHiddenFromQueries'` — Expected: PASS
  - [ ] 10.2.e Commit — `feat(mnemonic): hide expired edges from queries, preserve for history`
- [ ] 10.3 `[AFK]` mem graph CLI shows temporal status of edges
  - [ ] 10.3.a Write failing test (`TestMemGraphTemporalStatus`): create edges with various temporal states (active, expired, future valid_from); invoke `mem graph` CLI; verify the output shows temporal status (active/expired) for each edge; verify the CLI includes `valid_from` and `valid_to` in the output
  - [ ] 10.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemGraphTemporalStatus'` — Expected: FAIL
  - [ ] 10.3.c Minimal implementation — extend the `mem graph` CLI subcommand to include temporal status in the output; display `valid_from`, `valid_to`, and a computed status (`active` / `expired` / `pending`) for each edge
  - [ ] 10.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemGraphTemporalStatus'` — Expected: PASS
  - [ ] 10.3.e Commit — `feat(mnemonic): show temporal status in mem graph CLI`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/codeindex/ -run 'TestEdgeValidFromSetOnCreate\|TestExpiredEdgesHiddenFromQueries'` | PASS | | |
| Acceptance `@step-10` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/codeindex/` | PASS | | |
| Rollback boundary | verify existing edges work with valid_to=NULL default | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): temporal knowledge graph edges with valid_from/valid_to`

---

