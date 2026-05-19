---
name: enforced-tdd-protocol
description: >
  Enforces test-driven development as a mandatory quality gate.
  All implementation must follow RED-GREEN-REFACTOR cycle with failing tests first.
  Trigger: Invoked automatically before any code generation or modification.
license: Apache-2.0
metadata:
  author: aiskillgrid-integration
  version: "1.0"
  enforcement: mandatory
  triggers:
    - "before_implementation"
    - "sdd-apply_start"
    - "task_execution"
---

# Enforced TDD Protocol

## Overview

This skill enforces the Test-Driven Development iron law across all implementation work. It is automatically invoked before any code generation or modification and acts as a mandatory quality gate.

**Core Principle:** NO PRODUCTION CODE WITHOUT A FAILING TEST FIRST

## When to Use

This skill triggers automatically:
- Before `sdd-apply` begins any task
- Before any subagent is dispatched to write code
- Before any manual code generation
- At the start of every implementation task

## Enforcement Rules

### 1. Pre-Implementation Check

For each task, verify in this order:

**Check A: Test file exists or will be created**
- If modifying existing code → corresponding test file must already exist
- If new functionality → test file must be part of the task's file list

**Check B: Test shows desired behavior**
- Test name must clearly describe the expected behavior
- Test must exercise the actual code (not just mocks)
- One assertion per test principle

**Check C: Test fails before implementation**
- Run the test and capture failure output
- Store failure evidence in task context
- Test must fail for the RIGHT reason (feature absent, not syntax error)

If any check fails → BLOCK execution and return:
```
TDD VIOLATION: Implement failing test first.
Required: Write test → Verify failure → Then implement.
```

### 2. RED-GREEN-REFACTOR Sequence Validation

Monitor task execution for correct sequence:

```
Phase 1 (RED):   Write failing test    → Capture failure evidence
Phase 2 (GREEN): Minimal implementation → Make test pass
Phase 3 (REFACTOR): Clean up only        → Keep tests green
```

**Invalid sequences detected:**
- Implementation code appears before test code → DELETE implementation, restart
- Test added after implementation → REJECT, start over
- Refactoring before green phase → STOP, complete green first

### 3. No-Placeholders Enforcement

TDD tasks must not contain placeholders:
- ❌ "Implement the actual logic later"
- ❌ "Add proper error handling"
- ❌ "Fill in the rest"
- ❌ "TBD", "TODO" in implementation sections

Every code block must be complete and runnable.

### 4. Test Quality Standards

Tests must demonstrate:
- **Clear intent** — name describes what's being tested
- **One behavior** — single assertion or tightly related assertions
- **Real code** — avoid over-mocking; test actual behavior
- **Edge cases** — error paths, boundary conditions
- **No test-only production code** — don't add methods solely for testing

### 5. Implementation Minimality Check

During GREEN phase, code review ensures:
- No features beyond test requirements (YAGNI)
- No "while I'm here" additions
- No premature optimization
- No unrelated refactoring

### 6. Refactor Discipline

REFACTOR phase restricted to:
- Remove duplication
- Improve names
- Extract helpers
- Keep behavior identical
- All tests must stay green

**No new behavior during refactor.**

## Task Tagging Requirements

Every TDD task must include:
```
[Label: AFK]  (if TDD execution is fully deterministic)
[Label: HITL] (if human decision needed on test design)
[TDD: required]
```

AFK tasks must have complete test specification upfront.

## Evidence Collection

For audit trail, capture:
1. Test code written (full content)
2. Test execution output (failure message)
3. Implementation code (full content)
4. Final test execution (green output)
5. Refactor changes (diff or summary)

Store as task artifact: `.agents/tasks/{task-id}/tdd-evidence.md`

## Violation Handling

**Transaction rollback approach:**
- If TDD sequence violated, implementation code is discarded
- Task marked `blocked` with violation details
- Implementer must restart task correctly
- Repeat violations → escalate to human

**Automatic cleanup:**
- Detect code written before test → identify affected files
- Revert those changes automatically when safe (feature branch, no unpushed shared commits)
- Restore clean state for restart

## Integration Points

**Invoked by:**
- `sdd-apply` pre-check routine
- `sequential-agent-executor` before task dispatch
**Cooperates with:**
- `granular-planning` — ensures tasks include TDD steps
- `spec-compliance-verifier` — uses tests as evidence
- `code-quality-reviewer` — reviews TDD adherence

## Configuration

`.skillgrid/config.json`:
```json
{
  "tdd": {
    "enforcement": "strict",
    "allow_hitl_tests": true,
    "evidence_retention_days": 30,
    "auto_cleanup_on_violation": true
  }
}
```

## See Also

- Superpowers reference: `skills/test-driven-development/SKILL.md` (source methodology)
- Related skill: `skillgrid-tdd` (existing TDD guidance)
- Enforcement: `.agents/skills/_shared/sdd-enforcement-contract.md`
