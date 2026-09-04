---
id: TASK-002
title: '[FEATURE] SDD plan for 006-structured-session-handoff (mnemonic)'
status: ready-for-agent
assignee: []
created_date: '2026-09-04'
labels: []
dependencies: []
priority: medium
---

## Description

Track implementation of the approved SDD plan for change **006-structured-session-handoff** (Cleave-style **Session Relay**). Continuity across context fill / session end without Hermes facts/skills or 003 tiered/semantic/commit/trail cores.

**Current State:**
- Intent approved; plan authored at `docs/skillgrid/changes/006-structured-session-handoff/plan.md`
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

## Technical Notes

- Affected paths: `skillgrid-cli/internal/mnemonic/{store/migrations,relay,mcp}/`, `skillgrid-cli/cmd/skillgrid/{session,main}.go`, `.gitignore`
- Plan: `docs/skillgrid/changes/006-structured-session-handoff/plan.md`
- Source proposal: `docs/plan/06-structured-session-handof.md`
- Migration slot: `012_session_relay.sql` (leaves `009`/`010`/`011` for 001/003/004)

## Priority

Medium–High after 003 paths; orthogonal to 004/005.

## Comments

<!-- Conversation appends here. -->

- 2026-09-04: Created via `backlog task create`. Status/priority/notes enriched via filesystem edit because `backlog task edit` SIGTRAPs on this host (same Bun arch mismatch as intermittent crashes). Duplicate-search: no prior 006/session-handoff tickets in `.backlog/tasks/`.
