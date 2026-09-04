---
id: TASK-003
title: '[FEATURE] SDD tasks punch-list for 006-structured-session-handoff (mnemonic)'
status: ready-for-agent
assignee: []
created_date: '2026-09-04'
labels: []
dependencies:
  - TASK-002
priority: medium
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
