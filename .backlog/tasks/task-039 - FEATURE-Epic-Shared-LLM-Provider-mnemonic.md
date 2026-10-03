---
id: TASK-039
title: '[FEATURE] Epic: Shared LLM Provider (mnemonic)'
status: in-progress
assignee: []
created_date: '2026-10-03 09:36'
labels: []
dependencies: []
references:
  - .skillgrid/specs/2026-10-02-mnemonic-llm-provider/briefing.md
  - .skillgrid/specs/2026-10-02-mnemonic-llm-provider/blueprint.md
  - .skillgrid/specs/2026-10-02-mnemonic-llm-provider/tasks.md
  - .skillgrid/specs/2026-10-02-mnemonic-llm-provider/acceptance.feature
documentation:
  - .skillgrid/specs/2026-10-02-mnemonic-llm-provider/briefing.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Ask, dedup, extraction, and dream each lack a production LLM client, so task-029 stays open and install never wires a local or external model. One OpenAI-compatible Completer attaches every seam at boot, and skillgrid install sets up that provider.

Spec: .skillgrid/specs/2026-10-02-mnemonic-llm-provider/
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 One stdlib Completer serves ask, dedup, extraction, and dream
- [ ] #2 skillgrid install wires Local Ollama or an external OpenAI-compatible host
- [ ] #3 task-029 is closed by this change
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 Change-level Definition of Done in briefing.md fully checked
- [ ] #7 No new LLM SDK dependency in go.mod
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Follow blueprint.md tasks 1-6 in ticket order
2. Drive from tasks.md and acceptance.feature
3. Verify focused package tests before closing
<!-- SECTION:PLAN:END -->
