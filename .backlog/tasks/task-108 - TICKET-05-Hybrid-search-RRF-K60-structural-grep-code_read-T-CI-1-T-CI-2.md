---
id: TASK-108
title: >-
  TICKET-05 Hybrid search (RRF K=60) + structural grep + code_read +
  T-CI-1/T-CI-2
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
See .skillgrid/specs/2026-10-31-mnemonic-v2-code-index/tasks.md TICKET-05 for scope/acceptance/ownership.
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
TICKET-05 hybrid search — RRF K=60, 3 legs (FTS+structural+semantic), code_read, T-CI-1/T-CI-2. Review F1-F4 fixed (54b94034). Green.
<!-- SECTION:FINAL_SUMMARY:END -->
