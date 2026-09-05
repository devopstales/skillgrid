---
id: TASK-002
title: '[FEATURE] SDD plan for 006-structured-session-handoff (mnemonic)'
status: ready-for-agent
assignee: []
created_date: '2026-09-04'
updated_date: '2026-09-05'
labels: []
dependencies: []
priority: medium
type: feature
references:
  - docs/skillgrid/changes/006-structured-session-handoff/change.md
  - docs/skillgrid/changes/006-structured-session-handoff/tasks.md
  - docs/skillgrid/changes/006-structured-session-handoff/acceptance.feature
  - docs/plan/06-structured-session-handof.md
  - skillgrid-cli/internal/mnemonic/
documentation:
  - docs/skillgrid/changes/006-structured-session-handoff/change.md
  - .agents/skills/_shared/conventions/sdd-structure.md
---

## Description

Track implementation of the approved SDD plan for change **006-structured-session-handoff** (Cleave-style **Session Relay**). Continuity across context fill / session end without Hermes facts/skills or 003 tiered/semantic/commit/trail cores.

**Current State:**
- Intent approved; plan authored at `docs/skillgrid/changes/006-structured-session-handoff/change.md`
- No session_handoff / `.cleave/` / `skillgrid session` surface yet

**Expected State:**
- Steps 01–05 implemented per plan (schema → handoff/resume → status/compact → CLI → optional watchdog)
- `go test ./...` passes for touched packages

## Acceptance Criteria

<!-- AC:BEGIN -->
- [ ] #1 Plan artifact exists (filesystem + Engram `sdd/006-structured-session-handoff/plan`)
- [ ] #2 Steps 01–05 deliver Session Relay per plan Step WHAT
- [ ] #3 Threat-matrix RED tests for new MCP session tools (02, 03)
- [ ] #4 Soft-after 003 L0 paths; `.cleave/` gitignored; `go test ./...` passes for touched packages
<!-- AC:END -->

## Definition of Done

<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 Change-level Definition of Done in `change.md` fully checked
- [ ] #7 `.skillgrid/.cleave/` gitignored by default
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Soft-after 003 L0 path conventions only — no hard SQL dep on `010_*`; use migration slot `012_session_relay.sql`.
2. Follow `docs/skillgrid/changes/006-structured-session-handoff/change.md` Step Blueprint: 01 schema → 02 handoff/resume → 03 status/compact → 04 CLI → 05 optional watchdog.
3. Drive each step from `steps/*/tasks.md` and `acceptance.feature`; RED threat tests for MCP session tools in 02 and 03 before production tools.
4. Keep relay deep module owning SQL + `.skillgrid/.cleave/` FS; MCP/CLI are thin callers. Do not pull in 004 Fact Memory.
5. Verify with `go test ./...` on touched packages; mark AC/DoD and archive when change DoD is green.
<!-- SECTION:PLAN:END -->

## Technical Notes

- Affected paths: `skillgrid-cli/internal/mnemonic/{store/migrations,relay,mcp}/`, `skillgrid-cli/cmd/skillgrid/{session,main}.go`, `.gitignore`
- Plan: `docs/skillgrid/changes/006-structured-session-handoff/change.md`
- Source proposal: `docs/plan/06-structured-session-handof.md`
- Migration slot: `012_session_relay.sql` (leaves `009`/`010`/`011` for 001/003/004)

## Priority

Medium–High after 003 paths; orthogonal to 004/005.

## Comments

<!-- Conversation appends here. -->

- 2026-09-04: Created via `backlog task create`. Status/priority/notes enriched via filesystem edit because `backlog task edit` SIGTRAPs on this host (same Bun arch mismatch as intermittent crashes). Duplicate-search: no prior 006/session-handoff tickets in `.backlog/tasks/`.
- 2026-09-05: Filled missing `type`, `references`, Definition of Done, and Implementation Plan via filesystem fallback (CLI still SIGILL on edit).
