---
id: TASK-066
title: TICKET-08 Update skillgrid install to install mnemonic binary
status: ready-for-agent
assignee: []
created_date: '2026-10-07 12:51'
updated_date: '2026-10-07 12:52'
labels: []
milestone: m-8
dependencies:
  - TASK-063
references:
  - .skillgrid/specs/2026-10-06-mnemonic-standalone/briefing.md
modified_files:
  - skillgrid-cli/internal/install/install.go
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Make skillgrid install copy/install the mnemonic binary alongside skillgrid (from dist/ or built from source), so an install yields ~/.skillgrid/bin/mnemonic (or ~/.local/bin/mnemonic).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 cd skillgrid-cli && go run ./cmd/skillgrid install --dry-run --yes output mentions copying/installing the mnemonic binary
- [ ] #2 after a real install, the mnemonic binary exists in the bin dir
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
