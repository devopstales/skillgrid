---
id: TASK-113
title: 'TICKET-07: Skill Improvement Loop (Observe + Inspect + Amend + Evaluate)'
status: needs-triage
assignee: []
created_date: '2026-10-24 13:18'
updated_date: '2026-10-24 13:19'
labels:
  - mnemonic
  - global-memory
dependencies:
  - TASK-107
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
**Scope:** Migration 073 (skill_usage.outcome + feedback). C2 fix: `execute.go` logs failure/timeout rows. `skills.Inspect()` — aggregate usage, exact-match stderr grouping, top-N. `skills.Amend()` — LLM proposes patch → `<code_path>.amended`. `skills.Evaluate()` — before/after outcome rate, regression → discard, improvement → apply + git commit. Project-scoped.

**Acceptance:** Skill exit 1 → usage row with `outcome='failure'` + stderr. Timeout → `outcome='error'`. Inspect: 5 runs, 3 failures "timeout" → 40% success, "timeout" as top pattern. Amend: `<code_path>.amended` created, original unchanged. Evaluate regression: 80%→50% → discard, original restored. Evaluate improvement: 50%→80% → apply + commit.

**SATISFIES:** `happy path — skill failure is logged with outcome and stderr`, `happy path — inspect detects recurring failure pattern`, `happy path — amend writes a new file without modifying the original`, `edge — evaluate detects regression and rolls back`, `edge — evaluate detects improvement and applies`, `failure — skill timeout is logged as error outcome`

**Files:** `mnemonic/internal/store/migrations/073_skill_usage_outcome.sql` (modify: add outcome + feedback columns), `mnemonic/internal/skills/loop.go` (create), `mnemonic/internal/skills/loop_test.go` (create), `mnemonic/internal/skills/execute.go` (modify), `mnemonic/internal/skills/skills.go` (modify), `mnemonic/internal/mcp/tools_skills.go` (modify)

**Size:** ~700 (M)
**One-way door:** Yes (migration 073 + execute.go behavior change)
**Blocked by:** TICKET-01
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 skill exit 1 → usage row with outcome='failure' + stderr
- [ ] #2 timeout → outcome='error'
- [ ] #3 inspect: 5 runs, 3 failures → 40% success, top failure pattern listed
- [ ] #4 amend: <code_path>.amended created, original unchanged
- [ ] #5 evaluate regression: 80%→50% → discard, original restored
- [ ] #6 evaluate improvement: 50%→80% → apply + git commit
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
