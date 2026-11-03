---
id: TASK-114
title: 'TICKET-08: CLI Surface + ADR-0049 + Threat-Matrix RED Tests'
status: needs-triage
assignee: []
created_date: '2026-10-24 13:18'
updated_date: '2026-10-24 13:19'
labels:
  - mnemonic
  - global-memory
dependencies:
  - TASK-111
  - TASK-112
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
**Scope:** CLI `--scope`, `--target`, `pending/approve/reject`, `prefetch` subcommands. ADR-0049 (Context/Decision/Consequences, references ADR-0012/0048/0023/0045/0031). Threat-matrix RED tests: isolation, overflow, fused, injection shape, batch atomicity, trail FK, migration.

**Acceptance:** `memory fact add --scope global --target memory` → row in global store. `memory global pending` → lists pending facts. `memory global approve all` → in store. `memory global prefetch` → snapshot format. `--scope invalid` → non-zero exit + usage error. ADR file exists with correct format. All threat-matrix RED tests pass.

**SATISFIES:** `happy path — CLI adds a global fact with scope and target`, `happy path — CLI lists and approves pending facts`, `edge — CLI prefetch prints the frozen snapshot`, `failure — CLI with invalid scope returns usage error`

**Files:** `mnemonic/cmd/mnemonic/memory.go` (modify), `mnemonic/internal/facts/facts_test.go` (modify: threat-matrix tests), `.skillgrid/decisions/04-adr-0049-global-operational-memory.md` (create)

**Size:** ~400 (S)
**Blocked by:** TICKET-05, TICKET-06
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 memory fact add --scope global --target memory → row in global store
- [ ] #2 memory global pending → lists pending facts
- [ ] #3 memory global approve all → in store
- [ ] #4 memory global prefetch → snapshot format
- [ ] #5 --scope invalid → non-zero exit + usage error
- [ ] #6 ADR-0049 file exists with correct format
- [ ] #7 all threat-matrix RED tests pass
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
