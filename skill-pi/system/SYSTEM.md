# Skillgrid — pi system prompt delta

This file is the skillgrid DELTA only. It is appended to pi's default system
prompt. It does not restate pi's base prompt, and it does not duplicate the
persona or memory-protocol blocks — those live in `AGENTS.md` and are read
directly by pi.

## Persona + memory protocol (pointer, not a copy)

Persona and the persistent-memory protocol are defined in the repo's
`AGENTS.md` — see its persona and engram-protocol blocks. Pi reads
`AGENTS.md` directly for those; this delta only layers the skillgrid
operational rules on top. Do not copy those blocks here (drift).

## Zone discipline

- **Spec zone before code zone.** Changes under `.skillgrid/specs/` commit
  before the code that satisfies them; never both in one commit (the pre-commit
  zone guard enforces this).
- **Serial development.** One change at a time; no parallel branches.
- **Repo is the source of truth** (ADR-0013): when an external tracker, session
  memory, or dashboard disagrees with a committed artifact, the committed
  artifact wins.

## Commit contract

- **Conventional commits only** (`feat(scope): ...`, `fix(scope): ...`, etc.).
  No `Co-Authored-By` or any AI-attribution trailer (commit-msg hook enforces).
- **Atomic + independently-revertable** work-unit commits; one logical change
  per commit (~100 lines soft ceiling).
- **`[skillgrid-context]` block** in every work-unit commit body (Task /
  Decisions / Remaining / Tried) — it feeds the session event stream and keeps
  the `task → commit → diff` chain recoverable from git alone. A commit that
  closes a Backlog task names that task ID in the block.
- Commit **after** a verified gate is green, **before** any long
  install/build command.
