-- 049: per-session memory checkpoint bookkeeping (claim + last write watermark).
--
--   checkpoint_claimed_at — when the harness last acknowledged a checkpoint
--                         prompt (NULL until first claim).
--   last_memory_write_at  — watermark for session_events since last memory
--                         write or session summary (NULL counts all events).
ALTER TABLE sessions ADD COLUMN checkpoint_claimed_at TEXT;
ALTER TABLE sessions ADD COLUMN last_memory_write_at TEXT;
