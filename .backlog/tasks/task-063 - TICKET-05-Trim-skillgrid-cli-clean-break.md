---
id: TASK-063
title: TICKET-05 Trim skillgrid-cli (clean break)
status: ready-for-agent
assignee: []
created_date: '2026-10-07 12:49'
updated_date: '2026-10-07 12:55'
labels: []
milestone: m-8
dependencies:
  - TASK-062
references:
  - .skillgrid/specs/2026-10-06-mnemonic-standalone/briefing.md
modified_files:
  - skillgrid-cli/cmd/skillgrid/main.go
  - skillgrid-cli/go.mod
priority: high
type: refactor
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Contract phase of the module extraction. Trim skillgrid-cli/cmd/skillgrid/main.go to install/in/sync-repo/help only (remove all mnemonic cases + run* refs + imports), delete skillgrid-cli/internal/logging/, delete the 61 MB skillgrid-cli/skillgrid binary, and drop mnemonic-only deps from go.mod + go mod tidy. This is the one-way-door clean break.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 cd skillgrid-cli && go build ./... && go test ./... PASS
- [ ] #2 skillgrid --help shows only install + sync-repo
- [ ] #3 skillgrid mcp prints unknown command
- [ ] #4 skillgrid-cli/skillgrid and skillgrid-cli/internal/logging/ are gone
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
created: 2026-10-07 12:55
---
EXECUTOR NOTE: mostly committed (only internal/install remains in skillgrid-cli/internal/, mnemonic cmd files removed). Verify internal/logging + binary deleted and go.mod trimmed; finish if not. One-way door.
---
<!-- COMMENTS:END -->
