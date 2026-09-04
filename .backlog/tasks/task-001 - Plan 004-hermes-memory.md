---
id: task-001
title: '[FEATURE] SDD plan for 004-hermes-memory (mnemonic)'
status: ready-for-agent
assignee: []
created_date: '2026-09-04'
labels: []
dependencies: []
priority: medium
---

## Description

Track implementation of the approved SDD plan for change **004-hermes-memory** (Hermes Fact Memory & Agent Skills). Extends change **003** with a new Fact Memory store and Agent Skill registry; does not redo Tiered Storage.

**Current State:**
- Intent approved; plan authored at `docs/skillgrid/changes/004-hermes-memory/plan.md`
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

## Technical Notes

- Affected paths: `skillgrid-cli/internal/mnemonic/{store,facts,skills,vec,mcp}/`, `skillgrid-cli/cmd/skillgrid/{memory,skill,main}.go`
- Plan: `docs/skillgrid/changes/004-hermes-memory/plan.md`
- Source proposal: `docs/plan/05-hermes-memory.md`

## Priority

Medium — builds on 003; Medium–High product urgency after 003 lands.

## Comments

<!-- Conversation appends here. -->

- 2026-09-04: Created via emergency filesystem fallback because `backlog` CLI (Bun 1.3.14 linux-x64) SIGILL/segfaults on this host. Reconcile with CLI once fixed (`backlog task create` / import). Duplicate-search: no prior hermes/004 tickets in `.backlog/tasks/`.
