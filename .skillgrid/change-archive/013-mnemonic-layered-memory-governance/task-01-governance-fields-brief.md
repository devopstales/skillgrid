## 01-governance-fields

### Goal

Every memory is a governed asset — owner, version history (append), status, retrieval usage, visibility (private-by-default) — shared only by an explicit `mem_share`, surfaced by `mem_governance`, with existing `mem_*` tools unchanged.

### Out of scope / Non-Goals

- Layering / `observation_layers` / `personas` / `mem_layers` (step 02)
- Retrieval budgets / layered retrieval (step 03)
- Multi-tenant teams/roles/proxy; changing `mem_*` tool names/required params

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-01` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Produces contracts listed under Interfaces are available to dependents
- [ ] No Global Constraint violated

> Depends on: none

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/store/migrations/015_layered_memory_governance.sql`
- Create: `skillgrid-cli/internal/mnemonic/memory/governance.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_memory_governance.go`
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go`
- Test: `skillgrid-cli/internal/mnemonic/memory/governance_test.go`, `skillgrid-cli/internal/mnemonic/store/migrations_test.go`, `skillgrid-cli/internal/mnemonic/mcp/tools_memory_governance_test.go`, `skillgrid-cli/internal/mnemonic/service/governance_test.go`

**Interfaces:**
- Consumes: 005 observation store + `mem_save`/`mem_update`/`mem_search`/`mem_get_observation` (names + required params unchanged)
- Produces: `015_*` governance columns (owner, status, visibility, retrieval_usage) + `observation_versions` + `acl_grants`; `mem_share <id> <team|restricted|agent> [acl]`; `mem_governance <id>`; additive governance fields on existing `mem_*` responses — relied on by steps 02/03

### Tasks

<!-- [RED] items are the applicable threat-matrix rows (Mnemonic tool surface + Data leak/visibility), ordered BEFORE the production [AFK] tasks. Each uses the TDD micro-cycle a–e. Scenario names are the referenceable strings from acceptance.feature @step-01. -->

- [ ] 01.1 `[RED]` Threat "Mnemonic tool surface" — governance tools registered + private-by-default + append-version + existing `mem_*` schema stable + bad args rejected (Scenarios: `governance-tools-and-existing-schema-stable`, `new-observation-private-by-default-with-owner`, `mem-update-appends-recoverable-version`, `bad-governance-args-rejected`)
  - [ ] 01.1.a Write failing test
  - [ ] 01.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run GovernanceTools -count=1` — Expected: FAIL
  - [ ] 01.1.c Minimal implementation
  - [ ] 01.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run GovernanceTools -count=1` — Expected: PASS
  - [ ] 01.1.e Commit — `feat(memory): register mem_share/mem_governance + stable mem_* schema`
- [ ] 01.2 `[RED]` Threat "Data leak / visibility" — private invisible to 2nd owner until share; restricted ACL enforced; no-grant restricted is owner-only; private absent from admin cross-owner list (Scenarios: `private-observation-invisible-to-second-owner-until-share`, `restricted-acl-grant-enforced`, `restricted-with-no-grants-is-owner-only`, `private-observation-absent-from-admin-cross-owner-list`)
  - [ ] 01.2.a Write failing test
  - [ ] 01.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/... -run Visibility -count=1` — Expected: FAIL
  - [ ] 01.2.c Minimal implementation
  - [ ] 01.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/memory/... -run Visibility -count=1` — Expected: PASS
  - [ ] 01.2.e Commit — `feat(memory): private-by-default visibility + restricted ACL enforcement`
- [ ] 01.3 `[AFK]` Additive `015_*` governance schema (governance columns + `observation_versions` + `acl_grants`) — `Run: go test ./skillgrid-cli/internal/mnemonic/store/... -count=1` — Expected: PASS
- [ ] 01.4 `[AFK]` `mem_save` private-by-default + owner; second owner's search excludes until shared (Scenarios: `new-observation-private-by-default-with-owner`, `private-observation-invisible-to-second-owner-until-share`) — `Run: go test ./skillgrid-cli/internal/mnemonic/service/... -run PrivateDefault -count=1` — Expected: PASS
- [ ] 01.5 `[AFK]` `mem_share <id> <team|restricted|agent> [acl]` is the only explicit widen; unknown owner/agent/role rejected, visibility unchanged (Scenario: `mem-share-unknown-target-rejected`) — `Run: go test ./skillgrid-cli/internal/mnemonic/service/... -run MemShare -count=1` — Expected: PASS
- [ ] 01.6 `[AFK]` `mem_update <id>` appends a row to `observation_versions`; prior content recoverable via `mem_governance`; latest version is the read path; `revision_count` advances (Scenario: `mem-update-appends-recoverable-version`) — `Run: go test ./skillgrid-cli/internal/mnemonic/service/... -run UpdateVersion -count=1` — Expected: PASS
- [ ] 01.7 `[AFK]` `superseded`/`archived` status set explicitly, never inferred (Scenario: `superseded-status-set-explicitly`) — `Run: go test ./skillgrid-cli/internal/mnemonic/service/... -run StatusExplicit -count=1` — Expected: PASS
- [ ] 01.8 `[AFK]` Search hit increments retrieval usage count, distinct from `duplicate_count` (Scenario: `retrieval-usage-count-increments-on-search`) — `Run: go test ./skillgrid-cli/internal/mnemonic/service/... -run UsageCount -count=1` — Expected: PASS
- [ ] 01.9 `[AFK]` `mem_governance <id>` returns owner/version history/status/usage/visibility (Scenario: `mem-governance-surfaces-asset-fields`) — `Run: go test ./skillgrid-cli/internal/mnemonic/service/... -run GovernanceQuery -count=1` — Expected: PASS
- [ ] 01.10 `[AFK]` Existing `mem_*` names + required params unchanged; additive fields only; bad governance args rejected clearly (Scenarios: `governance-tools-and-existing-schema-stable`, `bad-governance-args-rejected`) — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/memory/... ./skillgrid-cli/internal/mnemonic/store/... -count=1` | PASS | | |
| Acceptance `@step-01` / `@p0` | `go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -count=1` + BDD `@step-01 @p0` | PASS | | |
| Runtime harness | `go test ./skillgrid-cli/... -count=1` | PASS | | |
| Rollback boundary | drop `015_*` governance portion + `memory/governance.go` + `tools_memory_governance.go`; 005 store + `mem_*` intact | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(memory): governance fields (owner/version/status/usage/visibility) + mem_share/mem_governance`

---

