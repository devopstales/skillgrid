---
id: TASK-045
title: 'TICKET-04: Config merge writes new live-chat + embed models'
status: done
assignee: []
created_date: '2026-10-06 08:17'
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
Verify mergeHomeProviderConfig writes localLLMModel/localEmbedModel (from TICKET-02) and the runtime default (TICKET-03) so the merged home config carries llm.model=llama3.2:1b + embedder.provider=ollama/embedder.model=embeddinggemma:300m, preserving unrelated keys. SATISFIES: happy path config merge writes new live-chat + embed models. Blocked by: TICKET-02, TICKET-03. Blocks: TICKET-05.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go test ./skillgrid-cli/internal/install -count=1 -run TestConfigMergeWritesNewModels passes
- [ ] #2 Merged config has llm.model=llama3.2:1b, embedder.provider=ollama, embedder.model=embeddinggemma:300m, base_url=ollamaBaseURL/v1, dimension+profile preserved
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
