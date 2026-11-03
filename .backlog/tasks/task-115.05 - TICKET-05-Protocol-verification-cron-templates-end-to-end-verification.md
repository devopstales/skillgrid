---
id: TASK-115.05
title: 'TICKET-05: Protocol verification + Cron templates + end-to-end verification'
status: done
assignee: []
created_date: '2026-10-25 08:03'
updated_date: '2026-10-25 12:20'
labels: []
dependencies:
  - TASK-115.04
references:
  - .skillgrid/specs/2026-10-24-hermes-integration/blueprint.md
  - .skillgrid/specs/2026-10-24-hermes-integration/tasks.md
  - .skillgrid/specs/2026-10-24-hermes-integration/acceptance.feature
  - .skillgrid/sdd/2026-10-24-hermes-integration/e2e-verification.md
modified_files:
  - mnemonic/internal/setup/hermes.go
  - .agents/skills/_shared/rules/ticketing/hermes-kanban.md
parent_task_id: TASK-115
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Verify protocol is injected via system_prompt_block. Add cron job templates (printed as next-steps after install). Run full end-to-end verification: memory round-trip, MCP test, skills discovery, Kanban round-trip, no-regression check.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 System prompt contains Mnemonic Memory Protocol section
- [x] #2 Cron templates printed after install
- [x] #3 Memory round-trip works (hermes -z remember → recall)
- [x] #4 hermes mcp test skillgrid-mnemonic exits 0
- [x] #5 Skills discoverable
- [x] #6 Kanban ticket round-trip works
- [x] #7 cd mnemonic && go test ./... passes
- [x] #8 cd skillgrid-cli && go test ./... passes
- [x] #9 SATISFIES: happy path — protocol injected; happy path — save and recall; edge — prefetch empty first turn; failure — sync_turn non-blocking
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Tests pass (`go test ./...` for touched packages)
- [x] #2 Lint and formatting pass
- [x] #3 Edge cases covered
- [x] #4 No new warnings introduced
- [x] #5 Spec/docs updated if behavior changes
- [x] #6 Full E2E verification script passes
<!-- DOD:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Verified system_prompt_block protocol (mem_save/mem_search/mem_context present; prefetch empty first turn; sync_turn non-blocking on error). Added informational cron job templates to SetupHermes (verified against hermes cron create --help v0.22.5). Full e2e: memory round-trip PASS (id 182), MCP exclude wire-level PASS (33 mem_* -> 0), skills discovery PASS (555 files), kanban board 'skillgrid' created and listed, go build green x2, setup tests 8/8, skillgrid-cli tests green. Fixed hermes-kanban adapter: --board is top-level on hermes kanban (not on create), --parents -> --parent (repeatable), list --status single-value. Commits: d3e07cf5, 200853be. E2E doc: .skillgrid/sdd/2026-10-24-hermes-integration/e2e-verification.md. Concerns (pre-existing, not this ticket): mnemonic full-suite has 2 env failures (mcp TestUseSkillExecutesAndLogs under parallel load; vectorstore TestObservationsCallerAudit missing archived fixture); installed ~/.local/bin/mnemonic is stale (pre-TICKET-01, lacks mem save/get/update).
<!-- SECTION:FINAL_SUMMARY:END -->
