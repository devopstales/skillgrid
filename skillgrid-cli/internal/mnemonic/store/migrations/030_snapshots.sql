-- 030: multi-version store snapshots (change 014, step 20.1).
--
-- A snapshot is a point-in-time view of a project's observations state:
-- the serialized rows as a BLOB plus a SHA-256 integrity hash. Snapshots are
-- multi-version — every Snapshot() call adds a new row, so a project can roll
-- back to any previous capture. The table is purely additive: it adds no
-- columns to observations and touches no existing schema.
--
--   id          — surrogate primary key (monotonic; newest snapshot = highest id)
--   project_id  — the project bucket this snapshot captures
--   state_hash  — SHA-256 (hex) of the serialized data BLOB (integrity check)
--   data        — the serialized observations state (JSON array of rows)
--   created_at  — RFC3339 UTC timestamp of the capture

CREATE TABLE IF NOT EXISTS snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id TEXT NOT NULL,
    state_hash TEXT NOT NULL,
    data BLOB NOT NULL,
    created_at TIMESTAMP NOT NULL
);
