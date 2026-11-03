-- 054: indexed code files (project-scoped) + FTS5 over (name, kind, language, content)
-- Backs the second leg of ctx_query two-leg RRF fusion (context_harness).
-- start_line/end_line give per-chunk line provenance (ctx search reports the
-- matched line range); the unique index (project, name, start_line) is the
-- INSERT ... ON CONFLICT target that makes IndexFile re-indexing idempotent.

CREATE TABLE IF NOT EXISTS indexed_files (
    id          INTEGER PRIMARY KEY,
    project     TEXT NOT NULL,
    name        TEXT NOT NULL,
    kind        TEXT NOT NULL,
    language    TEXT NOT NULL,
    content     TEXT NOT NULL,
    start_line  INTEGER NOT NULL DEFAULT 0,
    end_line    INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_indexed_files_project ON indexed_files(project);
CREATE UNIQUE INDEX IF NOT EXISTS idx_indexed_files_upsert
    ON indexed_files (project, name, start_line);

CREATE VIRTUAL TABLE IF NOT EXISTS indexed_files_fts USING fts5(
    name,
    kind,
    language,
    content,
    content='indexed_files',
    content_rowid='id',
    tokenize='porter'
);

CREATE TRIGGER IF NOT EXISTS indexed_files_fts_insert AFTER INSERT ON indexed_files BEGIN
    INSERT INTO indexed_files_fts(rowid, name, kind, language, content)
    VALUES (new.id, new.name, new.kind, new.language, new.content);
END;
CREATE TRIGGER IF NOT EXISTS indexed_files_fts_delete AFTER DELETE ON indexed_files BEGIN
    INSERT INTO indexed_files_fts(indexed_files_fts, rowid, name, kind, language, content)
    VALUES ('delete', old.id, old.name, old.kind, old.language, old.content);
END;
