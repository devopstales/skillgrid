-- 031: handoff cursor (change 014, step 21).
--
-- The handoff artifact (prefix + delta) is generated on demand. The delta
-- ("changed since the last handoff") needs a persistent, cross-process cursor
-- because the memory package cannot depend on the service layer and a fresh CLI
-- process cannot rely on in-process state. This tiny key/value table stores the
-- last-handoff timestamp (RFC3339) under a fixed key. Purely additive: a new
-- table, no changes to any existing schema.
--
--   key   — fixed key (handoff:cursor); one row per project store
--   value — RFC3339 UTC timestamp of the most recent handoff generation

CREATE TABLE IF NOT EXISTS handoff_meta (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
