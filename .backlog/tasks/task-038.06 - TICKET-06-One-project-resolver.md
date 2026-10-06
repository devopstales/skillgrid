---
id: TASK-038.06
title: 'TICKET-06: One project resolver'
status: done
assignee: []
created_date: '2026-10-02 14:47'
updated_date: '2026-10-06 08:35'
labels: []
milestone: m-3
dependencies: []
references:
  - .skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/briefing.md
  - .skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/tasks.md
  - .skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/acceptance.feature
documentation:
  - .skillgrid/specs/2026-10-02-mnemonic-memory-checkpoint/briefing.md
parent_task_id: TASK-038
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Hooks send directory only; server resolves via project.Resolve. SATISFIES prime-and-hooks-share-one-project, directory-without-project-param, prime-in-another-repository.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 POST with only directory stores under project.Resolve(dir)
- [x] #2 prime and HTTP agree; resolveProject gone from capture script
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 go test ./internal/mnemonic/http ./cmd/skillgrid -run TestToolCalls_ResolvesProjectFromDirectory|TestPrime_ProjectMatchesHTTPStore is ok
<!-- DOD:END -->

## Implementation Plan
<!-- SECTION:PLAN:BEGIN -->
1. Write TestToolCalls_ResolvesProjectFromDirectory and TestPrime_ProjectMatchesHTTPStore.
2. projectFromRequest falls back to project.Resolve(directory); capture script sends directory only.
3. Delete resolveProject from tool-call-capture.js.
<!-- SECTION:PLAN:END -->
