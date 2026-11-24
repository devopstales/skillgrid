# Verification-Gap Audit

Adapted from BMAD `bmad-code-review/review-prompts/verification-gap.md`.

**Goal:** Find changed behavior that could break without reliable verification catching it.

**Ask one question:** *"If the behavior this change is supposed to produce broke where it's actually used, would verification fail?"*

**QA-side audit artifact, not the lens.** The rows below render into the change's `report.md`. The review lens with the same name (`reviewers/verification-gap.md`, the lens set in `references/review-and-fix-loop.md`) is the dispatched subagent's template — it files lens findings into the triage buckets; this audit is the coordinator's own pass over the change, and its table is the audit record in the report.

## Four Gap Shapes

1. **Regression gap** — the changed code regresses where it's used, and no test covering that use would fail.
2. **Missing-adoption gap** — a place that should now use the new behavior doesn't (caller not updated, feature flag not flipped, migration not run).
3. **Broken-verification gap** — a test appears to cover the changed behavior, but would not actually protect it because it is:
   - skipped or disabled
   - flaky (passes sometimes, fails sometimes, no clear signal)
   - not run in the normal verification path (excluded from CI, filtered out)
   - too weak to observe the regression (mock-only, snapshot-only, source-text assertion, asserts the mock not the code)
4. **Missing-oracle gap** — a requirement's happy-path scenario has no `G<n>` entry (no runnable `CHECK:`+`EXPECT:`, no `manual`, no `ABANDON`). A scenario with no oracle is a traceability gap, not a completed behavior.

## Evidence Rule

Every finding must be **evidence-grounded**. Before filing a gap:

- Read the test before claiming what it covers. A test name is not evidence.
- Search the whole repo by symbol + import references before claiming no test exists.
- Run the test (or the suite that contains it) to confirm it actually executes.

**Triage trusts a gap finding as filed.** Each block must stand on its own evidence — do not write "see above" or reference a prior finding.

## Output Format

Each finding is a row in the audit table:

| Gap | Location | Shape | Evidence | Smallest Regression |
|-----|----------|-------|----------|---------------------|
| <one-line description> | <file:line or test file::name> | regression / missing-adoption / broken-verification / missing-oracle | <what you read that proves the gap> | <name the smallest change a consumer would observe> |

**Smallest Regression** is the demonstration: name the specific input, state, or call sequence that would break, and the observable difference a user or downstream caller would see. "The API returns 500" is not specific enough. "When the cache is cold and two requests arrive within 10ms, the second request reads a half-written entry and returns null instead of the cached value" is.

## Route by Shape

- A **regression gap** (no covering test) and a **missing-oracle gap** are FAIL-floor findings — the behavior is unverified where the change touched it (a scenario with no covering test / no oracle).
- A **missing-adoption gap** is a non-answer: CONCERNS when low-risk — named in the report with a re-run path.
- A **broken-verification gap** is a WARNING finding. The human can route any of them.

## Writing the Report Section

Render the audit as its own table in `report.md`'s `## Findings` section (above the per-severity blocks):

```
## Findings

| Gap | Location | Shape | Evidence | Smallest Regression |
|-----|----------|-------|----------|---------------------|
| <gap> | <loc> | <shape> | <evidence> | <smallest regression> |
```

FAIL-floor gaps are also named in `## Gate Decision`'s unmet criteria. A clean audit renders `Verification-gap audit: no gaps` in the section, never silence.

## What Is NOT a Gap

- A test that covers a different (but adjacent) behavior — that's a coverage question, not a verification gap.
- A scenario in `acceptance.feature` with no test yet — that's a traceability gap, reported in the intent area's scenario accounting (`## Intent` `Scenarios:` line), not here.
- A test that is slow but reliable — that's a test-hygiene issue, not a verification gap.
