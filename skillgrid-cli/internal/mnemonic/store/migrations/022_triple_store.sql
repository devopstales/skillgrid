-- 022: triple-store cross-linkage (change 014, step 07).
--
-- Cross-links the relational store (observations, long_term_memories), the
-- vector store (embeddings), and the graph store (symbols) inside the single
-- per-project SQLite database. Additive: every new column is nullable, so
-- existing rows and every existing query path are untouched. The cross-link
-- query (memory.CrossLinkQuery) and the integrity check
-- (memory.CheckCrossLinkIntegrity) are opt-in methods, not part of the
-- default search path.
--
--   observations.graph_ref      — codeindex symbols.id when the observation's
--                                 source file has an indexed symbol; NULL
--                                 otherwise (best-effort, never blocks Save).
--   long_term_memories.embedding_blob — optional precomputed embedding BLOB
--                                 (little-endian float32), mirroring the
--                                 observations.embedding layout.
--
-- NOTE: the `embeddings` table (migration 011) already bridges symbols to
-- vectors and lives in the SAME store as observations, so it is reused as
-- the symbol_embeddings bridge instead of creating a duplicate table. The
-- codeindex package exposes it via the symbol_embeddings.go accessors.

ALTER TABLE observations ADD COLUMN graph_ref INTEGER;

CREATE INDEX IF NOT EXISTS idx_obs_graph_ref
    ON observations (graph_ref)
    WHERE graph_ref IS NOT NULL;

ALTER TABLE long_term_memories ADD COLUMN embedding_blob BLOB;
