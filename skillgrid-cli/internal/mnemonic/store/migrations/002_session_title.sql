-- 002: named sessions. The `title` column shows the session name in the web
-- dashboard session list (mem-sessions). Fresh databases get the column here;
-- the updated 001 omits it so this is the single source of truth.
ALTER TABLE sessions ADD COLUMN title TEXT;
