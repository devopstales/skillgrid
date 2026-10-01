---
id: TASK-024
title: 'TICKET-03: DedupLLM.Classify 4-way + runDedupCheck switch'
status: in-progress
assignee: []
created_date: '2026-10-01 18:18'
updated_date: '2026-10-01 20:25'
labels:
  - bitemporal-audn
dependencies:
  - TASK-023
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
TICKET-03 of 2026-09-24-bitemporal-audn (blueprint Task 3, work unit 2).

Current State: the dedup path is binary — DedupLLM.Dedup returns a yes/no duplicate verdict; runDedupCheck maps it onto a single decision; there is no add/update/delete/noop distinction.

Expected State: DedupVerdict/DedupDecision types; Classify added to the DedupLLM seam (legacy Dedup kept deprecated); runDedupCheck switches to Classify returning (DedupDecision, string); Save adapted to the new shape until TICKET-04 rewrites it; isExtractedDuplicate stays binary.

Blocked by: TICKET-02 (task-023).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Fake Classify seam returning each of the 4 verdicts → runDedupCheck returns the decision unchanged (reason "llm")
- [ ] #2 Seam error → zero decision (reason "hash"), no panic
- [ ] #3 No seam → zero decision (reason "hash")
- [ ] #4 isExtractedDuplicate unchanged and still binary
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 SATISFIES scenarios green: Save returns an action label (classifier half); LLM error degrades to deterministic floor (seam half)
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. RED: extend memory/types_test.go with TestRunDedupCheckClassify (4 verdicts + seam error + no seam); confirm failing.
2. Add DedupVerdict/DedupDecision types + Classify to the DedupLLM interface in memory/types.go (~37); keep legacy Dedup marked deprecated.
3. Switch runDedupCheck (~127-156) to Classify returning (DedupDecision, string): seam ok → decision, reason "llm"; seam error or absent → zero decision, reason "hash".
4. Adapt Save in memory/service.go to the new runDedupCheck shape only (TICKET-04 rewrites it); isExtractedDuplicate (~282) stays binary.
5. GREEN: go test ./internal/mnemonic/memory/ -run 'TestRunDedupCheck|TestSave' -count=1.
6. Commit: feat(memory): DedupLLM.Classify 4-way + runDedupCheck switch. Refs: task-024.
<!-- SECTION:PLAN:END -->
