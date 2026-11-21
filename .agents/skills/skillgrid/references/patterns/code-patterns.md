# Code patterns

The measured ranges and shapes a reviewer checks in the code. Signals, not laws: a deliberate exception names its reason.

When to load: Phase 3 (writing), Phase 4 and 5 (review). Role: reviewer.

## Size and shape

| Signal | Healthy | Investigate | Source |
|---|---|---|---|
| Function length | under ~40 lines | over ~60 | this skill's review experience; matches common lint defaults |
| File length | under ~400 lines | over ~600 | same |
| Nesting depth | 3 or fewer | 5 or more | same |
| Parameters | 4 or fewer | 6 or more | same |
| Cyclomatic complexity | under 10 | over 15 | common linter thresholds |
| Duplication | deliberate and named | copy-paste with small edits | `jscpd` guidance in `skillgrid:jscpd` |

A long function with one linear job is fine. A short function with five branches is not.

## Names

- A name says what the thing is or does, not how it is built. `retry_with_backoff` beats `do_retry_v2`.
- No abbreviations the glossary does not define.
- Booleans read as assertions: `is_ready`, `has_access`, not `flag`.
- One term per concept, everywhere. A synonym is a drift finding (`references/specialists/consistency.md`).

## Errors

- Every error says what failed, on what input, and what the caller can do.
- Wrap with context at each boundary; never swallow silently.
- One taxonomy per program: codes, not free text, where a caller must branch.
- A catch-all handler is a defect unless it re-raises with context or ends the process deliberately.

## Tests

- The test name states the behavior: `rejects a rename onto an existing path`.
- One reason to fail per test.
- A test that asserts nothing, or asserts a mock was called, proves nothing.
- The failure path is tested, not just the happy path.
- Recorded output, not "tests pass": the command and its result.

## Duplication

- Three similar lines are cheaper than an abstraction. Three similar blocks with one varying detail are a parameterized function.
- A duplicated rule in two modules is a future inconsistency. Centralize the rule, not the formatting.
- A generated file is not duplication. Say so and exclude it.

## Comments and docs

- Comments explain why. If they explain what, rename something.
- A stale comment is worse than none. Fix it or delete it.
- Public functions carry a contract: inputs, outputs, errors, side effects.
