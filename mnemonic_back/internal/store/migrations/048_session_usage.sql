-- 048: per-session token usage and cost (Gryph-style cost tracking).
--
--   input_tokens/output_tokens/cache_tokens — running totals from
--                       POST /sessions/{id}/usage (harness-reported only).
--   model            — last model the harness reported for the session.
--   cost_usd         — running cost; NULL when no reported usage had a known
--                       price (shown as n/a, never guessed).
ALTER TABLE sessions ADD COLUMN input_tokens INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN output_tokens INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN cache_tokens INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN model TEXT;
ALTER TABLE sessions ADD COLUMN cost_usd REAL;
