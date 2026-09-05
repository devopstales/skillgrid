---
id: TASK-001
title: '[FEATURE] SDD plan for 004-hermes-memory (mnemonic)'
status: ready-for-agent
assignee: []
created_date: '2026-09-04'
updated_date: '2026-09-05'
labels: []
dependencies: []
priority: medium
type: feature
references:
  - docs/skillgrid/changes/004-hermes-memory/change.md
  - docs/skillgrid/changes/004-hermes-memory/tasks.md
  - docs/skillgrid/changes/004-hermes-memory/acceptance.feature
  - docs/plan/05-hermes-memory.md
  - skillgrid-cli/internal/mnemonic/
documentation:
  - docs/skillgrid/changes/004-hermes-memory/change.md
  - .agents/skills/_shared/conventions/sdd-structure.md
---

## Description

Track implementation of the approved SDD plan for change **004-hermes-memory** (Hermes Fact Memory & Agent Skills). Extends change **003** with a new Fact Memory store and Agent Skill registry; does not redo Tiered Storage.

**Current State:**
- Intent approved; plan authored at `docs/skillgrid/changes/004-hermes-memory/change.md` (and legacy `plan.md` if present)
- No Fact Memory / Agent Skill MCP tools or CLI yet

**Expected State:**
- Steps 01–05 implemented per plan (schema → fact tools → skills registry → sandboxed use_skill + hybrid → commit hooks + CLI)
- `go test ./...` passes for touched packages

## Acceptance Criteria

<!-- AC:BEGIN -->
- [ ] #1 Plan artifact exists (filesystem + Mnemonic `sdd/004-hermes-memory/plan`)
- [ ] #2 Steps 01–05 deliver Fact Memory + Agent Skills per plan Step WHAT
- [ ] #3 Threat-matrix RED tests for executable skills (04) and new MCP tools (02–04)
- [ ] #4 Depends on 003 `010_*` before applying `011_facts_skills.sql`
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
1. Confirm 003 `010_*` landed; reserve `011_facts_skills.sql` slot.
2. Follow `docs/skillgrid/changes/004-hermes-memory/change.md` Step Blueprint in order: 01 schema → 02 fact tools → 03 skills registry → 04 sandboxed `use_skill` + hybrid → 05 commit hooks + CLI.
3. Drive each step from `steps/*/tasks.md` and `acceptance.feature`; write RED threat tests before production tools for steps 02–04.
4. Keep Fact/Skill vectors on sqlite-vec Seam; do not redo 003 Tiered Storage / core semantic / trail surfaces.
5. Verify with `go test ./...` on touched packages; mark AC/DoD and archive when change DoD is green.
<!-- SECTION:PLAN:END -->

## Technical Notes

- Affected paths: `skillgrid-cli/internal/mnemonic/{store,facts,skills,vec,mcp}/`, `skillgrid-cli/cmd/skillgrid/{memory,skill,main}.go`
- Plan: `docs/skillgrid/changes/004-hermes-memory/change.md`
- Source proposal: `docs/plan/05-hermes-memory.md`

## Priority

Medium — builds on 003; Medium–High product urgency after 003 lands.

## Comments

<!-- Conversation appends here. -->

- 2026-09-04: Created via emergency filesystem fallback because `backlog` CLI (Bun 1.3.14 linux-x64) SIGILL/segfaults on this host. Reconcile with CLI once fixed (`backlog task create` / import). Duplicate-search: no prior hermes/004 tickets in `.backlog/tasks/`.
- 2026-09-05: Filled missing `type`, `references`, Definition of Done, and Implementation Plan via filesystem fallback (CLI still SIGILL on edit).
