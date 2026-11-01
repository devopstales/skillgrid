---
id: TASK-086
title: >-
  [FEATURE] TICKET-03 Intelligence seams (LLM + Embedder + degradation policy)
  (mnemonic-v2)
status: needs-triage
assignee: []
created_date: '2026-11-01 12:41'
labels: []
milestone: m-11
dependencies:
  - TASK-084
references:
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory/blueprint.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory/tasks.md
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory/acceptance.feature
documentation:
  - .skillgrid/specs/2026-10-31-mnemonic-v2-memory/blueprint.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The LLM-first seams with explicit deterministic degradation (INV-3): the LLM interface (4-way Classify, Judge, SuggestCandidates), the Embedder interface, an OpenAI-compatible provider over one base URL, and the degrade package (bounded retry, degraded flag + reason, never a silent floor). Every semantic operation routes through these seams, so their contract — including the zero-value-decision = degraded path — must exist before the save pipeline can degrade correctly.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 httptest OpenAI-compat server: 4-way classify payload returns the typed DedupDecision
- [ ] #2 Nil seam -> zero-value decision + degraded: true with reason
- [ ] #3 Failing seam (HTTP 500) -> exactly one bounded retry then the deterministic floor with reason
- [ ] #4 Embedder round-trip returns fixed-dim vectors; null embedder -> degraded
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
