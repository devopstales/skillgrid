-- 042: vec0 virtual tables for the durable in-SQL vector path (ADR-0009, option G).
--
-- Additive, forward-only. Two vec0 tables mirror the BLOB embedding tables
-- (embeddings, chunk_embeddings) which remain the source of truth. The vec
-- tables are a derived index: if absent/empty/corrupt the semantic leg
-- degrades to the in-memory path (never fails).
--
-- Dimension is pinned to 768 (DefaultOnnxDim, the built-in embedder default).
-- The vec subpackage is cgo-free (modernc.org/sqlite/vec, blank-imported in
-- vectorstore). rowid is keyed to the parent table's PK (symbol_id /
-- chunk_id) so a top-K hit maps 1:1 back to the BLOB row.
CREATE VIRTUAL TABLE IF NOT EXISTS vec_symbols USING vec0(
    rowid     INTEGER PRIMARY KEY,
    embedding float[768]
);
CREATE VIRTUAL TABLE IF NOT EXISTS vec_chunks USING vec0(
    rowid     INTEGER PRIMARY KEY,
    embedding float[768]
);
