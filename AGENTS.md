# Repository Instructions

This is the **real source of truth for the operational rules** a change must respect — the hard limits a build, commit, or change must follow. The onboarding skill renders the `<!-- skillgrid:start -->` … `<!-- skillgrid:end -->` block below into this file.

**What belongs here:** operational rules (toolchain floor, commit format, dependency policy, spec/code zone order, enforcement hooks). **What does NOT belong here:** design *decisions* — those are ADRs, indexed in `.skillgrid/ASSUMPTIONS.md` § `LOCKED` with their bodies in `.skillgrid/artifacts/04-adr-NNNN-slug.md`. If a rule is really "we decided X over Y because Z," it is an ADR, not an operational rule.

**What this is:** skillgrid — a local-first engine for agent memory + code intelligence. Single Go binary, three transports (CLI, MCP stdio, HTTP/REST + embedded SPA). Decisions, observations, and code context persist outside the chat so AI coding agents keep working across sessions. See `.skillgrid/ASSUMPTIONS.md` § VERIFIED for the full product + stack description.

<!-- skillgrid:start -->
## Skillgrid

This project is configured with Skillgrid. Project: **skillgrid-v2**.

Config: `.skillgrid/config.yaml` (static) — read it before running any Skillgrid skill.
State: `.skillgrid/state.yaml` (dynamic) — where the project is right now (phase, current change, progress).

### Artifacts

| Artifact | Path |
|----------|------|
| Project knowledge (PRD, architecture, terms, ADRs, research) | `.skillgrid/artifacts/` |
| Project state (phase, current change, progress) | `.skillgrid/state.yaml` |
| Specs (briefing, blueprint, tasks) | `.skillgrid/specs/` |
| Execution ledger | `.skillgrid/sdd/` (gitignored) |

**Domain model:** read the vocabulary + ADRs under `.skillgrid/artifacts/` (index: `README.md`) before designing or implementing. The ADR index is `.skillgrid/ASSUMPTIONS.md` § `LOCKED`; open the linked file for the full Context / Decision / Consequences.

**Rules & standards:** the operational rules live under `### Rules` below. Shared standards are referenced, never inlined — each loads via the skills that apply it.

- **Coding conventions** — reference `.agents/skills/_shared/rules/code-standards.md` for detailed coding conventions.
- **Testing conventions** — reference `.agents/skills/_shared/references/strict-tdd.md` for the TDD cycle and testing conventions.
- **Commits & verification** — reference `.agents/skills/_shared/rules/commits.md` for the commit contract, `.agents/skills/_shared/rules/verification-ladder.md` for the verification ladder, and `.agents/skills/_shared/planning/rigor-tiers.md` for rigor tiers.
- **Memory, code index & web cache** — reference the `skillgrid:mnemonic` skill (shared rules: `.agents/skills/_shared/rules/mnemonic-memory.md`, `.agents/skills/_shared/rules/mnemonic-artifacts.md`, `.agents/skills/_shared/rules/mnemonic-code-indexing.md`).

### Rules

*Operational rules only — the real source of truth is this section (edit here). Design decisions do not live here; they are ADRs in `.skillgrid/ASSUMPTIONS.md` § `LOCKED` → `.skillgrid/artifacts/04-adr-NNNN-slug.md`.*

- **Go 1.22+ minimum to build** (ADR-0031). A `go.mod` below 1.22 is a drift signal; raising the floor is a new superseding ADR.
- **No new dependencies without an ADR.** Adding a third-party dependency is a decision; record it in `.skillgrid/ASSUMPTIONS.md` § `LOCKED` before it lands (ADR-0008 is the `yaml`-package precedent).
- **Trivy is advisory-only** — findings are reported, never blocking the QA gate.
- **Serial development:** one change at a time, no parallel branches.
- **Conventional commits only;** no AI-attribution trailers (commit-msg hook enforces).
- **Spec-zone changes commit before code-zone changes** (pre-commit zone guard enforces).
- **The repo is the source of truth** (ADR-0013): when an external tracker, session memory, or dashboard disagrees with a committed artifact, the committed artifact wins. A commit that closes a Backlog task names that task ID in the `[skillgrid-context]` block so the `task → commit → diff` chain is recoverable from git alone.

### Issue Tracker

Backlog.md — tasks in `.skillgrid/tasks/`, tracked via the `backlog` MCP tools.

### Memory

Mnemonic (local SQLite + FTS5): `mem_*`, `code_*`, `web_*` over MCP and HTTP.

### Workflow

`brainstorming` → `writing-blueprints` → `slicing` → `ticketing` → execution (`subagent-execution` or `simple-execution`) → `qa` → `requesting-code-review` → `receiving-code-review` → `ship` → `reflect`

Run `skillgrid:onboarding` to update config after stack changes.
<!-- skillgrid:end -->
