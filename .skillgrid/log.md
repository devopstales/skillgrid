# Log

Changelog of knowledge-base writes. Newest first. One entry per change that
touches the bundle: what moved, what was authored, what was swept.

## 2026-10-20

- **R6 bundle dissolution (TICKET-14, ADR-0048).** `.skillgrid/artifacts/`
  dissolved into the bundle root: 45 ADRs → `decisions/`, terms →
  `glossary/`, PRD/research/roadmaps/plans → `reference/`,
  `05-locked-constraints.md` → `invariants/`, `README.md` → `reference/`,
  `03-adr-index.md` → `decisions/`. New `workflows/sdd-pipeline.md`
  authored. Root catalogs (`index.md`, `log.md`, `reverse-index.md`) +
  per-dir `index.md` catalogs created. `archive/` renamed to `xarchive/`
  (history intact via `git log --follow`). Reference sweep: `ASSUMPTIONS.md`
  LOCKED rows, `ARCHITECTURE.md`, `config.yaml` path keys, skills, hooks,
  scripts, `AGENTS.md`. No Go behavior change.
