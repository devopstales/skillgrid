---
id: TASK-058
title: '[FEATURE] Full-suite green + held-out backstop (mnemonic)'
status: needs-triage
assignee: []
created_date: '2026-10-07 11:33'
updated_date: '2026-10-07 11:35'
labels: []
milestone: m-7
dependencies:
  - TASK-052
  - TASK-053
  - TASK-055
  - TASK-056
  - TASK-057
references:
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/briefing.md
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/tasks.md
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/acceptance.feature
documentation:
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/briefing.md
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/acceptance.feature
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Current State: implementation incomplete.\n\nExpected State: full mnemonic-module suite green + clean build without scanner binaries; BDD scan-findings.feature passes; pre-existing tool-surface tests unchanged. Verification-only — fixups only if a regression surfaces.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 cd mnemonic && go test ./... -count=1 && go build ./... exits 0 with no scanner binaries present (fixture-driven)
- [ ] #2 acceptance-tests/features/scan-findings.feature passes
- [ ] #3 no pre-existing test fails (existing tool surface unchanged)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 Held-out backstop scenarios from blueprint Must-Haves exercised
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Follow blueprint Task 9 (tasks.md TICKET-09); verification-only
2. cd mnemonic && go test ./... -count=1 && go build ./... (no scanner binaries present; fixture-driven) -> exit 0
3. Run acceptance-tests/features/scan-findings.feature per repo acceptance runner -> PASS
4. Verify existing surface unchanged: cd mnemonic && go test ./internal/mnemonic/mcp/ -run 'Test.*Registered' -v
5. Commit only if touched: test(mnemonic): full-suite green for scan findings + dep graph + Refs: TASK-058
<!-- SECTION:PLAN:END -->
