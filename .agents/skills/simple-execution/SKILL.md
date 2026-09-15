---
name: simple-execution
description: Use when you have a written implementation plan to execute in a separate session with review checkpoints
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: superpowers:executing-plans
---

# Simple Execution

## Overview

Load plan, review critically, execute all tasks, report when complete.

**Announce at start:** "I'm using the skillgrid:simple-execution skill to implement this plan."

**Config:** Read `.skillgrid/config.yaml` before starting. Use `testing.runner` for the test command, `commands.build` for build, `commands.lint` for lint. If `ticketing.enabled: true` AND a `tasks.md` with tracker IDs exists (produced by `skillgrid:ticketing`), update ticket status at each transition: `ready` (wave start) → `in-progress` (ticket started) → `review` (verification passed) → `done` (review clean). If the file doesn't exist, use the verifications specified in the plan.

**Note:** This skill works much better with access to subagents (Claude Code, Codex CLI, Codex App, Copilot CLI, and Gemini CLI all qualify; see the per-platform tool refs in `../using-skillgrid/references/`). If subagents are available, use skillgrid:subagent-execution instead of this skill.

## When to Use

- You have a written implementation plan (or `tasks.md` from `skillgrid:slicing`) to execute in a separate session
- You need review checkpoints and a QA gate before integration
- You are on a platform without subagent access, so execution is inline

**When NOT to use:** subagents are available (use skillgrid:subagent-execution), or there is no written plan yet (use skillgrid:brainstorming / skillgrid:planning first).

## The Process

### Step 1: Load and Review Plan
1. Ensure an isolated workspace: use skillgrid:isolated-workspace to create one or verify the existing one
2. Read plan file. If a `tasks.md` exists alongside the plan (produced by `skillgrid:slicing`), read it instead: tickets replace tasks, and execution follows the wave order.
3. Review critically - identify any questions or concerns about the plan
4. If concerns: Raise them with the user before starting
5. If no concerns: Create todos for the plan items (or tickets, if sliced) and proceed

### Step 2: Execute Tasks

For each task (or ticket, if sliced):
1. Mark as in_progress
2. Follow each step exactly (plan has bite-sized steps; ticket has scope + acceptance criteria)
2.5. **Lazy check** (per `skillgrid:ponytail`, fully active): before writing code, climb the ladder once — does this need to exist at all (YAGNI), is it already in the codebase, does stdlib or a native platform feature cover it, is it one line? Take the first rung that holds; ship the laziest working version and mark any real corner cut with a `ponytail:` ceiling comment. Never simplify away input validation at trust boundaries, data-loss error handling, security, accessibility, or anything the spec/blueprint explicitly requests.
3. **Gates first:** before writing code, ensure the `#### Gates` block for this task's requirements in `acceptance.feature` is complete — every happy-path scenario has a `G<n>` (runnable `CHECK:`+`EXPECT:`, or `manual`/`ABANDON` with a reason). Author any missing gate now; an `ABANDON` becomes a handoff you surface, not a pass. (Per `skillgrid:test-driven-verification`.)
4. **TDD is always on** (`testing.tdd: true`): for implementation work, follow skillgrid:test-driven-development — write the failing test first, watch it fail, then write the minimal code to pass. Never start implementation without a failing test. **BDD is always on** — the failing test is the task's `SATISFIES` scenario: confirm the acceptance scenario is RED before writing code, and name TDD Evidence after the scenario.
5. Run verifications as specified
6. **Zone rule (BDD is always on):** edit `.skillgrid/specs/` *or* code in a single commit — never both uncommitted. Commit spec changes before code changes; the spec is the contract, the code satisfies it. The `pre-commit` zone guard (skillgrid:work-unit-commits) blocks a commit that stages both.
7. **Commit the task's changes** per `skillgrid:work-unit-commits` — conventional subject (no AI-attribution trailer), atomic + independently-revertable, a `[skillgrid-context]` block in the body, then `checkpoint-state.sh snapshot`. Commit *after* the task's gate is green, *before* any long install/build. The git hooks (`pre-commit`, `commit-msg`) enforce the guards; each commit is a checkpoint you can roll back to.
8. Mark as completed

