-- 020: TTL config + extraction metadata (change 014, step 01).
-- Change-014's migration, sequenced as 020 (the brief's 014_ prefix collides
-- with the existing 014_process_flows.sql); sorts after 019_session_relay.sql
-- so it applies last. Additive: new tables only; no existing schema is touched.

CREATE TABLE IF NOT EXISTS ttl_config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS extraction_metadata (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT,
    content_hash TEXT,
    extracted_at TIMESTAMP,
    model TEXT
);
