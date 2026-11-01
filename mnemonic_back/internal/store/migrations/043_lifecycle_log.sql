-- 043: second-brain lifecycle audit (mem_lifecycle + mem_ask floor).
--
-- Additive only. lifecycle_log is the append-only audit trail for every
-- mutating lifecycle op (archive/restore/dedup/consolidate): each op writes a
-- pending row then flips it to completed/failed with completed_at.
-- observations gains the soft-archive pair (archive_reason, archived_at) so
-- archive/restore is reversible without touching deleted_at.
--
--   archive_reason — human-readable why the observation was archived.
--   archived_at    — RFC3339 stamp set on archive, cleared on restore.
CREATE TABLE IF NOT EXISTS lifecycle_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project TEXT NOT NULL,
    action TEXT NOT NULL,
    subaction TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    input_ids TEXT,
    output_ids TEXT,
    details TEXT,
    created_at TEXT NOT NULL,
    completed_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_lifecycle_log_project ON lifecycle_log (project, created_at);

ALTER TABLE observations ADD COLUMN archive_reason TEXT;
ALTER TABLE observations ADD COLUMN archived_at TEXT;
