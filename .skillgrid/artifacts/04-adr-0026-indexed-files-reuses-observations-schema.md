# indexed_files reuses the observations schema shape

---
status: "accepted"
supersedes: none
date: 2026-10-06
---

## Context and Problem Statement

`ctx index <path>` lets the agent or operator proactively index a file or directory into a searchable store. The question is which store. We already have three FTS5 surfaces: `observations_fts` (project-scoped memory, 7-day TTL), `chunks_fts` (code index, tree-sitter-tuned for source), and the new `tool_outputs` sandbox (session-scoped, captured output). The user's direction was explicit: "use the same [schema] that for Second Brain."

Second Brain has no table of its own. `secondbrain/ask.go:74` proves it: `AskCited` calls `session_inject.HybridObservations` over `observations` + `observations_fts`. "The same that for Second Brain" therefore means the `observations` schema shape: `id`, `title`, `type`, `content`, `project_id`, `session_id`, `created_at`, `updated_at`, `valid_at`, `invalid_at`, `superseded_by`, `status`, plus the `observations_fts` FTS5 table over `(title, content)`.

The lifecycle question is the real trade-off. `tool_outputs` is session-scoped and purged at session end. Proactively-indexed files are meant to persist for the project — you index your docs once, search them all week. Merging them into `tool_outputs` means `ctx purge` at session end destroys your indexed docs. Merging them into `observations` means `ctx index` and `mem_save` write to the same store, and an indexed file's `content` is confused with an observation's `content` (one is a document, the other is a memory).

## Considered Options

- Reuse `tool_outputs`: one FTS store for captured + indexed, `ctx_search` queries one table
- Reuse `observations`: `ctx index` writes observations with `type='indexed_file'`, `ctx_search` queries `observations_fts`
- Separate `indexed_files` table reusing the `observations` schema shape: same columns, same FTS5 shape, separate table, separate lifecycle

Chosen option: "Separate `indexed_files` table reusing the `observations` schema shape," because the lifecycle is different (project-scoped, 7-day TTL like observations, not session-scoped like `tool_outputs`) and the semantics are different (a document, not a memory). Reusing the schema shape means the FTS5 tokenization, the RRF fusion, and the privacy filters are all the same — the new code is a table + a `ctx_search` leg, not a new retrieval engine.

- `indexed_files` has the same columns as `observations` (`id`, `title`, `type`, `content`, `project_id`, `created_at`, `updated_at`, `valid_at`, `invalid_at`, `status`) plus `source_path` (the indexed file's repo-relative path) and `chunk_index` (for files chunked into multiple rows). It is created by migration `051_indexed_files.sql` (project-scoped, 7-day TTL like `observations`).
- `indexed_files_fts` is a separate FTS5 virtual table over `(title, content)`, mirroring `observations_fts`.
- `ctx_search` queries both `tool_outputs_fts` (session-scoped) and `indexed_files_fts` (project-scoped) and fuses the two legs with RRF (reusing `hybrid.RRF`, rrfK=60).
- `ctx index <path>` reads the file, chunks it (by line, threshold ~2KB per chunk), and inserts rows into `indexed_files`. A file already indexed is upserted by `source_path` + `chunk_index`.
- `ctx purge` clears `tool_outputs` only (session-scoped). `indexed_files` is cleared by `ctx index --clear` or by the 7-day TTL (same as observations).

### Consequences

- Good, because the schema reuse means the FTS5 tokenization, RRF fusion, and privacy filters are the same — no new retrieval engine, just a new table and a new `ctx_search` leg.
- Good, because the lifecycle is honest: `tool_outputs` is purged at session end, `indexed_files` persists for the project (7-day TTL), and `ctx purge` does not destroy your indexed docs.
- Good, because `ctx index` is a thin writer: read file, chunk, insert. The retrieval side is `ctx_search`, which is a two-leg RRF fusion.
- Bad, because `indexed_files` and `observations` have nearly identical schemas, which invites confusion. The `source_path` and `chunk_index` columns are the only differentiators, and the `type` column should be set to a distinct value (e.g. `indexed_file`) to make the difference queryable.
- Bad, because a large file indexed in one `ctx index` call can produce many `indexed_files` rows (one per chunk). The `ctx index` command must be bounded (e.g. max 100 chunks per call) or the agent indexes a 10,000-line file and floods the table.
