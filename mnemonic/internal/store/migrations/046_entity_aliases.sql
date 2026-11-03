-- Migration 046: case-insensitive alias → symbol qualified_name.
-- Separate from 045 because 045 may already be recorded as query_cache only.

CREATE TABLE IF NOT EXISTS entity_aliases (
    project TEXT NOT NULL,
    alias TEXT NOT NULL COLLATE NOCASE,
    qualified_name TEXT NOT NULL,
    source TEXT NOT NULL DEFAULT 'index',
    PRIMARY KEY (project, alias, qualified_name)
);
