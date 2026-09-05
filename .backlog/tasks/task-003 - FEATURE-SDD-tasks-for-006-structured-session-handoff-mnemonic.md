---
id: TASK-003
title: '[FEATURE] SDD tasks punch-list for 006-structured-session-handoff (mnemonic)'
status: ready-for-agent
assignee: []
created_date: '2026-09-04'
updated_date: '2026-09-05'
labels: []
dependencies:
  - TASK-002
priority: medium
type: feature
references:
  - docs/skillgrid/changes/006-structured-session-handoff/tasks.md
  - docs/skillgrid/changes/006-structured-session-handoff/change.md
  - docs/skillgrid/changes/006-structured-session-handoff/acceptance.feature
  - docs/skillgrid/changes/006-structured-session-handoff/steps/
documentation:
  - docs/skillgrid/changes/006-structured-session-handoff/change.md
  - .agents/skills/_shared/conventions/sdd-structure.md
---

## Description

Track the SDD **tasks.md** punch-lists for change **006-structured-session-handoff** (Session Relay steps 01–05). Acceptance features are authored; implementation waits on human choice between `sdd-apply` and `sdd-propose`.

**Current State:**
- Plan tracked by TASK-002; step folders and `tasks.md` exist under `docs/skillgrid/changes/006-structured-session-handoff/steps/`
- Per-step `acceptance.feature` written (spec phase); no production Session Relay code yet

**Expected State:**
- Steps 01–05 task checkboxes completed during `sdd-apply`
- RED threat tests for MCP session tools (steps 02, 03) land before production tools
- `go test ./...` passes for touched packages

## Acceptance Criteria

<!-- AC:BEGIN -->
- [ ] #1 Tasks punch-lists exist for steps 01–05 (filesystem + Engram `sdd/006-structured-session-handoff/tasks`)
- [ ] #2 Acceptance features cover happy/edge/failure per step; threat-matrix scenarios in 02 and 03
- [ ] #3 Apply marks tasks `[x]` in dependency order (01 → 02 → 03 → 04; 05 after 02)
- [ ] #4 Soft-after 003 L0 paths; `.cleave/` gitignored; `go test ./...` passes for touched packages
<!-- AC:END -->

## Definition of Done

<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 Every step `tasks.md` Verdict is PASS or PASS WITH WARNINGS
- [ ] #7 Apply order 01 → 02 → 03 → 04 (05 after 02) respected
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Gate on TASK-002 plan readiness and human Implement vs Revise decision.
2. On Implement: run `sdd-apply` per step folders `01-relay-schema` → `02-handoff-resume` → `03-status-compact` → `04-session-cli`; `05-handoff-watchdog` after 02.
3. Check off punch-list items in each `steps/*/tasks.md` as Verdict PASS; keep Engram `sdd/006-structured-session-handoff/tasks` in sync.
4. Ensure RED threat scenarios for MCP session tools (02, 03) land before production tool code.
5. Close when all step checkboxes + change DoD are green; archive with TASK-002.
<!-- SECTION:PLAN:END -->

## Technical Notes

- Steps: `01-relay-schema`, `02-handoff-resume`, `03-status-compact`, `04-session-cli`, `05-handoff-watchdog`
- Paths: `docs/skillgrid/changes/006-structured-session-handoff/steps/*/tasks.md`
- Spec: Engram `sdd/006-structured-session-handoff/spec`; carry-through plan ticket TASK-002
- Open: watchdog usage signal (client `%` vs token estimate) decided in step 05

## Priority

Medium–High after 003 paths; orthogonal to 004/005.

## Comments

<!-- Conversation appends here. -->

- 2026-09-04: Created for `force_ticket_creation` on sdd-spec (tasks.md artifact). `backlog` CLI SIGTRAPs on this host (Bun arch mismatch); ticket written as filesystem task file matching TASK-001/002 frontmatter. Duplicate-search: TASK-002 covers plan only; no prior tasks-punch-list ticket for 006.
- 2026-09-05: Filled missing `type`, `references`, Definition of Done, and Implementation Plan via filesystem fallback (CLI still SIGILL on edit).
