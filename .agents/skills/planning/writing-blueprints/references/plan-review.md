## Plan Review

After self-review passes, run a fresh-eyes structural review of the blueprint
before handing off to execution. This is NOT a re-do of self-review — it
checks what self-review can't: whether the plan will survive contact with
reality.

### When to run

- Always, for blueprints with 3+ tasks or any one-way-door decision
- Skip for 1-2 task blueprints with no migrations or contract breaks
  (self-review is sufficient)

### How to run

**Option A — Subagent (preferred for 3+ tasks):**
Dispatch a `general-purpose` subagent with the blueprint file path, the spec
file path, and the in-force ADR manifest. Prompt:

> You are a plan reviewer. Read the blueprint and spec. Do NOT review code
> quality — review the PLAN. Check:
> 1. **Failure modes:** For each task, what can go wrong at runtime that the
>    plan doesn't address? (network failure, partial state, concurrent access,
>    empty data, timeout) Name specific failure → specific mitigation.
> 2. **Scope coherence:** Does the plan do MORE than the spec asks (scope
>    creep) or LESS (scope gap)? Flag both.
> 3. **Feasibility:** Does any task assume a capability the codebase doesn't
>    have (library, API, infra)? Check against the codebase if needed.
> 4. **Ordering:** Are dependencies correct? Could a later task be done
>    first (parallelism) or does an earlier task block on something that
>    doesn't exist yet?
> 5. **Test strategy:** Does each task have a verifiable "done" state? Are
>    the acceptance scenarios (BDD) sufficient to prove the feature works?
> 6. **ADR compliance:** Does any task contradict an in-force ADR? If so,
>    flag it.
>
> Output: a findings list (Critical / Important / Minor) with file:line
> references to the blueprint, plus a verdict: READY FOR EXECUTION |
> NEEDS REVISION.

**Option B — Inline (1-2 tasks, no one-way doors):**
Run the same 6 checks yourself against the blueprint. If all pass, proceed.
If any fail, fix inline.

### Acting on findings

Apply `skillgrid:receiving-code-review` triage rules:
- **Fix now:** structural gaps that would block execution (missing dependency,
  wrong ordering, ADR violation) — fix the blueprint, re-commit
- **Defer:** nice-to-have refinements (extra test cases, edge case notes) —
  add as a note in the blueprint's "Global Constraints" or the task's step
- **Human look:** scope questions the user should answer — surface and ask
- **Noise:** reviewer overreach — note and drop

If the verdict is NEEDS REVISION: fix the Critical/Important items, re-run
the self-review checklist, then hand off. Do NOT enter a full review loop —
one pass is the cap. The execution phase has its own review; this gate
catches structural problems, not code quality.

### Output

Append a brief review summary to the blueprint file:

```markdown
## Plan Review

- Verdict: READY FOR EXECUTION
- Findings: 0 Critical, 1 Important (fixed: added error handling for
  timeout in Task 3), 2 Minor (deferred: retry logic, logging)
- Reviewed: [date]
```

