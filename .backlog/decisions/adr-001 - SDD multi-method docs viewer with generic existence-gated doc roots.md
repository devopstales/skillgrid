---
id: adr-001
title: SDD multi-method docs viewer with generic, existence-gated doc roots
date: '2026-09-17'
status: accepted
---
## Context

The Skillgrid Web Admin Dashboard (change `2026-09-08-web-admin-dashboard`) ships a
Docs view that renders repo markdown with Mermaid across the SDD methods the team
uses: Skillgrid specs, OpenSpec, SpecKit, Superpowers, and Backlog.md tasks/docs.

The dashboard is a **single Go binary** that must work from *any* repo it is started
in — `skillgrid serve` reads the repo at its working directory. Each of those SDD
methods lives at a *different* canonical on-disk location, and not every repo uses
every method:

- Skillgrid: `.skillgrid/specs/<change>/{briefing,tasks,acceptance}.md`
- OpenSpec: `openspec/specs/**`, `openspec/changes/**`
- SpecKit: `specs/<NNN-feature>/{spec,plan,tasks}.md`, `.specify/memory/constitution.md`
- Superpowers: `docs/superpowers/plans/**`, `docs/superpowers/specs/**`
- Backlog.md: `.backlog/tasks/*.md`, `.backlog/docs/**`, `.backlog/decisions/**`

An early implementation hard-coded a single `.skillgrid/specs` root, which broke
the moment the dashboard was pointed at a repo that used SpecKit or Superpowers
instead. We needed a way to expose whichever methods a repo actually uses, without
the dashboard knowing the repo in advance.

## Decision

The Docs bridge exposes **doc roots as per-method artifact patterns that are
existence-gated**: each root is a list of candidate repo-relative paths, and only
the directories that actually exist in the current repo are walked, resolved, and
searched. The same binary therefore works from any repo using any subset of the
methods, with no configuration.

### Key Requirements

- Roots are declared as `map[string][]string` (selector → candidate paths), not a
  single directory. `GET /docs/tree?root=all` walks every candidate that exists.
- Path resolution checks roots **most-specific first** (in a fixed `mdRootOrder`)
  so a file under `openspec/` resolves to the `openspec` root, not the broader
  `.` catch-all.
- Roots with no markdown return `[]` (never `null`) so the SPA's `nodes.map/reduce`
  never runs on null and renders a graceful empty state.
- Reads stay **sandboxed**: every path is cleaned + prefix-checked against the
  declared roots (`..`/absolute → 400, unknown → 404); read-only; rendered, never
  executed.
- The Skillgrid root shows only spec/task files (briefing/tasks/acceptance);
  research side-products (findings, adr, digests, literature) and research-only
  branches are pruned so the tree stays focused on plan/spec artifacts.
- Each served doc carries a `docType` (task|doc|adr|prd|spec|note) inferred from
  path + frontmatter, and ADRs additionally carry a `decisionStatus`
  (proposed/accepted/rejected/superseded), so the SPA can pick a schema-aware view
  (ADR = Context/Decision/Consequences + status lifecycle chip).

### Rationale

1. **Repo-agnostic**: existence-gating means "works from anywhere" with zero config
   — the dashboard adapts to whatever SDD method(s) the repo uses.
2. **Honest empty states**: a missing method shows "No markdown in this root"
   instead of a crash or a phantom empty tab.
3. **Schema-aware without a DB**: classifying `docType`/`decisionStatus` from
   frontmatter keeps the read path file-based (no index) while still letting the UI
   render ADRs and PRDs distinctly from plain notes.
4. **Sandbox preserved**: the generic roots do not widen the traversal surface —
   the same clean + prefix-check guards every candidate path.

## Consequences

### Positive
- One binary serves Skillgrid, OpenSpec, SpecKit, Superpowers, and Backlog.md
  artifacts from whatever repo it runs in.
- ADRs and PRDs get first-class, schema-aware rendering (status lifecycle,
  Context/Decision/Consequences sections) with no backend schema migration.
- Empty/absent methods degrade to a clean empty state, not an error.

### Negative
- The root list is a fixed vocabulary; a novel method (not in the table) won't be
  surfaced until its candidate paths are added to `mdRoots`.
- Existence-gating means a repo mid-initialization (empty `specs/`) shows nothing
  for that method until real files land.

### Mitigation
- `mdRoots`/`mdRootOrder` are a single declaration; adding a method is one line.
- `SKILLGRID_DOCS_CWD` remains an override for tests and unusual layouts.
- `docType` classification is a conservative heuristic (path + frontmatter); an
  unrecognized file falls back to `note` and still renders as plain markdown.

## References

- Change: `2026-09-08-web-admin-dashboard` (Phase 3 — docs)
- Research: github/spec-kit artifact layout (`.specify/`, `specs/<NNN>/`),
  obra/superpowers (`docs/superpowers/plans|specs`), MrLesk/Backlog.md
  (`backlog/{tasks,docs,decisions}/` ADR + doc-NNN schema).
