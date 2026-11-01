# Artifacts — Topic Index

Durable project knowledge: what you need to know to work on this project, organized by topic and surviving across changes. This zone replaces the former `docs/PRD.md`, `docs/ARCHITECTURE.md`, `.skillgrid/glossary/`, and `.skillgrid/adr/`.

**Read order at session start:** `.skillgrid/ASSUMPTIONS.md` (product statement, verified facts, the in-force table) → the File path for a decision you are touching → this index.

> 2026-10-31: the old-mnemonic-v1 product zone was deleted for the mnemonic-v2 reset — `00-prd.md`, the `01`/`02` terms glossaries, `06-spec-second-brain-learnings.md`, `07`/`08`/`09` reference docs, and 27 ADRs (see `03-adr-index.md`). Their vocabulary and requirements return with the mnemonic-v2 specs (`.skillgrid/specs/2026-10-31-mnemonic-v2-*/`). The v1 PRD and glossaries remain recoverable in git history and `.skillgrid/archive/`.

## CANON (do not change without a locked constraint or an ADR)

- `05-locked-constraints.md` — user-locked project-wide boundaries (the "bible")
- `00-prd.md` — to be re-created by the mnemonic-v2 specs (created lazily on first use)
- `00-architecture.md` — system architecture (components, boundaries, data flow) — created lazily on first use

## VOCABULARY

- `01-business-terms.md` / `02-technical-terms.md` — to be re-created by the mnemonic-v2 specs (a glossary, nothing else)

## DECISIONS

The in-force set is the table in `.skillgrid/ASSUMPTIONS.md` § `### In-force set`. Each row's Record column is the path. The body is the file.

- `04-adr-NNNN-slug.md` — one locked decision (global, immutable, `supersedes` trail). `ASSUMPTIONS.md` stores the path only.

## RESEARCH

- `06-research-findings.md` — cross-change distilled research (carried forward from per-change `findings.md`)

## REFERENCE

- *(empty — the old-mnemonic reference docs `07`/`08`/`09` were deleted 2026-10-31 with the rest of mnemonic-v1; mnemonic-v2 reference material lands under `.skillgrid/specs/2026-10-31-mnemonic-v2-*/`)*

## What is NOT here

- Per-change artifacts (briefing, blueprint, tasks, per-change `findings.md`, the per-change `adr.md` manifest) — those live in `.skillgrid/specs/<topic>/` and are archived with the change.
- Project state (phase, current change, progress) — that is `.skillgrid/state.yaml` (dynamic), not a durable artifact.
- Static config (stack, testing, commands, rules) — that is `.skillgrid/config.yaml`.