### Step 2.5: QA Gate

After all tasks are complete, run `skillgrid:qa`. It produces the test plan, goal-backward verification, verification-gap audit, traceability matrix, and TDD evidence audit, and renders the four-state gate (PASS / CONCERNS / FAIL / WAIVED).

- **PASS** → proceed to Step 2.6.
- **CONCERNS** → fix the in-scope open items with tests, re-run QA (re-verification mode), then proceed.
- **FAIL** → fix the CRITICAL findings with a failing test first, re-run QA. Cap at 3 rounds, then escalate to the human.
- **WAIVED** → record the waiver, proceed to Step 2.6.

### Step 2.6: Request Review and Fix Findings

After the QA gate passes:
1. Request code review (follow `skillgrid:requesting-code-review`)
2. On findings: follow `skillgrid:receiving-code-review` (triage → fix →
   validate → log)
3. Loop until review is clean or capped at 3 rounds
4. Update ticket status → `done` for clean tickets (if `ticketing.enabled`)

### Step 3: Complete Development

After all tasks complete and verified:
- Announce: "I'm using the skillgrid:ship skill to integrate and close this change."
- **REQUIRED SUB-SKILL:** Use skillgrid:ship
- Follow that skill to verify tests, present options, execute choice

## When to Stop and Ask for Help

**STOP executing immediately when:**
- Hit a blocker (missing dependency, test fails, instruction unclear)
- Plan has critical gaps preventing starting
- You don't understand an instruction
- Verification fails repeatedly

**Ask for clarification rather than guessing.**

## When to Revisit Earlier Steps

**Return to Review (Step 1) when:**
- Partner updates the plan based on your feedback
- Fundamental approach needs rethinking

**Don't force through blockers** - stop and ask.

## Remember
- Review plan critically first
- Follow plan steps exactly
- Don't skip verifications
- Commit after every task per skillgrid:work-unit-commits — commits are your checkpoints
- Reference skills when plan says to
- Stop when blocked, don't guess
- Never start implementation on main/master branch without explicit user consent
- If tasks.md exists, execute tickets in wave order, not file order

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "The plan is clear enough, skip the critical review" | Step 1 says review critically and raise concerns before starting. A gap caught in review is one line; caught in execution it is a rework of tasks that depended on it. |
| "I'll run the tests at the end" | `testing.tdd: true` means a failing test exists before code. Verifications are per-task (step 2.5), so a red gate at the end can't tell you which task broke. |
| "This task is small, no need for a gate" | Step 2.4 requires a `G<n>` gate per requirement before code. Small tasks are where a missing gate silently skips the acceptance scenario. |
| "I'll fix the commit zone guard later" | The zone rule is one commit each for `.skillgrid/specs/` and code; the `pre-commit` hook blocks a commit that stages both. "Later" means a commit that was already wrong. |
| "The review found minor issues, ship anyway" | Step 2.6 caps at 3 rounds, but a review only closes clean or escalated. Shipping with open findings skips the gate that exists to catch them. |

## Red Flags

- Implementation started before the acceptance scenario was confirmed RED
- A commit stages both `.skillgrid/specs/` and code (zone guard violation)
- Task marked completed with no captured test output or exit code
- QA gate hit FAIL and the fix was applied without a failing test first
- Ticket status jumped past `review` to `done` without a clean review
- Work started on main/master without explicit user consent

## Verification

- [ ] Plan (or `tasks.md`) was read and critically reviewed; concerns raised and resolved before execution started
- [ ] Every task's gate is green: `CHECK:`+`EXPECT:` ran (or `manual` executed, `ABANDON` surfaced as a handoff) with captured output
- [ ] Test runner from `testing.runner` exits 0; build from `commands.build` and lint from `commands.lint` are clean
- [ ] QA gate (step 2.5) returned PASS or WAIVED with the waiver recorded; FAIL findings were fixed with a failing test first
- [ ] Code review is clean or capped at 3 rounds; open findings logged
- [ ] If `ticketing.enabled: true`, every ticket's status reflects its real state (`ready` → `in-progress` → `review` → `done`)
