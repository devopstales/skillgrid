# Definition of Done

A task is done only when ALL of the following are true. If any item fails, the task is in progress — say so.

## Code

- Implements exactly what was requested — no extra features, no speculative abstractions.
- Follows existing code style and conventions.
- Every changed line traces back to the request.
- No dead code introduced, no secrets or keys committed.
- Related tests added or updated covering the new behavior and the bug it fixes.

## Verification

- Build passes (run it, don't assume).
- Tests pass (run them, confirm output).
- Lint passes if the project has linters.
- For CLI tools: the exact command the user would run was executed and its exit code + output checked.

## Knowledge

- Decisions, bug fixes, and non-obvious discoveries saved to Engram memory.
- Session summary written if the session is ending.

## Communication

- Summary states what changed, how it was verified, and any caveats — evidence, not claims.
- Anything not done is named explicitly, with the reason.
