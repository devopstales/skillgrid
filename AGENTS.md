<!-- skillgrid:start -->
## Skillgrid

This project is configured with Skillgrid. Project: **skillgrid**.

Config: `.skillgrid/config.yaml` (static) — read it before running any Skillgrid skill.
State: `.skillgrid/state.yaml` (dynamic) — where the project is right now (phase, current change, progress).

### Artifacts

| Artifact | Path |
|----------|------|
| Understanding + decisions (PRD facts, ADRs, locked constraints) | `.skillgrid/ASSUMPTIONS.md` |
| Live repo/program structure | `.skillgrid/ARCHITECTURE.md` |
| Terms glossary (business + technical) | `.skillgrid/artifacts/01-business-terms.md`, `02-technical-terms.md` |
| Research findings + tool-surface docs | `.skillgrid/artifacts/06-research-findings.md`, `07-mnemonic-tool-surface.md` |
| Feasibility spikes (permanently retained) | `.skillgrid/spikes/NNN-name/` |
| Project state (phase, current change, progress) | `.skillgrid/state.yaml` |
| Specs (briefing, blueprint, tasks) | `.skillgrid/specs/` |
| Execution ledger | `.skillgrid/sdd/` (gitignored) |

**Domain model:** Before designing or implementing, read `.skillgrid/ASSUMPTIONS.md` (VERIFIED facts, ADR in-force set, locked constraints), the terms glossary (`artifacts/01-business-terms.md` + `02-technical-terms.md`) for vocabulary, and `.skillgrid/ARCHITECTURE.md` for the structure you're touching. When terms resolve or a hard-to-reverse decision is made, update them via `skillgrid:architectural-decision-records` (ADRs are `### ADR-NNNN` entries inside `ASSUMPTIONS.md`; the terms files are a glossary and nothing else).

### Rules

*Rendered from `### Locked constraints` in `.skillgrid/ASSUMPTIONS.md` — that file is the source of truth; edit the constraint there, then mirror it here.*

- Go 1.22+ minimum to build.
- No new dependencies without an ADR.
- Trivy is advisory-only — findings are reported, never blocking the QA gate.
- Serial development: one change at a time, no parallel branches.
- Conventional commits only; no AI-attribution trailers (commit-msg hook enforces).
- Spec-zone changes commit before code-zone changes (pre-commit zone guard enforces).
- The repo is the source of truth (ADR-0013): when an external tracker, session memory, or dashboard disagrees with a committed artifact, the committed artifact wins. A commit that closes a Backlog task names that task ID in the `[skillgrid-context]` block so the `task → commit → diff` chain is recoverable from git alone.
- Session-inject uses a two-layer mechanism: (1) auto-prepend a slim token-capped L1 summary only on resume (not fresh sessions), and (2) an on-demand `mem_inject_session` tool for deeper BM25/semantic retrieval. See `.skillgrid/artifacts/07-mnemonic-tool-surface.md`.
- Session-inject privacy: tag-by-default — secrets, full local paths, and `private`-tagged content are auto-excluded from injection; project-relative paths, commit SHAs, tool names, and task IDs are included by default.
- Session-inject scope: project-scoped by default; cross-project injection only when the agent explicitly passes `all_projects: true` (reuses the existing `mem_search` flag).
- Session-inject selection is hybrid (BM25 + semantic, RRF-fused) by default. The vector leg degrades to BM25-only when no embedder is active (reuses the existing degrade-to-Null design); BM25-only is a degraded state, not the target model.

### Issue Tracker

Backlog.md — tasks under `.backlog/` (prefix `task`, zero-padded IDs).

### Memory

Mnemonic is active (`skillgrid:mnemonic`) — `mem_save` decisions and `mem_code` for code orientation; data at `~/.skillgrid/mnemonic`.

### Workflow

`brainstorming` → `writing-blueprints` → `slicing` → `ticketing` → execution (`subagent-execution` or `simple-execution`) → `qa` → `requesting-code-review` → `receiving-code-review`

Run `skillgrid:onboarding` to update config after stack changes.
<!-- skillgrid:end -->
