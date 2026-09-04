---
name: requesting-code-review
description: Use after completing an SDD step that carries high-risk signals, before sdd-archive, or optionally when stuck / before refactor / after a complex bug fix. Dispatches a fresh code-reviewer sub-agent with crafted context — never your session history — and acts on the findings.
license: MIT
metadata:
  author: skillgrid
  version: "1.0"
  source: derived from superpowers requesting-code-review, adapted for skillgrid's Mnemonic + orchestrator model
---

# Requesting Code Review

Dispatch a code-reviewer sub-agent to catch issues before they cascade. The reviewer gets precisely crafted context for evaluation — never your session's history.

**Core principle:** review early, review often, but only when the risk justifies the cost.

## When to Request Review in skillgrid

**Mandatory (high-risk signals — any one triggers a review):**
- The change's `Review Workload Forecast` in `tasks.md` says `400-line budget risk: High` or `Chained PRs recommended: Yes`
- The `change.md` `## Threat Matrix` has any `Applicable` row
- This is the **last step before `sdd-archive`** (`### Verification` in `tasks.md` already passed; this is the second lens)
- The change touches any `_shared/conventions/*` file or any Mnemonic tool contract

**Optional but valuable:**
- When stuck (fresh perspective)
- Before refactoring (baseline check)
- After fixing a complex bug

**Skip:**
- Pure doc changes (vocabulary table edits, comment updates)
- Mechanical, one-line, low-blast-radius fixes where the test suite is the real reviewer
- Changes already covered by an exhaustive `sdd-verify` PASS WITH no applicable threat rows

## How to Request

### 1. Get git SHAs

```bash
BASE_SHA=$(git rev-parse HEAD~1)              # or origin/main, or the previous step's commit
HEAD_SHA=$(git rev-parse HEAD)
```

For multi-step changes, prefer per-step ranges — review what each step actually produced, not the cumulative pile.

### 2. Inject project standards (orchestrator responsibility)

Before dispatching, the orchestrator injects **pre-resolved compact rules** from `_shared/conventions/*` and the skill registry into the reviewer prompt as a `## Project Standards (auto-resolved)` block. The reviewer reads rules; it does not browse the registry.

### 3. Dispatch the reviewer sub-agent

Use the `general` sub-agent type. Fill the template at [references/code-reviewer.md](references/code-reviewer.md) with the placeholders:

- `{DESCRIPTION}` — brief summary of what was built
- `{PLAN_OR_REQUIREMENTS}` — the path to `change.md` or the per-step WHAT block inside it
- `{BASE_SHA}` / `{HEAD_SHA}` — the git range
- `## Project Standards (auto-resolved)` — the compact rules block

The reviewer is **read-only** and **self-contained**: it inspects via `git diff` / `git show` / `git log` and may check out other revisions into a temporary `git worktree add /tmp/review-<SHA> <SHA>` — but never mutates this checkout's HEAD, index, or working tree.

The reviewer **does not dispatch further sub-agents**. If the diff is too large for one pass, the reviewer does it in passes and says so.

### 4. Act on feedback

| Severity | Action |
|---|---|
| **Critical** | Fix immediately, before any further work. Push back only with file:line + test evidence. |
| **Important** | Fix before proceeding to the next step or to `sdd-archive`. |
| **Minor** | Note for later; do not block the change. |
| **Reviewer wrong** | Push back with technical reasoning; show code/tests that prove it works. |

**Push-back principle:** review is a tool, not an oracle. If the reviewer is wrong, say so with evidence (file:line + test result), not deference.

## Example

```text
[Just completed Step 02 of change 001-oauth-login]
[change.md said: "Implement refresh-token rotation atomically"]
[Workload forecast: 400-line budget risk: High — CHAIN trigger]

→ requesting-code-review fires (high-risk signal)

BASE_SHA=$(git rev-parse HEAD~1)   # = step 01 end
HEAD_SHA=$(git rev-parse HEAD)      # = step 02 end

[Dispatch general-purpose sub-agent with code-reviewer.md template]
  DESCRIPTION: Step 02 — refresh-token rotation in internal/auth/refresh.go
  PLAN_OR_REQUIREMENTS: docs/skillgrid/changes/001-oauth-login/change.md (Step 02 WHAT block)
  BASE_SHA: a7981ec
  HEAD_SHA: 3df7661
  ## Project Standards (auto-resolved)
    [compact rules from _shared/conventions/*]

[Sub-agent returns]
  Strengths: Atomic rotation via single SQL UPDATE; row-level lock acquired
  Issues:
    Critical: (none)
    Important: Missing test for the race where two rotations happen within the same millisecond
    Minor: Magic number (32) for the new token length
  Assessment: Ready to merge with fixes

[Fix the race-condition test → re-run sdd-verify → proceed to sdd-archive]
```

## Common Rationalizations

| Excuse | Reality |
|---|---|
| "I'll just review the diff myself instead of dispatching a reviewer" | You're the coordinator. Reviewing the diff inline burns the context window you need to keep driving the work. Dispatch a reviewer — the diff and the evaluation live in its context, only the findings come back. |
| "The reviewer needs my whole session history" | Hand precisely crafted context, never session history. The reviewer works the diff, not your thought process. |
| "This is too small to need review" | If the workload forecast flagged it, the budget risk is real. If it didn't, this skill is opt-out (see **Skip** above). |
| "I trust the test suite — that's the review" | The test suite proves behavior. The reviewer catches *what the tests don't ask*: design, contract drift, hidden coupling, dead code, vocabulary drift. They are different lenses. |

## Red Flags

**Never:**
- Skip review because "it's simple" when a high-risk signal fired
- Ignore Critical issues
- Proceed to `sdd-archive` with unfixed Important issues
- Argue with valid technical feedback

**If reviewer wrong:**
- Push back with technical reasoning
- Show code/tests that prove it works
- Request clarification

## Integration with skillgrid

- **`sdd-apply`** fires this skill automatically when the change-level `Review Workload Forecast` in `tasks.md` has any high-risk signal, OR when the assigned step carries an applicable threat-matrix row, OR when it is the last step before `sdd-archive`.
- **`sdd-verify`** (fills `### Verification` in `tasks.md`) is the *first* lens (acceptance-test driven). `requesting-code-review` is the *second* lens (`change.md`/quality driven). Both run; the order is `sdd-verify` first, then this skill on the high-risk subset.
- **`isolated-workspace`** is the up-front workspace-prep step; this skill runs at the close, not the start.

## References

- [references/code-reviewer.md](references/code-reviewer.md) — the full reviewer sub-agent prompt template (read-only, no sub-spawning, severity-anchored, verdict format).
- [../_shared/conventions/commits.md](../_shared/conventions/commits.md) — commit hygiene the reviewer will check (Conventional Commits, no AI trailers, one logical change per commit).
- [../_shared/conventions/mnemonic-memory.md](../_shared/conventions/mnemonic-memory.md) — `topic_key` isolation rules; the orchestrator injects them.
