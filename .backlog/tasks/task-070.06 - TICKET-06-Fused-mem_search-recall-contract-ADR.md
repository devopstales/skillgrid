---
id: TASK-070.06
title: 'TICKET-06: Fused mem_search recall + contract ADR'
status: needs-triage
assignee: []
created_date: '2026-10-11 15:34'
updated_date: '2026-10-11 15:39'
labels:
  - second-brain-learnings
milestone: m-9
dependencies:
  - TASK-070.03
references:
  - .skillgrid/specs/2026-10-06-second-brain-learnings/blueprint.md
parent_task_id: TASK-070
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Project-scoped web_cache_fts MATCH with expires_at exclusion, RRF fuse, labeled web hits; vector leg untouched; additive contract ADR (~0040, verify highest, LOCKED OK bundled with re-gate).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 TestMemSearch_FusedWebCache PASS, expired rows absent
- [ ] #2 Contract ADR filed; re-baselines recorded in commit message
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
