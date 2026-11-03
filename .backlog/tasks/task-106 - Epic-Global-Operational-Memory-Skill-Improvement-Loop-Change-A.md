---
id: TASK-106
title: 'Epic: Global Operational Memory + Skill Improvement Loop (Change A)'
status: needs-triage
assignee: []
created_date: '2026-10-24 13:16'
labels:
  - mnemonic
  - global-memory
  - skill-loop
dependencies: []
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Cross-project fact store in `__global__.sqlite` with AKL importance, char limit, prefetch injection, background distill, write approval, and a skill improvement loop (observe → inspect → amend → evaluate). Change A of the 2026-10-24-global-llm-wiki spec (ADR-0049).

Spec: `.skillgrid/specs/2026-10-24-global-llm-wiki/briefing.md`
Blueprint: `.skillgrid/specs/2026-10-24-global-llm-wiki/blueprint.md`
Tasks: `.skillgrid/specs/2026-10-24-global-llm-wiki/tasks.md`
Acceptance: `.skillgrid/specs/2026-10-24-global-llm-wiki/acceptance.feature`

Delivery: 4 chained PRs (stacked-to-main), ~1100-1500 lines, 5 waves.
One-way doors: T01 (migration 072), T07 (migration 073 + execute.go change).
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
