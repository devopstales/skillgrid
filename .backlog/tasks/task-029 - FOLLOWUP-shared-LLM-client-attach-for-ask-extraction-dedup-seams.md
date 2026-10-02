---
id: TASK-029
title: 'FOLLOWUP: shared LLM client attach for ask/extraction/dedup seams'
status: needs-triage
assignee: []
created_date: '2026-10-02 07:14'
labels:
  - bitemporal-audn
  - followup
  - llm-wiring
dependencies: []
references:
  - .skillgrid/specs/2026-09-24-bitemporal-audn/blueprint.md
documentation:
  - skillgrid-cli/internal/mnemonic/service/dedup_llm.go
  - skillgrid-cli/internal/mnemonic/service/service.go
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Deferred from TICKET-05 review (bitemporal-audn). No production LLM client is constructed anywhere in the codebase: SetAskLLM, EnableExtractionLLM, and the new dedupLLMFunc seam are all only populated in tests. When mnemonic.dedup.llm (or .extraction.llm / ask llm-mode) is on, the seam is armed and invoked but returns "no LLM configured" → hash floor. A single shared completion client (one SetXLLMFunc wired where the real model client is built) should back all three seams so wiring a model populates one place, not three parallel seams.
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
