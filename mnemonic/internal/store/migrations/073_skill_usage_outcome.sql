-- 073_skill_usage_outcome: skill improvement loop outcome tracking (change
-- 2026-10-24-global-llm-wiki, Change A, TICKET-07). One-way: the runner
-- applies it once (tracked in index_meta); no down-migration.
--
-- skill_usage (011) gains two columns:
--
--   outcome  — per-execution result: 'success' | 'failure' | 'error'
--              (error = sandbox timeout or process start failure). Pre-073
--              rows take the NOT NULL DEFAULT 'success' value, which is
--              exactly what 011's schema implied (only successful runs were
--              logged); the C2 execute.go fix now writes all three.
--   feedback — optional user/agent note on the run (nullable).
--
-- The C2 execute.go change logs a row on EVERY outcome, so the usage table
-- becomes a success-rate signal (Inspect/Evaluate), not just an activity log.

ALTER TABLE skill_usage ADD COLUMN outcome TEXT NOT NULL DEFAULT 'success';
ALTER TABLE skill_usage ADD COLUMN feedback TEXT;

CREATE INDEX IF NOT EXISTS idx_skill_usage_skill_outcome
    ON skill_usage (skill_id, outcome);

--
-- pending_facts (TICKET-06, appended to this migration per the task brief):
-- the write-approval pending queue for PASSIVE (background distill) fact
-- writes. When write_approval is enabled, a distill's extracted facts are
-- staged here (status='pending') instead of going directly to facts; the
-- row is a table row, so the queue is durable across server restart.
-- Approve inserts the content into facts and deletes this row (two-phase);
-- reject deletes it (discarded). Explicit agent writes (fact_add) bypass the
-- queue entirely — they never produce a row here.
--
--   content    — the distilled fact text.
--   target     — the fact category: 'memory' (default) | 'user'.
--   scope      — the store the fact routes to on approval: 'project'
--                (the caller's project store) | 'global' (__global__ store).
--   created_at — staging time (RFC3339 UTC).
--   status     — 'pending' until approved or rejected (the row is deleted
--                on either; the column is a read filter, not a lifecycle).
CREATE TABLE IF NOT EXISTS pending_facts (
    id         INTEGER PRIMARY KEY,
    content    TEXT NOT NULL,
    target     TEXT NOT NULL DEFAULT 'memory',
    scope      TEXT NOT NULL DEFAULT 'global',
    created_at TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'pending'
);
