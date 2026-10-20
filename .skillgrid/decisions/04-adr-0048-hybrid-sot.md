---
type: TechnicalDecision
title: "Hybrid Source of Truth"
status: "accepted"
supersedes: none
date: 2026-10-20
---

# Hybrid Source of Truth
**Amends:** `04-adr-0012-sqlite-as-second-brain.md` (adds the Markdown
bundle as the durable SoT for the knowledge base; SQLite remains the SoT
for observations and code intelligence — this is an ADDITION to the store,
not a second retrieval index over the same data).

## Context and Problem Statement

ADR-0012 (SQLite as the second brain) rejected llm-wiki-style Markdown
files as the *retrieval* engine: the store is the engine, and every
observation, decision reference, and code fact flows through the SQLite
FTS/vector path. That decision held for **episodic** knowledge —
observations, session logs, discoveries — and for **code intelligence**.

But durable, *curated* project knowledge had a different shape: ADR bodies,
locked constraints, glossaries, research distillations, roadmaps, and plans
lived as Markdown files under `.skillgrid/artifacts/` and were only
*referenced* by the store, never *indexed* as first-class content. The
result was a split: the repo's Markdown was the durable record, SQLite was
the live retrieval engine, and the two were connected only by pointer
rows. The layout (`artifacts/` as a flat topic zone, `archive/` for closed
changes) served that hybrid but was invisible to the OKF v0.2 contract —
the knowledge was not a bundle: no `index.md`, no `log.md`, no
`reverse-index.md`, no per-concept frontmatter.

The second-brain-learnings change (2026-10-06) introduced OKF
(TICKET-09 through TICKET-14) and named the target: an OKF bundle at the
`.skillgrid/` root with typed concepts and committed catalogs. This ADR
records the hybrid SoT that the R6 dissolution implements, so the layout
is a *decision* (reversible by supersession) rather than a migration
artifact.

## Considered Options

- **SQLite-only SoT (ADR-0012 as-is):** every durable concept ingested into
  observations with a `knowledge_class` (ADR-0046). Retrieval is uniform,
  but curation (edits, supersedes trails, git history) lives in the store,
  not the repo — violating ADR-0013 (the repo is the source of truth) for
  the curated tier.
- **Markdown-only SoT (llm-wiki style):** everything in the bundle, SQLite
  demoted to an index. Curation is perfect, but the live memory (session
  observations, code facts) loses its engine — ADR-0012's core finding.
- **Hybrid SoT (chosen):** the two SoTs are *disjoint by content kind*.
  SQLite (mnemonic) is the SoT for **observations** and **code
  intelligence** — the high-volume, machine-written, retrieval-first tier.
  The `.skillgrid/` **Markdown bundle** is the SoT for the **knowledge
  base** — `decisions/` (ADRs), `invariants/` (locked constraints),
  `glossary/`, `reference/` (PRD, research, roadmaps, plans), `workflows/`
  — the curated, human-written, git-history-bearing tier. `xarchive/`
  holds closed change folders (immutable). The bundle is the durable
  record; SQLite is the retrieval engine over the live tier and an index
  (via `mem_save` ingest) into the durable tier — never the writer of
  durable concepts.

## Decision Outcome

Chosen: **hybrid source of truth**, disjoint by content kind:

1. **SQLite (mnemonic) is the SoT for observations + code
   intelligence.** Unchanged from ADR-0012. Observations (bitemporal,
   ADR-0011), web cache, and the code index live in the per-project store;
   retrieval is FTS/vector over that store.
2. **The `.skillgrid/` Markdown bundle is the SoT for the knowledge
   base.** `decisions/` (ADRs, the in-force set indexed by
   `ASSUMPTIONS.md` § LOCKED — ADR-0019 unchanged: the index holds the
   path, the file holds the body), `invariants/`, `glossary/`,
   `reference/`, `workflows/`. Concepts carry OKF v0.2 frontmatter; the
   bundle carries committed catalogs (`index.md`, `log.md`,
   `reverse-index.md`, per-dir `index.md`); `okf-lint.mjs` guards drift.
3. **`xarchive/` holds closed changes.** Renamed from `archive/`
   (2026-10-20) so the SDD archive is not confused with the knowledge
   bundle; closed folders are immutable. `git log --follow` carries
   history through both the artifact dissolution and this rename.
4. **The store indexes the bundle; it does not own it.** Durable concepts
   are ingested into the store (e.g. `knowledge_class: doc`, ADR-0046) for
   recall, but the Markdown file is the writer and the SoT — a store row
   that disagrees with the file loses (ADR-0013).
5. **Layout:** the bundle root is `.skillgrid/` itself (it carries
   `index.md`); `artifacts/` no longer exists. The OKF lint scopes to the
   bundle root when it carries `index.md`, with the `knowledge/` and
   pre-R6 `artifacts/` layouts kept for other projects.

## Consequences

- **One reversible unit.** The layout is a decision with a file:
  superseding it (e.g. a `knowledge/` subdir, or SQLite-only) is a new
  ADR + table flip, never an in-place edit of an accepted ADR (IRON RULE).
- **Curation stays in git.** ADR supersession, constraint unlocking, and
  glossary growth are commits with history — the store row is a derived
  index, re-derivable from the Markdown.
- **Retrieval stays uniform for the live tier.** Nothing about the
  observation/code path changes; agents recall the durable tier through
  the same `mem_search` surface (fused, ADR-0047), with the bundle file as
  the cited source.
- **Drift is guarded.** `okf-lint.mjs` (TICKET-11) exits non-zero on
  missing catalogs, broken links, or stale reverse-index rows, and the
  QA gate runs it; `scripts/check-adr-invariants.mjs` keeps the
  ADR file ↔ LOCKED row bijection.
- **The ADR-0013 chain is preserved.** Every LOCKED row points at a
  bundle path (`decisions/04-adr-NNNN-slug.md`); the task → commit → diff
  chain is recoverable from git alone, including through this move.
