-- 019: structured session handoff (change 006, step 01).
--
-- Additive relay schema on top of 001's session/observation store. Every
-- statement is CREATE TABLE IF NOT EXISTS / CREATE INDEX IF NOT EXISTS so an
-- existing database upgrades without a rewrite; the 001 sessions and
-- observations tables are never dropped, altered, or rebuilt.
--
-- The relay is the continuity unit for a session: a handoff row points at the
-- on-disk cleave bundle (PROGRESS.md / KNOWLEDGE.md / NEXT_PROMPT.md under
-- .skillgrid/.cleave/) and an archive row records a handoff that was folded
-- into a resumable archive. Steps 02–05 (relay FS, MCP tools, CLI, watchdog)
-- build on these tables; no row here is ever written without its cleave
-- bundle on disk (fail closed).
--
--   handoff_id       — the operator-facing identifier (also the cleave
--                      bundle directory name); unique, resume target.
--   source_session   — the sessions.id this handoff was taken from.
--   status           — pending | archived. Pending by default; flipped
--                      explicitly when the handoff is archived.
--   cleave_path      — repo-relative path of the cleave bundle directory.
--   context_summary  — optional short context note written at handoff time.

CREATE TABLE IF NOT EXISTS session_handoffs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project TEXT NOT NULL,
    handoff_id TEXT NOT NULL,
    source_session TEXT REFERENCES sessions(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    cleave_path TEXT NOT NULL,
    context_summary TEXT,
    created_at TEXT NOT NULL,
    archived_at TEXT,
    UNIQUE (project, handoff_id)
);

-- session_archives — one row per handoff that was archived. path is the
-- on-disk archive (tarball or directory) the next session resumes from.
CREATE TABLE IF NOT EXISTS session_archives (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project TEXT NOT NULL,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE SET NULL,
    -- No FK to session_handoffs: SQLite FKs must target a PRIMARY KEY or
    -- UNIQUE index, and handoff_id's uniqueness is scoped (project,
    -- handoff_id), so the link is enforced by the application (step 02).
    handoff_id TEXT NOT NULL,
    path TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (project, handoff_id)
);

CREATE INDEX IF NOT EXISTS idx_handoffs_source
    ON session_handoffs (project, source_session);
CREATE INDEX IF NOT EXISTS idx_handoffs_status
    ON session_handoffs (project, status);
CREATE INDEX IF NOT EXISTS idx_archives_session
    ON session_archives (project, session_id);
