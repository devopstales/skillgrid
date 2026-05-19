---
name: sdd-diagnose
description: >
  Disciplined diagnosis loop for hard bugs and performance regressions.
  Reproduce → minimise → hypothesise → instrument → fix → regression-test.
  Trigger: User reports a bug, something broken/throwing/failing, performance regression, or says "diagnose this" / "debug this".
license: Apache-2.0
metadata:
  author: devopstales
  version: "2.0"
triggers:
  - "diagnose"
  - "debug"
  - "bug"
  - "broken"
  - "failing"
  - "throwing"
  - "performance regression"
  - "something is wrong"
tools:
  - file_system
  - execute_command
---

# Diagnose

## Norse persona invocations (coordinator)

When orchestrating `/sdd-diagnose`, dispatch per `skills/_shared/sdd-persona-delegation.md`:

| Required | Persona | Capability |
| --- | --- | --- |
| yes | vidar | root-cause-analysis |
| no | loki | assumption-stress-test |

A discipline for hard bugs. Skip phases only when explicitly justified.

When exploring the codebase, use the project's domain glossary to get a clear mental model of the relevant modules, and check ADRs in the area you're touching.

## Phase 0 — Preflight

Before any investigation:
1. Check `sdd-verify` last status — was this change already verified? Might be spec gap vs bug
2. Read `.skillgrid/project/CONTEXT.md` for domain assumptions
3. Determine: is this a bug in current code or a failing test from ongoing `sdd-apply`?

If bug originates from uncommitted work in progress → redirect to `sdd-apply` for fix, not diagnose.

---

## Phase 1 — Build a Feedback Loop (Systematic Debugging Phase 1: Root Cause Investigation)

**This is the skill.** Everything else is mechanical. If you have a fast, deterministic, agent-runnable pass/fail signal for the bug, you will find the cause. If you don't have one, no amount of staring at code will save you.

Spend disproportionate effort here. **Be aggressive. Be creative. Refuse to give up.**

### Evidence Gathering Checklist

Before hypothesizing, collect:

**A. Error documentation:**
- Full error message verbatim
- Complete stack trace (all frames, not just top)
- Error codes or HTTP status
- Timestamp and environment

**B. Reproduction script:**
- Build a deterministic repro (script, test, curl command)
- Capture exact steps, inputs, expected vs actual
- Note reproduction rate (always/sometimes/intermittent)

**C. Recent changes correlation:**
```bash
git log --oneline -20
git diff origin/main...HEAD --name-only
# What changed recently that could cause this?
```

**D. Multi-component boundaries** (if system has layers):
- Instrument each layer entry/exit
- Log state propagation
- Verify config/env propagation
- Check secrets/credentials at boundaries

**E. Data flow trace** (for deep errors):
- Trace bad value backward to its origin
- Find first caller that introduces incorrect state
- Fix at source, not at symptom

### Constructing the Feedback Loop

Try in order:
1. **Failing test** — unit/integration/e2e at the bug's seam
2. **CLI invocation** with fixture input, diff stdout
3. **HTTP script** against running dev server
4. **Headless browser** script (for UI bugs)
5. **Replay captured trace** (HAR file, network log)
6. **Throwaway harness** — minimal subset exercising bug path
7. **Property/fuzz loop** — 1000 random inputs to find pattern
8. **Bisection harness** — automate state boot to bisect bug origin
9. **Differential loop** — compare old vs new version/config
10. **HITL bash script** — drive human with structured repro steps

**Non-deterministic bugs:** Loop 100×, parallelize, add stress, raise repro rate to >50%. A 1% flake is not debuggable.

### When You Genuinely Cannot Build a Loop

Stop explicitly. List all attempts. Ask user for:
- Access to reproducing environment
- Captured artifact (HAR, logs, core dump, screen recording)
- Permission to add temporary production instrumentation

**Do NOT proceed to hypothesis without a loop.**

