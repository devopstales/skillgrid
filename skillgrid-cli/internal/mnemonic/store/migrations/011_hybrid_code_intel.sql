-- 011 hybrid code intelligence schema (additive; do not rewrite files/chunks)
-- Change 005, step 01.

CREATE TABLE IF NOT EXISTS symbols (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    qualified_name TEXT,
    kind TEXT NOT NULL,
    language TEXT,
    signature TEXT,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    content_hash TEXT NOT NULL,
    uid TEXT NOT NULL UNIQUE
);
CREATE INDEX IF NOT EXISTS idx_symbols_file ON symbols(file_id);
CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols(name);

CREATE TABLE IF NOT EXISTS edges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kind TEXT NOT NULL,
    from_id INTEGER NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
    file_id INTEGER REFERENCES files(id) ON DELETE CASCADE,
    to_id INTEGER REFERENCES symbols(id) ON DELETE CASCADE,
    to_name TEXT,
    target_path TEXT,
    confidence TEXT NOT NULL DEFAULT 'EXTRACTED',
    line INTEGER,
    UNIQUE(kind, from_id, file_id, to_id, to_name, target_path, line)
);
CREATE INDEX IF NOT EXISTS idx_edges_from ON edges(from_id);
CREATE INDEX IF NOT EXISTS idx_edges_file ON edges(file_id);
CREATE INDEX IF NOT EXISTS idx_edges_to ON edges(to_id);
CREATE INDEX IF NOT EXISTS idx_edges_kind ON edges(kind);

CREATE TABLE IF NOT EXISTS rationale (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol_id INTEGER REFERENCES symbols(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'note',
    line INTEGER
);
CREATE INDEX IF NOT EXISTS idx_rationale_symbol ON rationale(symbol_id);

CREATE VIRTUAL TABLE IF NOT EXISTS symbol_fts USING fts5(
    name,
    qualified_name,
    signature,
    kind,
    language,
    content='symbols',
    content_rowid='id',
    tokenize='unicode61'
);

CREATE TRIGGER IF NOT EXISTS symbol_fts_insert AFTER INSERT ON symbols BEGIN
    INSERT INTO symbol_fts(rowid, name, qualified_name, signature, kind, language)
    VALUES (new.id, new.name, new.qualified_name, new.signature, new.kind, new.language);
END;

CREATE TRIGGER IF NOT EXISTS symbol_fts_delete AFTER DELETE ON symbols BEGIN
    INSERT INTO symbol_fts(symbol_fts, rowid, name, qualified_name, signature, kind, language)
    VALUES ('delete', old.id, old.name, old.qualified_name, old.signature, old.kind, old.language);
END;

CREATE TRIGGER IF NOT EXISTS symbol_fts_update AFTER UPDATE ON symbols BEGIN
    INSERT INTO symbol_fts(symbol_fts, rowid, name, qualified_name, signature, kind, language)
    VALUES ('delete', old.id, old.name, old.qualified_name, old.signature, old.kind, old.language);
    INSERT INTO symbol_fts(rowid, name, qualified_name, signature, kind, language)
    VALUES (new.id, new.name, new.qualified_name, new.signature, new.kind, new.language);
END;

CREATE TABLE IF NOT EXISTS embeddings (
    symbol_id INTEGER PRIMARY KEY REFERENCES symbols(id) ON DELETE CASCADE,
    model TEXT NOT NULL,
    dim INTEGER NOT NULL,
    vector BLOB NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS embed_meta (
    key TEXT PRIMARY KEY,
    value TEXT
);

CREATE TABLE IF NOT EXISTS lsh_buckets (
    bucket INTEGER NOT NULL,
    symbol_id INTEGER NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
    PRIMARY KEY (bucket, symbol_id)
);
CREATE INDEX IF NOT EXISTS idx_lsh_bucket ON lsh_buckets(bucket);

CREATE TABLE IF NOT EXISTS index_freshness (
    key TEXT PRIMARY KEY,
    value TEXT
);
