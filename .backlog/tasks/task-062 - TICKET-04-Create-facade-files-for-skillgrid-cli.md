---
id: TASK-062
title: TICKET-04 Create facade files for skillgrid-cli
status: ready-for-agent
assignee: []
created_date: '2026-10-07 12:49'
updated_date: '2026-10-07 12:55'
labels: []
milestone: m-8
dependencies:
  - TASK-060
references:
  - .skillgrid/specs/2026-10-06-mnemonic-standalone/briefing.md
modified_files:
  - mnemonic/*.go
  - skillgrid-cli/internal/install/install.go
  - skillgrid-cli/internal/install/agents.go
  - skillgrid-cli/go.mod
priority: high
type: refactor
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Create top-level facade .go files in mnemonic/ (package mnemonic) re-exporting exactly the symbols skillgrid-cli/internal/install needs (setup, config, store). Point skillgrid-cli imports at the facade and add replace/require in skillgrid-cli/go.mod. Facade files mnemonic/setup.go and mnemonic/ui.go already exist; confirm internal/install resolves through the facade and add any missing re-export.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 skillgrid-cli/internal/install imports github.com/devopstales/skillgrid/mnemonic (facade), not any mnemonic/internal/... path
- [ ] #2 cd skillgrid-cli && go build ./... PASS
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
EXECUTOR NOTE: partially committed — facade files mnemonic/setup.go + mnemonic/ui.go exist. Confirm internal/install resolves through the facade and add any missing re-export. Finish, not verify-only.
---
<!-- COMMENTS:END -->
