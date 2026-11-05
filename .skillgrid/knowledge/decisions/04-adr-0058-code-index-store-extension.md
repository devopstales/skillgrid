---
status: in-force
supersedes: none
date: 2026-11-02
---

# ADR-0046 — Code-index `Store` interface extension (additive)


Change: `2026-10-31-mnemonic-v2-code-index` (spec zone: `adr.md` § ADR-0046)

## Context

The code-index adds the 7 content + 2 FTS tables + the graph reads + the vector methods
to the **same** `Store` the memory change owns (the shared project DB, spec
`002-schema` §4). The extension must be **additive** (no existing memory method
signature changes) and must include a **table-adding seam** (T21) the KB change's
`KBStore` uses to add its `kb_*` tables without forking the dialect code.

## Decision

Extend the `Store` interface (`mnemonic-v2/internal/core/store/store.go`) with:

- **Write:** `UpsertFile` (returns the file rowid as `int64` — K1/Minor fold) /
  `UpsertSymbols` / `ReplaceEdgesForFile` / `ReplaceUnresolvedForFile` /
  `ReplaceChunksForFile` / `DeleteFile` / `PruneOrphanSymbols` /
  `UpdateIndexFreshness`
- **Read:** `EdgesByFrom` / `EdgesByTo` / `EdgesByNameOnlyToName` / `SymbolByName` /
  `SymbolByUID` / `FileByPath`
- **Vector (M2):** `EmbedSymbol` / `EmbedChunk` / `ClearVectors` — the
  `vectorstore.Engine` holds the Store, not a `*sql.DB`; the vector SQL is
  dialect-rendered in the adapters (SQLite: BLOB + vec0 + HNSW; PG: BLOB + pgvector)
- **Trail seam (M7/T15):** `RecordRetrieval` (the shared `retrieval_trails` table,
  migration 003 — the memory change owns the table, the code-index owns the code-side
  writes)
- **DDL seam (T21):** `ExecDDL` (the table-adding seam the KB re-uses)
- **Transaction surface (K1):** `Begin` / `Txn` — the single per-file tx the
  idempotency claim rests on (the caller composes the five per-file writes inside one
  tx; the composite `ApplyFileGraph` is the driver's own composition, not a Store
  method)

## Alternatives

- **A separate `CodeStore` interface** — rejected: the code-index tables live in the
  same DB + share the WAL/write-retry + the migration runner; a separate store would
  fork the dialect code (the T21 seam exists to avoid exactly that).
- **The KB forking `sqlite.go`/`postgres.go`** — rejected: that is the v1 pattern; the
  KB spec (`004-index` §3.1) explicitly requires "same store, different query surface".

## Consequences

**One-way door (D2)** — adding a method later is additive; reshaping one is a contract
break for the memory change, the code-index, and the KB. The migration numbering (100,
to follow the memory change's 001–004 range, decided in TICKET-01; TICKET-08c mirrors
the same number) is **one-way door (D3)** — a collision is a rebase.
