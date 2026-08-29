-- 003: give sessions that have NEITHER a title NOR a summary a placeholder
-- title so they render in the web dashboard session list (mem-sessions)
-- instead of being dropped by the (title OR summary) filter in RecentContext.
-- Sessions that DO have a summary get their title derived from the "## Goal"
-- line by the memory service (deriveSessionTitle) at end/summary/read time —
-- kept in Go because modernc.org/sqlite lacks the regexp/char functions needed
-- for SQL-level extraction.
UPDATE sessions
SET title = 'Untitled session'
WHERE (title IS NULL OR TRIM(title) = '')
  AND (summary IS NULL OR TRIM(summary) = '');
