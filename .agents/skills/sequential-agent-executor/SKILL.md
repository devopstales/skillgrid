---
name: sequential-agent-executor
description: >
  Executes granular implementation plans by dispatching fresh subagents per task
  with two-stage review (spec compliance, then code quality) after each.
  Continuous execution without human checkpoints between tasks.
  Trigger: Invoked by sdd-apply after plan validation and TDD pre-checks.
license: Apache-2.0
metadata:
  author: aiskillgrid-integration
  version: "1.0"
  dependencies:
    - "enforced-tdd-protocol"
  mode: orchestrator
  triggers:
    - "execution_phase"
    - "sdd-apply_body"
---

# Sequential Agent Executor

## Overview

Orchestrates task execution by dispatching fresh subagents per task, performing two-stage review after each, and maintaining continuous execution until all tasks complete or a blocker is hit.

**Core principle:** One fresh subagent per task + two-stage review = high quality, fast iteration

## When to Use

Invoked automatically:
- After `sdd-apply` completes pre-checks (plan exists, branch ready, TDD enforced)
- When `granular-planning` produces sufficient task detail
- For sequential execution of tasks in a single change slice

Not used when:
- Manual execution requested (operator controls)
- Parallel execution across independent slices (use `parallel-delegate` with separate branches per lane instead)

## Execution Pipeline

### Phase 0: Setup

```
1. Read tasks file (openspec/changes/<id>/tasks.md or equivalent)
2. Parse into atomic task objects (id, title, files, phase, TDD tag)
3. Create TodoWrite tracker with all tasks
4. Mark task 1 as current
5. Record starting SHA for rollback capability
6. Verify git state (clean working tree, on feature branch, tests passing)
```

### Phase 1: Per-Task Dispatch Loop

For each task in order:

**Step A: Prepare Task Context**
- Extract full task text
- Include slice spec context (acceptance criteria)
- Include design decisions that affect implementation
- Include relevant existing code patterns (read from filesystem)
- Provide TDD phase expectations

**Step B: Dispatch Implementer Subagent**

```
Subagent role: Implementer
Model tier: Fast model for simple tasks, standard for integration
Context:
  - Task specification (complete)
  - Slice spec (bounded context)
  - Current state of files on the branch
  - TDD protocol requirements (enforced-tdd-protocol reference)
  - Any relevant existing code patterns

Instructions:
  1. Review task scope and acceptance criteria
  2. If TDD: RED phase → write failing test FIRST, run it, capture output
  3. Implement minimal code to satisfy test
  4. Run tests → GREEN confirmed
  5. Refactor without changing behavior
  6. Self-review against task specification
  7. Commit with conventional message
  8. Mark task DONE or DONE_WITH_CONCERNS

Expected output: implementation summary + git SHAs + self-review pass/fail
```

**Step C: Capture Implementer Result**

Implementer reports one of:
- `DONE` — task complete, ready for review
- `DONE_WITH_CONCERNS` — complete but flagged doubts (read before proceeding)
- `NEEDS_CONTEXT` — needs additional information (provide before proceeding)
- `BLOCKED` — cannot proceed (assess: missing context → provide; too large → split; wrong plan → escalate)

**Step D: Spec Compliance Review (Stage 1)**

```
Subagent role: Spec Compliance Reviewer
Model tier: Standard model (knows spec requirements)
Context:
  - Slice spec (acceptance criteria, scope, out-of-scope)
  - Implemented code (full content)
  - Tests written
  - Task specification

Checklist:
  ✅ All task objectives satisfied?
  ✅ Tests exercise required scenarios?
  ✅ No out-of-scope features added?
  ✅ Edge cases handled as per spec?
  ✅ Interface matches specification exactly?
  ✅ No hardcoded test values leaking to production?

If any ❌: task FAILS compliance → return to implementer for fixes
If ✅: proceed to Stage 2
```

**Loop:** Implementer fixes → Spec reviewer re-reviews → repeat until compliance PASS

**Step E: Code Quality Review (Stage 2)**

```
Subagent role: Code Quality Reviewer
Model tier: Capable model (style, patterns, architecture)
Context:
  - Implemented code
  - Existing codebase patterns
  - Lint/style configuration (if present)

Checklist:
  ✅ Readability: clear names, no magic numbers
  ✅ DRY: no duplication
  ✅ Error handling: explicit, logged, user-friendly
  ✅ Test quality: realistic, not over-mocked, edge cases
  ✅ Security: no secrets in code, input validation
  ✅ Performance: O(n) not O(n²) without reason
  ✅ Consistency: follows project conventions

Severity levels:
  CRITICAL: blocks merge (security flaw, data loss, crash)
  IMPORTANT: should fix before merge (readability, DRY violations)
  MINOR: optional polish (naming tweaks, comment clarity)

If CRITICAL/IMPORTANT issues found → task FAILS quality → implementer fixes → re-review
If only MINOR: record but allow task completion
```