### Iron Law — No Hypotheses Before Evidence

```
NO FIXES WITHOUT ROOT CAUSE INVESTIGATION FIRST
```

All 4 phases must be completed in order:
- Phase 1: Root cause investigation (evidence gathering)
- Phase 2: Pattern analysis (find working examples, compare)
- Phase 3: Hypothesis and minimal testing
- Phase 4: Implementation (create test, fix, verify)

**3-fixes threshold:** If 3+ attempted fixes fail → STOP. Question architecture. Escalate to HITL.

---

## Phase 2 — REPRODUCE

Run the loop. Watch the bug appear.

**Confirm:**
- [ ] Loop produces exact user-described failure (not adjacent failure)
- [ ] Failure reproducible across multiple runs (or high-enough rate for flaky)
- [ ] Symptom captured exactly (error message, wrong output, timing)
- [ ] Minimal repro isolated to smallest possible trigger

**Do NOT proceed until bug is reproduced reliably.**

---

## Phase 3 — ISOLATE

Binary narrowing to shrink failing surface.

**Techniques:**
- Comment out code halves until bug disappears
- Add logs at critical boundaries
- Use debugger to step through stack
- Substitute mocks/fakes to isolate component

**Output:** "Isolated to `src/auth/middleware.ts:45-50` — token parsing branch"

---

## Phase 4 — UNDERSTAND / ROOT CAUSE (Systematic Debugging Phase 3: Pattern Analysis + Hypothesis)

### Pattern Analysis (Phase 2 of 4-phase model)

**Before hypothesizing:**

1. **Find working examples** in same codebase:
   - Similar features that work correctly
   - Adjacent components with correct pattern
   
2. **Compare working vs broken:**
   - What's different? List every difference
   - Don't assume small differences don't matter

3. **Read reference implementations COMPLETELY** if using external pattern
   - Every line, not skim

4. **Understand dependencies:**
   - Required config?
   - Environment/setup assumptions?
   - External service behavior?

### Hypothesise (Phase 3 of 4-phase model)

Generate **3–5 ranked hypotheses** before testing any.

**Each hypothesis MUST be falsifiable:**
> "If X is root cause, then changing Y will make bug disappear (or Z will make it worse)."

Show ranked list to user (they often re-rank based on domain knowledge).

If user AFK, proceed with your ranking.

**Rationale per hypothesis:** 1-2 sentences of evidence supporting it.

---

## Phase 5 — INSTRUMENT (Systematic Debugging Phase 3 continued)

Each probe maps to specific hypothesis prediction. **Change one variable at a time.**

**Tool preference:**
1. Debugger / REPL inspection (one breakpoint > ten logs)
2. Targeted logs at hypothesis-distinguishing boundaries
3. Never "log everything and grep"

**Tag every log with unique prefix:** `[DEBUG-{id}]`. Cleanup becomes single grep. Untagged logs survive; tagged logs die.

**Perf branch:** For performance, establish baseline measurement (profiler, timing harness), then bisect. Measure first, fix second.

---

## Phase 6 — FIX + REGRESSION TEST (Systematic Debugging Phase 4: Implementation)

**Write regression test BEFORE fix** — but only if correct seam exists.

A correct seam exercises the **real bug pattern** at call site. If only shallow seam available (single-caller test when bug needs multi-caller chain), a test there gives false confidence.

**If no correct seam exists → that IS the finding.** Flag architecture issue. Codebase prevents clean regression lock-down. Escalate to `sdd-architecture-review`.

### Fix sequence:

1. **Turn minimized repro into failing test** at the correct seam
2. **Watch it fail** (verify test catches bug)
3. **Apply minimal root-cause fix** (no extra changes)
4. **Watch it pass**
5. **Re-run original Phase 1 feedback loop** against full original scenario (not just minimized version)

---

## Phase 7 — CLEANUP + POST-MORTEM

Required before declaring done:

