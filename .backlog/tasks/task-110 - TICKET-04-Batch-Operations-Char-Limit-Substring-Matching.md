---
id: TASK-110
title: 'TICKET-04: Batch Operations + Char Limit + Substring Matching'
status: needs-triage
assignee: []
created_date: '2026-10-24 13:18'
updated_date: '2026-10-24 13:19'
labels:
  - mnemonic
  - global-memory
dependencies:
  - TASK-108
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
**Scope:** `facts.Batch()` — atomic operations array, char limit checked on final result only. `OverflowError` struct (current_entries + usage). Exact dedup. Substring matching for patch/remove (`old_text` param, ambiguous-match error, no-match with closest_entries). Char limits from `config.d/indexing.yaml` (memory=2200, user=1375).

**Acceptance:** Batch free 200 + add 250 at 2100/2200 → success (final 2150). Overflow at 2150/2200 + 100-char add → `*OverflowError` with entries + usage. Exact duplicate → "no duplicate added". Patch with `old_text` substring → updated. Ambiguous match → error. No match → error with closest_entries.

**SATISFIES:** `happy path — batch frees space and adds new fact`, `edge — overflow returns structured error with current entries`, `edge — exact duplicate is rejected`, `edge — patch uses substring matching`, `failure — ambiguous substring match returns error`, `failure — no-match substring returns error with closest entries`

**Files:** `mnemonic/internal/facts/batch.go` (create), `mnemonic/internal/facts/batch_test.go` (create), `mnemonic/internal/facts/facts.go` (modify), `mnemonic/internal/mcp/tools_facts.go` (modify)

**Size:** ~600 (M)
**Blocked by:** TICKET-02
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 batch free 200 + add 250 at 2100/2200 → success (final 2150)
- [ ] #2 overflow → *OverflowError with entries + usage string
- [ ] #3 exact duplicate → no duplicate added
- [ ] #4 patch with old_text substring → updated
- [ ] #5 ambiguous match → error
- [ ] #6 no match → error with closest_entries
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
