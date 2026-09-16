---
name: receiving-code-review
description: Use when receiving code review feedback, before implementing suggestions, especially if feedback seems unclear or technically questionable - requires technical rigor and verification, not performative agreement or blind implementation
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: superpowers:receiving-code-review
---

# Receiving Code Review

**Announce at start:** "I'm using the skillgrid:receiving-code-review skill to process this feedback."

## Overview

Code review requires technical evaluation, not emotional performance.

**Core principle:** Verify before implementing. Ask before assuming. Technical correctness over social comfort.

## When to Use

- You receive code review feedback (user, reviewer, or automated) and are about to act on it
- A comment is unclear, technically questionable, or conflicts with a prior decision
- A review produces multiple findings to triage and fix
- You're about to agree to, or push back on, a reviewer's suggestion

**When NOT to use:** No feedback exists yet — that's the `skillgrid:code-review` side. A single agreed, already-verified typo fix also doesn't need the full loop.

## The Response Pattern

```
WHEN receiving code review feedback:

1. READ: Complete feedback without reacting
2. UNDERSTAND: Restate requirement in own words (or ask)
3. VERIFY: Check against codebase reality
4. EVALUATE: Technically sound for THIS codebase?
5. RESPOND: Technical acknowledgment or reasoned pushback
6. IMPLEMENT: One item at a time, test each
```

## Forbidden Responses

**NEVER:**
- "You're absolutely right!" (explicit instruction-file violation)
- "Great point!" / "Excellent feedback!" (performative)
- "Let me implement that now" (before verification)

**INSTEAD:**
- Restate the technical requirement
- Ask clarifying questions
- Push back with technical reasoning if wrong
- Just start working (actions > words)

## Handling Unclear Feedback

```
IF any item is unclear:
  STOP - do not implement anything yet
  ASK for clarification on unclear items

WHY: Items may be related. Partial understanding = wrong implementation.
```

**Example:**
```
the user: "Fix 1-6"
You understand 1,2,3,6. Unclear on 4,5.

❌ WRONG: Implement 1,2,3,6 now, ask about 4,5 later
✅ RIGHT: "I understand items 1,2,3,6. Need clarification on 4 and 5 before proceeding."
```

## Source-Specific Handling

### From the user
- **Trusted** - implement after understanding
- **Still ask** if scope unclear
- **No performative agreement**
- **Skip to action** or technical acknowledgment

### From External Reviewers
```
BEFORE implementing:
  1. Check: Technically correct for THIS codebase?
  2. Check: Breaks existing functionality?
  3. Check: Reason for current implementation?
  4. Check: Works on all platforms/versions?
  5. Check: Does reviewer understand full context?

IF suggestion seems wrong:
  Push back with technical reasoning

IF can't easily verify:
  Say so: "I can't verify this without [X]. Should I [investigate/ask/proceed]?"

IF conflicts with the user's prior decisions:
  Stop and discuss with the user first
```

**the user's rule:** "External feedback - be skeptical, but check carefully"

## YAGNI Check for "Professional" Features

```
IF reviewer suggests "implementing properly":
  grep codebase for actual usage

  IF unused: "This endpoint isn't called. Remove it (YAGNI)?"
  IF used: Then implement properly
```

**the user's rule:** "You and reviewer both report to me. If we don't need this feature, don't add it."

## Implementation Order

```
FOR multi-item feedback:
  1. Clarify anything unclear FIRST
  2. Then implement in this order:
     - Blocking issues (breaks, security)
     - Simple fixes (typos, imports)
     - Complex fixes (refactoring, logic)
  3. Test each fix individually
  4. Verify no regressions
```

## Triage and Fix Loop

When a review produces findings, you own the loop from findings to clean:

### 1. Triage (the human's call)

Sort findings before touching code. If direction is unclear, surface them
grouped and ASK rather than fixing everything by default:

| Bucket | Meaning | Action |
|--------|---------|--------|
| **Fix now** | Real, in-scope, belongs with this change | Fix in this pass |
| **Defer** | Real but later; don't bloat this change | Log as a tracker issue (skillgrid:ticketing) or note in the spec |
| **Human look** | Needs manual inspection or testing before trusting | Flag it, don't silently auto-fix |
| **Noise** | Won't-fix or misread | Say why, drop it |

Don't let the reviewer dictate scope — "real, but later" is a valid and
common call. A clean small change beats a sprawling one.

### 2. Fix the "fix now" set — one at a time

For each:
1. Restate what was wrong (verify you agree with the finding)
2. Make the fix
3. Create and run a test that proves it (TDD: the test was RED before the fix)
4. Commit

### 3. Validate

Run the full verification suite (test runner + lint + build per config).
If a fix broke something, the finding was deeper than it looked — re-triage
that item.

### 4. Log the rest

- Deferred items: create a tracker ticket or add a note to the change's
  `tasks.md` / spec so they surface in the next cycle.
- Human-look items: flag in your final report with what to inspect.
- Noise: state the reason in your report so the reviewer (or a future
  reviewer) sees the ruling.

