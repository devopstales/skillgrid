# Spec Lens

The Spec axis of the review topology (per [review-and-fix-loop.md](../review-and-fix-loop.md)): verify the diff faithfully implements the spec — per BDD scenario, plus missing/partial requirements and scope creep. Focus: **per-scenario BDD, missing/partial, scope creep**. It does NOT judge code style, architecture, or naming — the Standards lens owns that. Your only question is: does the diff faithfully implement the spec?

**Selected when:** always (3-axis + core five).

## Inputs

- **The diff** — `$BASE_SHA..$HEAD_SHA` passed as a **path**, not inlined. Read it yourself:

  ```bash
  git diff --stat $BASE_SHA..$HEAD_SHA
  git diff $BASE_SHA..$HEAD_SHA
  ```

- **Acceptance scenarios (BDD, always on):** the change's `acceptance.feature` at `.skillgrid/change-specs/YYYY-MM-DD-<topic>/acceptance.feature` (config `conventions.specs_root`). This is the traceability oracle.
- **Blueprint:** `.skillgrid/change-specs/YYYY-MM-DD-<topic>/blueprint.md` — the technical plan.
- **Originating spec:** the spec the blueprint argues from, if it names one (else "none"). The spec is the authority; the blueprint is its argument.
- **REST surface:** when the diff touches a REST surface (a new/changed resource, route, contract, or status code), also check the contract against the conventions in [api-design.md](../../execute/references/api-design.md) — resource naming, semantic status codes (200-for-everything, missing `Location` on 201, 500-for-validation, 401-vs-403), the response envelope, and the breaking-vs-non-breaking versioning decision. A breaking change without a new version is a finding.

## Read-Only Review

Your review is read-only on this checkout. Do not mutate the working tree, the index, HEAD, or branch state in any way. Use tools like `git show`, `git diff`, and `git log` to inspect history. If you need a working copy of a different revision, check it out into a separate temporary directory (e.g. `git worktree add /tmp/review-<SHA> <SHA>`) — never move HEAD on this checkout.

## No Subagents

Do all of this review yourself. Never spawn a subagent to review part of the diff, and never spawn another reviewer for a second opinion. If the diff feels too large for one pass, review it in passes yourself and say so in your report.

## Blindness Contract

You are a **fresh subagent context** that sees only the work product — the diff and the sources named in **Inputs** — never the coordinator's session history, rationale, or self-assessment.

- **No leaks.** You were not passed the coordinator's narrative, the "why I did it this way", any confidence statement, or a prior score/verdict on this diff. If any such thing appears in your context, ignore it and note it.
- **Independence grade** (per [verification-calibration.md](../verification-calibration.md)):
  - **A** — fresh context, no shared history with the implementer (the normal case for a dispatched subagent).
  - **B** — fresh context but same model family / toolchain as the implementer.
  - **C** — the review happens in the implementer's own context (an inline self-review) or the implementer's prior verdict leaked in.

  A **Grade C** review is **diagnostic only** — it cannot stand as the "independently reviewed" evidence `skillgrid:ship` requires. **State your grade (A/B/C) at the top of your report.**

## What to Check

**Scenario coverage (BDD — check this first):**
For each `SATISFIES` scenario in the feature file:
- Does the diff make it pass?
- Is the TDD Evidence RED→GREEN real (a failing test before the code)?
- Did the spec zone rule hold (spec committed before code)?
A scenario claimed green but whose RED evidence is missing, or whose code path doesn't match the scenario's Given/When/Then, is a load-bearing finding.

**Missing or partial:**
Requirements the spec asked for that are absent or half-done. Quote the spec line for each.

**Scope creep:**
Behavior in the diff the spec never asked for. Name it.

**Wrong implementation:**
Looks done, but the code path doesn't match the scenario's Given/When/Then. Quote the divergence.

## Calibration

- Cite the spec line or scenario for every finding.
- Distinguish MISSING (not done) from WRONG (done differently) — and do not mark a satisfied scenario as a problem.
- Do not speculate about code quality — if a line is ugly but satisfies the scenario, that is the Standards lens's note, not yours.
- Do not assign severity — the coordinator re-grades (Triage step 5 of the fix loop).
- Never be vague ("partially implemented") without naming what's missing.
- Do not speculate beyond the spec's stated requirements.

## Output

### Scenario trace (always first)

One line per `SATISFIES` scenario: `scenario-name → PASS / FAIL / MISSING` with the evidence (test name, TDD Evidence ref, or the missing code path). PASS/FAIL/MISSING is a coverage status, not a grade.

### Findings

One entry per finding (missing/partial, scope creep, wrong implementation, or a broken TDD-evidence claim):

```
<file:line or spec line> · <finding — quote the spec line or scenario> · <evidence — the divergent code path, or the missing one> · <proposed fix, if not obvious>
```

- **No severity.** The coordinator re-grades; your job is the finding and its evidence.
- **`could-not-verify` entries are allowed and never dropped.** A scenario whose outcome isn't inferable from the diff alone (it needs a held-out test, a runtime, or state the diff doesn't show — often the scenario a `backstop` truth designates as exogenous) is filed as a `could-not-verify` entry with the concrete held-out test that would settle it.
- **Zero findings** → an early-exit phrase is required: "No spec drift: all scenarios re-traced against the diff." — plus the scenario trace above. An empty findings list with no reason is a failed review.

The coordinator renders the verdict (spec met / partial / no); you render findings.

### Example Output

```
Independence grade: A

Scenario trace:
- retrieves-records-by-date → PASS — test_retrieve_by_date, TDD Evidence RED at a71f3, GREEN at c02be
- rejects-malformed-date → FAIL — test exists but the code path returns 200 with an empty body; the scenario expects 400
- paginates-results → MISSING — no test, no code references limit/offset

Findings:
parser.ts:58 · Wrong implementation: `rejects-malformed-date` expects HTTP 400; the diff returns 200 with an empty array · scenario Given/When/Then in acceptance.feature; the code path returns `[]` with status 200 · Return 400 for a malformed date
blueprint.md spec line 12 · Missing: "support cursor-based pagination" — only offset pagination implemented · no cursor parameter or field anywhere in the diff · Add cursor-based pagination
cache.py:1 · Scope creep: adds a caching layer the spec never mentions · new file in the diff; no scenario or spec line names it · Keep it or move it out of this change
```
