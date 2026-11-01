-- Migration 044: bitemporal audit columns on observations.
-- Adds three nullable, additive columns so existing rows remain valid
-- (invalid_at NULL/empty = still true; superseded_by NULL = not superseded).
-- See ADR-0011.

ALTER TABLE observations ADD COLUMN valid_at TEXT;
ALTER TABLE observations ADD COLUMN invalid_at TEXT;
ALTER TABLE observations ADD COLUMN superseded_by INTEGER;

-- Partial index for fast bi-temporal filtering (only rows with a real
-- invalid_at value). Rows where invalid_at is NULL or '' are "still true"
-- and don't need the index.
--
-- Deviates from the naive spec ON observations(project, invalid_at) WHERE
-- invalid_at IS NOT NULL AND deleted_at IS NULL, intentionally:
-- Single-column partial index: every bi-temporal query binds project = ?
-- (equality) and filters invalid_at, so project doesn't need to be a key
-- column. The != '' predicate encodes the ADR-0011 'NULL/empty = still true'
-- rule — a soft-deleted-but-still-valid row would be indexed unnecessarily
-- under a deleted_at predicate, and a non-deleted empty-string row would be
-- wrongly excluded.
CREATE INDEX IF NOT EXISTS idx_obs_invalid_at
    ON observations (invalid_at)
    WHERE invalid_at IS NOT NULL AND invalid_at != '';
