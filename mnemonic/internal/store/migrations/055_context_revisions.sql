-- 055_context_revisions.sql — CLM session-scoped revision audit (ADR-0027/0028,
-- TICKET-07). One row per accepted mirror edit (context revision). The Go side
-- owns state/config/mirror; the Node request path renders the mirror, applies
-- withhold, and the checkpoint capture path persists validated edits here.
--
-- Raw history + the mirror are the source of truth (ADR-0028); this table is a
-- session-scoped audit, purged at session end. It carries the revision number,
-- the anchor (count + SHA-256 digest) used to re-validate on resume, the
-- edited messages, the size estimate + calibration factor, and the withhold /
-- edit trace.
CREATE TABLE IF NOT EXISTS context_revisions (
    id                 INTEGER PRIMARY KEY,
    session_id         TEXT NOT NULL,
    project_id         TEXT NOT NULL,
    revision           INTEGER NOT NULL,
    anchor_count       INTEGER NOT NULL,
    anchor_digest      TEXT NOT NULL,
    messages           TEXT NOT NULL,
    size_estimate      INTEGER NOT NULL,
    calibration_factor REAL NOT NULL DEFAULT 4.0,
    withhold_ids       TEXT,
    edit_trace         TEXT,
    created_at         TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_context_revisions_session ON context_revisions(session_id);
CREATE INDEX IF NOT EXISTS idx_context_revisions_rev ON context_revisions(session_id, revision);
