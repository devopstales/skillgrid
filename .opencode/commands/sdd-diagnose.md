---
name: sdd-diagnose
id: sdd-diagnose
category: Workflow
description: Structured debugging session to reproduce, isolate, hypothesize, fix, and verify issues. Use when user reports a bug, unexpected behavior, or test failure.
---

<context-awareness>

## Read CONTEXT.md

Before starting, read `.skillgrid/project/CONTEXT.md` if it exists. Note any relevant domain terms, assumptions, or success criteria that might relate to the issue.

If the issue involves a term that conflicts with the glossary, flag it immediately (see `.agents/rules/skillgrid-context-contract.mdc`).

</context-awareness>

<workflow>

## The Diagnostic Loop

Execute these phases in order. Do not skip phases.

### 1. REPRODUCE
- Capture exact repro steps and reproduction rate (always/sometimes/intermittent).
- Record expected vs actual behavior.
- If repro is unclear, request missing evidence (logs, stack traces, inputs, environment details) and loop.
- Output: *“Reproduced under [conditions], rate: [x]. Expected: [...]. Actual: [...].”*

### 2. ISOLATE
- Identify responsible component/function/module.
- Build minimal repro or failing test/script when possible.
- Use binary narrowing to shrink the failing surface.
- Output: *“Isolated to [component/path]. Minimal repro: [...].”*

### 3. UNDERSTAND (ROOT CAUSE)
- Produce a root-cause hypothesis supported by evidence.
- Run a concise 5-whys chain.
- Distinguish symptom from actual defect location.
- Output: *“Root cause: [...]. Why chain: [...]. Evidence: [...].”*

### 4. FIX & VERIFY
- Implement minimal root-cause fix.
- Re-run original repro and related safety checks.
- Add or recommend regression coverage.
- Output: *“Fix applied and verified. Regression prevention: [...].”*

</workflow>

<guidelines>

## Behavioural rules

- **Don't guess** – If you cannot reproduce, ask for more data. Do not propose fixes until you have a confirmed reproduction.
- **One change at a time** – When isolating, change only one variable per attempt.
- **Preserve evidence** – Keep a log of steps tried and outputs.
- **Use existing tests** – If the project has a test suite, try to add a failing test before fixing (Red-Green style). Ask the user if they want that.
- **Fix root cause, not symptom** – Temporary symptom masking is not closure.
- **Profile before optimizing** – For performance issues, measure bottlenecks first.
- **Treat intermittent as timing/race candidates** – Prioritize ordering, concurrency, and external dependency behavior.

## Investigation strategy by issue type

- Runtime errors: start from full stack trace and null/type boundaries.
- Logic bugs: trace data flow and compare expected vs actual transformations.
- Performance issues: profile first; then optimize the measured hotspot.
- Intermittent/flaky behavior: inspect race conditions, timing windows, retries, dependency instability.
- Memory issues: inspect listeners, caches, closure retention, and long-lived objects.

## Anti-patterns

- Random fixes without evidence.
- Ignoring stack traces or environment differences.
- Multiple investigative edits at once.
- Declaring success without verification.

</guidelines>

<example>

## Example interaction

User: “The login button does nothing when clicked.”

Agent (Reproduce):
> Can you open the browser console and click the button? What error appears?

User: “`Uncaught TypeError: submitForm is not a function`”

Agent (Isolate):
> That suggests the function is undefined. Let me check `login.js`. On line 42, I see `onclick="submitForm()"` but the function is named `submitLoginForm`. Is that the only button on the page?

User: “Yes.”

Agent (Understand):
> Root cause: The event handler name in HTML does not match the declared function.  
> Why chain: click fails -> missing function -> wrong handler name in template -> drift from renamed JS function.

User: “Go ahead.”

Agent (Fix & Verify):
> Changed line 42. Now clicking the button shows no error and submits. Verified.

</example>

<error-handling>

## When the loop fails

If after 5 isolation attempts the cause is still unclear:
- **Pause** and summarise what you’ve eliminated.
- **Ask the user** to provide additional logs, a debugger session, or a different reproduction environment.
- **Do not** propose random fixes.

## Completion checklist

Before closing, verify all are true:
- Reproduction and expected behavior documented.
- Root cause and 5-whys chain recorded.
- Fix verified against original reproduction.
- Regression-prevention action captured.

</error-handling>

<integration>

## Connection to other SDD commands

- After a successful diagnose, the user may run `/sdd-apply` to implement the fix more thoroughly (if the fix was a temporary change).
- If the diagnosis reveals a missing success criterion or domain assumption, update `CONTEXT.md` via `/sdd-clarify` or ask the user.
- If the issue requires an architectural decision, load `architectural-decision-records` and draft an ADR in `.skillgrid/adr/` (or offer `/sdd-explore <decision-topic>` for a focused ADR pass).


## Return envelope

Return the standard SDD envelope per `.agents/skills/_shared/sdd-return-envelope.md`. Capture repro, isolation, root cause, fix, and verification in `detailed_report` and `executive_summary.overview`.


</integration>