-- 061_ctx_traffic_memories.sql — CTX (ADR-0033/0012) traffic-derived memories.
-- One row per memory distilled from observed proxy traffic (error recovery,
-- environment, preference, architecture). Schema verbatim from the
-- context-orchestrator plan §9.3 (TICKET-02 / TASK-091).
CREATE TABLE IF NOT EXISTS ctx_traffic_memories (
    memory_id             TEXT PRIMARY KEY,
    session_id            TEXT,
    category              TEXT,     -- error_recovery | environment | preference | architecture
    memory_text           TEXT,
    evidence_json         TEXT,
    confidence            REAL,
    created_at            INTEGER
);
