-- 056_ctx_proxy_sessions.sql — CTX (ADR-0033/0012) proxy session ledger. One
-- row per proxied agent session on the shared project store. INTEGER columns
-- are unix-millis epoch (codebase convention). Schema verbatim from the
-- context-orchestrator plan §9.3 (TICKET-02 / TASK-091).
CREATE TABLE IF NOT EXISTS ctx_proxy_sessions (
    session_id            TEXT PRIMARY KEY,
    agent                 TEXT,     -- detected from headers/UA
    mode                  TEXT,     -- token | cache | passthrough
    started_at            INTEGER,
    ended_at              INTEGER,
    requests              INTEGER,
    tokens_before         INTEGER,
    tokens_after          INTEGER,
    cache_hits            INTEGER,
    cache_invalidations   INTEGER
);
