---
id: TASK-043
title: 'TICKET-02: Six-model catalog + role smoke + floor-gated pull'
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
  - .skillgrid/specs/2026-10-03-local-ollama-models/findings.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Add modelRole/modelEntry/localModels (six tags, glm:vision-tools excluded), heavyModels, the ollamaVersion HTTP seam, smokeProbe (non-fatal), and rewrite pullMissingModels to iterate the catalog and skip heavy models below the floor with a version-named warning. Update localLLMModel to llama3.2:1b, localEmbedModel to embeddinggemma:300m. SATISFIES: happy path catalog pull list; happy path version floor gates heavy models; happy path version at or above floor pulls all. Blocked by: TICKET-01. Blocks: TICKET-04, TICKET-05.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go test ./skillgrid-cli/internal/install -count=1 -run 'TestCatalogPullList|TestSmokeProbeNonFatal|TestFloorGatesHeavyModels|TestAtFloorPullsAll' passes
- [ ] #2 Below floor: tev1/clef-flash not pulled, warning emitted, install succeeds
- [ ] #3 At/above floor: all six pulled
- [ ] #4 glm:vision-tools never appears in pull invocations
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
