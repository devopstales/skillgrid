---
id: TASK-026
title: 'TICKET-05: Config mnemonic.dedup.llm + newDedupLLMBackend wiring'
status: done
assignee: []
created_date: '2026-10-01 18:21'
updated_date: '2026-10-02 07:14'
labels:
  - bitemporal-audn
dependencies:
  - TASK-025
references:
  - .skillgrid/specs/2026-09-24-bitemporal-audn/briefing.md
  - .skillgrid/specs/2026-09-24-bitemporal-audn/tasks.md
  - .skillgrid/specs/2026-09-24-bitemporal-audn/acceptance.feature
  - .skillgrid/artifacts/04-adr-0011-observations-are-bitemporal.md
documentation:
  - .skillgrid/specs/2026-09-24-bitemporal-audn/blueprint.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
TICKET-05 of 2026-09-24-bitemporal-audn (blueprint Task 5, work unit 3).

Current State: the 4-way classifier seam from TICKET-03 exists but nothing arms it — there is no config knob and no backend wiring, so Classify is unreachable in production.

Expected State: Dedup config section (mnemonic.dedup.llm, default false) in config/load.go mirroring Extraction; newDedupLLMBackend() in service/service.go (wraps the extraction LLM client, AUDN prompt) + if cfg.Dedup.LLM { ... } wiring next to EnableExtractionLLM.

Blocked by: TICKET-04 (task-025).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Load with no dedup section → Dedup.LLM == false
- [ ] #2 With mnemonic.dedup.llm: true → Dedup.LLM == true
- [ ] #3 With the seam armed the Classify path is reachable end-to-end
- [ ] #4 config + service packages green
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 SATISFIES scenarios green: LLM error degrades to deterministic floor (wiring that arms the seam); Save returns an action label (LLM path reachable)
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. RED: extend config/load_test.go with TestDedupLLMDefaultFalse + TestDedupLLMEnabled; confirm failing.
2. Add the Dedup config section (mnemonic.dedup.llm, default false) in config/load.go mirroring the Extraction section (~66/221/453).
3. Add newDedupLLMBackend() in service/service.go wrapping the extraction LLM client with the AUDN prompt; wire if cfg.Dedup.LLM { ... } next to EnableExtractionLLM (~411).
4. GREEN: go test ./internal/mnemonic/config/ ./internal/mnemonic/service/ -count=1.
5. Commit: feat(memory): mnemonic.dedup.llm config + newDedupLLMBackend wiring. Refs: task-026.
<!-- SECTION:PLAN:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
mnemonic.dedup.llm config section (default false) mirroring Extraction at all touchpoints; newDedupLLMBackend() adapter in service/dedup_llm.go satisfying memory.DedupLLM (4-way Classify + deprecated binary Dedup), injectable dedupLLMFunc seam; openProject arms SetDedupLLM+EnableDedupLLM(true) when cfg.Dedup.LLM. Candidate *int distinguishes 0 (first) from null. Review: PASS; 2 deferred WARNINGs (no production LLM client is constructed anywhere in the codebase — pre-existing across ask/extraction/dedup — so the seam is structurally reachable but functionally inert until a shared client attach step lands; the parallel seam mirrors the existing SetAskLLM convention). 2 SUGGESTIONS fixed (prompt clarity, add/noop candidate-drop pin). Commits: f06f2a8c + 1a7c5b1d.
<!-- SECTION:FINAL_SUMMARY:END -->
