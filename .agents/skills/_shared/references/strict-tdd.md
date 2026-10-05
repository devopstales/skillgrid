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

Every task in the execution ledger MUST carry a row in the TDD Cycle Evidence table below. The table is the **machine-checkable contract** between the implementer and `qa`: `qa` validates each cell against git history and the working tree, not against the implementer's narrative.

| Task | SATISFIES | RED (commit + exit) | GREEN (commit + exit) | TRIANGULATE (commit + exit) | REFACTOR (suite exit) | Verdict |
|---|---|---|---|---|---|---|
| <ticket-id> | <scenario-name> | `<short-hash>` exit 1 | `<short-hash>` exit 0 | `<short-hash>` exit 0 / `n/a` | exit 0 | OK |

**Column rules:**

- **SATISFIES** — the `acceptance.feature` scenario name this task implements. `qa` cross-references this against the scenario list; a mismatch is `STALE`.
- **RED** — the commit hash (first 7 chars) of the commit that *adds the failing test*, plus the test runner's exit code. The exit code MUST be non-zero. `qa` verifies: (1) the commit exists in git history, (2) the test file added in that commit exists on disk, (3) the exit code is non-zero.
- **GREEN** — the commit hash of the commit that *adds the implementation*, plus exit code 0. `qa` verifies: (1) the commit exists, (2) re-running the named test now passes.
- **TRIANGULATE** — the commit hash of the second test (different inputs, same behavior), plus exit code 0. For tasks where the spec has exactly one scenario, write `n/a` and note the scenario count. `qa` flags `WARNING` if the spec has multiple scenarios but only one test case.
- **REFACTOR** — exit code of the full relevant suite run after refactoring. Must be 0.
- **Verdict** — one of: `OK`, `MISSING_RED`, `MISSING_GREEN`, `STALE`, `FAILED`.
  - `OK` — all cells validated against git and the working tree.
  - `MISSING_RED` — no commit adds the test before the implementation commit.
  - `MISSING_GREEN` — no commit makes the test pass.
  - `STALE` — the SATISFIES scenario name or test file referenced does not match what exists on disk (the evidence refers to a different iteration).
  - `FAILED` — test-runner infrastructure failure mid-cycle; the row is marked and reported as `partial`.

A task completed without a test written first is marked `MISSING_RED`. `qa` rejects work whose TDD Evidence table is missing, has rows with no commit hash, or contains any `MISSING_RED` / `STALE` verdict.

## Assertion Quality Pre-Commit Checklist

Before committing the GREEN cell, the implementer MUST scan the test file for the following patterns. Any hit is a CRITICAL that must be fixed before the commit; `qa` re-checks at audit time and a missed pattern is a CRITICAL finding in the report.

| Pattern | What it looks like | Why it's wrong |
|---|---|---|
| **Tautology** | `expect(true).toBe(true)`, `assert 1 == 1` | Proves nothing — passes regardless of the code |
| **Ghost loop** | `for (item of emptyArr) { expect(item.x).toBe(...) }` | The loop body never runs if the collection is empty — the test always passes |
| **Pass-always** | test that would still pass if the feature were removed | The test does not actually test the feature |
| **Snapshot-only** | a snapshot that captures the bug rather than the expected behavior | Freezes the bug as the baseline |
| **Implementation-detail coupling** | `expect(el.className).toContain("text-xs")`, `expect(mock.calls.length).toBe(3)` | Tests internal state, not behavior — breaks on refactor with no behavior change |
| **Smoke-test-only** | `render(); expect(container).toBeInTheDocument()` with no behavioral assertion | "Renders without crash" is not a test — it must assert WHAT was rendered |
| **Mock-heavy** | more `mock()` calls than `expect()` calls (ratio > 2:1) | The test exercises the mock, not the code — wrong test layer |

**No snapshot-only** rule: a snapshot is evidence only when it is *generated from a known-correct run* and then frozen. A snapshot taken while the bug is present is a bug captured in amber — it proves the bug was there, not that the fix works.

## Changed-File Coverage

When a coverage tool is configured (`testing.coverage` in `config.yaml`), the implementer MUST record per-file line coverage for every file created or modified in the change, in the execution ledger:

| File | Line % | Uncovered lines |
|---|---|---|
| `path/to/file.ext` | <N>% | <line ranges or "—"> |

**Threshold:** `quality.coverage_min` (default 80). A changed file below the threshold is a WARNING in the QA report (not CRITICAL — coverage is a signal, not a gate, unless `quality.coverage_min` is explicitly set higher by the project). The gate for *whole-project* coverage is separate and lives in `qa`'s Code Quality Gate.

If no coverage tool is available, record: `Coverage: n/a — no coverage tool detected`. This is not a failure.

## Test Layer Selection

- **Unit**: single function/method, mocked dependencies.
- **Integration**: multiple components, real I/O (DB, file, network).
- **E2E**: full user path through the system.

Select the highest layer that fits. If a lower layer already covers the behavior, use the lower layer (Duplicate Coverage Guard).

## No Silent Fallback

If Strict TDD is active and you hit a test-runner infrastructure failure mid-cycle, mark the row `FAILED` and report `partial` — do NOT fall back to "just write the code first." That hides a real test-infra defect.
