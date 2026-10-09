# Briefing — ADR-aware knowledge index + retrieval

## Problem

The `mnemonic` code index is blind to the project's own decision records. `DefaultIndexing()`
excludes `**/.skillgrid/**` (`mnemonic/internal/config/load.go:624`), so every ADR
(`.skillgrid/artifacts/04-adr-NNNN-slug.md`, 32+ files), `ASSUMPTIONS.md`, the glossary, the
PRD, specs, and shipped reports are invisible to `code_search` / `code_explore`. An agent asking
"what did we decide about vector search?" gets nothing from the index, even though ADR-0006/0009
are the answer on disk.

This is the same gap the knowledge-projection (wiki) attempt hit and rolled back
(`state.yaml` P26-09-30 note): source knowledge from the local `.skillgrid/` files. That attempt
was rolled back for scope, not direction. This change re-approaches it as two small, independent
pieces.

## Design (approved in chat)

### Part 1 — Index `.skillgrid/`

The exclude-before-include walk (`indexer.go:152-157`, `fingerprint.go:202-207`) means a simple
include cannot override `**/.skillgrid/**`, and the `shouldSkipDir` prune removes the whole
subtree. So a dedicated `DocInclude` field is threaded through the walker: a file matching
`DocInclude` is indexed even when it matches `Exclude` (it still must match `Include` when
`Include` is non-empty). `DefaultIndexing()` sets `DocInclude: ["**/.skillgrid/**/*.md"]`.

This is additive and reversible — it changes which files are indexed, not the schema, the graph,
or the embedder.

### Part 2 — ADR-aware retrieval (`adr_search`)

A new `adr_search` MCP tool. It FTS-queries `chunks_fts` for the terms, filters hits to paths
matching `**/04-adr-*.md`, parses each ADR's frontmatter (`status`, `supersedes`, `date`) + H1
title, and returns structured results. For a query that matches a superseded ADR it also surfaces
the superseding ADR (follows the `supersedes` chain one hop). It reuses the existing
`chunks`/`files`/`chunks_fts` tables — no new table, no migration.

Per the approved design, a hit returns frontmatter + snippet + score; the agent `code_read`s the
path for the full Context/Decision/Consequences body. No full-body return in this change.

## What this does NOT do (YAGNI)

- No new table or migration — reuses `chunks`/`files`/`chunks_fts`.
- No embedding change — ADRs enter the FTS index via Part 1.
- No auto-injection of the ADR index into agent context (the "Standing knowledge" piece) — a
  separate change.
- No change to the `code_search` FTS query builder — `adr_search` is a distinct tool.

## Acceptance

1. After `code_index`, an ADR file under `.skillgrid/artifacts/` appears in the index and is
   returned by `adr_search` for a term from its body.
2. `adr_search "vector search"` returns ADR-0006/ADR-0009 with `status`, `supersedes`, `date`,
   `title`, `snippet`, `score`.
3. A superseded ADR (one whose frontmatter is flipped or that another ADR `supersedes`) is
   flagged, and the superseding ADR is surfaced in the same result set.
4. A file that matches only `Exclude` (e.g. a `.skillgrid/` file that is not under `artifacts/`
   or not `.md`) is still not indexed — the carve-out is scoped to the `DocInclude` glob.
5. `adr_search` on an index with no ADR chunks returns an empty list, not an error.

## One-way doors / risk

- **One-way door: none.** Part 1 is a config-driven include (reversible by removing the
  `DocInclude` glob). Part 2 is a new read-only tool (reversible by un-registering it). No
  schema migration, no data transform.
- **Risk: Low.** Additive config field + one new query path. The blast radius is "which files
  the indexer walks" — contained to the walk filter and the new tool.

## Scope / files

- `mnemonic/internal/codeindex/indexer.go` — `Config.DocInclude`, `Scan`/`shouldSkipDir`/walk filter
- `mnemonic/internal/codeindex/fingerprint.go` — fingerprint walk filter
- `mnemonic/internal/codeindex/watch.go` — `WatchConfig.DocInclude` + filter (comfort-layer parity)
- `mnemonic/internal/config/load.go` — `Indexing.DocInclude`, `mnemonicSection.doc_include`,
  `mergeIndexing`, `DefaultIndexing`
- `mnemonic/internal/search/adr.go` (new) — `AdrSearch` + frontmatter parse + supersedes chain
- `mnemonic/internal/mcp/tools_code.go` — register `adr_search` tool + handler
- `cmd/mnemonic/mcp.go` + `internal/service/service.go` + `internal/http/server.go` — thread
  `DocInclude` into the `codeindex.Config`/`WatchConfig` construction
- Tests: `codeindex` walk-filter test, `search/adr.go` unit test (seed ADR chunks), `mcp` handler test
