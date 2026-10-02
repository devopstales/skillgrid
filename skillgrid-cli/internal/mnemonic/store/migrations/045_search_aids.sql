-- Migration 045: query embedding cache.
-- The query text is never stored; query_hash is hex SHA-256 of model + newline + query.
-- Entity aliases are 046 so a store that already applied this file still receives them.

CREATE TABLE IF NOT EXISTS query_cache (
    query_hash TEXT PRIMARY KEY,
    embedding BLOB NOT NULL,
    model TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
