-- Skillgrid Mnemonic v1.2 — prompts capture + passive observation provenance
--
-- Prompts: captured user prompts (plugin side strips <private> tags, truncates
-- to 2000 chars). Kept for searchability across sessions and as a source of
-- context for "what were we working on" recall.
CREATE TABLE IF NOT EXISTS prompts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    project TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_prompts_session ON prompts(session_id);
CREATE INDEX IF NOT EXISTS idx_prompts_project_time ON prompts(project, created_at DESC);

CREATE VIRTUAL TABLE IF NOT EXISTS prompts_fts USING fts5(
    content,
    content='prompts',
    content_rowid='id',
    tokenize='porter'
);

CREATE TRIGGER IF NOT EXISTS prompts_fts_insert AFTER INSERT ON prompts BEGIN
    INSERT INTO prompts_fts(rowid, content) VALUES (new.id, new.content);
END;

CREATE TRIGGER IF NOT EXISTS prompts_fts_delete AFTER DELETE ON prompts BEGIN
    INSERT INTO prompts_fts(prompts_fts, rowid, content)
    VALUES ('delete', old.id, old.content);
END;

-- Provenance for observations. Values: "agent" (default, via mem_save),
-- "passive" (server-extracted from assistant/tool output), "prompt" (rare,
-- used if a prompt is later promoted to an observation), or "compact"
-- (saved during compaction).
ALTER TABLE observations ADD COLUMN source TEXT NOT NULL DEFAULT 'agent';

-- Per-project migration bookkeeping so `skillgrid setup` can detect when a
-- project id changed (e.g. rename, git remote change) and roll the data over.
-- The migration is one-time and driven from HTTP POST /projects/migrate.
CREATE TABLE IF NOT EXISTS project_migrations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    old_project TEXT NOT NULL,
    new_project TEXT NOT NULL,
    migrated_at TEXT NOT NULL
);
