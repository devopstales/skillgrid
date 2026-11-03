-- Migration 058: HNSW id-namespacing (TICKET-06, L-1 fix for ADR-0037).
--
-- 057 created hnsw_nodes/hnsw_links keyed by node_id (int64) with NO source
-- partition. The graph is shared by symbol_id (embeddings, FK over symbols)
-- and chunk_id (chunk_embeddings, FK over chunks) — two independent
-- autoincrement spaces that both start at 1. A symbol id 1 and a chunk id 1
-- therefore collide on node_id 1: one vector silently overwrites the other
-- and each leg serves the other's ids (the first row of any real codebase
-- has both id-1 rows).
--
-- Fix: a source TEXT column ('symbol' | 'chunk') partitions the graph; the
-- primary keys include it so (source, node_id) is unique per id space.
-- Rebuild, not ALTER: the tables are a derived index (the BLOB table is the
-- source of truth) with no production data yet (the first writer lands in
-- this ticket), so a fresh pair is byte-identical for a new store and the
-- drop is lossless for existing ones. A plain ALTER TABLE ADD COLUMN could
-- not re-key the PRIMARY KEY without a rebuild anyway.
--
-- Additive forward-only from the store's perspective (the 057 pair is
-- replaced by the 058 pair). CREATE TABLE IF NOT EXISTS keeps re-runs a
-- no-op; the DROP IF EXISTS is the one-way part of the swap and is
-- idempotent on re-run (the pair is empty until the writer populates it).
DROP TABLE IF EXISTS hnsw_links;
DROP TABLE IF EXISTS hnsw_nodes;

CREATE TABLE IF NOT EXISTS hnsw_nodes (
    source   TEXT NOT NULL,
    node_id  INTEGER NOT NULL,
    level    INTEGER NOT NULL,
    vector   BLOB NOT NULL,
    PRIMARY KEY (source, node_id)
);

CREATE TABLE IF NOT EXISTS hnsw_links (
    source      TEXT NOT NULL,
    node_id     INTEGER NOT NULL,
    layer       INTEGER NOT NULL,
    neighbor_id INTEGER NOT NULL,
    PRIMARY KEY (source, node_id, layer, neighbor_id)
);

CREATE INDEX IF NOT EXISTS idx_hnsw_links_neighbor
    ON hnsw_links (source, neighbor_id);

-- hnsw_meta is untouched by 057/058 (warm-load metadata stays global).
