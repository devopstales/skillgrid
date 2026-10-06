---
id: TASK-046
title: 'TICKET-05: Update existing install tests + acceptance gates + full verify'
status: done
assignee: []
created_date: '2026-10-06 08:17'
updated_date: '2026-10-06 11:55'
labels:
  - local-ollama
  - install
  - qa
milestone: 2026-10-03-local-ollama-models
dependencies: []
references:
  - .skillgrid/specs/2026-10-03-local-ollama-models/tasks.md
  - .skillgrid/specs/2026-10-03-local-ollama-models/acceptance.feature
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Update pre-existing TestSetupProviderLocal and TestSetupProviderEnsureSkipsPullWhenPresent to the six-model catalog (add ollamaVersion stub); run all G1-G6 acceptance gates; run go build ./... + the full install + embedder suite green. SATISFIES: happy path glm vision tools excluded; (regression) G10; (regression) ensure skips pull when models already present. Blocked by: TICKET-04.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go build ./... passes
- [ ] #2 go test ./skillgrid-cli/internal/install/... ./skillgrid-cli/internal/mnemonic/embedder/... -count=1 passes
- [ ] #3 Every G1-G6 CHECK in acceptance.feature returns its EXPECT
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
