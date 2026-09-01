-- 008: observation lifecycle + embedding columns.
--
-- Parity with Engram's observation lifecycle. Every column is additive
-- (ADD COLUMN with a safe default) so existing databases upgrade without a
-- rewrite. The FTS index stays untouched — search still runs on the same
-- title/content fields; the new fields change ordering and ranking, not
-- matching.
--
--   pinned             — first-class pin (mem_pin / mem_unpin). Pinned rows
--                        sort ahead of everything else in mem_context and
--                        boost mem_search ordering.
--   expires_at         — RFC3339 timestamp; when in the past the row is
--                        soft-excluded from search (mirrors Engram's TTL).
--   duplicate_count    — incremented on repeated Save() of the same hash;
--                        powers Engram-style "seen N times" recency
--                        weighting without storing a full counter of hits.
--   last_seen_at       — updated whenever a duplicate save happens. Gives
--                        search a recency signal independent of revision
--                        count.
--   embedding          — optional BLOB payload of floating32 vector. Only
--                        populated when MNEMONIC_EMBED=1 and an embedder is
--                        available; otherwise NULL, and search falls back
--                        to FTS5-only (reciprocal rank fusion only when a
--                        non-NULL embedding set is present).
--   embedding_model    — model name / version that produced the embedding,
--                        for provenance and to gate re-embedding on model
--                        swap.
--   embedding_created_at — when the embedding was written.

-- SQLite has no DROP COLUMN and ADD COLUMN must have a constant default,
-- so the pinned and lifecycle columns are added individually with their
-- defaults set at the column level.

ALTER TABLE observations ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0;

-- expires_at is nullable — most rows never expire.
ALTER TABLE observations ADD COLUMN expires_at TEXT;

-- duplicate_count starts at 0 (NOT Engram's 1) so the first Save() is
-- unambiguous; we only count observed duplicates past the first write.
-- SQLite does not support default expressions, so the initial value is 0
-- for ALL rows (including pre-existing ones).
ALTER TABLE observations ADD COLUMN duplicate_count INTEGER NOT NULL DEFAULT 0;

-- last_seen_at is the timestamp of the most recent duplicate save, in UTC
-- RFC3339 format, same as created_at.
ALTER TABLE observations ADD COLUMN last_seen_at TEXT;

-- P4 — embedding triplet. Stored as BLOB because SQLite has no native
-- vector type; we own the byte layout (little-endian float32, contiguous)
-- and the dimension is inferred from len/4 when needed. The model string
-- is free-form so we can embed with a local ONNX endpoint or a remote URL
-- depending on MNEMONIC_EMBED_ENDPOINT.
ALTER TABLE observations ADD COLUMN embedding BLOB;
ALTER TABLE observations ADD COLUMN embedding_model TEXT;
ALTER TABLE observations ADD COLUMN embedding_created_at TEXT;

-- Ordering index so mem_context and mem_search can sort by pinned DESC,
-- recency DESC with a single seek. SQLite ignores leading columns with
-- NULL values, so this is cheap to maintain.
CREATE INDEX IF NOT EXISTS idx_obs_pinned
    ON observations (pinned DESC, last_seen_at DESC, created_at DESC)
    WHERE deleted_at IS NULL;

-- TTL sweep: expires_at is the only TTL source (no separate "expiring"
-- marker).
CREATE INDEX IF NOT EXISTS idx_obs_expires
    ON observations (expires_at)
    WHERE expires_at IS NOT NULL AND deleted_at IS NULL;

-- Dedup helper: identical to Engram's idx_obs_dedupe. Used by Save() to
-- locate existing rows for duplicate_count / last_seen_at updates without
-- a full table scan.
CREATE INDEX IF NOT EXISTS idx_obs_dedupe
    ON observations (normalized_hash, project, scope, type, title, created_at DESC);

-- Embedding lookup: when enabled, we fetch embeddings by project so a
-- subsequent search can compute cosine over the right candidate set. The
-- index only exists to prune non-embedded rows; the BLOB itself is not
-- searchable.
CREATE INDEX IF NOT EXISTS idx_obs_embedding
    ON observations (project, embedding_created_at DESC)
    WHERE embedding IS NOT NULL;