**Step F: Advance Task**

- Mark task ✅ in TodoWrite
- Store artifacts:
  - `tdd-evidence/` — test output, code diffs
  - `spec-compliance-report/` — reviewer findings
  - `quality-review/` — issues and resolutions
- Record git SHAs (before/after) for traceability
- Proceed to next task

### Phase 2: Continuous Execution

Execute ALL tasks in sequence without human intervention unless:
- Task reports `BLOCKED` and cannot self-resolve
- Spec or quality review fails 3 times for same task (escalation threshold)
- User interrupt requested (Ctrl+C) — saves state and exits cleanly

No progress summaries needed — run until done or stuck.

### Phase 3: Final Integration Review

After last task:

```
Dispatch final reviewer subagent (cross-slice integration check):
  - All specifications satisfied?
  - No conflicts between slices?
  - End-to-end flow works?
  - Test suite fully green?
  - No leftover debug code?

Output: Final gate PASS/FAIL with remediation
```

## Subagent Prompt Templates

**Implementer prompt** (`sequential-agent-executor/prompts/implementer.md`):
```
You are implementing task #{task-id}: {title}

Context:
{slice-spec}
{design-decisions}
{existing-code-patterns}

TDD phase: {RED|GREEN|REFACTOR}

Instructions:
1. If RED: write failing test first, run it, capture output
2. If GREEN: write minimal code to pass
3. If REFACTOR: clean up, keep behavior identical
4. Self-review before marking complete

Proceed.
```

**Spec Compliance Reviewer** (`sequential-agent-executor/prompts/spec-reviewer.md`):
```
Review implementation against spec:

Spec requirement IDs: {req-ids}
Implemented: {files-changed}
Tests: {test-files}

Check each requirement has traceable evidence.
Report: PASS or FAIL with missing/extra items.
```

**Code Quality Reviewer** (`sequential-agent-executor/prompts/quality-reviewer.md`):
```
Code quality assessment for files: {files}

Evaluate: style, duplication, error handling, test quality, security.

Report:
CRITICAL: [list or none]
IMPORTANT: [list or none]
MINOR: [list or none]

Strengths: [positive notes]

Recommendation: APPROVED or CHANGES_REQUESTED
```

## State Persistence

During execution:
```
agents/tasks/{change-id}/execution-state.json
{
  "current_task_index": 3,
  "completed_tasks": ["1.1", "1.2", "2.1"],
  "last_good_commit": "abc123",
  "failures": [
    {"task": "2.2", "issue": "spec mismatch", "fixed": true}
  ]
}
```

On completion or fatal error, update `sdd/apply-status` in engram.

## Error Handling

**Implementer reports `BLOCKED`:**
- If missing context → provide and re-dispatch
- If task too large → split task, create new tasks.md entry, continue
- If plan wrong → escalate to human with clear description

**Reviewer reports `FAIL` twice on same task:**
- Escalate to `sdd-review` with human-in-loop
- Mark task `[HITL]` retroactively

**Unrecoverable git state** (detached HEAD, merge conflict mid-run, corrupted branch):
- Abort execution
- Return to last known-good commit or base branch
- Notify user with recovery instructions

## Cost Optimization

- Use fast model for RED/GREEN tasks on existing, well-specified code
- Use standard model for integration and REFACTOR
- Use capable model for complex domain logic or architecture-sensitive changes
- Cached context: provide same spec/design references to avoid re-reading

## Configuration

`.skillgrid/config.json`:
```json
{
  "executor": {
    "default_model_tier": "fast",
    "max_retries_per_task": 2,
    "stop_on_first_failure": false,
    "enable_final_integration_review": true,
    "parallel_slices": false
  }
}
```

## Integration With Other Skills

Invokes:
- `enforced-tdd-protocol` — pre-task TDD validation
- `spec-compliance-verifier` — stage 1 review (reuses skill directly)
- `code-quality-reviewer`  — stage 2 review (reuses skill directly)

Coordinates with:
- `pre-merge-verification` — receives pass/fail from final integration review

## See Also

- Superpowers reference: `skills/subagent-driven-development/SKILL.md` (source methodology)
- Alternative: `batch-executor` for checkpointed execution
- Related: `parallel-delegate` for parallel slice execution when lanes are independent
