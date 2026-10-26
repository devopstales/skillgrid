---
type: TechnicalDecision
title: "Global operational memory: the cross-project fact store (__global__)"
status: "accepted"
supersedes: none
date: 2026-10-24
---

# Global operational memory: the cross-project fact store (__global__)
**Amends:** `04-adr-0012-sqlite-as-second-brain.md` (extends the SQLite
second-brain store with a cross-project fact tier; SQLite remains the engine).
**Related:** the `2026-10-24-global-llm-wiki` spec (Change A) and its
blueprint threat matrix; `04-adr-0051-llm-wiki-pipeline` (Change B, not yet authored) is the
sister change that builds the wiki pipeline over this store's layout.

## Context and Problem Statement

ADR-0012 made SQLite (mnemonic) the second-brain engine: observations,
decisions, and code facts flow through the per-project store's FTS/vector
path. That store is **per-project** — each project resolves to its own
`<project>.sqlite`. Operational knowledge that holds *across* projects had
nowhere to live: the user's preferences, tool behaviors, and environment
facts that are true for the human, not for any one repo. Agents re-learn them
per project, or an operator copy-pastes them into every project's memory.

The Hermes agent's memory pattern (two bounded files: MEMORY.md at 2,200
chars for operational notes, USER.md at 1,375 chars for the user profile,
injected as a frozen snapshot at session start, with a write-approval queue
for passive writes) showed the shape that works: small, bounded, injected,
and human-auditable. The open question: where does the cross-project tier
live, and how does it route, search, and get injected without a second
source of truth?

## Considered Options

- **Markdown files (llm-wiki style) as the cross-project tier:** curation is
  git-native, but ADR-0012's core finding still applies — a second retrieval
  surface over the same data, token-cost proportional to file size, and a
  sync path between the file and the store. The sister change (Change B,
  ADR-0051) owns the Markdown wiki pipeline; the operational tier needs the
  live engine, not a second one.
- **A new database (PostgreSQL/Redis) for cross-project memory:** access
  control and scale come free, but the tool is local-first and single-user
  (ADR-0012's premise); a new server process is a new failure mode for a
  tier that must be fail-open.
- **A `__global__` pseudo-project in the existing SQLite store (chosen):**
  one engine, one migration path, one retrieval surface. The cross-project
  tier is a project that every project can route to. The store file is
  `__global__.sqlite` under the same data dir; it is invisible to
  `ListProjects` and `RunTTLExpiry` (it is curated explicitly, not by TTL).

## Decision Outcome

Chosen: **the `__global__` pseudo-project fact store**, a cross-project,
cross-session, cross-agent tier inside the ADR-0012 SQLite store:

1. **Store:** `~/.skillgrid/mnemonic/__global__.sqlite` opens through the
   standard `store.Open` path (the id bypasses `NormalizeID`, which would
   collapse its underscores). Migration 072 adds `facts.category`
   (`memory` | `user`, `NOT NULL DEFAULT 'memory'`) — the Hermes two-target
   split; migration 073 adds `skill_usage.outcome`/`feedback` and the
   `pending_facts` write-approval queue. The global store is skipped by
   `ListProjects` and `RunTTLExpiry`.
2. **Scope routing:** every `fact_*` tool (MCP and CLI) accepts
   `scope: project | global | all` and, on writes, `target: memory | user`.
   A fact operation needs two store roles — the **data store** (where the
   row lives) and the **trail store** (where the `session_events` row goes).
   `scope=project` uses the caller's store for both (unchanged behavior);
   `scope=global` routes the data to `__global__` and the trail to the
   caller's project store (the session — and its FK — lives there);
   `scope=all` is read-only (fused search). Unknown session → clean error
   naming it, no partial write.
3. **Fused search:** `scope=all` opens both stores and stable-merges by each
   store's own BM25 rank (not RRF — the stores are disjoint, so RRF's
   overlap benefit does not apply). Every hit is labeled `scope` +
   `category`; the search trail goes to the project store only.
