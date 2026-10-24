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
