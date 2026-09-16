-- 037: AST-boundary chunking for the semantic tier + language partition on
-- the vector store. Borrowed from cocoindex-code (the 4th external index in
-- docs/NOTES.md), whose whole identity is "AST-based semantic code search":
-- it chunks at Tree-Sitter logical boundaries (~1000 chars) and stores the
-- embedding's language as a partition key for exact index-level filtering.
-- Mnemonic chunked by fixed line windows (ChunkLines) and had no language
-- column on the vector store, so a language-scoped semantic search full-scanned
-- and filtered in Go.

--
-- 1. Chunk lineage
--
-- A chunk is now either a fixed line window (kind='lines', the previous
-- behavior) or an AST symbol boundary (kind='ast', one per function/class/
-- method). The embedding tier prefers ast chunks — a function-level chunk is
-- contextually coherent AND fits a 512-token encoder window — and falls back
-- to line chunks for non-code or when a symbol exceeds the char target.
ALTER TABLE chunks ADD COLUMN kind TEXT NOT NULL DEFAULT 'lines';

--
-- 2. Chunk-level semantic embedding
--
-- The vector store was symbol-only (one row per symbol, PK symbol_id). The
-- embedding TIER now also embeds chunk text (the ~1000-char coherent unit),
-- so semantic recall works for non-symbol code (lambdas, config blocks, doc
-- spans) and for the chunk-level recall cocoindex-code is built around. A
-- symbol still keeps its name+signature vector in `embeddings`; chunk vectors
-- live here, keyed by chunk_id.
CREATE TABLE IF NOT EXISTS chunk_embeddings (
    chunk_id   INTEGER PRIMARY KEY REFERENCES chunks(id) ON DELETE CASCADE,
    model      TEXT NOT NULL,
    dim        INTEGER NOT NULL,
    vector     BLOB NOT NULL,
    updated_at TEXT NOT NULL
);

--
-- 3. Language partition on the vector store
--
-- The cocoindex partition key, expressed relationally: a `language` column on
-- both vector tables so a "search only Go" semantic query is an exact
-- index-level filter (WHERE language = ?) instead of a full-scan + Go-side
-- filter. An index on the column makes the filter cheap.
ALTER TABLE embeddings ADD COLUMN language TEXT NOT NULL DEFAULT '';
ALTER TABLE chunk_embeddings ADD COLUMN language TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_embeddings_language ON embeddings (language);
CREATE INDEX IF NOT EXISTS idx_chunk_embeddings_language ON chunk_embeddings (language);