### 5. Repeat if findings remain

If validation reveals new issues or a fix was incomplete, re-triage the
residuals and loop. Cap at 3 rounds — beyond that, surface the remaining
items to the human with the rulings you made and why.

## When To Push Back

Push back when:
- Suggestion breaks existing functionality
- Reviewer lacks full context
- Violates YAGNI (unused feature)
- Technically incorrect for this stack
- Legacy/compatibility reasons exist
- Conflicts with the user's architectural decisions

**How to push back:**
- Use technical reasoning, not defensiveness
- Ask specific questions
- Reference working tests/code
- Involve the user if architectural

**If you're uncomfortable pushing back out loud:** Name that tension, then tell your partner about the issue you've seen. They'll appreciate your honesty.

## Acknowledging Correct Feedback

When feedback IS correct:
```
✅ "Fixed. [Brief description of what changed]"
✅ "Good catch - [specific issue]. Fixed in [location]."
✅ [Just fix it and show in the code]

❌ "You're absolutely right!"
❌ "Great point!"
❌ "Thanks for catching that!"
❌ "Thanks for [anything]"
❌ ANY gratitude expression
```

**Why no thanks:** Actions speak. Just fix it. The code itself shows you heard the feedback.

**If you catch yourself about to write "Thanks":** DELETE IT. State the fix instead.

## Gracefully Correcting Your Pushback

If you pushed back and were wrong:
```
✅ "You were right - I checked [X] and it does [Y]. Implementing now."
✅ "Verified this and you're correct. My initial understanding was wrong because [reason]. Fixing."

❌ Long apology
❌ Defending why you pushed back
❌ Over-explaining
```

State the correction factually and move on.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "The reviewer is wrong, ignore it" | Verify first — wrong AND unverified is how regressions ship. State why with technical reasoning, in-thread. |
| "I'll implement all the comments at once" | Batched fixes hide which one broke what. One at a time, test each; clarify unclear items before starting. |
| "It's a professional feature, keep it" | Grep for actual usage first. Unused = YAGNI; the user's rule is you don't add features you don't need. |
| "The reviewer is right, I'll just agree" | Performative agreement skips verification. Restate the requirement or push back — then act. |
| "I'm too uncomfortable to push back" | Discomfort is not a technical argument. Name the tension, state the issue, reference working tests/code. |

## Red Flags

- "Great point!" / "You're absolutely right!" / any "Thanks for..." in the reply — a Forbidden Response
- Code changed before any verification against the codebase
- Unclear items left as TODOs while understood items get implemented first
- A fix with no test proving it (the test was never RED)
- Multiple items fixed and tested together, so a regression can't be attributed
- A pushback that's defensive or apologetic instead of technical

## Verification

- [ ] Every actionable comment fixed or explicitly pushed back on with a technical reason in-thread
- [ ] Each fix has a test that was RED before the fix and is GREEN after (test output captured)
- [ ] Full suite (tests + lint + build) passes after all fixes — exit 0
- [ ] Diff reviewed comment-by-comment: no item silently dropped, deferred, or re-triaged without a logged ruling
- [ ] No Forbidden Response in any thread reply (grep the replies for "Thanks"/"Great point")
- [ ] If you pushed back and were wrong: correction stated factually in-thread, no apology essay

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Performative agreement | State requirement or just act |
| Blind implementation | Verify against codebase first |
| Batch without testing | One at a time, test each |
| Assuming reviewer is right | Check if breaks things |
| Avoiding pushback | Technical correctness > comfort |
| Partial implementation | Clarify all items first |
| Can't verify, proceed anyway | State limitation, ask for direction |

## Real Examples

**Performative Agreement (Bad):**
```
Reviewer: "Remove legacy code"
❌ "You're absolutely right! Let me remove that..."
```

**Technical Verification (Good):**
```
Reviewer: "Remove legacy code"
✅ "Checking... build target is 10.15+, this API needs 13+. Need legacy for backward compat. Current impl has wrong bundle ID - fix it or drop pre-13 support?"
```

**YAGNI (Good):**
```
Reviewer: "Implement proper metrics tracking with database, date filters, CSV export"
✅ "Grepped codebase - nothing calls this endpoint. Remove it (YAGNI)? Or is there usage I'm missing?"
```

**Unclear Item (Good):**
```
the user: "Fix items 1-6"
You understand 1,2,3,6. Unclear on 4,5.
✅ "Understand 1,2,3,6. Need clarification on 4 and 5 before implementing."
```

## GitHub Thread Replies

When replying to inline review comments on GitHub, reply in the comment thread (`gh api repos/{owner}/{repo}/pulls/{pr}/comments/{id}/replies`), not as a top-level PR comment.

## Next

Once findings are triaged and the in-scope set is fixed (or the review is clean), the change is ready for **`skillgrid:ship`** — the integration + archive-move step. Ship verifies tests on the integrated tree, merges / opens a PR / keeps the branch, then moves the change folder to `.skillgrid/archive/`. After ship, **`skillgrid:reflect`** closes the cycle.
