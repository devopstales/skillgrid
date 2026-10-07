---
id: TASK-059
title: TICKET-01 Delete stray nested git repo + inline logging
status: done
assignee: []
created_date: '2026-10-07 12:48'
updated_date: '2026-10-07 13:46'
labels: []
milestone: m-8
dependencies: []
references:
  - .skillgrid/specs/2026-10-06-mnemonic-standalone/briefing.md
modified_files:
  - skillgrid-cli/internal/mnemonic/mcp/.git
priority: high
type: refactor
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Extract mnemonic from skillgrid-cli into a standalone module. First step: remove the stray nested .git/.skillgrid under skillgrid-cli/internal/mnemonic/mcp/ and inline the logging helpers into mnemonic/setup so it no longer imports skillgrid-cli/internal/logging. This is the expand phase of the module extraction.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 cd skillgrid-cli && go build ./... && go test ./... PASS
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->

## Comments

<!-- COMMENTS:BEGIN -->
created: 2026-10-07 12:54
---
EXECUTOR NOTE: already committed (inlined setup/logging.go present, 4 setup files use local log*). Verify only — re-run acceptance. No new code expected.
---
<!-- COMMENTS:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
T01 verified: stray nested git repo removed, inline logging moved out of skillgrid-cli into mnemonic/internal/logging. No skillgrid-cli/internal/logging refs remain in the main tree.
<!-- SECTION:FINAL_SUMMARY:END -->
