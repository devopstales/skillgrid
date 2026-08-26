-- Skillgrid Mnemonic v1 initial schema

PRAGMA foreign_keys = ON;

-- Memory (Engram-aligned)
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    project TEXT NOT NULL,
    directory TEXT NOT NULL,
    started_at TEXT NOT NULL,
    ended_at TEXT,
    summary TEXT,
    status TEXT NOT NULL DEFAULT 'active'
);

CREATE TABLE IF NOT EXISTS observations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL REFERENCES sessions(id),
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    project TEXT,
    scope TEXT,
    topic_key TEXT,
    normalized_hash TEXT,
    revision_count INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);

CREATE VIRTUAL TABLE IF NOT EXISTS observations_fts USING fts5(
    title,
    content,
    type,
    project,
    content='observations',
    content_rowid='id',
    tokenize='porter'
);

CREATE TRIGGER IF NOT EXISTS observations_fts_insert AFTER INSERT ON observations BEGIN
    INSERT INTO observations_fts(rowid, title, content, type, project)
    VALUES (new.id, new.title, new.content, new.type, new.project);
END;

CREATE TRIGGER IF NOT EXISTS observations_fts_delete AFTER DELETE ON observations BEGIN
    INSERT INTO observations_fts(observations_fts, rowid, title, content, type, project)
    VALUES ('delete', old.id, old.title, old.content, old.type, old.project);
END;

CREATE TRIGGER IF NOT EXISTS observations_fts_update AFTER UPDATE ON observations BEGIN
    INSERT INTO observations_fts(observations_fts, rowid, title, content, type, project)
    VALUES ('delete', old.id, old.title, old.content, old.type, old.project);
    INSERT INTO observations_fts(rowid, title, content, type, project)
    VALUES (new.id, new.title, new.content, new.type, new.project);
END;

-- Code index (OpenClaw-aligned)
CREATE TABLE IF NOT EXISTS files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT NOT NULL UNIQUE,
    mtime_ns INTEGER NOT NULL,
    size INTEGER NOT NULL,
    content_hash TEXT NOT NULL,
    indexed_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS chunks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    text TEXT NOT NULL,
    content_hash TEXT NOT NULL
);

CREATE VIRTUAL TABLE IF NOT EXISTS chunks_fts USING fts5(
    text,
    path UNINDEXED,
    content='chunks',
    content_rowid='id',
    tokenize='trigram'
);

CREATE TRIGGER IF NOT EXISTS chunks_fts_insert AFTER INSERT ON chunks BEGIN
    INSERT INTO chunks_fts(rowid, text, path)
    SELECT new.id, new.text, (SELECT path FROM files WHERE id = new.file_id);
END;

CREATE TRIGGER IF NOT EXISTS chunks_fts_delete AFTER DELETE ON chunks BEGIN
    INSERT INTO chunks_fts(chunks_fts, rowid, text, path)
    VALUES (
        'delete',
        old.id,
        old.text,
        (SELECT path FROM files WHERE id = old.file_id)
    );
END;

CREATE TRIGGER IF NOT EXISTS chunks_fts_update AFTER UPDATE ON chunks BEGIN
    INSERT INTO chunks_fts(chunks_fts, rowid, text, path)
    VALUES (
        'delete',
        old.id,
        old.text,
        (SELECT path FROM files WHERE id = old.file_id)
    );
    INSERT INTO chunks_fts(rowid, text, path)
    SELECT new.id, new.text, (SELECT path FROM files WHERE id = new.file_id);
END;

-- Web cache (Neuledge-aligned)
CREATE TABLE IF NOT EXISTS web_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project TEXT,
    source TEXT NOT NULL,
    cache_key TEXT NOT NULL,
    url TEXT,
    title TEXT,
    query TEXT,
    library_id TEXT,
    version_tag TEXT,
    content TEXT NOT NULL,
    metadata_json TEXT,
    content_hash TEXT NOT NULL,
    fetched_at TEXT NOT NULL,
    expires_at TEXT,
    session_id TEXT,
    created_at TEXT,
    UNIQUE(project, source, cache_key)
);

CREATE VIRTUAL TABLE IF NOT EXISTS web_cache_fts USING fts5(
    title,
    content,
    query,
    url,
    source,
    library_id,
    content='web_cache',
    content_rowid='id',
    tokenize='porter'
);

CREATE TRIGGER IF NOT EXISTS web_cache_fts_insert AFTER INSERT ON web_cache BEGIN
    INSERT INTO web_cache_fts(rowid, title, content, query, url, source, library_id)
    VALUES (new.id, new.title, new.content, new.query, new.url, new.source, new.library_id);
END;

CREATE TRIGGER IF NOT EXISTS web_cache_fts_delete AFTER DELETE ON web_cache BEGIN
    INSERT INTO web_cache_fts(web_cache_fts, rowid, title, content, query, url, source, library_id)
    VALUES ('delete', old.id, old.title, old.content, old.query, old.url, old.source, old.library_id);
END;

CREATE TRIGGER IF NOT EXISTS web_cache_fts_update AFTER UPDATE ON web_cache BEGIN
    INSERT INTO web_cache_fts(web_cache_fts, rowid, title, content, query, url, source, library_id)
    VALUES ('delete', old.id, old.title, old.content, old.query, old.url, old.source, old.library_id);
    INSERT INTO web_cache_fts(rowid, title, content, query, url, source, library_id)
    VALUES (new.id, new.title, new.content, new.query, new.url, new.source, new.library_id);
END;

-- Meta
CREATE TABLE IF NOT EXISTS index_meta (
    key TEXT PRIMARY KEY,
    schema_version INTEGER NOT NULL
);

INSERT OR IGNORE INTO index_meta (key, schema_version) VALUES ('schema_version', 1);
