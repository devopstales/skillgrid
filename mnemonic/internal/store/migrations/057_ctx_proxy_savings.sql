-- 057_ctx_proxy_savings.sql — CTX (ADR-0033/0012) per-turn proxy savings
-- ledger. One row per saved turn; the savings ratio is the observed
-- before/after compression factor. Schema verbatim from the
-- context-orchestrator plan §9.3 (TICKET-02 / TASK-091).
CREATE TABLE IF NOT EXISTS ctx_proxy_savings (
    saving_id             TEXT PRIMARY KEY,
    session_id            TEXT,
    turn                  INTEGER,
    upstream              TEXT,
    mode                  TEXT,
    tokens_before         INTEGER,
    tokens_after          INTEGER,
    savings_ratio         REAL,
    created_at            INTEGER
);
