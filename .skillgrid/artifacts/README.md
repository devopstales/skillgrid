# Artifacts — Topic Index

Durable project knowledge: what you need to know to work on this project, organized by topic and surviving across changes. This zone replaces the former `docs/PRD.md`, `docs/ARCHITECTURE.md`, `.skillgrid/glossary/`, and `.skillgrid/adr/`.

**Read order at session start:** `.skillgrid/ASSUMPTIONS.md` (product statement, verified facts, the in-force table) → the Record path for a decision you are touching → this index → the terms files. Open `00-prd.md` when a change touches scope or acceptance.

## CANON (do not change without a locked constraint or an ADR)

- `00-prd.md` — product requirements (users, personas, MoSCoW features, flows, NFRs, release line). Reference; `ASSUMPTIONS.md` § VERIFIED wins on conflict.
- `00-architecture.md` — system architecture (components, boundaries, data flow) — created lazily on first use
- `05-locked-constraints.md` — user-locked project-wide boundaries (the "bible")

## VOCABULARY

- `01-business-terms.md` — domain / product / workflow terms (a glossary, nothing else)
- `02-technical-terms.md` — architecture / platform / protocol terms (a glossary, nothing else)

## DECISIONS

The in-force set is the table in `.skillgrid/ASSUMPTIONS.md` § `### In-force set`. Each row's Record column is the path. The body is the file.

- `04-adr-NNNN-slug.md` — one locked decision (global, immutable, `supersedes` trail). `ASSUMPTIONS.md` stores the path only.

## RESEARCH

- `06-research-findings.md` — cross-change distilled research (carried forward from per-change `findings.md`)

## REFERENCE

- `07-mnemonic-tool-surface.md` — planned function areas for `skillgrid-mnemonic` (memory/observations/decisions, code index/semantic/vector, monitoring/session tracking/tool calls, web cache, session handoff) + session-inject design grounded in 5 reference projects
- `08-second-brain-roadmap.md` — capability roadmap for making mnemonic *feel* like a second brain (autonomous capture, `mem_ask` synthesis, knowledge lifecycle, passive learning); claude-os reference mapping + P0–P3 change list
- `09-claude-os-deep-dive.md` — verified deep-dive of the claude-os repo (MCP/SQLite/installer/dashboard/integrations); steal list ranked by "brain feel" per effort; headline: we already beat it on real vec0 + real FTS5

## What is NOT here

- Per-change artifacts (briefing, blueprint, tasks, per-change `findings.md`, the per-change `adr.md` manifest) — those live in `.skillgrid/specs/<topic>/` and are archived with the change.
- Project state (phase, current change, progress) — that is `.skillgrid/state.yaml` (dynamic), not a durable artifact.
- Static config (stack, testing, commands, rules) — that is `.skillgrid/config.yaml`.
