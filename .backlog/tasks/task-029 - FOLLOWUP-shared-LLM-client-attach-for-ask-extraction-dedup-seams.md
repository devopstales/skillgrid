---
id: TASK-029
title: 'FOLLOWUP: shared LLM client attach for ask/extraction/dedup seams'
type: feature
status: done
assignee: []
created_date: '2026-10-02 07:14'
updated_date: '2026-10-05 20:15'
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

**Owned by:** `.skillgrid/specs/2026-10-02-mnemonic-llm-provider`. That change delivers one OpenAI-compatible Completer, `AttachSharedLLM` for ask + dedup + extraction + dream, opt-in feature flags, fail-open floors. Queue: after `2026-10-02-mnemonic-project-init`. Close this FOLLOWUP when that change ships — do not implement a parallel client here.

**Status: DONE — superseded by and closed via `2026-10-02-mnemonic-llm-provider`.** The shared OpenAI-compatible Completer (`internal/mnemonic/llm`) and the single boot attach (`service.AttachSharedLLM` → `SetAskLLM` + `SetDedupLLMFunc` + `mem.SetExtractionLLM`, with the Dream adapter built for the lazy distill path) now populate these seams in production when `mnemonic.llm.enabled` is set, so the "no LLM configured → floor" gap is closed. No parallel client was implemented. See ADR-0023.
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [x] #1 Tests pass (`go test ./...` for touched packages)
- [x] #2 Lint and formatting pass
- [x] #3 Edge cases covered
- [x] #4 No new warnings introduced
- [x] #5 Spec/docs updated if behavior changes
- [x] #6 Closed by shipping `2026-10-02-mnemonic-llm-provider` (or marked superseded)
<!-- DOD:END -->
