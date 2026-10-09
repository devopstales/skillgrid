-- 060_ctx_stats_history.sql — CTX (ADR-0033/0012) aggregated stats history.
-- One row per (granularity, bucket) rollup; the store-level Rollup query
-- re-aggregates these into tokens_saved/requests/sessions/cache_hits grouped
-- by granularity + bucket. Schema verbatim from the context-orchestrator
-- plan §9.3 (TICKET-02 / TASK-091).
CREATE TABLE IF NOT EXISTS ctx_stats_history (
    bucket_id             TEXT PRIMARY KEY,
    granularity           TEXT,     -- hourly | daily | weekly | monthly
    bucket_start          INTEGER,
    tokens_saved          INTEGER,
    requests              INTEGER,
    sessions              INTEGER,
    cache_hits            INTEGER,
    created_at            INTEGER
);
