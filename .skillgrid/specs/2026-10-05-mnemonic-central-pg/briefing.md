# Briefing — Mnemonic Central PG (team-shared memory state)

> **STATUS:** `accepted` (2026-10-05)

**Topic:** 2026-10-05-mnemonic-central-pg
**Date:** 2026-10-05
**Classification:** high-risk (new trust boundary: network + PG auth + shared data) — T3
**Build shape:** Tracer thread (one observation syncs end-to-end before anything else thickens)
**Queued behind:** 2026-10-02-mnemonic-llm-provider
**Reference design:** engram cloud (`~/git/ai-test/engram` — `internal/cloud/autosync/manager.go`, `internal/cloud/cloudstore/cloudstore.go`, `internal/cloud/auth/foundation.go`)
**Plan source:** `/Users/paladm/.local/share/opencode/plans/mnemonic-central-pg.md` (opencode plan-mode artifact, 2026-10-05)

## Problem / Intent

Mnemonic is a local-first per-machine second brain: each project gets its own SQLite file under `~/.skillgrid/mnemonic/`. Two people working the same codebase each accumulate private memory — decisions, facts, skills, web research — that the other person's agent cannot see. Engram proved the shape for this exact problem: local store stays source of truth per machine, a thin sync process pushes/pulls against a central Postgres, LWW + tombstones, bearer tokens with per-project grants. We want the same capability for mnemonic so a **team** can share one central PG of memory state while every machine keeps working offline.

## Locked decisions (user, 2026-10-05)

