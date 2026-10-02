<!-- skillgrid:start -->
## Skillgrid

This project is configured with Skillgrid. Project: **skillgrid**.

Config: `.skillgrid/config.yaml` (static) — read it before running any Skillgrid skill.
State: `.skillgrid/state.yaml` (dynamic) — where the project is right now (phase, current change, progress).

### Artifacts

| Artifact | Path |
|----------|------|
| Understanding (facts, in-force ADR index, locked constraints) | `.skillgrid/ASSUMPTIONS.md` |
| Product requirements (reference) | `.skillgrid/artifacts/00-prd.md` |
| Locked decisions (one file each; the index stores the path) | `.skillgrid/artifacts/04-adr-NNNN-slug.md` |
| Live repo/program structure | `.skillgrid/ARCHITECTURE.md` |
| Terms glossary + research + tool-surface docs | `.skillgrid/artifacts/` (index: `README.md`) |
| Feasibility prototypes (permanently retained) | `.skillgrid/prototypes/NNN-name/` |
| Project state (phase, current change, progress) | `.skillgrid/state.yaml` |
| Specs (briefing, blueprint, tasks) | `.skillgrid/specs/` |
| Execution ledger | `.skillgrid/sdd/` (gitignored) |

**Domain model:** Before designing or implementing, read `.skillgrid/ASSUMPTIONS.md` (VERIFIED facts, ADR in-force set, locked constraints) and the terms glossary under `.skillgrid/artifacts/` for vocabulary. When terms resolve or a hard-to-reverse decision is made, update them via `skillgrid:architectural-decision-records`.

**Rules & standards:** locked project constraints render under `### Rules` below. Shared standards are referenced, never inlined — each loads via the skills that apply it.

- **Coding conventions** — reference `.agents/skills/_shared/rules/code-standards.md` for detailed coding conventions.
- **Testing conventions** — reference `.agents/skills/_shared/references/strict-tdd.md` for the TDD cycle and testing conventions.
- **Commits & verification** — reference `.agents/skills/_shared/rules/` for the commit contract, verification ladder, and rigor tiers.
- **Memory, code index & web cache** — reference the `skillgrid:mnemonic` skill (shared rules: `.agents/skills/_shared/rules/mnemonic-memory.md`).

### Rules

*Locked constraints only (source of truth: `### Locked constraints` in `.skillgrid/ASSUMPTIONS.md` — edit there, then mirror here).*

- Go 1.22+ minimum to build.
- No new dependencies without an ADR.
- Trivy is advisory-only — findings are reported, never blocking the QA gate.
- Serial development: one change at a time, no parallel branches.
- Spec-zone changes commit before code-zone changes.
- The repo is the source of truth: when an external tracker, session memory, or dashboard disagrees with a committed artifact, the committed artifact wins.

### Issue Tracker

Backlog.md — tasks under `.backlog/` (prefix `task`, zero-padded IDs).

### Memory

Mnemonic is active — protocol in the `skillgrid:mnemonic` skill; shared rules in `.agents/skills/_shared/rules/mnemonic-memory.md`.

### Workflow

`brainstorming` → `writing-blueprints` → `slicing` → `ticketing` → execution (`subagent-execution` or `simple-execution`) → `qa` → `requesting-code-review` → `receiving-code-review` → `ship` → `reflect`

Run `skillgrid:onboarding` to update config after stack changes.
<!-- skillgrid:end -->
