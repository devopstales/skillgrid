# Artifacts — Topic Index

Durable project knowledge: what you need to know to work on this project, organized by topic and surviving across changes. This zone replaces the former `docs/PRD.md`, `docs/ARCHITECTURE.md`, `.skillgrid/glossary/`, and `.skillgrid/adr/`.

**Read order at session start:** this index → `05-locked-constraints.md` (the hard boundaries) → the terms files and ADR index for the area you're touching.

## CANON (do not change without a locked constraint or an ADR)

- `00-prd.md` — product requirements (scope, features, target state)
- `00-architecture.md` — system architecture (components, boundaries, data flow) — created lazily on first use
- `05-locked-constraints.md` — user-locked project-wide boundaries (the "bible")

## VOCABULARY

- `01-business-terms.md` — domain / product / workflow terms (a glossary, nothing else)
- `02-technical-terms.md` — architecture / platform / protocol terms (a glossary, nothing else)

## DECISIONS

- `03-adr-index.md` — ADR index (in-force set: number, title, status, supersedes, gist)
- `04-adr-NNNN-slug.md` — ADR records (global, immutable, `supersedes` trail). Read the index first.

## RESEARCH

- `06-research-findings.md` — cross-change distilled research (carried forward from per-change `findings.md`)

## REFERENCE

- `07-mnemonic-tool-surface.md` — planned function areas for `skillgrid-mnemonic` (memory/observations/decisions, code index/semantic/vector, monitoring/session tracking/tool calls, web cache, session handoff) + session-inject design grounded in 5 reference projects

## What is NOT here

- Per-change artifacts (briefing, blueprint, tasks, per-change `findings.md`, the per-change `adr.md` manifest) — those live in `.skillgrid/specs/<topic>/` and are archived with the change.
- Project state (phase, current change, progress) — that is `.skillgrid/state.yaml` (dynamic), not a durable artifact.
- Static config (stack, testing, commands, rules) — that is `.skillgrid/config.yaml`.
