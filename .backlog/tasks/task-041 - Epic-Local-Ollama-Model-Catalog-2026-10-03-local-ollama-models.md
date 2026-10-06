---
id: TASK-041
title: 'Epic: Local Ollama Model Catalog (2026-10-03-local-ollama-models)'
status: done
assignee: []
created_date: '2026-10-06 08:15'
updated_date: '2026-10-06 11:55'
labels:
  - local-ollama
  - install
  - embedder
milestone: 2026-10-03-local-ollama-models
dependencies: []
references:
  - .skillgrid/specs/2026-10-03-local-ollama-models/briefing.md
  - .skillgrid/specs/2026-10-03-local-ollama-models/tasks.md
  - .skillgrid/specs/2026-10-03-local-ollama-models/acceptance.feature
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Make the local Ollama install path pull a six-model local catalog (live chat llama3.2:1b, system-one qwen2.5:1.5b, embed embeddinggemma:300m, research tev1:0.8b/gemma2:2b/clef-flash) instead of the single llama3.2:3b/nomic-embed-text pair, gate the two heavy research models behind an Ollama >= 0.35.1 version floor, and move the runtime Ollama embedder default to embeddinggemma:300m. No new LLM SDK (ADR-0023); fail-open floors stay (ADR-0016). Single PR, Medium 400-line risk.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 All 5 child tickets done
- [ ] #2 go build ./... passes
- [ ] #3 go test ./skillgrid-cli/internal/install/... ./skillgrid-cli/internal/mnemonic/embedder/... passes
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
