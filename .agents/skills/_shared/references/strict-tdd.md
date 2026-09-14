# Strict TDD Cycle (shared reference)

> Canonical source for the RED → GREEN → TRIANGULATE → REFACTOR cycle.
> Loaded by `subagent-execution` implementer briefs and audited by `qa`.

## The Cycle

For every implementation task:

### 1. RED
- Write the failing test first. The test MUST reference a `SATISFIES` scenario from `acceptance.feature`.
- Run it. It MUST fail (compile error, assertion failure, or missing symbol).
- **Evidence**: commit or dry-run output showing the test failed. Record the exact command + output.

### 2. GREEN
- Write the minimal implementation to make the test pass.
- Run the test. It MUST pass.
- **Evidence**: commit or suite output showing the test passed. Record the exact command + output.

### 3. TRIANGULATE
- Add a second test with different inputs that exercises the same behavior.
- This prevents trivial implementations (e.g., `return 42` passing a single test).
- Run both tests. Both MUST pass.
- **Evidence**: the second test's command + output.

### 4. REFACTOR
- Clean up the implementation while keeping all tests green.
- Run the full relevant test suite.
- **Evidence**: suite output.

## TDD Cycle Evidence Table

Every task in the execution ledger MUST carry:

| Task | RED (cmd + result) | GREEN (cmd + result) | TRIANGULATE (cmd + result) | REFACTOR (suite result) |
|---|---|---|---|---|

A task completed without a test written first is marked `FAILED` in the table. `qa` rejects work whose TDD Evidence table is missing or incomplete.

## Assertion Quality Rules

- **No tautologies**: `expect(true).toBe(true)` proves nothing.
- **No ghost loops**: an assertion inside a `for` over an empty collection never runs.
- **No pass-always**: the test must fail if the feature is removed.
- **No snapshot-only**: a snapshot that captures the bug is not a test.
- **No implementation-detail coupling**: assert on behavior, not internal state.

## Test Layer Selection

- **Unit**: single function/method, mocked dependencies.
- **Integration**: multiple components, real I/O (DB, file, network).
- **E2E**: full user path through the system.

Select the highest layer that fits. If a lower layer already covers the behavior, use the lower layer (Duplicate Coverage Guard).

## No Silent Fallback

If Strict TDD is active and you hit a test-runner infrastructure failure mid-cycle, mark the row `FAILED` and report `partial` — do NOT fall back to "just write the code first." That hides a real test-infra defect.
