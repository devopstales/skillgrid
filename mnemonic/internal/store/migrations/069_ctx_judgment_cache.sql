CREATE TABLE IF NOT EXISTS ctx_judgment_cache (
    fingerprint  TEXT PRIMARY KEY,
    decision     TEXT NOT NULL,
    score        REAL NOT NULL,
    judged_at    INTEGER NOT NULL
);
