---
id: TASK-044
title: 'TICKET-03: Runtime default embedder model to embeddinggemma:300m'
status: done
assignee: []
created_date: '2026-10-06 08:17'
updated_date: '2026-10-06 11:55'
labels:
  - local-ollama
  - embedder
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
Change embedder.DefaultOllamaModel from nomic-embed-code to embeddinggemma:300m in ollama.go, with a default-model test. SATISFIES: happy path ollama default model. Blocked by: none. Blocks: TICKET-04. REVERSIBILITY: one-way (forwards the runtime embedder default for existing local installs — the user-confirmed requirement-2 lock; stop for a human checkpoint before this ticket).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go test ./skillgrid-cli/internal/mnemonic/embedder -count=1 -run TestDefaultOllamaModelIsEmbeddinggemma passes
- [ ] #2 DefaultOllamaModel == embeddinggemma:300m
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
