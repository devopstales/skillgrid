---
id: TASK-109
title: >-
  TICKET-06 Watch loop (fsnotify + debounce + burst-collapse + re-check +
  isolation)
status: done
assignee: []
created_date: '2026-11-02 15:52'
updated_date: '2026-11-03 18:24'
labels: []
milestone: 2026-10-31-mnemonic-v2-code-index
dependencies: []
priority: high
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
See .skillgrid/change-specs/2026-10-31-mnemonic-v2-code-index/tasks.md TICKET-06 for scope/acceptance/ownership.
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
TICKET-06 watch loop — fsnotify, debounce, burst-collapse, re-check, M10 delete, isolation (6ab0f3ef + OnSync error seam 3952f322). -race clean.
<!-- SECTION:FINAL_SUMMARY:END -->
