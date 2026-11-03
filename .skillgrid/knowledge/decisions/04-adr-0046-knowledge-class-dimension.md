---
type: TechnicalDecision
title: "Knowledge class dimension: nullable knowledge_class column (episodic|convention|doc|unclassified) on observations, heuristic inference on save, re-runnable backfill, optional search filter (FTS-leg-only), migration 070"
status: "accepted"
supersedes: none
date: 2026-10-18
---

# Knowledge Class Dimension
**Related:** `04-adr-0012-sqlite-as-second-brain.md` (this dimension is
ADDITIVE on that store — it does not introduce a second source of truth).

## Context and Problem Statement

Mnemonic observations carry `type` (the coarse skill taxonomy: decision,
convention, session_log, discovery, ...) and `scope` (project/user/global),
but no **durability** dimension. An agent's "second brain" holds two
fundamentally different kinds of content in the same table:

- **episodic** knowledge — session-specific notes ("found a parser bug
  today"), which decay with the session;
- **convention** knowledge — durable rules ("we use token rotation for
  auth"), which should outlive every session that produced them.

Without a label, retrieval cannot distinguish the two: a search for "auth"
ranks this week's session log next to the standing decision that it
describes, and the caller has to re-derive durability from the type string
every time. The dimension should be cheap (inferred, not curated), optional
(existing rows and unfiltered reads keep their exact behavior), and
re-runnable (a backfill can be invoked any number of times).

## Considered Options

- **KB-per-class (separate stores per durability class):** clean separation,
  but a fourth class means a fourth store, four migration surfaces, and a
  split source of truth — directly at odds with ADR-0012 (SQLite as the
  single source of truth) and with the project-scoped store model.
- **A nullable `knowledge_class` column + type-driven heuristic (chosen):**
  one column on the live `observations_core` table; inference at save time
  (the heuristic is non-load-bearing — it labels, it does not gate); a
  re-runnable backfill for pre-existing NULL rows; an optional exact-match
  search filter. Additive, no behavior change unfiltered, single source of
  truth preserved.
- **Reuse `type` as the durability axis:** no new column, but `type` is an
  OPEN taxonomy (14 values, MCP tool aliases) whose semantics are
  "what kind of observation is this", not "how durable is this". Overloading
  it would make every type addition a durability decision.

## Decision Outcome

Chosen: a nullable `knowledge_class` column
(`episodic | convention | doc | unclassified`) on `observations_core`
(migration `070_add_knowledge_class.sql`, one-way — no down-migration), with:

1. **Heuristic inference on save/update** — `inferKnowledgeClass(type)`:
   `decision, convention, architecture, config, standing → convention`;
   `session_log, learning, discovery, bugfix → episodic`;
   `doc, prd, research, roadmap → doc`; anything else / empty →
   `unclassified`. A caller-supplied `knowledge_class` is validated against
   the closed four-class set and is never overwritten by inference.
2. **Re-runnable backfill** — `BackfillKnowledgeClass` is a pure
   `UPDATE ... WHERE project = ? AND knowledge_class IS NULL`; the
   `IS NULL` predicate is the idempotency guard, so every run after the
   first is a no-op. It is called explicitly (not from the migration
   runner), mirroring the store-level backfill precedent (the 062 pre-split
   backfill).
3. **Optional search filter** — `SearchWithKnowledgeClass` adds an exact-
   match `WHERE` fragment only when a class is supplied; an empty class
   leaves the query byte-identical to the pre-0046 shape (the
   no-regression contract). Every hit on this path carries its per-row
   class label (NULL surfaced as `unclassified`).
4. **Additive read contract** — the column is appended to the compat view
   and to the hot-read / view column lists; the 31-column
   `scanObservations` contract grows by one trailing `NullString`, and
   `Observation.KnowledgeClass` is `omitempty`, so pre-backfill rows keep
   the existing JSON contract.

The dimension is NOT a KB-per-class: all four classes live in the same
store, so ADR-0012 (single source of truth) holds unchanged.

## Consequences

- **Additive, no behavior change unfiltered.** The default `mem_search`
  path is the pre-0046 blend; the class filter is opt-in per call.
- **Backfill is re-runnable and side-effect-free on already-labeled
  rows.** Safe to call from any init path, any number of times.
- **The heuristic is non-load-bearing.** It may be wrong (e.g. a
  `session_log` row that is actually durable); a caller can always
  override with an explicit `knowledge_class`. The class is a label for
  retrieval, not a routing decision — nothing gates on it.
- **One-way migration.** `070` adds the column, rebuilds the compat view
   and the three INSTEAD OF triggers (they name explicit column lists, so
   they must carry the new column), and creates
   `idx_obs_knowledge_class (knowledge_class, project)`. There is no
   down-migration; the pre-0046 schema is recoverable only from a pre-070
   dump.
- **Search filter is FTS-leg only.** The class-filtered path rides the
   keyword leg (no vector fusion) — the class is a structural label, and a
   class-scoped search is advisory. A vector leg can be added later by
   extending `SearchByVector` with the same clause if precision requires it.
- **The compat view grows to 41 columns.** Pre-0038 consumers that
   `SELECT *` see one extra trailing NULL column; the parity test dumps an
   explicit column list (minus `knowledge_class`) to stay valid across the
   one-way dimension.
