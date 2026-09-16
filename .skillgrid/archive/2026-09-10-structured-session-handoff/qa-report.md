# QA Plan — 006-structured-session-handoff

**Gate**: human QA must be ACCEPTED or explicitly WAIVED before `sdd-archive`.
**Runtime proof**: all 16 `@step-NN` scenarios passed at runtime (`store`/`relay`/`mcp`/`cmd/skillgrid` `-race` ok; 005/008/010/011/013 baselines ok — one pre-existing flaky 011 pdg wall-clock timeout, passes alone).

## What a human should exercise

The CLI is the operator surface (`skillgrid session ...`). The MCP tools (`session_handoff` / `session_resume` / `session_status` / `knowledge_compact`) are the agent surface and mirror the same store.

### Happy
1. **Handoff**: `skillgrid session handoff --session-id <active-session> --progress "..." --knowledge "..." --next-prompt "..."` → exits 0; prints `handoff_id` + paths; `.skillgrid/.cleave/{PROGRESS,KNOWLEDGE,NEXT_PROMPT}.md` exist; a `session_handoffs` row is recorded.
2. **Resume**: `skillgrid session resume --handoff-id <id>` → prints the NEXT_PROMPT (the resume prompt) + handoff_id.
3. **Resume + archive**: `skillgrid session resume --handoff-id <id> --archive` → prints the prompt + archive_id; the handoff status flips to `archived`; a `session_archives` row exists.
4. **Status**: `skillgrid session status` → prints `handoff_count` (>=1) (+ optional cost/context only when supplied).
5. **MCP parity**: drive `session_handoff` via MCP, then `session_resume`/`session_status` — outcomes match the CLI on the same store.

### Edge
6. **Resume unknown id**: `skillgrid session resume --handoff-id no-such-id` → non-zero exit; clear "unknown handoff id" error; no invented prompt.
7. **Resume missing `.cleave/`**: delete the `.cleave` bundle for a handoff, then resume → non-zero exit; clear "missing cleave" error; no invented prompt.
8. **Status with no handoffs**: fresh store, `skillgrid session status` → `handoff_count: 0`, no crash.
9. **Compact empty**: `knowledge_compact` on a store with no handoffs/notes → a minimal (non-empty) `KNOWLEDGE.md`, no error.

### Failure
10. **Bad flags / missing id**: `skillgrid session handoff --session-id <id>` (no `--next-prompt`) → non-zero exit (2); stderr names the missing flag; NO partial `.cleave/` bundle left behind.
11. **No usable store**: point at a store path that can't be opened (e.g. a file where a dir is expected) → non-zero exit (1); clear stderr; no partial bundle.
12. **Fail-closed (no orphan)**: make the `.cleave/` dir unwritable (e.g. `chmod 555` the parent) then handoff → non-zero exit; NO `session_handoffs` row written (the row is only written after the files succeed).
13. **Watchdog off by default**: with `SKILLGRID_HANDOFF_WATCHDOG` unset, the watchdog check is a no-op — never auto-hands-off.
14. **Watchdog enabled + past threshold**: `SKILLGRID_HANDOFF_WATCHDOG=1 SKILLGRID_HANDOFF_WATCHDOG_THRESHOLD=0.8` + usage 0.95 → a handoff runs (row + 3 cleave files) via the same path.
15. **Watchdog invalid config**: `SKILLGRID_HANDOFF_WATCHDOG=1 SKILLGRID_HANDOFF_WATCHDOG_THRESHOLD=abc` (or `<0` / `>1`) → clear config error; NO auto-handoff.

## Environments / data / accounts
- macOS, Go toolchain. A temp project store (use a `t.TempDir()`-style scratch or a throwaway `MNEMONIC_PROJECT` + data dir so you don't pollute a real store). An active session id for handoff.
- `.skillgrid/.cleave/` is gitignored by default — confirm it does NOT appear in `git status`.

## Pass / fail criteria
- **PASS**: all 15 items behave as specified (exit codes, stderr messages, file/row presence, no partial bundles, no orphans, off-by-default).
- **FAIL**: any orphan `session_handoffs` row without files; an invented resume prompt; a crash on empty status/compact; the watchdog auto-handing-off when disabled/default/invalid; `.cleave/` showing in `git status`.

## How to waive
If the CLI/MCP behavior is covered by the runtime suite (it is — 16/16 scenarios COMPLIANT, `-race` clean) and the change is additive (new `relay` package + 019 migration + 4 session MCP tools + `session` CLI, no behavior change to existing tools), the operator may **waive** hands-on QA with a one-line rationale, since the same code paths are exercised at runtime by the tests. Waiver is acceptable here because the load-bearing invariants (fail-closed no-orphan, off-by-default watchdog, additive surface) are all asserted by passing runtime tests.
