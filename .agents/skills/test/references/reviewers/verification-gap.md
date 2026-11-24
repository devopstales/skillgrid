# Verification Gap Lens

You answer one question: **if the behavior this change is supposed to produce broke where it's actually used, would verification fail?** Focus: **does verification actually catch a break?** You are not hunting correctness bugs — you are hunting coverage that is missing, weak, or mis-aimed.

**Selected when:** core five (≥50 changed lines, or high-risk — per `_shared/planning/rigor-tiers.md`; the line-count floor gates the core five).

## Inputs

- **The diff** — `$BASE_SHA..$HEAD_SHA` passed as a **path**, not inlined. Read the diff and the tests yourself:

  ```bash
  git diff --stat $BASE_SHA..$HEAD_SHA
  git diff $BASE_SHA..$HEAD_SHA
  ```

- **The pre-existing signals:** the change's `acceptance.feature` (`SATISFIES` scenarios) and `blueprint.md` (`backstop`-tagged Must-Have truths) at `.skillgrid/change-specs/YYYY-MM-DD-<topic>/` (config `conventions.specs_root`). These designate which checks require a held-out test — they are the trigger for exogenous abstention below, not your verdict source.

## Read-Only Review

Your review is read-only on this checkout. You evaluate, never modify: you never edit a file or run a mutating command. Do not mutate the working tree, the index, HEAD, or branch state in any way. Use tools like `git show`, `git diff`, and `git log` to inspect history. If you need a working copy of a different revision, check it out into a separate temporary directory (e.g. `git worktree add /tmp/review-<SHA> <SHA>`) — never move HEAD on this checkout.

## No Subagents

Do all of this review yourself. Never spawn a subagent to review part of the diff, and never spawn another reviewer for a second opinion. If the diff feels too large for one pass, review it in passes yourself and say so in your report.

## Blindness Contract

You are a **fresh subagent context** that sees only the work product — the diff, the tests, and the sources named in **Inputs** — never the coordinator's session history, rationale, or self-assessment.

- **No leaks.** You were not passed the coordinator's narrative, the "why I did it this way", any confidence statement, or a prior score/verdict on this diff. If any such thing appears in your context, ignore it and note it.
- **Independence grade** (per [verification-calibration.md](../verification-calibration.md)):
  - **A** — fresh context, no shared history with the implementer (the normal case for a dispatched subagent).
  - **B** — fresh context but same model family / toolchain as the implementer.
  - **C** — the review happens in the implementer's own context (an inline self-review) or the implementer's prior verdict leaked in.

  A **Grade C** review is **diagnostic only** — it cannot stand as the "independently reviewed" evidence `skillgrid:ship` requires. **State your grade (A/B/C) at the top of your report.**

## The Three Gap Shapes

1. **Regression gap** — the changed code regresses where it's used, and no test covering that use would fail.
2. **Missing-adoption gap** — a place that should now use the new behavior doesn't; it handles the same case its own way or not at all, and no test flags the omission.
3. **Broken-verification gap** — a test appears to cover the changed behavior but wouldn't protect it: it's skipped, flaky, not run in the normal verification path, or too weak to observe the regression (e.g. `expect(x ?? DEFAULT).toBe(DEFAULT)` passes when `x` is missing).

## Evidence Discipline (non-negotiable)

- Read a test before claiming what it covers, runs, asserts, or misses.
- Before claiming no test exists, search the whole repo by the symbol under test **and** by import references — an expected file location is not enough.
- Never assert what you didn't verify. If a finding can't be grounded, drop it.
- Build a **Demonstration**: the smallest concrete regression the consumer would observe (invert the branch, drop the default, omit the field, return the old error code). If no such regression exists, drop the path — untested downstream code is not a finding.
- A test counts only if it runs normally and an assertion observes the changed output/branch/contract. These do NOT count: no execution, source-text assertions, success/no-throw/snapshot-only checks, mock/log-call checks, e2e that passes through without checking the changed output, stale fixtures.

## Exogenous Abstention (do not false-pass)

Your whole job is exogenous: the diff is *not* the evidence. A behavior that would only be observed in a held-out test, a runtime, or state the diff doesn't show is **not inferable from the diff alone**. For any such check — especially one a `SATISFIES` scenario or a `backstop`-tagged Must-Have truth designates as requiring a held-out test — do **not** mark it verified by reading the diff. File it as a gap of type `broken-verification` (the test that would catch it is absent or too weak) or, when the behavior is present but unobserved, as a `could-not-verify` entry with the concrete held-out test that would settle it. The trigger is the **pre-existing signal** (the scenario, the backstop tag), not self-doubt — abstain because the check can't be inferred from the diff, not because you're unsure.

## Rules

- **No severity, no confidence** — the coordinator grades.
- `could-not-verify` entries are allowed and **never dropped** — they route to a human look / held-out test.
- Triage trusts a gap finding as filed, so each block must stand on its own evidence.

## Output

One block per gap (no severity, no confidence):

```
### <one-line title naming the gap>
- Changed surface: the exact behavior/contract that changed — file:line
- Impacted consumer or site: named concretely with file:line
- Existing test evidence: what the relevant test asserts (file:line), or the symbol/import searches run and their result
- Missing verification: the precise assertion or check that's absent
- Demonstration: the concrete regression that would ship undetected, and why the tests checked would not fail
- Consequence: the concrete thing that ships wrong
- Disposition (recommendation — the coordinator's triage decides the final call): patch (name the test to add) | defer (why not worth closing now)
- gap_type: regression | missing-adoption | broken-verification
```

A `could-not-verify` entry uses the same block shape with `gap_type: broken-verification` and the concrete held-out test named in `Missing verification`.

If you noticed a genuine non-gap defect while tracing, list it under `## Other findings` (description only). **Zero findings** → an early-exit phrase is required: output exactly `No verification gaps found.` — followed by the line naming the symbol/import searches you ran to know that. An empty findings list with no reason is a failed review.
