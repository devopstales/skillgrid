# Verification-Gap Audit

Adapted from BMAD `bmad-code-review/review-prompts/verification-gap.md`.

**Goal:** Find changed behavior that could break without reliable verification catching it.

**Ask one question:** *"If the behavior this change is supposed to produce broke where it's actually used, would verification fail?"*

## Three Gap Shapes

1. **Regression gap** — the changed code regresses where it's used, and no test covering that use would fail.
2. **Missing-adoption gap** — a place that should now use the new behavior doesn't (caller not updated, feature flag not flipped, migration not run).
3. **Broken-verification gap** — a test appears to cover the changed behavior, but would not actually protect it because it is:
   - skipped or disabled
   - flaky (passes sometimes, fails sometimes, no clear signal)
   - not run in the normal verification path (excluded from CI, filtered out)
   - too weak to observe the regression (mock-only, snapshot-only, source-text assertion, asserts the mock not the code)

## Evidence Rule

Every finding must be **evidence-grounded**. Before filing a gap:

- Read the test before claiming what it covers. A test name is not evidence.
- Search the whole repo by symbol + import references before claiming no test exists.
- Run the test (or the suite that contains it) to confirm it actually executes.

**Triage trusts a gap finding as filed.** Each block must stand on its own evidence — do not write "see above" or reference a prior finding.

## Output Format

Each finding is a row in the QA report's Verification-Gap Audit table:

| Gap | Location | Shape | Evidence | Smallest Regression |
|-----|----------|-------|----------|---------------------|
| <one-line description> | <file:line or test file::name> | regression / missing-adoption / broken-verification | <what you read that proves the gap> | <name the smallest change a consumer would observe> |

**Smallest Regression** is the demonstration: name the specific input, state, or call sequence that would break, and the observable difference a user or downstream caller would see. "The API returns 500" is not specific enough. "When the cache is cold and two requests arrive within 10ms, the second request reads a half-written entry and returns null instead of the cached value" is.

## What Is NOT a Gap

- A test that covers a different (but adjacent) behavior — that's a coverage question, not a verification gap.
- A scenario in `acceptance.feature` with no test yet — that's a traceability gap, reported in the Traceability Matrix, not here.
- A test that is slow but reliable — that's a test-hygiene issue, not a verification gap.
