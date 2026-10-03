---
id: TASK-039.01
title: '[FEATURE] TICKET-01 Completer and mnemonic.llm config (mnemonic)'
status: in-progress
assignee: []
created_date: '2026-10-03 09:36'
labels: []
dependencies: []
references:
  - .skillgrid/specs/2026-10-02-mnemonic-llm-provider/blueprint.md
  - .skillgrid/specs/2026-10-02-mnemonic-llm-provider/acceptance.feature
parent_task_id: TASK-039
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Without a stdlib chat client and a mnemonic.llm config block, later attach and install have nothing to wire. Defaults stay off so existing floors keep working until an operator enables a host and model.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 happy path openai-compatible complete succeeds
- [ ] #2 non-2xx and timeout Complete return an error
- [ ] #3 happy path llm config defaults off
- [ ] #4 enabled config without base_url and model fails Validate
- [ ] #5 api key resolves from SKILLGRID_LLM_API_KEY then OPENAI_API_KEY
- [ ] #6 go.mod gains no LLM SDK
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 G1-G4 gates green
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. RED TestCompleteSuccess, TestCompleteHTTPError, TestCompleteTimeout
2. Implement llm.Client with stdlib HTTP
3. RED TestLLMConfigDefaultOff, TestLLMConfigRequiresURLAndModel, TestLLMAPIKeyFromEnv
4. Add LLM to indexing config and merge
<!-- SECTION:PLAN:END -->
