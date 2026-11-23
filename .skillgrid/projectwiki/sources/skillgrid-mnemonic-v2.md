---
type: SourceSummary
title: "skillgrid/mnemonic (v2) — predecessor study"
status: stable
sources:
  - "local: ../skillgrid/mnemonic (Go 1.26, module github.com/devopstales/skillgrid/mnemonic)"
  - "local: ../skillgrid/mnemonic/internal/core/store/migrations/001_schema.sql"
  - "local: ../skillgrid/mnemonic/internal/codeindex/vectorstore/store.go"
---

# skillgrid/mnemonic (v2) — predecessor study

The memory package inside the `skillgrid` repo (`../skillgrid/mnemonic`): SQLite
(`modernc.org/sqlite`) **and** Postgres (`lib/pq` + `migrations_pg`) dialect-aware
store, FTS5, a tree-sitter code index with a vector layer, a projectwiki generator,
and MCP/CLI transport. Studied 2026-11-23 as the primary design predecessor for
[[entities/mnemonic]].

## What it proves (inherited by the new design)

- **Observations model** — `sessions` + `observations` with `topic_key` upsert,
  `normalized_hash` dedup, `revision_count`, `duplicate_count`, `scope`
  (project|user|global), `superseded_by`, `importance`, `embedding` BLOB +
  `embedding_model`. The DDL in `001_schema.sql` is the reference.
- **Vector layer decision** — BLOB is the source of truth; ANN preference
  HNSW → vec0 → brute-force; dim-strict (a dim change is a re-embed, never a
  silent truncate); the `Embedder` interface seam with a nil = "absent" state.
  See [[concepts/in-file-vector-layer]].
- **Hash-64 floor** — deterministic 64-dim token hash so the vector leg exists
  before any model is present. See [[concepts/degradation-floor]].
- **Transport seam (K6)** — CLI and MCP share one Handler; byte-identical
  envelopes. See [[concepts/single-handler-two-adapters]].
- **Project detection chain** — flag > env > git toplevel > basename; exit
  codes 0/1/2; WAL + `busy_timeout`.

## What to cut (scope creep the new design avoids)

- Postgres dialect + `migrations_pg` (cloud tier) — v1 is SQLite-only.
- Projectwiki generation — a skill-level concern, not a memory concern.
- `team_*` MCP tools and SDD artifact coupling.
- The ~30-tool code-graph surface (`code_god_nodes`, `code_pdg_query`, …) —
  the new `code` tool (deferred) starts from ~5 tools.
- Its default semantic leg is an opt-in 768-dim API embedder with hash floor —
  no local-model path. The new design inverts this: local ONNX is the default,
  API is opt-in ([[decisions/0006-local-onnx-default-embedder]]).

## Related

- Supersedes: none (study, not a decision).
- Consumed by: [[syntheses/mnemonic-core]], [[entities/mnemonic]].
- Depends on: [[concepts/in-file-vector-layer]], [[concepts/degradation-floor]],
  [[concepts/single-handler-two-adapters]].
