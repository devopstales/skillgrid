-- 011: Fact Memory + Agent Skills tables (change 2026-09-04-hermes-memory, step 01).
--
-- Additive only — creates five new tables and never touches observations,
-- sessions, or any other existing table. Soft-delete convention: deleted_at
-- is NULL for a live row (mirrors observations). The 014 AKL importance
-- columns land on facts with NOT NULL defaults so a fresh fact is always
-- rankable (importance_score=1.0, maturity_tier='new').
--
--   facts        — durable, retrievable facts (Add-only in step 02; search /
--                  forget / decay arrive with the TICKET-02 tools).
--   facts_fts    — FTS5 external-content index over facts.content, kept in
--                  sync by triggers (same pattern as observations_fts).
--                  The vector leg (vec0) is intentionally absent here: a
--                  missing vec0 degrades ranking to FTS-only and never fails.
--   skills       — Agent Skill registry metadata (the code itself lives in the
--                  ContentPlane; code_path points at it).
--   skills_fts   — FTS5 external-content index over name/description.
--   skill_usage  — per-execution log (stdout/stderr captured by the sandbox).
CREATE TABLE IF NOT EXISTS facts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    content TEXT NOT NULL,
    importance_score REAL NOT NULL DEFAULT 1.0,
    recency_decay REAL NOT NULL DEFAULT 0.0,
    maturity_tier TEXT NOT NULL DEFAULT 'new',
    retrieval_usage INTEGER NOT NULL DEFAULT 0,
    deleted_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- Importance-desc ranking for the decay/purge pass (014 AKL). facts lives in
-- a per-project store with no project column, so the 025 (project, ...)
-- observations pattern is not applicable here.
CREATE INDEX IF NOT EXISTS idx_facts_importance
    ON facts (importance_score DESC)
    WHERE deleted_at IS NULL;

CREATE VIRTUAL TABLE IF NOT EXISTS facts_fts USING fts5(
    content,
    content='facts',
    content_rowid='id',
    tokenize='porter'
);

CREATE TRIGGER IF NOT EXISTS facts_fts_insert AFTER INSERT ON facts BEGIN
    INSERT INTO facts_fts(rowid, content) VALUES (new.id, new.content);
END;

CREATE TRIGGER IF NOT EXISTS facts_fts_delete AFTER DELETE ON facts BEGIN
    INSERT INTO facts_fts(facts_fts, rowid, content)
    VALUES ('delete', old.id, old.content);
END;

CREATE TRIGGER IF NOT EXISTS facts_fts_update AFTER UPDATE ON facts BEGIN
    INSERT INTO facts_fts(facts_fts, rowid, content)
    VALUES ('delete', old.id, old.content);
    INSERT INTO facts_fts(rowid, content) VALUES (new.id, new.content);
END;

CREATE TABLE IF NOT EXISTS skills (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    language TEXT NOT NULL,
    description TEXT,
    code_path TEXT NOT NULL,
    deleted_at TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE VIRTUAL TABLE IF NOT EXISTS skills_fts USING fts5(
    name,
    description,
    content='skills',
    content_rowid='id',
    tokenize='porter'
);

CREATE TRIGGER IF NOT EXISTS skills_fts_insert AFTER INSERT ON skills BEGIN
    INSERT INTO skills_fts(rowid, name, description)
    VALUES (new.id, new.name, new.description);
END;

CREATE TRIGGER IF NOT EXISTS skills_fts_delete AFTER DELETE ON skills BEGIN
    INSERT INTO skills_fts(skills_fts, rowid, name, description)
    VALUES ('delete', old.id, old.name, old.description);
END;

CREATE TRIGGER IF NOT EXISTS skills_fts_update AFTER UPDATE ON skills BEGIN
    INSERT INTO skills_fts(skills_fts, rowid, name, description)
    VALUES ('delete', old.id, old.name, old.description);
    INSERT INTO skills_fts(rowid, name, description)
    VALUES (new.id, new.name, new.description);
END;

CREATE TABLE IF NOT EXISTS skill_usage (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    skill_id INTEGER NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    session_id TEXT,
    timestamp TEXT NOT NULL,
    stdout TEXT,
    stderr TEXT
);

CREATE INDEX IF NOT EXISTS idx_skill_usage_skill ON skill_usage (skill_id);
