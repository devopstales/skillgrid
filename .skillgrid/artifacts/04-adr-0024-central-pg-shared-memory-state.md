# Team-shared memory state: local SQLite + outbox journal against a VM PostgreSQL

---
status: "accepted"
supersedes: none
date: 2026-10-05
---

## Context and Problem Statement

Mnemonic is a local-first, per-machine second brain: each project owns one SQLite file
(ADR-0012). A team working the same codebase accumulates private, per-machine memory —
decisions, facts, skills, web research, sessions — that no other member's agent can see.
The user asked for "something similar to engram cloud": a central remote DB a team shares.

Engram's shipped design (reference, `~/git/ai-test/engram`) established the shape:
local store stays source of truth per machine; a thin sync process pushes and pulls
mutation batches against a central Postgres; conflicts are last-write-wins per entity
key plus tombstones; credentials are bearer tokens with per-project grants.

ADR-0012's revisit trigger (1) anticipated exactly this: "the second brain must serve
multiple agents/sessions concurrently with access control … then per-user DB-backed
memory … note the store is already DB-backed, so this is a concurrency/authorization
change, not a storage swap."

## Considered Options

- **A — Sync layer (local SQLite + outbox journal ↔ central PG).** Each machine keeps its
  SQLite store; every shared-entity write also appends to a local outbox; a
  lease-guarded background manager pushes batches to and pulls batches from one shared
  PostgreSQL. LWW per `sync_id` + tombstones. Code index/embeddings never sync.
- **B — PG as the store (driver swap).** Replace the SQLite driver at `store.Open`;
  the team shares one real database, no sync lag. Cost: rewrite FTS5→tsvector/GIN across
  ~8 packages, vec0→pgvector, a parallel PG migration set, and the `ATTACH`-based project
  migrate. Weeks, and it destroys offline-first.
- **C — Engram-style sidecar storage server.** A separate Go service owns the PG;
  clients speak its HTTP protocol. Adds an always-on process the team must host and
  patch; the user explicitly wanted "mnemonic only connects to [the PG]".
- **D — Files over a sync protocol (restic/syncthing-style).** Sync the SQLite files
  themselves. File-level sync conflicts with WAL and per-process locks; no per-row LWW.

## Decision Outcome

Chosen option: **A — sync layer**, with these locked specifics (user decisions,
2026-10-05):

1. **Local SQLite remains the working store on every machine** (offline-first; zero
   change to the MCP/HTTP/CLI read paths). PG holds only the shared state.
2. **The central DB is a standard PostgreSQL (14+) on a team-ops VM**, installed and
   operated by the team admin. Mnemonic is the only client (no sidecar service —
   option C rejected). `pg_hba.conf` scram-sha-256; TLS mode `require` by default.
3. **Conflict policy: last-write-wins per entity key + tombstones.** Stable
   client-generated `sync_id` (UUIDv7); upsert wins on later `occurred_at`
   (tie-break `created_by`); delete is a tombstone journal row, pruned after
   `tombstone_retention` (default 30d). No CRDTs, no version vectors, no merge —
   engram's documented model.
4. **Synced set:** observations (+versions), prompts, facts, skills, web_cache,
   sessions (+session_events), memory edges, teams/team_members. **Never synced:**
   code index (chunks/symbols/communities/processes), embeddings, vec tables, PDG/taint,
   snapshots, query/retrieval caches, config — all per-machine derived state.
5. **Access: managed tokens = PG roles.** Token `egm_<env>_<prefix>_<secret>` maps to
   PG role `mem_<prefix>`; grants are per-project schema usage; deny-by-default;
   HMAC-peppered secret hash in `mem_token_meta`; auth denials audited. OIDC later (M3).
6. **One PG schema per project** (`mem_<project>`) — grants stay plain `GRANT`s; no RLS.
7. **All scopes sync** (`project`, `user`, `global`). Scope is a search/tag filter,
   never a privacy boundary (engram `docs/TEAM-USAGE.md:28-37`; local "personal" is
   normalized to `user` at the MCP boundary).
8. **Client token storage: `<dataDir>/central.json`, mode 0600.**

### Consequences

- Good, because offline-first survives: PG down = outbox grows, nothing is lost, local
  behavior is unchanged (ADR-0016 fail-open floors hold).
- Good, because the agent-facing surface (MCP/HTTP/CLI) is untouched; sync is a new
  background concern, not a new read path.
- Good, because the PG journal is content-complete: it doubles as the bootstrap/import
  path for a future full-PG mode (option B) without re-designing the wire.
- Good, because grants are plain `GRANT`s on per-project schemas — no RLS policy code,
  no provider-specific auth.
- Bad, because team visibility is eventual (poll interval, default 30s), not real-time.
- Bad, because `scope: user/global` rows are team-visible by decision — an agent that
  saved a "personal" note under `user` scope shares it with the team.
- Bad, because two machines diverging on the same `sync_id` converge by wall-clock
  `occurred_at`, which can skew slightly across machines; tie-break is `created_by`,
  never user input.

Revisit triggers: (1) sync latency contract (<30s) is violated by workload — add
long-poll or a notify channel; (2) the team needs per-user isolation — the scope-as-tag
model must give way to per-user rows/grants; (3) full-PG mode (option B) is adopted —
the journal becomes the import source and the outbox is retired; (4) OIDC is adopted —
token minting moves to an IdP-anchored flow, data plane unchanged.
