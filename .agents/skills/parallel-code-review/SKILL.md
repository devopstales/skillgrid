---
name: parallel-code-review
# based on bmad:bmad-code-review + gstack:review
description: Use for large or high-risk diffs when a single review pass isn't enough — dispatch several specialist reviewers in parallel, then triage, deduplicate, and route the findings into the fix loop. The heavy counterpart to skillgrid:requesting-code-review.
---

# Parallel Code Review

Dispatch several independent specialist reviewers **in parallel**, then verify,
deduplicate, and triage their findings before routing them into the fix loop.

**Announce at start:** "I'm using the skillgrid:parallel-code-review skill to
fan out specialist reviewers on this diff."

**When to use:** this is the **heavy** review. Reach for it when a single pass
(`skillgrid:requesting-code-review`) isn't enough:

- the diff is large (50+ changed lines), or
- the diff is high-risk (auth, data migration, money, concurrency, public API)

For small or low-risk diffs, stay on `skillgrid:requesting-code-review` — the
fan-out costs more than it finds.

**Relationship to the other review skill:** same target (the diff), different
topology. `requesting-code-review` is one reviewer doing two axes (Standards +
Spec). This skill is N specialist reviewers, each a narrow lens, run in
parallel. The two-axis templates are **reused** here as two of the specialists
(not copied).

## Config

Read `.skillgrid/config.yaml` before starting. Use `testing.runner` for any
verification the triage step needs, and the glossary/ADR paths
(`conventions.glossary`, the change's ADR Review Manifest at
`.skillgrid/specs/YYYY-MM-DD-<topic>/adr.md`) for the Standards specialist.

## The Process

### Step 1: Pin the diff and size it

```bash
BASE_SHA=$(git merge-base <base-branch> HEAD)   # or the SHA the work started from
HEAD_SHA=$(git rev-parse HEAD)
git diff --stat $BASE_SHA..$HEAD_SHA
```

Capture the changed-line count **and the file types touched**. This drives
specialist selection:

- **< 50 changed lines** → run only Standards + Spec (the two-axis set). The
  other specialists are skipped; say so.
- **≥ 50 changed lines, or high-risk** → run the core five: Standards, Spec,
  Edge cases, Verification gaps, Security.
- **Accessibility** — add it when the diff touches **UI** (markup, components,
  styles, or interactive elements: buttons, links, forms, inputs, navigation,
  modals, focus handling).
- **Performance** — add it when the diff touches **data access, request
  handling, rendering, or hot paths** (queries, loops over collections, async
  work, caching, network calls, bundle/build, per-request or per-render paths).
- **Red team** — always last (it reads the other specialists' merged findings).

A small diff that *does* touch UI or data access still earns the Accessibility
or Performance specialist on top of Standards + Spec — the line-count floor
gates the *core* five, not the domain specialists. Say which specialists you ran
and why, including any you skipped for lack of surface.

If the diff is empty, stop — there is nothing to review.

### Step 2: Fan out the specialists (one message, parallel)

Launch **every selected specialist in a single message** (multiple subagent
calls) so they run in parallel, each with fresh context and no bias from the
others. Use a `general-purpose` subagent per specialist. Each specialist's
prompt is the template in this skill's `reviewers/` directory (or, for
Standards/Spec, the shared templates):

| Specialist | Template | Lens | Selected when |
|------------|----------|------|---------------|
| Standards | [../requesting-code-review/code-reviewer.md](../requesting-code-review/code-reviewer.md) | glossary, in-force ADRs, smell baseline | always (core) |
| Spec | [../requesting-code-review/spec-reviewer.md](../requesting-code-review/spec-reviewer.md) | per-scenario BDD, missing/partial, scope creep | always (core) |
| Edge cases | [reviewers/edge-case-hunter.md](reviewers/edge-case-hunter.md) | unhandled branches, boundaries, deletion regressions | core five |
| Verification gaps | [reviewers/verification-gap.md](reviewers/verification-gap.md) | does verification actually catch a break? | core five |
| Security | [reviewers/security.md](reviewers/security.md) | injection, authz, exposure, secrets, dep risk | core five |
| Accessibility | [reviewers/accessibility.md](reviewers/accessibility.md) | keyboard, semantics, forms, ARIA, contrast | diff touches UI |
| Performance | [reviewers/performance.md](reviewers/performance.md) | N+1, hot paths, network, caching, rendering | diff touches data/render |
| Red team | [reviewers/red-team.md](reviewers/red-team.md) | what the others missed | always, runs last |

