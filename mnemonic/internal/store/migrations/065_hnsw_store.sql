-- Migration 057: HNSW graph store (TICKET-05, ADR-0037).
--
-- The HNSW section of the planned 055_hnsw_store.sql split. 055 is taken by
-- 055_obs_split_fk_repoint.sql (TICKET-03); 056 is the IVF section (TICKET-04).
-- The HNSW graph lands at 057 (next free).
--
-- hnsw_nodes: one row per graph node (node_id = vector_id, level = highest
-- layer, vector = little-endian float32 blob matching the BLOB layout).
-- hnsw_links: one row per directed edge (node_id -> neighbor_id at layer l).
-- hnsw_meta: key-value pairs (top node, node count, dim) for warm-load.
--
-- All tables are keyed by node_id (int64) — the same vector_id space as the
-- IVF index (migration 056). The BLOB table (embeddings) remains the source
-- of truth; the HNSW graph is a derived index (ADR-0037).
--
-- Additive, forward-only. CREATE TABLE IF NOT EXISTS is idempotent.
CREATE TABLE IF NOT EXISTS hnsw_nodes (
    node_id  INTEGER NOT NULL PRIMARY KEY,
    level    INTEGER NOT NULL,
    vector   BLOB NOT NULL
);

CREATE TABLE IF NOT EXISTS hnsw_links (
    node_id    INTEGER NOT NULL,
    layer      INTEGER NOT NULL,
    neighbor_id INTEGER NOT NULL,
    PRIMARY KEY (node_id, layer, neighbor_id)
);

CREATE INDEX IF NOT EXISTS idx_hnsw_links_neighbor
    ON hnsw_links (neighbor_id);

CREATE TABLE IF NOT EXISTS hnsw_meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
