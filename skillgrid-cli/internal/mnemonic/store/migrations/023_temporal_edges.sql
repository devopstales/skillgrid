-- 023: temporal bounds on graph edges (change 014, step 10).
--
-- Adds valid_from / valid_to to the `edges` table so a relationship carries the
-- time window in which it was true. Additive only:
--
--   valid_from INTEGER NOT NULL DEFAULT 0   — UNIX seconds when the relationship
--                                             was first observed. 0 is the
--                                             backfill for edges created before
--                                             this migration ("unknown/always
--                                             active": 0 <= now, so the
--                                             current-state filter keeps them).
--   valid_to   INTEGER                      — NULL = the edge is still active;
--                                             a past UNIX second = the edge
--                                             expired at that moment and is
--                                             hidden from current-state queries
--                                             but preserved for history.
--
-- SQLite ADD COLUMN cannot take an expression, so valid_from's default is the
-- constant 0; new edge INSERTs set valid_from = time.Now().Unix() explicitly
-- (codeindex), so the 0 default only applies to rows created before this
-- migration. valid_to has no default (NULL) — a fresh edge is active.
--
-- Current-state edge queries filter on
--     valid_from <= now AND (valid_to IS NULL OR valid_to > now)
-- so expired (valid_to <= now) and not-yet-active (valid_from > now) edges are
-- hidden from graph traversal, while the history read (QueryEdgesWithHistory)
-- applies no filter.

ALTER TABLE edges ADD COLUMN valid_from INTEGER NOT NULL DEFAULT 0;
ALTER TABLE edges ADD COLUMN valid_to INTEGER;

-- Partial index on the active window: the current-state filter scans this
-- instead of the full edges table. valid_to IS NULL rows (the active
-- majority) are included; expired rows (valid_to set) are pruned as they age.
CREATE INDEX IF NOT EXISTS idx_edges_valid_to
    ON edges (valid_from, valid_to)
    WHERE valid_to IS NULL;
