---
id: TASK-029
title: 'FOLLOWUP: shared LLM client attach for ask/extraction/dedup seams'
type: feature
status: needs-triage
assignee: []
created_date: '2026-10-02 07:14'
updated_date: '2026-10-03 07:39'
labels:
  - bitemporal-audn
  - followup
  - llm-wiring
dependencies: []
references:
  - .skillgrid/specs/2026-10-02-mnemonic-llm-provider/briefing.md
  - .skillgrid/specs/2026-09-24-bitemporal-audn/blueprint.md
documentation:
  - skillgrid-cli/internal/mnemonic/service/dedup_llm.go
  - skillgrid-cli/internal/mnemonic/service/service.go
priority: medium
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Deferred from TICKET-05 review (bitemporal-audn). No production LLM client is constructed anywhere in the codebase: SetAskLLM, EnableExtractionLLM, and the new dedupLLMFunc seam are all only populated in tests. When mnemonic.dedup.llm (or .extraction.llm / ask llm-mode) is on, the seam is armed and invoked but returns "no LLM configured" → hash floor.

**Owned by:** `.skillgrid/specs/2026-10-02-mnemonic-llm-provider` (draft). That change delivers one OpenAI-compatible Completer, `AttachSharedLLM` for ask + dedup + extraction + dream, opt-in feature flags, fail-open floors. Queue: after `2026-10-02-mnemonic-project-init`. Close this FOLLOWUP when that change ships — do not implement a parallel client here.
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 Closed by shipping `2026-10-02-mnemonic-llm-provider` (or marked superseded)
<!-- DOD:END -->
