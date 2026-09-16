# Test-Rigor Contracts

Adapted from gsd-core `TESTING-STANDARDS.md` and gentleman-ai `sdd-apply/strict-tdd.md`.

A test that passes regardless of whether the feature it describes is implemented is worse than no test: it inflates the count while providing false confidence.

## The Six Contracts

| Contract | Rule | Violation Example |
|----------|------|-------------------|
| **Real code** | Test exercises production code, not a re-statement of the mock. | `expect(mockFn).toHaveBeenCalled()` as the only assertion. |
| **No vacuous truths** | Assertion can actually fail for a real input. | `expect(result).toBeDefined()` when the function always returns an object. |
| **No pass-always** | Test fails if the feature it describes is removed. | A test that passes with or without the retry logic. |
| **Test claimed path** | Test exercises the code path the test name claims. | Test named "retries 3 times" but the mock short-circuits on the first call. |
| **Complete mocks** | Mock returns a realistic shape, not a partial stub that hides the real interface. | Mock returns `{ id: 1 }` when the real interface has 12 fields and the code accesses field 7. |
| **Counter-test** | For "does not X" behavior, a test asserts the negative case explicitly. | Claiming "no side effect" without a test that asserts the side effect is absent. |

## Banned Assertion Catalogue

These patterns are named anti-patterns. If you see one, file it in the QA report's Test Quality Audit:

| Pattern | Why It's Banned |
|---------|-----------------|
| `expect(true).toBe(true)` | Tautology. Always passes. Tests nothing. |
| `expect(result).toBeDefined()` as the only assertion | Vacuous. Almost always true. |
| `expect(fn).toHaveBeenCalled()` without checking args | Tests that the mock was called, not that the code did the right thing. |
| Snapshot as the only assertion | Detects any change, including the ones you want. Does not verify behavior. |
| Source-text assertion (grep the source in a test) | Tests the code's shape, not its behavior. Refactor breaks the test. |
| Test that mocks the very thing it's testing | Circular. The mock returns the expected value, the test asserts it. |
| Test with no `When` (only `Given` + `Then`) | Precondition list, not a scenario. Nothing is exercised. |

## Triangulation

Adapted from gentleman-ai `strict-tdd.md`.

After a test goes GREEN, add a second test with **different inputs** that exercises the same behavior. If the first implementation was a coincidence (hardcoded value, lucky edge case), the second test breaks it and forces generalization.

**Default: triangulation is required for P0 behaviors.** Skip only with a compelling reason recorded in the test plan.

## Safety Net

Before modifying an existing file, run the existing tests for that file. Capture the baseline: `{N} tests passing`. Report pre-existing failures to the orchestrator — do not fix them in the same change. A test that was already failing before your change is not your finding.

## Assert the Degraded Verdict

When testing error handling, assert the specific degraded output, not just "did not throw."

```typescript
// Bad — only proves it didn't crash
await expect(riskyOperation()).not.toThrow();

// Good — proves what the degraded state actually is
const result = await riskyOperation();
expect(result.status).toBe('degraded');
expect(result.fallback).toBe('cached-value');
```