- [ ] Original repro no longer reproduces (re-run Phase 1 loop)
- [ ] Regression test passes
- [ ] All `[DEBUG-…]` instrumentation removed (grep for prefix)
- [ ] Throwaway prototypes deleted or moved to clearly-marked debug location
- [ ] Hypothesis that was correct stated in commit/PR message (so next debugger learns)
- [ ] If architecture friction prevented clean regression test → hand off to `sdd-architecture-review`

**Then ask:** "What would have prevented this bug?"

If answer involves:
- No good test seam
- Tangled callers
- Hidden coupling
- Missing abstraction

→ escalate to `sdd-architecture-review` **after** fix is in (you have more information now than at start).

---

## Phase 8 — GATE ESCALATION (3-Fixes Threshold)

**Hard rule:**

```
IF 3+ attempted fixes failed → STOP → QUESTION ARCHITECTURE
```

Pattern indicating architectural problem (not failed hypothesis):
- Each fix reveals new shared state/coupling in different place
- Fixes require "massive refactoring" to implement
- Each fix creates new symptoms elsewhere

**Action:**
1. STOP attempting more fixes
2. Document: "Three fixes attempted, each revealed architectural issue X"
3. Escalate to `sdd-architecture-review` with evidence
4. Mark as HITL pending architectural decision

**Do NOT try fix #4.** This is a wrong architecture pattern, not a hard bug.

---

## Integration with SDD Workflow

- After diagnosis complete → optional `sdd-apply` to implement thorough fix (if initial fix was minimal/quick)
- If diagnosis reveals missing CONTEXT.md entry → `/sdd-clarify` to update
- If architectural friction found → `sdd-architecture-review` before more work
- Diagnosis report saved to: `.skillgrid/tasks/research/<issue-id>-diagnosis.md`

## Rules Summary

- **Don't guess** — cannot reproduce → ask for data
- **One change at a time** — isolate with single variable changes
- **Preserve evidence** — log every step, command, output
- **Use existing tests** — add failing test first if TDD project
- **Fix root cause** — never mask symptoms
- **Profile first** — for performance issues
- **Intermittent = race/time** — inspect ordering, concurrency, dependency stability
- **3 fixes threshold** → architectural review
- **No hypothesizing before evidence gathering** (Phase 1 complete)
- **Show hypotheses to user** before testing (cheap checkpoint)

## Red Flags — STOP and Return to Phase 1

- "Quick fix first, investigate later"
- "Just try changing X and see"
- "Add multiple changes, run tests"
- "Skip loop, I'll manually verify"
- "It's probably X, let me fix that"
- "I don't fully understand but this might work"
- "One more fix attempt" (when already tried 2+)
- Each fix reveals different new problem
- Proposing solutions before tracing data flow

**ALL mean: STOP. Return to Phase 1. If 3+ fixes attempted → question architecture.**

---

## Command Output Format

```
[DIAGNOSE REPORT]

Issue: [description]

Phase 1 — Evidence gathered:
  - Reproduction: [script/steps]
  - Error: [exact message]
  - Recent changes: [git diff summary]

Phase 2 — Isolated to:
  [file:line-range]

Phase 3 — Root cause hypothesis:
  [clearest cause with evidence]

Phase 4 — Fix applied:
  [description + commit SHA]

Phase 5 — Verification:
  [repro no longer triggers]

Phase 6 — Next steps:
  [optional: sdd-apply for thorough implementation]

Status: COMPLETED
```

---

## Return

- **Return:** the standard SDD envelope per `skills/_shared/sdd-return-envelope.md` (capture phase outcomes in `detailed_report` and `executive_summary.overview`).

## See Also

- Superpowers source: `skills/systematic-debugging/SKILL.md` (4-phase model)
- TDD integration: `enforced-tdd-protocol` for test-first bug fixes
- Architecture review: `sdd-architecture-review` if architectural friction found
- Related: `spec-compliance-verifier` (check if bug is spec gap instead)
