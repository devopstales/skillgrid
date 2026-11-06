## 01-community-detection

### Goal

Additive `012_*` schema + pure-Go Leiden (`bluuewhale/loom`) + god nodes + LLM-free community labels + `code_communities` / `code_god_nodes` / `code_explain_community` MCP/CLI tools so agents see subsystems and the most-connected concepts.

### Out of scope / Non-Goals

- Doc/config/SQL knowledge nodes (step 03)
- Process flows (step 02)
- Hybrid search; changing 005 tools

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-01` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Produces contracts listed under Interfaces are available to dependents
- [ ] No Global Constraint violated

> Depends on: none (005 done; 010 not required for this step)

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/store/migrations/012_community_knowledge_graph.sql`
- Create: `skillgrid-cli/internal/mnemonic/community/leiden.go`
- Create: `skillgrid-cli/internal/mnemonic/community/godnodes.go`
- Create: `skillgrid-cli/internal/mnemonic/community/labels.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_code_community.go`
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go` (community facade)
- Modify: `skillgrid-cli/cmd/skillgrid/code_intel.go` (CLI community commands)
- Modify: `skillgrid-cli/go.mod` (add `bluuewhale/loom`)
- Test: `skillgrid-cli/internal/mnemonic/community/...`, `skillgrid-cli/internal/mnemonic/store/...`, `skillgrid-cli/internal/mnemonic/mcp/...`, `skillgrid-cli/cmd/skillgrid/...`

**Interfaces:**
- Consumes: 005's symbols/edges tables, existing store migration runner, 005 tool registration baseline
- Produces: `communities` / `community_meta` tables + god-node ranking; LLM-free community labels; `code_communities`, `code_god_nodes` (`--exclude-hubs`), `code_explain_community` MCP tools + CLI parity; seeded Leiden partition (content-hash cache key)

### Tasks

- [ ] 01.1 `[RED]` Mnemonic tool surface — 005 `code_search` schema stable before community tools land (Scenario: code_communities returns labeled subsystems and 005 tools stay stable) — threat: Mnemonic tool surface
  - [ ] 01.1.a Write failing test — assert 005 `code_*` tool names + required params are unchanged (baseline lock) before community tools register
  - [ ] 01.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run ToolSurfaceBaseline -count=1` — Expected: FAIL (or red until baseline lock exists)
  - [ ] 01.1.c Minimal implementation — lock 005 tool-surface baseline assertions
  - [ ] 01.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run ToolSurfaceBaseline -count=1` — Expected: PASS
  - [ ] 01.1.e Commit — `test(mnemonic): lock 005 tool surface baseline`
- [ ] 01.2 `[RED]` Leiden over 005 edges produces labeled communities (Scenario: code_communities returns labeled subsystems and 005 tools stay stable)
  - [ ] 01.2.a Write failing test — fixture graph of symbols/edges; after the community pass, `communities` rows partition the graph coherently, each community carries an LLM-free label; `code_communities` returns the partition; 005 tools still registered
  - [ ] 01.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/community/... ./skillgrid-cli/internal/mnemonic/mcp/... -run Communities -count=1` — Expected: FAIL
  - [ ] 01.2.c Minimal implementation — `012_*` migration + `community/leiden.go` (loom `NodeRegistry` + `LeidenOptions{Seed, Resolution, MaxIterations, NumRuns}`, partition → community rows) + `labels.go` + `go.mod` dep + `tools_code_community.go` `code_communities` + service facade
  - [ ] 01.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/community/... ./skillgrid-cli/internal/mnemonic/mcp/... -run Communities -count=1` — Expected: PASS
  - [ ] 01.2.e Commit — `feat(mnemonic): leiden community detection with llm-free labels`
- [ ] 01.3 `[RED]` Mnemonic tool surface — community tools register and reject bad args (Scenario: community tools reject bad args clearly) — threat: Mnemonic tool surface
  - [ ] 01.3.a Write failing test — `code_communities` / `code_god_nodes` / `code_explain_community` registered with distinct `code_*` names; bad/missing args (e.g. non-existent community id) rejected clearly with a validation error, no invented communities
  - [ ] 01.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run CommunityToolsArgs -count=1` — Expected: FAIL
  - [ ] 01.3.c Minimal implementation — arg validation in `tools_code_community.go` + service
  - [ ] 01.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run CommunityToolsArgs -count=1` — Expected: PASS
  - [ ] 01.3.e Commit — `feat(mnemonic): community tool arg validation`
- [ ] 01.4 `[RED]` God nodes ranked by degree with exclude-hubs (Scenario: code_god_nodes ranks hubs and exclude-hubs suppresses them)
  - [ ] 01.4.a Write failing test — `code_god_nodes` returns symbols ranked by degree; `--exclude-hubs` suppresses utility super-hubs from the ranking
  - [ ] 01.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/community/... -run GodNodes -count=1` — Expected: FAIL
  - [ ] 01.4.c Minimal implementation — `community/godnodes.go` (degree ranking + `--exclude-hubs`) + `code_god_nodes` tool + CLI flag
  - [ ] 01.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/community/... -run GodNodes -count=1` — Expected: PASS
  - [ ] 01.4.e Commit — `feat(mnemonic): god-node ranking with exclude-hubs`
- [ ] 01.5 `[AFK]` code_explain_community returns members and entry points (Scenario: code_explain_community explains a subsystem) — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -count=1` — Expected: PASS
- [ ] 01.6 `[AFK]` Tiny graph (< 2 nodes) yields one trivial community, no crash (Scenario: Tiny graph yields a single trivial community) — `Run: go test ./skillgrid-cli/internal/mnemonic/community/... -run TinyGraph -count=1` — Expected: PASS
- [ ] 01.7 `[AFK]` Community label falls back to community-N when no god node (Scenario: Community label falls back when no god node exists) — `Run: go test ./skillgrid-cli/internal/mnemonic/community/... -count=1` — Expected: PASS
- [ ] 01.8 `[AFK]` Communities are seeded, pinned, and cached by content-hash (Scenario: Community partition is reproducible and cached by content-hash) — `Run: go test ./skillgrid-cli/internal/mnemonic/community/... ./skillgrid-cli/internal/mnemonic/codeindex/... -count=1` — Expected: PASS
- [ ] 01.9 `[AFK]` CLI parity for community commands via `code_intel.go` (Scenario: CLI community commands return the same views) — `Run: go test ./skillgrid-cli/cmd/skillgrid/... -count=1` — Expected: PASS

### Verification

Verdict: `PENDING`  <!-- PASS | PASS WITH WARNINGS | FAIL -->

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/community/... ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/internal/mnemonic/service/... -count=1` | PASS | | |
| Acceptance `@step-01` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `skillgrid index` on fixture repo; `code_communities` / `code_god_nodes` via MCP + CLI | PASS | | |
| Rollback boundary | Drop `012_*` + `community/` + community tools + `go.mod` dep | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): leiden community detection, god nodes, and community tools`

---

