-- 004_sync_lease.sql — the sync lease (Task 17: the autosync manager's
-- single-worker guarantee). Two MCP servers on one data dir must not
-- double-push: the manager acquires sync_state.lease_owner/lease_until
-- (60 s TTL, Engram store.go:8198-8233 semantics — a dead worker's lease is
-- re-acquirable once it expires) before a cycle and releases it (clear only
-- if the owner matches) after. The columns are additive on the existing
-- sync_state row (target_key PRIMARY KEY) — no new table, no rewrite.
ALTER TABLE sync_state ADD COLUMN lease_owner TEXT;
ALTER TABLE sync_state ADD COLUMN lease_until TEXT;
