-- 047: harness label on sessions (Gryph-style agent observability).
--
--   agent — the harness that owns the session id (cursor, opencode, kilo, …).
--           Set by the session-start hook or the first tool-call capture;
--           NULL for sessions minted by mem_session_start.
ALTER TABLE sessions ADD COLUMN agent TEXT;
