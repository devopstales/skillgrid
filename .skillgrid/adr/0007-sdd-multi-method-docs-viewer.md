# SDD multi-method docs viewer with generic, existence-gated doc roots

---
status: "accepted"
supersedes: none
date: 2026-09-17
---

## Context and Problem Statement

The Web Admin Dashboard (change `2026-09-08-web-admin-dashboard`) ships a Docs
view that renders repo markdown with Mermaid across the SDD methods the team
uses: Skillgrid specs, OpenSpec, SpecKit, Superpowers, and Backlog.md
tasks/docs/decisions.

`skillgrid serve` is a **single Go binary** that must work from *any* repo it is
started in — it reads the repo at its working directory. Each of those SDD
methods lives at a *different* canonical on-disk location, and not every repo
uses every method:

- Skillgrid: `.skillgrid/specs/<change>/{briefing,tasks,acceptance}.md`,
  `.skillgrid/adr/*.md`, `.skillgrid/prd/*.md`
- OpenSpec: `openspec/specs/**`, `openspec/changes/**`
- SpecKit: `specs/<NNN-feature>/{spec,plan,tasks}.md`, `.specify/memory/constitution.md`
- Superpowers: `docs/superpowers/plans/**`, `docs/superpowers/specs/**`
- Backlog.md: `.backlog/tasks/*.md`, `.backlog/docs/**`, `.backlog/decisions/**`

An early implementation hard-coded a single `.skillgrid/specs` root, which broke
the moment the dashboard was pointed at a repo that used SpecKit or Superpowers
instead. We needed a way to expose whichever methods a repo actually uses,
without the dashboard knowing the repo in advance.

## Considered Options

- **Single hard-coded root** — serve only `.skillgrid/specs`. Simple, but the
  dashboard is useless for a repo that uses any other method.
- **Per-repo config file** — declare roots in a config the operator writes.
  Explicit, but it adds a config file and "works from anywhere" degrades to
  "works once configured."
- **Existence-gated generic roots** — declare a fixed vocabulary of
  per-method candidate paths; walk only the directories that exist in the
  current repo.

## Decision Outcome

Chosen option: **Existence-gated generic roots.** The Docs bridge declares its
roots as `map[string][]string` (selector → candidate repo-relative paths), and
only the directories that actually exist in the current repo are walked,
resolved, and searched. The same binary therefore works from any repo using any
subset of the methods, with no configuration.

Key requirements:

- Path resolution checks roots **most-specific first** (in a fixed
  `mdRootOrder`) so a file under `openspec/` resolves to the `openspec` root,
  not the broader `.` catch-all.
- Roots with no markdown return `[]` (never `null`) so the SPA's
  `nodes.map/reduce` never runs on null and renders a graceful empty state.
- Reads stay **sandboxed**: every path is cleaned + prefix-checked against the
  declared roots (`..`/absolute → 400, unknown → 404); read-only; rendered,
  never executed.
- The Skillgrid root is `.skillgrid` and shows spec/task files **plus**
  first-class decision (`adr/`) and product-doc (`prd/`) artifacts; research
  side-products (digests, literature, findings, notes, assets) are pruned.
- Each served doc carries a `docType` (task|doc|adr|prd|spec|note) inferred
  from path + frontmatter, and ADRs additionally carry a `decisionStatus`
  (proposed/accepted/rejected/superseded), so the SPA renders a schema-aware
  view (ADR = lifecycle chip + Context/Decision/Consequences; PRD/Doc = type
  chips; Task = acceptance + DoD checklists).

### Consequences

- Good, because one binary serves Skillgrid, OpenSpec, SpecKit, Superpowers,
  and Backlog.md artifacts from whatever repo it runs in, with zero config.
- Good, because ADRs and PRDs get first-class, schema-aware rendering (status
  lifecycle, Context/Decision/Consequences) with no backend schema migration.
- Good, because empty/absent methods degrade to a clean empty state, not an
  error or a crash.
- Tension: the root vocabulary is fixed; a novel method not in the table won't
  be surfaced until its candidate paths are added to `mdRoots`.
- Tension: existence-gating means a repo mid-initialization (empty `specs/`)
  shows nothing for that method until real files land.
- Mitigation: `mdRoots`/`mdRootOrder` are a single declaration (adding a method
  is one line); `SKILLGRID_DOCS_CWD` remains an override for tests and unusual
  layouts; `docType` classification is a conservative heuristic (unrecognized
  files fall back to `note` and still render as plain markdown).

## References

- Change: `2026-09-08-web-admin-dashboard` (Phase 3 + 3b — docs)
- Research: MrLesk/Backlog.md backlog schema (`.backlog/{tasks,docs,decisions}/`,
  ADR + doc-NNN pattern), github/spec-kit artifact layout (`.specify/`,
  `specs/<NNN>/`), obra/superpowers (`docs/superpowers/plans|specs`).
