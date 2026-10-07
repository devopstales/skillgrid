---
id: TASK-068
title: TICKET-10 Final verification + cleanup
status: ready-for-agent
assignee: []
created_date: '2026-10-07 12:51'
updated_date: '2026-10-07 12:53'
labels: []
milestone: m-8
dependencies:
  - TASK-063
  - TASK-064
  - TASK-065
  - TASK-066
  - TASK-067
references:
  - .skillgrid/specs/2026-10-06-mnemonic-standalone/briefing.md
priority: high
type: chore
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
End-to-end gate for the module extraction: full build from repo root via go.work, go vet + go test in each module, mnemonic binary subcommand help (mcp/serve/index/orient/search/mem), skillgrid binary trimmed (only install + sync-repo; skillgrid mcp errors), zero stale skillgrid-cli/internal/mnemonic / skillgrid-cli/internal/logging refs outside mnemonic/, and all external configs reference mnemonic.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 both modules go vet + go test PASS
- [ ] #2 go build ./... from repo root via go.work PASS
- [ ] #3 mnemonic mcp --help / serve --help / index --help all work
- [ ] #4 skillgrid --help shows only install + sync-repo and skillgrid mcp errors
- [ ] #5 grep -r 'skillgrid-cli/internal/mnemonic' --include=*.go . | grep -v ^mnemonic/ is empty
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
