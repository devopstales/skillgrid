---
id: TASK-019.09
title: 'TICKET-07: Delete relay, hub HTTP, envelope field, bash snapshot/restore'
status: done
assignee: []
created_date: '2026-09-21 15:11'
updated_date: '2026-09-22 07:56'
labels: []
dependencies:
  - TASK-019.07
  - TASK-019.06
references:
  - .skillgrid/specs/2026-09-21-session-events-layer/blueprint.md
  - .skillgrid/specs/2026-09-21-session-events-layer/tasks.md
  - .skillgrid/specs/2026-09-21-session-events-layer/acceptance.feature
parent_task_id: TASK-019
priority: high
type: refactor
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Delete internal/mnemonic/relay/, http/handoff.go, envelope HandoffRefs, session.go handoff/resume/status; strip snapshot/restore from checkpoint-state.sh (keep guards).

Current: cleave relay, hub HTTP, snapshot/restore subcommands live.

Expected: snapshot/restore report unknown subcommand; rg for the five table names plus cleave in Go sources is empty; test-hooks.sh green.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 snapshot/restore gone; table-name grep empty; hook tests green
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Failing checks (RED)
2. Deletions plus bash strip (GREEN)
3. Hook tests plus build green, commit
<!-- SECTION:PLAN:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Wave 4 review: dispatch failed on infra; coordinator recovered the worktree directly. Verified: build clean, cmd session tests green, hook tests 25/25, snapshot/restore unknown-subcommand, table-name grep clean (only 039/019 schema tests remain for TICKET-10). Committed as 17a3b79.
<!-- SECTION:NOTES:END -->
