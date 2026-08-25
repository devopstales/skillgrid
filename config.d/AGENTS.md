# AGENTS.md

Global rules for all AI agents (Kilo, OpenCode, Cursor, Claude, Codex, Gemini).

## General

- Concise, direct responses. No preamble, no filler.
- Prefer editing existing files over creating new ones.
- Follow existing code style and conventions before introducing new patterns.
- Never commit secrets or keys.
- Verify work before claiming it is done: run the relevant tests/build/lint.

## Environment

- Tooling managed by aiskillgrid lives in `~/.aiskillgrid`:
  - Binaries: `~/.aiskillgrid/bin`
  - npm packages: `~/.aiskillgrid/node_modules` (binaries in `~/.aiskillgrid/node_modules/.bin`, packages installed with `--prefix "$HOME/.aiskillgrid"`)
- Engram persistent memory is available: update it on decisions, bug fixes, and non-obvious discoveries.

## Conventions

- Commit messages: short imperative, one line (e.g. `fix: merge MCP entries without clobbering keys`).
- Tests: keep unit tests for config merge logic and `--dry-run` output passing.
- YAML config files in `config.d/` are the single source of truth for what gets installed; do not hardcode package names in code.
