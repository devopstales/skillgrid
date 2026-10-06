---
id: TASK-042
title: 'TICKET-01: Ollama version floor comparison'
status: done
assignee: []
created_date: '2026-10-06 08:16'
updated_date: '2026-10-06 11:55'
labels:
  - local-ollama
  - install
milestone: 2026-10-03-local-ollama-models
dependencies: []
references:
  - .skillgrid/specs/2026-10-03-local-ollama-models/tasks.md
  - .skillgrid/specs/2026-10-03-local-ollama-models/blueprint.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add ollamaRequiredVersion (0.35.1), ollamaVersionAtLeast, parseOllamaVersion, versionLessThan to provider.go, with a pure-function test table. SATISFIES: happy path version floor gates heavy models (parsing half). Blocked by: none. Blocks: TICKET-02.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go test ./skillgrid-cli/internal/install -count=1 -run TestOllamaVersionAtLeast passes
- [ ] #2 ollamaVersionAtLeast(0.35.1)=true, (0.35.0)=false, ('')=false, (not-a-version)=false, (0.35.10)=true
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
