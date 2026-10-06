---
id: TASK-039.03
title: '[FEATURE] TICKET-03 Natural install provider setup (mnemonic)'
status: ready-for-agent
assignee: []
created_date: '2026-10-03 09:39'
updated_date: '2026-10-06 08:35'
labels: []
milestone: m-4
dependencies:
  - TASK-039.01
references:
  - .skillgrid/specs/2026-10-02-mnemonic-llm-provider/blueprint.md
  - .skillgrid/specs/2026-10-02-mnemonic-llm-provider/acceptance.feature
parent_task_id: TASK-039
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Install never ensures the embedder and chat host that code indexing already needs. A normal install step should ensure local Ollama or wire an external OpenAI-compatible host into the home indexing config, and stay skippable and non-fatal.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 happy path install yes ensures local ollama and wires config
- [ ] #2 external provider wires without ollama commands
- [ ] #3 skip-provider is a no-op
- [ ] #4 dry-run writes nothing
- [ ] #5 provider setup failure is non-fatal
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 G10-G11 gates green
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. RED TestSetupProviderLocal, Skipped, External, EnsureSkipsPull, DryRun, FailureNonFatal
2. Implement setupProvider with a fakeable runner
3. Wire install after security tools and add --provider and --skip-provider
<!-- SECTION:PLAN:END -->
