# Archive report — 008-mnemonic-community-knowledge-graph

**Archived**: docs/skillgrid/archive/008-mnemonic-community-knowledge-graph/
**Commit**: e23ab1c docs(sdd): archive 008-mnemonic-community-knowledge-graph
**Move**: git mv changes/ -> archive/; source gone; diff -r EMPTY (snapshot vs archive) ✅
**Ticket**: none (change.md has no Ticket:)

## Gates
- verify: PASS WITH WARNINGS (change); 01/02/03 all PASS. 36 @step-NN scenarios COMPLIANT at runtime.
- tasks: no open `- [ ]` (all 01.x/02.x/03.x [x]).
- human QA: ACCEPTED (user, this turn).
- DoD: satisfied.
- Workspace: allowed edit roots (skillgrid-cli/, docs/skillgrid/).

## Warning (not a 008 defect)
Parallel-session internal/mnemonic/setup refactor (helpers -> new untracked internal/install/agents.go; opencode.go mid-edit) breaks setup build; transitively fails mcp go-build e2e (TestE2EMemoryExtTools, TestE2EEngramParityGaps) + cmd TestTrail*. go build ./... fails ONLY in setup. All 008 packages build + pass. Re-run those e2e once setup/install lands.

## Final state
- Migrations: 012 (community), 013 (framework routes, from 010), 014 (process flows), 015 (knowledge graph) — additive.
- Tool surface: 63 -> 71 additive (code_route/code_navigates [010], code_processes/code_process [008-02], 4 knowledge tools [008-03]).
- New packages: community/, eval/, hybrid/{rerank,confidence,snippet}, route/ [010], process/, knowledge/, graph/span.go.
- 005/010 baselines preserved; CGo-free; go 1.26 (loom-forced).
- 008 SDD cycle COMPLETE.