1. **Local SQLite stays the working store** on every machine (offline-first, zero UX change). Shared state lives in PG.
2. **Standard PostgreSQL on a team-ops VM** — the team admin installs and operates it. Mnemonic is the only client. No managed-provider quirks: plain `pg_hba.conf`, full role control, `CREATE EXTENSION` available.
3. **Conflicts: last-write-wins per entity key + tombstones.** No CRDTs, no merge. (Engram's documented model.)
4. **Access: managed tokens = PG roles with per-project grants now; OIDC/SSO later (M3).** Deny-by-default.
5. **All scopes sync** (`project`, `user`, `global`). Scope is a search/tag filter, never a privacy boundary (engram `docs/TEAM-USAGE.md:28-37` model; local "personal" is normalized to `user` at the MCP boundary, `mcp/tools_memory.go:240`).
6. **One PG schema per project.** Grants are plain `GRANT`s — no RLS policy code.
7. **Client token storage: file, mode 0600** at `<dataDir>/central.json` (engram `internal/cloudconfig` shape).

## Shared vs local data

**Synced to PG (durable memory layer):** observations (+ versions), prompts, facts, skills, web_cache, sessions (+ session_events), memory edges (edges + memory_relations + observation_relations), teams + team_members.

**Local only, never synced:** code index (chunks, symbols, communities, processes), embeddings (all `*_embeddings`, `vec_*`, `lsh_buckets`), PDG/taint, snapshots, query/retrieval caches, index freshness, config. Rationale: the code index is a derived per-machine artifact of the machine's repo checkout + embedder (ADR-0009, ADR-0016); syncing it would force cross-machine reindex and couple sync latency to embedder availability.

## Success Criteria

- **Purpose:** two machines against one VM Postgres see each other's `mem_save` observations, facts, skills, web_cache, and sessions within one sync cycle (≤30s default), while either machine works fully offline; the agent-facing MCP/HTTP/CLI surface is unchanged.
- **Success criteria (verifiable):**
  1. Machine A saves an observation; machine B sees it in `mem_search` within one sync cycle (30s default, configurable).
  2. Machine A saves, goes offline (PG down); saves queue in the local outbox; on reconnect the queued rows arrive at PG and to machine B with no data loss and no duplicate local rows.
  3. Both machines update the same `sync_id` concurrently: the later `occurred_at` wins on both machines after sync; neither machine errors.
  4. Delete on machine A propagates to B as a tombstone; B's local row soft-deletes; re-save of the same `sync_id` revives it.
  5. A token with no grant to project P cannot read or write P's schema (`42501`-class denial, audited).
  6. `mnemonic central status` shows server, enrolled projects, outbox depth, last push/pull, and backoff state; `mnemonic central doctor` validates DSN, role, grants, and schema version.
  7. With central disabled (default), behavior is byte-identical to today: no PG connection, no new tables touched on the read path, outbox empty.

## Out of scope

- Sharing the code index, embeddings, snapshots, or PDG (per-machine derived state).
- Real-time sync (WebSocket/long-poll) — poll-based, 30s contract; long-poll is a possible M3 add.
- OIDC/SSO (M3), per-project pause switch (M3), dashboard read-model page (M3).
- Full PG-backend swap of the local store (the "Option B" of the plan) — this change is the sync layer; the PG journal is its import path if that ever happens.
- Multi-team / group ACLs, per-user data isolation (engram's documented gap, accepted).
- Supabase/Neon/RDS-specific anything — target is vanilla PostgreSQL 14+ on a VM.

## Requirements

1. **Outbox on every shared-entity write (local).**
   - Current: shared-entity writes (observation save, prompt save, fact save, skill save, web_cache save, session end, edge change, team change) commit to SQLite and stop.
   - Target: after local commit, one `outbox.Record(entity, key, op, payload)` appends to a local `sync_mutations` table (autoincrement seq, `disposition=pending`). Record failures are logged, never fail the local write (fail-open, per ADR-0016 floors).
   - Acceptance: each of the 8 write sites produces exactly one outbox row per local mutation; killing the process between commit and Record loses at most that one row (documented), no partial outbox row.
   - Acceptance scenario: `happy path observation save records outbox row`

2. **PG schema per project + migrations.**
   - Current: no PG anywhere in the codebase.
   - Target: embedded `pgmigrate/*.sql` applied at manager start, idempotent, versioned in `schema_migrations` (per ADR-0008-style discipline; hand-rolled runner like engram `cloudstore.migrate`, no new migration library without an ADR). Per-project schema `mem_<project>` with `mem_observations`, `mem_observation_versions`, `mem_prompts`, `mem_facts`, `mem_skills`, `mem_web_cache`, `mem_sessions`, `mem_session_events`, `mem_edges`, `mem_teams`, `mem_team_members`, `mem_mutations` (journal), `mem_project_meta`, `mem_auth_audit_log` (shared, in `mem_admin` schema).
   - Journal: `mem_mutations` with `seq BIGSERIAL`, `UNIQUE (entity, entity_key, payload_hash)` for idempotent re-push, `(project, entity, since_seq)` cursor index.
   - Materialized `mem_<entity>` tables carry `sync_id UUID`, `created_by`, `occurred_at`, `deleted_at`, `payload JSONB`.
   - Acceptance: fresh PG gets full schema from migrations; re-running migrations is a no-op; a partially-applied migration (process killed mid-run) resumes without error.
   - Acceptance scenario: `happy path pg migrations apply idempotently`

3. **Sync manager: push + pull with lease.**
   - Current: no background sync of any kind in mnemonic.
   - Target: `syncpg.Manager` goroutine started at service boot when `mnemonic.store.central.enabled`; SQLite lease row (60s TTL) in `sync_state` prevents duplicate workers across processes on the same machine; 500ms debounce on outbox dirty + 30s poll; push = batches ≤100 per enrolled project via pgx `COPY`/parameterized insert, ack by accepted count; pull = cursor loop from `last_pulled_seq` until exhausted, applied via per-entity `ApplyPulledMutation` that upserts local SQLite respecting LWW (skip when local `occurred_at` is newer).
   - Backoff on consecutive failures: `min(base·2^n, 5m)` ±25% jitter, paused after 10 with a reason code (`auth_failed`, `policy_denied`, `transport_failed`, `schema_mismatch`, `not_enrolled`).
   - Acceptance: two managers on the same machine never both hold the lease; a killed manager's lease expires within 60s and another takes over; 10 consecutive auth failures pause with reason code visible in `status`.
   - Acceptance scenario: `happy path lease guards single worker per machine`
   - Acceptance scenario: `error path auth failure backs off and pauses with reason code`

4. **LWW + tombstones, end to end.**
   - Current: local supersession is bi-temporal within one store (ADR-0011); no cross-machine story.
   - Target: every synced row carries a stable client-generated `sync_id` (UUIDv7); upsert = later `occurred_at` wins (tie-break: `created_by`); delete = `op=delete` journal row + `deleted_at`; tombstones pruned after `tombstone_retention` (default 30d); identical re-push of same `(entity, entity_key, payload_hash)` is a server no-op.
   - Acceptance: concurrent same-`sync_id` updates converge to the later `occurred_at` on both machines; delete→re-save revives; prune never removes a `sync_id` re-saved after its tombstone.
   - Acceptance scenario: `happy path lww converges concurrent updates`
   - Acceptance scenario: `happy path delete tombstone propagates and revives on re-save`

5. **Bootstrap (first enrollment of a project with local history).**
   - Current: a newly-enrolled project's existing local rows are invisible to the team.
   - Target: `mnemonic central enroll <project>` imports existing local shared-state rows into the outbox as `op=upsert` (LWW means the import loses to anything newer already in PG — documented, acceptable); subsequent writes flow normally.
   - Acceptance: enrolling a project with N local observations results in N outbox upsert rows and, after one cycle, PG holds all N (minus any lost to LWW against newer PG rows).
   - Acceptance scenario: `happy path enroll bootstraps local history`

6. **Tokens, grants, audit.**
   - Current: no PG identity; HTTP surface uses one optional bearer (`SKILLGRID_HTTP_TOKEN`).
   - Target: token format `egm_<env>_<prefix>_<secret>` (32 random bytes secret); client stores it 0600 in `<dataDir>/central.json`; PG stores role `mem_<prefix>` with `LOGIN PASSWORD` (scram-sha-256 via `pg_hba`); a `mem_admin` role (admin DSN in config/env) creates roles, grants per-project schema access, and records HMAC-SHA256(pepper+secret) in `mem_token_meta` for revocation + last-use. `mnemonic central issue-token --project p` prints the token once. Pull and push both filter to the caller's granted schemas. Denials → `mem_auth_audit_log`.
   - Acceptance: token for project A cannot SELECT from `mem_B.mem_observations` (PG-level denial); revoking a token drops/locks its role; last-use timestamp updates on push; audit log row on every denial.
   - Acceptance scenario: `happy path issue token grants only named project`
   - Acceptance scenario: `error path token without grant is denied and audited`

7. **Config + CLI surface.**
   - Current: `mnemonic.*` config exists with behavioral options only; no storage/transport section.
   - Target: new `mnemonic.store.central` block: `enabled` (default false), `dsn` (or `dsn_env`), `admin_dsn` (or `admin_dsn_env`), `projects` (enrollment list), `poll_interval` (default 30s), `push_batch` (default 100), `tombstone_retention` (default 30d), `tls` (default `require`). Env overrides: `SKILLGRID_MNEMONIC_CENTRAL_DSN`, `SKILLGRID_MNEMONIC_CENTRAL_ADMIN_DSN`, `SKILLGRID_MNEMONIC_CENTRAL_ENABLED=1`.
   - CLI: `mnemonic central status | enroll <project> | unenroll <project> | issue-token --project <p> [--role member|admin] | revoke-token <prefix> | doctor`.
   - Acceptance: config tests (defaults off; env wins over file; invalid DSN surfaces at `doctor`, not at save time); each CLI verb has a unit test against a test PG; `status` output is stable JSON for scripting.
   - Acceptance scenario: `happy path central config defaults off`
   - Acceptance scenario: `happy path central doctor validates connection and grants`

8. **No behavior change when disabled.**
   - Target: `enabled: false` (default) → no PG dial, no manager goroutine, outbox table exists but is never read; all existing tests pass unmodified.
   - Acceptance: `go test ./...` green with central unset on a machine without PG; `mnemonic central status` reports `disabled`.
   - Acceptance scenario: `happy path disabled central is a no-op`

## Global constraints (carried into the blueprint)

- Go 1.22+ minimum to build (locked).
- **No new dependencies without an ADR** (locked) — this change's ADR must name `github.com/jackc/pgx/v5` explicitly.
- Local store unchanged: per-project SQLite, WAL, pure-Go driver, embedded migrations (ADR-0012) — the sync layer adds tables, never rewrites existing ones.
- Bi-temporal save path (ADR-0011) remains the local classifier; sync carries the resulting rows, it does not re-classify.
- Fail-open: sync must never block or fail a local `mem_save` (ADR-0016 floors).
- Trivy advisory-only.
- BDD always on: every requirement above maps to a scenario in `acceptance.feature`.
