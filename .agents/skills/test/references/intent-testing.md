# Intent Testing

Does the change satisfy the stated requirement? This area verifies the *goal*, not the task list. The core principle: **task completion ≠ goal achievement; passing tests ≠ verified behavior.** A green suite proves the code the agent thought to test, not that the feature exists.

**Force stance:** assume the goal was NOT achieved until codebase evidence proves it. Your starting hypothesis: tasks completed, goal missed. Falsify the implementation narrative.

**When NOT to run this area:** a pure refactor or rename with no requirement affected — there is no goal to falsify. Record `N/A — no requirement affected (refactor)`.

## The Four Levels

Start from the briefing's falsifiable requirements (the truths). For each, walk down:

| Level | Question | Evidence That Passes |
|-------|----------|---------------------|
| **Truth** | Is the falsifiable claim from the briefing actually true? | A test or scenario that exercises the behavior end-to-end and passed. |
| **Artifact** | Does the named file/component exist, is it substantive, and is it wired into the system? | File exists with real content (not a stub) + an import/registration that connects it. |
| **Key Link** | Is the wiring between components actually exercised? | A test that calls through the link and asserts the output at the far end. |
| **Data Flow** | Does real data move through the system as claimed? | A test with a concrete input that produces an observable output at the boundary. |

## Status Definitions

- **VERIFIED** — a test exercises this at the behavior level and passed in the verification output.
- **PRESENT_BEHAVIOR_UNVERIFIED** — code exists and is wired, but no test exercises the state transition or the cancellation/cleanup/ordering invariant. **Never counts as VERIFIED.** Routes to the human.
- **UNVERIFIED** — no evidence found. Grep found nothing, no test references it, or the test that should cover it is skipped.

## Process

1. **List every truth.** From `briefing.md`, enumerate the falsifiable requirements. Each is a row.
2. **For each truth, run the named test** that covers it at the highest level the evidence supports (Truth → Data Flow). Run the specific test, not the full suite. The full suite is a regression net; the named test is the evidence.
3. **Check the acceptance scenarios.** For every scenario in `acceptance.feature`, confirm a test exercised it and it went RED→GREEN (TDD evidence references the scenario name). A scenario with no covering test is a gap.
4. **Classify each truth** VERIFIED / PRESENT_BEHAVIOR_UNVERIFIED / UNVERIFIED.
5. **Escalate the gaps.** An unresolvable `PRESENT_BEHAVIOR_UNVERIFIED` (cannot be resolved by running the test) surfaces to the human in the report. Never silently absorbed.

## Rules

- **Presence is not behavior.** Grep/file checks prove a symbol is present and wired — they do not prove a state transition, a cancellation invariant, or an ordering guarantee holds at runtime.
- **One named test per level.** Behavioral spot-checks over structural checks: prefer a test that calls the function and asserts the output over a grep that confirms the function exists.
- **Re-verification mode:** on a second run, failed items get the full 4-level check; passed items get regression-only (re-run the named test, confirm still green).
- **BDD is the oracle.** The `acceptance.feature` scenarios are the intent contract. A truth whose scenario was never confirmed RED before implementation is a TDD-evidence gap, not a pass.

## What Is NOT Intent Testing

- Checking that all tasks in `tasks.md` are `[x]` — that's task completion, not goal achievement.
- Confirming the test suite is green — that's the regression net, not the goal evidence. The goal evidence is a *named* test for *each* truth.
- Reading the code and confirming it "looks right" — that's a code review, not verification.

## Writing the Report Section

```
## Intent
Tier: T<n> (floor: L<n>)
| Truth (briefing) | Level | Named test | Status |
|---|---|---|---|
| <falsifiable claim> | Truth / Data Flow | <test name> | VERIFIED |
| <falsifiable claim> | Artifact | — | PRESENT_BEHAVIOR_UNVERIFIED → human |

Scenarios: <N> total, <M> covered+passed, <K> gaps (<names>)
```
