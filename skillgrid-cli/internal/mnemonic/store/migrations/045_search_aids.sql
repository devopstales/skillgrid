-- Migration 045: search aids. This revision adds query_cache only.
-- entity_aliases is appended by a later ticket in this same file.
-- The query text is never stored; query_hash is hex SHA-256 of
-- model + newline + query.

CREATE TABLE IF NOT EXISTS query_cache (
    query_hash TEXT PRIMARY KEY,
    embedding BLOB NOT NULL,
    model TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
