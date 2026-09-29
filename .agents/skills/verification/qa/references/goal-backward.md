# Goal-Backward Verification

Adapted from gsd-core `agents/gsd-verifier.md`.

**Force stance:** Assume the goal was NOT achieved until codebase evidence proves it. Your starting hypothesis: tasks completed, goal missed. Falsify the implementation narrative.

**Task completion ≠ Goal achievement.** A "create chat component" task can be complete with a placeholder file — task done, goal "working chat interface" missed. Verify the goal, not the task list.

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
- **PRESENT_BEHAVIOR_UNVERIFIED** — code exists and is wired, but no test exercises the state transition or the cancellation/cleanup/ordering invariant. **Never counts as VERIFIED.** Routes to human.
- **UNVERIFIED** — no evidence found. Grep found nothing, no test references it, or the test that should cover it is skipped.

## Rules

- **Presence is not behavior.** Grep/file checks prove a symbol is present and wired — they do not prove a state transition, a cancellation invariant, or an ordering guarantee holds at runtime.
- **One named test per level.** Run the specific test that covers the level, not the full suite. The full suite is a regression net; the named test is the evidence.
- **Behavioral spot-checks over structural checks.** Prefer a test that calls the function and asserts the output over a grep that confirms the function exists.
- **Re-verification mode:** on a second QA run, failed items get full 4-level check; passed items get regression-only (re-run the named test, confirm still green).
- **Escalation:** an unresolvable gap (PRESENT_BEHAVIOR_UNVERIFIED that cannot be resolved by running the test) surfaces to the human in the QA report. Never silently absorbed.

## What Is NOT Goal-Backward

- Checking that all tasks in `tasks.md` are `[x]` — that's task completion, not goal achievement.
- Confirming the test suite is green — that's the regression net, not the goal evidence. The goal evidence is a *named* test for *each* truth.
- Reading the code and confirming it "looks right" — that's a code review, not verification.
