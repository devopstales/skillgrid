-- Migration 056: IVF flat-code store (TICKET-04).
--
-- This is the IVF section of the planned 055_hnsw_store.sql split. 055 is
-- already taken by 055_obs_split_fk_repoint.sql (TICKET-03), so the ANN
-- tables land at the next free number. The HNSW section (hnsw_nodes /
-- hnsw_links / hnsw_meta) lands in TICKET-05 on its own migration.
--
-- ivf_centroids persists the k-means-lite centroids (one row per centroid,
-- centroid vector as a little-endian float32 blob matching the BLOB layout).
-- ivf_assignments persists the per-vector cluster assignment (one row per
-- vector id, keyed to the vector's rowid in the BLOB source-of-truth table).
--
-- Both tables are keyed by vector_id (int64) rather than a table-specific
-- rowid because the same IVF index is shared by vec_symbols / vec_chunks —
-- a vector_id collision across tables would otherwise require a namespace
-- column, which is over-engineering for the first slice (YAGNI: the indexer
-- owns the id-space and assigns non-overlapping ranges).
--
-- Additive, forward-only. CREATE TABLE IF NOT EXISTS is idempotent on a
-- fresh DB and a no-op on re-run.
CREATE TABLE IF NOT EXISTS ivf_centroids (
    k          INTEGER NOT NULL,
    dim        INTEGER NOT NULL,
    centroid   INTEGER NOT NULL PRIMARY KEY,
    vector     BLOB NOT NULL
);

CREATE TABLE IF NOT EXISTS ivf_assignments (
    vector_id  INTEGER NOT NULL PRIMARY KEY,
    cluster    INTEGER NOT NULL,
    dim        INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ivf_assignments_cluster
    ON ivf_assignments (cluster, vector_id);
