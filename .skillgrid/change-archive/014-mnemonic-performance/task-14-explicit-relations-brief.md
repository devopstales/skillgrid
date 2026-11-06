## 14-explicit-relations

### Goal

@relation annotations between observations.

### Out of scope / Non-Goals

- All other steps.

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-14` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 07

**Files:**
- `internal/mnemonic/memory/relations.go` — CREATE: @relation annotations between observations
- `internal/mnemonic/memory/relations_test.go` — CREATE: relations tests
- `internal/mnemonic/cmd/skillgrid/mem.go` — MODIFY: add `relations` subcommand

**Interfaces:**
- Consumes: observations table; triple-store linkage from step 07
- Produces: typed relation edges in memory store; `mem relations` CLI

### Tasks

- [ ] 14.1 `[RED]` Relation edges store source_id, target_id, relation_type, confidence
  - [ ] 14.1.a Write failing test (`TestRelationEdgeCreation`): create two observations; call `AddRelation(sourceID, targetID, "mentions", 0.9)`; query the relation and verify it has the correct `source_id`, `target_id`, `relation_type`, and `confidence`; verify each of the 5 relation types can be created (`mentions`, `depends_on`, `contradicts`, `supports`, `references`)
  - [ ] 14.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestRelationEdgeCreation'` — Expected: FAIL
  - [ ] 14.1.c Minimal implementation — create `relations.go` with an `observation_relations` table: `source_id INTEGER NOT NULL, target_id INTEGER NOT NULL, relation_type TEXT NOT NULL, confidence REAL NOT NULL, created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY (source_id, target_id, relation_type)`; implement `AddRelation`, `GetRelations`, `RemoveRelation`
  - [ ] 14.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestRelationEdgeCreation'` — Expected: PASS
  - [ ] 14.1.e Commit — `feat(mnemonic): add typed relation edges between observations`
- [ ] 14.2 `[RED]` mem relations CLI returns all related observations with relation types
  - [ ] 14.2.a Write failing test (`TestMemRelationsCLI`): create observation A with relations to B (`mentions`), C (`depends_on`), D (`contradicts`); invoke `mem relations A`; verify the output lists B, C, D with their relation types and confidence scores; verify the CLI handles observations with no relations gracefully (empty output)
  - [ ] 14.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemRelationsCLI'` — Expected: FAIL
  - [ ] 14.2.c Minimal implementation — add `mem relations <observation_id>` subcommand to `mem.go`; query `observation_relations` for both outgoing and incoming edges; format the output with relation type and confidence; handle missing observation ID with a clear error
  - [ ] 14.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemRelationsCLI'` — Expected: PASS
  - [ ] 14.2.e Commit — `feat(mnemonic): add mem relations CLI subcommand`
- [ ] 14.3 `[AFK]` Confidence filtering on relation queries
  - [ ] 14.3.a Write failing test (`TestRelationConfidenceFiltering`): create relations with confidence 0.3, 0.7, 0.95; query with `minConfidence=0.5`; verify only the 0.7 and 0.95 relations are returned; query with `minConfidence=0.0`; verify all relations are returned
  - [ ] 14.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestRelationConfidenceFiltering'` — Expected: FAIL
  - [ ] 14.3.c Minimal implementation — add `minConfidence` parameter to `GetRelations`; add `WHERE confidence >= ?` to the query; expose the filter via `mem relations <id> --min-confidence 0.5`
  - [ ] 14.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestRelationConfidenceFiltering'` — Expected: PASS
  - [ ] 14.3.e Commit — `feat(mnemonic): add confidence filtering to relation queries`

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestRelationEdgeCreation\|TestRelationConfidenceFiltering'` | PASS | | |
| Acceptance `@step-14` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/internal/mnemonic/memory/` | PASS | | |
| Rollback boundary | verify implicit similarity still works alongside explicit relations | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): explicit @relation annotations between observations`

---

