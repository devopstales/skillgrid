---
id: TASK-072
title: T09 Chronos injection + version-pair + supply-chain gate
status: done
assignee: []
created_date: '2026-10-09 13:08'
updated_date: '2026-10-14 09:33'
labels:
  - skill-pi
  - tracing
dependencies:
  - TASK-069
references:
  - .skillgrid/specs/2026-10-09-skill-pi/tasks.md
  - .skillgrid/specs/2026-10-09-skill-pi/acceptance.feature
modified_files:
  - skill-pi/extensions/chronos.ts
  - skill-pi/version.test.js
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement `chronos.ts` (date/time injection into pi session context). Extend `version.test.js` with version-pair equality + supply-chain assertions (exact pin, no ^/~/range, shrinkwrap present). Tracer A, Wave 3 (independent of T03/T04).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 A pi session's context includes the injected date/time from chronos.ts, matching the system clock (within 1s)
- [ ] #2 pi-version.txt content equals the package.json dependency string
- [ ] #3 The @earendil-works/pi-* deps are exact (no ^/~/range); shrinkwrap present
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
