<!-- skillgrid:start -->
## Skillgrid

This project is configured with Skillgrid. Project: **skillgrid**.

Config: `.skillgrid/config.yaml` (static) — read it before running any Skillgrid skill.
State: `.skillgrid/state.yaml` (dynamic) — where the project is right now (phase, current change, progress).

### Artifacts

| Artifact | Path |
|----------|------|
| Project knowledge (PRD, architecture, terms, ADRs, constraints, research) | `.skillgrid/artifacts/` |
| Project state (phase, current change, progress) | `.skillgrid/state.yaml` |
| Specs (briefing, blueprint, tasks) | `.skillgrid/specs/` |
| Execution ledger | `.skillgrid/sdd/` (gitignored) |

**Domain model:** Before designing or implementing, read `.skillgrid/artifacts/README.md` for the topic index, `01-business-terms.md` + `02-technical-terms.md` for vocabulary, and the ADRs (`04-adr-*.md`, in-force set via `03-adr-index.md`) for the area you're touching. When terms resolve or a hard-to-reverse decision is made, update them via `skillgrid:architectural-decision-records` — the terms files are a glossary and nothing else.

### Rules

- Go 1.22+ minimum to build.
- No new dependencies without an ADR.
- Trivy is advisory-only — findings are reported, never blocking the QA gate.
- Serial development: one change at a time, no parallel branches.
- Conventional commits only; no AI-attribution trailers (commit-msg hook enforces).
- Spec-zone changes commit before code-zone changes (pre-commit zone guard enforces).

### Issue Tracker

Backlog.md — tasks under `.backlog/` (prefix `task`, zero-padded IDs).

### Memory

Mnemonic is active (`skillgrid:mnemonic`) — `mem_save` decisions and `mem_code` for code orientation; data at `~/.skillgrid/mnemonic`.

### Workflow

`brainstorming` → `writing-blueprints` → `slicing` → `ticketing` → execution (`subagent-execution` or `simple-execution`) → `requesting-code-review` → `receiving-code-review`

Run `skillgrid:onboarding` to update config after stack changes.
<!-- skillgrid:end -->
