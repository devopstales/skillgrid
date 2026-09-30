---
id: TASK-001
title: '[FEATURE] SDD plan for 004-hermes-memory (mnemonic)'
status: done
assignee: []
created_date: '2026-09-04'
updated_date: '2026-09-30 07:30'
labels: []
dependencies: []
references:
  - .skillgrid/specs/2026-09-04-hermes-memory/briefing.md
  - .skillgrid/specs/2026-09-04-hermes-memory/tasks.md
  - .skillgrid/specs/2026-09-04-hermes-memory/acceptance.feature
  - docs/plan/05-hermes-memory.md
  - skillgrid-cli/internal/mnemonic/
documentation:
  - .skillgrid/specs/2026-09-04-hermes-memory/briefing.md
  - .agents/skills/_shared/conventions/sdd-structure.md
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Track implementation of the **revised** SDD plan for change **004-hermes-memory** (Hermes Fact Memory & Agent Skills). Revised 2026-09-29 to reuse 014 AKL importance, session-events-layer (migration 040), vector-db (migration 042), and 014 skills framework. Scope reduced from ~1600-2200 to ~800-1200 lines. Steps reduced from 5 to 4.

**Current State:**
- Plan revised at `.skillgrid/specs/2026-09-04-hermes-memory/briefing.md` (revised 2026-09-29)
- No Fact Memory / Agent Skill MCP tools or CLI yet
- Dependencies: session-events-layer (migration 040) must ship first; vector-db (042) shipped

**Expected State:**
- Steps 01-04 implemented per revised plan (schema → fact tools → skills registry+execute → commit hooks + hybrid + CLI)
- `go test ./...` passes for touched packages

**Key revisions from original:**
- No separate `vec/` Seam — reuses `vectorstore/` + `hybrid.Rank`
- No `forgetting_events` table — reuses 014 AKL importance + Dream Executor
- Retrieval trails → `session_events` (migration 040), not old `retrieval_trails`
- Skills extend 014 `memory_type="skill"` observation framework
- Sandbox (use_skill) merged into step 03 with registry
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Plan artifact exists and is revised (briefing.md + tasks.md + acceptance.feature)
- [ ] #2 Steps 01-04 deliver Fact Memory + Agent Skills per revised plan Step WHAT
- [ ] #3 Threat-matrix RED tests for executable skills (03) and new MCP tools (02, 03)
- [ ] #4 Depends on session-events (040) + vector-db (042) before applying 011_facts_skills.sql
- [ ] #5 Reuses 014 importance, vectorstore, hybrid.Rank — no reimplementation
<!-- AC:END -->

## Definition of Done

<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 Change-level Definition of Done in `change.md` fully checked
- [ ] #7 Soft-after 003 `010_*` migration order respected
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Confirm session-events-layer (migration 040) + vector-db (042) shipped; reserve `011_facts_skills.sql` slot.
2. Follow `.skillgrid/specs/2026-09-04-hermes-memory/briefing.md` Step Blueprint in order: 01 facts+skills schema → 02 fact MCP tools → 03 skills registry + sandboxed execute → 04 commit hooks + hybrid + CLI.
3. Drive each step from `tasks.md` and `acceptance.feature`; write RED threat tests before production tools for steps 02-03.
4. Reuse 014 importance (AKL), `vectorstore/` (vec0), `hybrid.Rank` (RRF), `session_events` (trails); do NOT re-implement.
5. Keep skills compatible with 014 `MatchSkills` (dual path: intent-match via observations + explicit registry via FS+SQL).
6. Verify with `go test ./...` on touched packages; mark AC/DoD and archive when change DoD is green.
<!-- SECTION:PLAN:END -->

## Comments

<!-- COMMENTS:BEGIN -->
created: 2026-09-21 07:13
---
2026-09-21: spec flagged STALLED (see .skillgrid/specs/2026-09-04-hermes-memory/tasks.md). 0/5 steps implemented since Sep 4 — importance/dream exist via 014, but the Fact Memory store + Agent Skills sandbox are unimplemented. Ticket kept open (ready-for-agent) for a future resume; not archived.
---

created: 2026-09-29 11:20
---
2026-09-29: Plan REVISED to account for shipped changes. Key changes: (1) 5→4 steps (sandbox merged into step 03), (2) reuse 014 AKL importance instead of new forgetting_events table, (3) reuse vectorstore/ + hybrid.Rank instead of separate vec/ Seam, (4) retrieval trails → session_events (migration 040), (5) skills extend 014 memory_type=skill framework, (6) scope reduced ~800-1200 lines, (7) hard dependency on session-events-layer shipping. All three spec files (briefing, tasks, acceptance) updated.
---
<!-- COMMENTS:END -->

## Technical Notes

- Affected paths: `skillgrid-cli/internal/mnemonic/{store,facts,skills,vec,mcp}/`, `skillgrid-cli/cmd/skillgrid/{memory,skill,main}.go`
- Plan: `.skillgrid/specs/2026-09-04-hermes-memory/briefing.md`
- Source proposal: `docs/plan/05-hermes-memory.md`

## Priority

Medium — builds on 003; Medium–High product urgency after 003 lands.

## Comments

<!-- Conversation appends here. -->

- 2026-09-04: Created via emergency filesystem fallback because `backlog` CLI (Bun 1.3.14 linux-x64) SIGILL/segfaults on this host. Reconcile with CLI once fixed (`backlog task create` / import). Duplicate-search: no prior hermes/004 tickets in `.backlog/tasks/`.
- 2026-09-05: Filled missing `type`, `references`, Definition of Done, and Implementation Plan via filesystem fallback (CLI still SIGILL on edit).