4. **Bounded curation:** the global store is char-bounded at the Hermes
   budgets (memory 2,200 / user 1,375, config
   `mnemonic.global_memory`). Batch `operations` (add/patch/remove) apply as
   a unit; the budget is checked on the final result only; overflow returns
   a structured `OverflowError` (current_entries + usage) and rolls back the
   whole batch. `add` dedups on exact content; `patch`/`remove` match
   `old_text` as a substring (ambiguous → error, no match → error with
   closest_entries).
5. **Prefetch injection:** `RenderFrozenSnapshot` renders the store into the
   two labeled sections (GLOBAL MEMORY + USER PROFILE) with usage headers
   and `\n§\n` separators, importance DESC. Injection is frozen at session
   start (the OpenCode V2 `context` hook per ADR-0045 memoizes by session
   id; the MCP `memory/global` resource and the CLI
   `memory global prefetch` render the same block). All adapters fail open:
   a dead store is the empty string, never an error that breaks the
   session.
6. **Background distill + write approval:** one LLM pass (ADR-0023
   OpenAI-compatible client) at session end extracts durable facts,
   classifying each `project` or `global` scope in a single call. When
   `mnemonic.global_memory.write_approval` is `true`, passive (distill)
   writes are staged in the durable `pending_facts` table (survives
   restart) and applied only on explicit approve (`memory global approve
   all` / `fact_pending_approve`); explicit agent writes bypass the gate.
7. **Skill improvement loop:** `skill_usage` records `outcome`
   (success/failure/error) + `feedback` (migration 073); `skill
   inspect/amend/evaluate` (project-scoped) observe, propose
   `<code_path>.amended`, and roll back amendments that do not improve the
   outcome rate.
8. **CLI surface:** `memory fact add --scope <s> --target <t>`, `search
   --scope all`, `global prefetch | pending | approve all | reject all`;
   invalid scope/target is a usage error (exit 2) listing the valid values,
   before any store opens.

## Consequences

- Good, because one source of truth (ADR-0012 intact): the cross-project
  tier is a store, not a second engine; retrieval stays constant-cost and
  zero-LLM in the hot path.
- Good, because the trail semantics (C3) keep the audit chain in the store
  where the session lives — a global `fact_add` is traceable from the
  project's `session_events` without a cross-database join.
- Good, because the bounded budgets make the injected prompt predictable:
  the system prompt never grows with the corpus, and overflow is a
  structured, actionable error (free space, retry) rather than a silent
  truncation.
- Good, because write approval is opt-in (`write_approval: false` is the
  default, Hermes-consistent): the first days of global memory are
  LLM-written and human-unreviewed. The pending queue (`memory global
  pending/approve/reject`) is the audit path once an operator opts in — and
  it is a table, so staged facts survive a server restart.
- Bad, because a fact lives in exactly one store and the routing is
  caller-declared: a `scope=global` fact is invisible to a
  `scope=project` search unless the caller asks for `scope=all`. The fused
  search exists precisely for that, but it is an explicit opt-in.
- Bad, because the trail is not atomically co-located with the data across
  stores (SQLite has no cross-connection transaction): the trail is written
  first, so a crash can leave an orphan trail row (no fact) but never an
  orphan fact without a trail. The tradeoff is documented in
  `facts.AddWith`.
- Bad, because `__global__` is a pseudo-project: any new sweep over "all
  stores" (list, TTL, export, migration tooling) must skip it or it leaks
  into project-facing surfaces. The skip is pinned by tests
  (`ListProjects`, `RunTTLExpiry`).

Revisit triggers: (1) the global tier must serve multiple users/agents with
access control — then per-user DB-backed memory (the ADR-0012 revisit
trigger), not a wider pseudo-project; (2) the char budgets stop fitting the
operational notes — then a ranked top-k render (the store is currently
bounded, not ranked) or a config-wired budget (the constants are a ponytail
ceiling in `facts.CharLimit`); (3) paraphrase-heavy global queries become
common — then embeddings (sqlite-vec) on the same store, per ADR-0012.
