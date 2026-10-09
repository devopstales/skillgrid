-- 058_ctx_proxy_mode_history.sql — CTX (ADR-0033/0012) proxy mode change
-- audit. One row per transition between proxy modes (token | cache |
-- passthrough) with the reason that triggered it. Schema verbatim from the
-- context-orchestrator plan §9.3 (TICKET-02 / TASK-091).
CREATE TABLE IF NOT EXISTS ctx_proxy_mode_history (
    change_id             TEXT PRIMARY KEY,
    session_id            TEXT,
    old_mode              TEXT,
    new_mode              TEXT,
    reason                TEXT,
    changed_at            INTEGER
);
