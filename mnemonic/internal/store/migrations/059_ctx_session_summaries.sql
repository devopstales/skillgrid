-- 059_ctx_session_summaries.sql — CTX (ADR-0033/0012) per-session summary.
-- One row per finished proxied session carrying the wide event counters
-- (judgment, prefilter, shaping, effort routing) and the full wide event
-- JSON. Schema verbatim from the context-orchestrator plan §9.3
-- (TICKET-02 / TASK-091).
CREATE TABLE IF NOT EXISTS ctx_session_summaries (
    session_id            TEXT PRIMARY KEY,
    agent                 TEXT,
    duration_ms           INTEGER,
    turns                 INTEGER,
    tokens_saved          INTEGER,
    judgment_keeps        INTEGER,
    judgment_truncs       INTEGER,
    prefilter_pins        INTEGER,
    prefilter_deads       INTEGER,
    cache_invalidations   INTEGER,
    shaping_applied       BOOLEAN,
    effort_routed         INTEGER,
    traffic_memories      INTEGER,
    summary_json          TEXT,     -- the full wide event
    created_at            INTEGER
);
