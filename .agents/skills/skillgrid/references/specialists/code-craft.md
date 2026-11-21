# Code craft specialist

Write code a maintainer can pick up tomorrow: clear entry points, honest seams, errors that say what to do, and tests that name behavior.

When to load: Phase 3 (writing), Phase 5 (revision moves). Role: implementer and fix editor.

## While writing

- **Entry points.** One obvious way in. The `main`/handler/router does three things: parse input, call the domain, render the result. Business rules live below it.
- **Seams.** Put a seam where the program talks to something it does not control: the network, the clock, the filesystem, another process. A seam is a named interface with one implementation and one fake; it is not an abstraction layer for its own sake.
- **Errors.** Every error says what failed, on what input, and what the caller can do. Wrap with context; never swallow. One taxonomy per program.
- **State.** Make the invalid state unrepresentable where the language allows it. If not, validate at the boundary and assert the invariant inside.
- **Tests.** Name the behavior, not the function: `rejects a rename onto an existing path` beats `test_rename_2`. One reason to fail per test. A test that cannot fail is not a test.
- **Comments.** Explain why, never what. If the code needs a comment to say what it does, rename something.

## Measured ranges

From `references/patterns/code-patterns.md`: functions under ~40 lines, files under ~400, nesting under 4, parameters under 5. These are signals, not laws; a deliberate exception is fine, an accidental one is a defect.

## Revision moves

When a unit is correct but lifeless, apply the smallest move that changes how it reads:

1. **Cut or merge** — remove the unit or fold it into a neighbour if it does not earn its place.
2. **Structural** — move a rule below the entry point, split a long function, name a seam.
3. **Connective** — replace a generic call with a named one; replace a silent catch with a typed error.
4. **Code** — tighten names, delete dead branches, make the test assert the claim.
5. **Factual** — fix a wrong comment, a stale doc, a misleading error.

Never apply a decorative abstraction to raise a score. A concrete seam beats a generic layer every time.

## Rules

- The implementer writes for the next maintainer, not for the rubric. It never sees the rubric.
- One unit, one job. If a unit needs "and" to describe it, split it.
- Leave the code better than the brief found it, but do not expand the brief.
- Every accepted unit has a test whose output is recorded in the run.