Give every specialist: the diff range (`$BASE_SHA..$HEAD_SHA`), the review
package path if one exists (e.g. from `skillgrid:subagent-execution`'s
`scripts/review-package`), and the spec sources (the change's
`acceptance.feature` and blueprint) for the Spec specialist. The diff is
passed as a **path**, not inlined text — each specialist reads it.

**Red team runs last:** dispatch it after the other specialists return, and hand
it their merged findings so it hunts for what they missed, not what they found.

**Each specialist returns machine-parseable findings** (JSON, per the template)
so the triage step can normalize them uniformly. A specialist that finds nothing
returns an empty list — that is a valid, clean result for that lens.

**Resilience:** if a specialist fails, times out, or returns empty, log its name
and continue with the survivors. If **all** specialists fail or return empty, do
**not** claim a clean review — report that the review may be incomplete and list
which lenses didn't run.

### Step 3: Triage (the coordinator verifies; specialists don't grade)

Do this yourself — the specialists file findings, you render the verdict. They
lack the full context to grade severity.

**1. Normalize.** Pull every finding from every specialist into one list. Each
carries: `location` (file:line), the finding, its **source** (specialist name),
and the specialist's own evidence.

**2. Deduplicate by fingerprint** (`path:line:category`). Findings from two or
more specialists pointing at the same defect collapse into one entry, tagged
`MULTI-REVIEWER CONFIRMED (a + b)`. Multi-confirmation is a strong signal the
finding is real.

**3. Group by root cause.** Two findings belong in one entry only when the same
defect produced both — same location alone is not enough, and neither is a
shared fix. The entry's severity is the highest of its members; `source` is the
contributing specialists joined with `+`.

**4. Verify each entry at its location.** Read the cited `file:line` and read
beyond it — follow callers, upstream guards, the surrounding code — until you can
answer: does the bad outcome actually occur? Judge whether the problem is real,
not whether the proposed fix is plausible. (Edge-case and verification-gap
findings arrive with their own traced evidence; trust that evidence as filed and
confirm the cited location matches.)

**5. Render exactly one verdict per entry:**

| Verdict | Meaning |
|---------|---------|
| `high` | The bad outcome is real and intolerable (bug, data loss, security break, broken feature) |
| `medium` | Real, tolerable — should be fixed before merge |
| `low` | Real, cosmetic or negligible |
| `false` | You checked; the bad outcome does not happen here. Say what disproves it. |
| `maybe-false` | You couldn't tell. Say what would settle it. |

**Reject** (drop) any entry that is `false`; any `low` whose fix adds more
complexity than the harm it removes; or any entry whose only fix is to edit the
spec under review.

### Step 4: Route into the fix loop

Route every surviving entry into `skillgrid:receiving-code-review`'s triage
buckets — **fix now** (real, in-scope) / **defer** (real, later) / **human
look** (needs manual inspection) / **noise** (won't-fix). The mapping is
straightforward: `high`/`medium` that are in-scope → fix now; `high`/`medium`
that are pre-existing or out of scope → defer; anything you can't confirm →
human look; `low`/`maybe-false` you chose to drop → noise.

Then run the receiving-code-review fix loop: fix the in-scope set one at a time
with tests, log the rest, validate, and loop until clean or capped at 3 rounds.

### Step 5: Report

Present the result per specialist (findings count + worst finding), the
deduplicated/grouped entries with their verdicts and source tags, and what was
fixed vs deferred vs needs a human look. If any specialist failed to run, say so
before the verdict. Never present a "clean review" when a lens didn't complete.

## Common Rationalizations

| Excuse | Reality |
|--------|---------|
| "One good reviewer is enough; fan-out is overkill" | For a large or risky diff, one reviewer's context is the bottleneck — it can't hold the whole diff *and* every lens at full depth. Specialists run in parallel, each fresh; the diff lives in their context, only findings come back. |
| "I'll grade severity as I read the findings" | The specialist that saw the hunk doesn't have your full context. You verify at the location and render the verdict; the specialist only files the candidate. |
| "Two specialists flagged the same line, so it's twice as bad" | No. Dedup it into one entry, tag it MULTI-REVIEWER CONFIRMED, and grade it once. Confirmation raises your confidence it's real; it doesn't double the severity. |
| "The red team is redundant if the others ran" | The red team reads the *merged* findings and hunts the gaps — cross-cutting and integration-boundary issues that per-lens checklists don't cover. It runs last, with the others' output in hand. |

## Red Flags

**Never:**
- Claim a clean review when a specialist failed or returned empty
- Let a specialist's self-assigned severity be the final verdict
- Merge two findings just because they share a file or a fix
- Skip the red team on a high-risk diff because "the others looked fine"
- Re-grade a verified verification-gap finding — trust its filed evidence, confirm the location
