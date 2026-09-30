---
id: TASK-004
title: '[FEATURE] SDD acceptance spec for 004-hermes-memory (mnemonic)'
status: done
assignee: []
created_date: '2026-09-04'
updated_date: '2026-09-30 07:30'
labels: []
dependencies:
  - TASK-001
references:
  - .skillgrid/specs/2026-09-04-hermes-memory/acceptance.feature
  - .skillgrid/specs/2026-09-04-hermes-memory/briefing.md
  - .skillgrid/specs/2026-09-04-hermes-memory/tasks.md
  - .skillgrid/specs/2026-09-04-hermes-memory
documentation:
  - .skillgrid/specs/2026-09-04-hermes-memory/briefing.md
  - .agents/skills/_shared/conventions/sdd-structure.md
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Track the SDD **acceptance.feature** specs for change **004-hermes-memory** (Hermes Fact Memory & Agent Skills, **revised 4 steps**). Implementation waits on session-events-layer shipping + human choice between `sdd-apply` and `sdd-propose`.

**Current State:**
- Plan tracked by TASK-001 (revised 2026-09-29); `tasks.md` + `acceptance.feature` updated
- Per-step `acceptance.feature` rewritten for 4 steps (was 5); no production Fact Memory / Agent Skill code yet
- Blocker: session-events-layer must ship (QA → ship → archive) before implementation starts

**Expected State:**
- Steps 01-04 task checkboxes completed during `sdd-apply` against these acceptance scenarios
- Threat RED scenarios covered: path escape / unknown language (03); Mnemonic tool surface (02, 03); session_events integrity (02, 03)
- `go test ./...` passes for touched packages

**Key revisions from original:**
- 5→4 steps (sandbox merged into step 03)
- "Retrieval Trail" → "session_events" in all scenarios
- "fails closed" → "degrades gracefully" for vec0 absent
- Hybrid search moved to step 04 (was step 04, now still step 04 but with CLI)
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Acceptance features exist for steps 01-04 (filesystem + Engram sdd/004-hermes-memory/spec)
- [ ] #2 Each step has @happy / @edge / @failure; threat-matrix scenarios in 02, 03
- [ ] #3 Apply marks tasks [x] in dependency order (01 → 02|03 → 04)
- [ ] #4 Depends on session-events (040) + vector-db (042); go test ./... passes for touched packages
<!-- AC:END -->

## Definition of Done

<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 Every `@step-NN` Feature has passing @happy / @edge / @failure
- [ ] #7 Threat-matrix RED coverage for steps 02, 03, 04 passed
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Gate on session-events-layer shipping (QA → ship → archive) and TASK-001 plan readiness.
2. On Implement: run sdd-apply against step acceptance features in order 01 → 02|03 → 04.
3. Keep per-step acceptance.feature and Engram sdd/004-hermes-memory/spec aligned as scenarios pass.
4. Prioritize threat RED: path escape / unknown language (03); Mnemonic tool surface (02, 03); session_events integrity (02, 03).
5. Close when all AC scenarios + change DoD are green; archive with TASK-001.
<!-- SECTION:PLAN:END -->

## Comments

<!-- COMMENTS:BEGIN -->
created: 2026-09-21 07:13
---
2026-09-21: spec flagged STALLED (see .skillgrid/specs/2026-09-04-hermes-memory/tasks.md). Kept open for a future resume; not archived.
---

created: 2026-09-29 11:21
---
2026-09-29: Acceptance.feature rewritten for 4-step structure (was 5). Blocker confirmed: session-events-layer must ship first. All scenario names updated to match revised tasks.md.
---
<!-- COMMENTS:END -->

## Technical Notes

- Steps: `01-facts-schema`, `02-fact-tools`, `03-skills-registry`, `04-skill-execute-hybrid`, `05-commit-hooks-cli`
- Paths: `.skillgrid/specs/2026-09-04-hermes-memory/acceptance.feature`
- Spec: Engram `sdd/004-hermes-memory/spec`; carry-through plan ticket TASK-001
- Open: sqlite-vec on modernc (extension vs CGO) decided in step 01

## Priority

Medium — builds on 003; product urgency after 003 lands.

## Comments

<!-- Conversation appends here. -->

- 2026-09-04: Created for `force_ticket_creation` on sdd-spec (acceptance/tasks artifact). `backlog` CLI SIGILL/SIGTRAP on this host (Bun arch mismatch); ticket written as filesystem task file matching TASK-001 frontmatter. Duplicate-search: TASK-001 covers plan only; no prior acceptance-spec ticket for 004.
- 2026-09-05: Filled missing `type`, `references`, Definition of Done, and Implementation Plan via filesystem fallback (CLI still SIGILL on edit).
