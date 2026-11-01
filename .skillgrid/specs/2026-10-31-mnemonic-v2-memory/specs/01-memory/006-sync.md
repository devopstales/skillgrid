# 006 — Sync (full replication protocol)

**Status:** DRAFT · **Owner:** memory change
**Depends on:** `001-storage` (dual-mode, INV-4), `002-schema` (§3 sync domain)
**Reference:** Engram `internal/sync/sync.go` (git chunks), `internal/cloud/autosync/
manager.go` (lease-guarded autosync), `internal/cloud/remote/transport.go`
(`MutationTransport`), `cloudstore.go` (cloud Postgres schema), `chunkcodec.go`
(chunk id = 8 hex of SHA-256).

---

## 1. Capability

The memory function is **local-first with opt-in remote replication**. Local SQLite is
always the source of truth (INV-4); the remote — **a self-hosted PostgreSQL the team
runs** — receives *replicated mutations*, never direct writes. This is the "full sync in
memory" decision — the complete replication protocol is specified here.

**There is no mnemonic-managed cloud.** No `mnemonic cloud serve`, no HTTP sync API, no
mnemonic auth service. The team hosts the Postgres; mnemonic connects to it over a DSN.

## 2. Project isolation (the core constraint)

**Each project's data is separated, on both sides. Never everything in one DB.**

- **Local (SQLite): one file per project.** `<project>.sqlite` under the data dir — the
  same isolation v1 had. A project's observations/sessions/sync journal never share a
  file with another project's.
- **Remote (Postgres): one schema per project.** The team's PG server is shared
  infrastructure, but each project is replicated into **its own Postgres schema**
  (`schema_<sanitized_project>`), so project A's tables are not interleaved with
  project B's. The DSN in config selects the *server*; the project is resolved at
  connect time and the store sets its `search_path` (or qualifies every table) to that
  project's schema. A connection that cannot resolve a project schema fails fast — it
  never falls back to a shared `public` schema.

| Side | Isolation unit | Consequence |
|---|---|---|
| SQLite | one `.sqlite` file per project | no cross-project query possible without opening the other file |
| Postgres | one schema per project | `schema_<project>.observations`, etc.; no cross-project JOIN |

This means the **PG side has no global `cloud_*` tables** (no shared users/principals/
grants) — those were a product of the dropped cloud-service model. Auth is per-project
PG roles; the schema *is* the isolation boundary.

## 3. The model: mutation journal + last-write-wins

```
Local Write
   → Store.AddObservation / Update / Delete / Relation
   → [one transaction] insert entity + FTS + enqueue sync_mutations row (seq++)
   → onWrite hook → Autosync.NotifyDirty()   (buffered-1 channel)
                       → debounce 500ms
                       → cycle():
                           ├─ AcquireSyncLease (SQLite)
                            ├─ push: ListPendingMutations → INSERT batch into
                            │         schema_<project>.mutations   (one PG transaction)
                            └─ pull: SELECT schema_<project>.mutations since cursor
                                     → Store.ApplyPulledMutation (defer if FK missing)
                        → status → `mnemonic sync status`
```

All remote I/O is **direct SQL to the project's PG schema** (§2) — no HTTP, no server.

**Conflict model: last-write-wins, no CRDT** (Engram's explicit stance). Ordering is
handled by `seq` + tombstone floors + deferred-apply. **Distributed atomicity is
disclaimed** — a receiver may defer a survivor upsert while its parent session is
missing; LWW can resurrect a source after a later peer edit. This is a known, accepted
limit, documented on the consolidation spec.

## 4. The mutation journal (`sync_mutations`)

Every local write enqueues a mutation **in the same transaction** as the entity write:

| Field | Value |
|---|---|
| `seq` | monotonic per store (the ordering cursor) |
| `target_key` | `cloud` \| `cloud:inbox` \| `local` |
| `entity` | `session` \| `observation` \| `prompt` \| `relation` \| `project` |
| `entity_key` | the `sync_id` (INV-6) |
| `op` | `upsert` \| `delete` |
| `payload` | the **canonical** entity — project-canonicalized (Engram
  `chunkcodec.CanonicalizeForProject`): effective project resolved, blank-project
  inheritance applied, redaction/truncation already done |
| `source` | `local` \| `remote` |
| `project` | |
| `acked_at` | set when the remote acknowledges |

**Enrollment:** only enrolled projects (`sync_enrolled_projects`) generate remote journal
entries. Unenrolled projects' writes never enqueue mutations for the remote; a delete on
an unenrolled project supersedes pending source mutations via the deletion lifecycle.

## 5. The transport

The remote is **a self-hosted PostgreSQL the team runs** — not a mnemonic-managed
service. There is **no `mnemonic cloud serve`** and no HTTP sync API. The store talks to
PG **directly over a DSN** (`001-storage` §6), scoped to the **project's schema** (§2).
Push/pull are **SQL transactions**, not HTTP calls.

`MutationTransport` (interface; the PG implementation issues SQL against
`schema_<project>`):

```go
type MutationTransport interface {
    PushMutations(ctx, project string, batch []Mutation) (ackedSeq int64, err error) // ≤100/batch
    PullMutations(ctx, project string, sinceSeq int64) ([]Mutation, error)
    EnsureSchema(ctx, project string) error   // CREATE SCHEMA IF NOT EXISTS schema_<project>
}
```

- **Push:** `INSERT` the batch of mutations (≤100, `sync.max_bytes` 8 MiB) into
  `schema_<project>.mutations`, in one transaction. A paused project is skipped and
  reported as `sync-paused` in `sync status`.
- **Pull:** `SELECT` from `schema_<project>.mutations` since a cursor, **scoped to that
  project's schema** — no cross-project reads.
- **Auth:** the PG **role/credentials in the DSN** (or a per-project PG role). No bearer
  token, no JWT, no separate auth service — Postgres authentication is the auth.
- **Schema bootstrap:** `EnsureSchema` creates `schema_<project>` (and its tables) on
  first sync for that project. A connect that cannot resolve the project schema fails
  fast; it never falls back to a shared `public` schema.

## 6. Applying pulled mutations (`ApplyPulledMutation`)

A pulled mutation is applied to the local store through the **same store methods** a
local write uses (no special remote path — INV-4: local is always the authority over
its own rows).

- **Deferred-apply:** if the mutation's FK dependency is missing (e.g. a relation whose
  endpoint observation hasn't been pulled yet, or a survivor whose parent session is
  missing), it goes to `sync_apply_deferred` and is retried. **5 failed replays → dead.**
  Dead rows are pruned; eligible dead relations re-arm only when the original scoped
  endpoint predicate is satisfied. No project rewriting or automatic ownership repair.
- **Tombstone floors:** `sync_delete_tombstones` + `…_remote_floors` prevent the
  classic race where a delete is followed by a *late* upsert that resurrects the row.
  A pulled delete for a live prompt records the live row's session-and-inbox identity,
  not conflicting identity fields from its payload.
- **Identity conflicts:** a pulled prompt upsert cannot move an established nonempty
  inbox identity to a different session; import adoption refuses cross-project
  reassignment. Backup import rejects a tombstone that would delete a matched live
  prompt from a different effective project (rolls back atomically).

## 7. Sync state and lifecycle (`sync_state`)

Per `target_key`: `lifecycle` (`idle`/`pending`/`running`/`healthy`/`degraded`/`inbox`),
`last_enqueued_seq`/`last_acked_seq`/`last_pulled_seq`, backoff, lease, reason codes.
`mnemonic sync status` reports this. The dashboard pill (when present) reads it.

## 8. Autosync manager (background)

A **lease-guarded background goroutine** started by `mnemonic mcp` (and `mnemonic serve`
alias, when present) on the **local** side when `sync.auto=true`
(`MNEMONIC_SYNC_AUTO=1`). There is no remote-side daemon — the team's Postgres is just a
database; autosync runs in whichever local process is active. Engram's invariants:

- **All network I/O in its own goroutine** — never holds locks shared with the SQLite
  write path. The `onWrite` hook is a buffered-1 channel; the store never blocks on the
  network.
- **Lease:** SQLite-backed. `StopForUpgrade` sets `PhaseDisabled` and does *not* release
  the lease (no other worker picks up sync during an upgrade). `ResumeAfterUpgrade`
  clears the flag and re-arms without restarting.
- **Debounce + backoff + jitter:** 500 ms debounce; exponential backoff on
  push/pull failure with jitter.
- **Phases:** `idle`/`pushing`/`pulling`/`push_failed`/`pull_failed`/`backoff`/
  `healthy`/`disabled`.

## 9. The two sync transports (parity)

Two dialect-neutral transports for getting a project's memory to/from the remote:

1. **Live mutations** (this spec, §3-7): the replication path to the team's Postgres —
   direct SQL into `schema_<project>.mutations`.
2. **Chunk export/import** (the git path): `mnemonic sync export` / `import` produce a
   portable archive — `manifest.json` + `chunks/*.jsonl.gz` (gzip JSONL). **Chunk id =
   first 8 hex of SHA-256 of the payload** (Engram `chunkcodec`). The archive is
   dialect-neutral (JSON), so a SQLite local can export and a peer (or the PG side, via
   import) can load it. This is the "carry my memory in git" path and the backup path.

Both share the canonical payload (project-canonicalized) and the tombstone/deferred
semantics.

## 10. Remote schema and controls (Postgres)

Because the remote is the team's own Postgres with **one schema per project** (§2), there
are **no global `cloud_*` tables** — no shared users/principals/grants (those were the
dropped cloud-service model). Each `schema_<project>` holds only that project's
replicated data:

| Table (in `schema_<project>`) | Purpose |
|---|---|
| `mutations` | the replicated mutation journal (`seq BIGSERIAL`, mirrors the local `sync_mutations`) |
| `observations`, `sessions`, `user_prompts`, `memory_relations` (+ FTS `tsv` columns) | the project's replicated rows (the same entities as local, PG-dialect) |
| `tombstones` | delete floors for the project |
| `apply_deferred` | pulled mutations deferred for missing FKs (5-retry cap) |
| `sync_state` | the project's per-target lifecycle / cursors (mirrors local) |
| `project_controls` | per-project sync pause flag (the only "control" row) |

- **Per-project sync pause:** a row in `schema_<project>.project_controls`. A paused
  project's push writes no rows and is reported as `sync-paused` in `sync status`; reads
  are unrestricted.
- **Auth:** per-project PG role (or the DSN's role). No bearer token, no JWT.
- **Allowlist:** which projects may sync is controlled locally by `sync_enrolled_projects`
  (`002-schema` §3.3) — the remote has no allowlist of its own; it just holds schemas for
  the projects that enrolled.

## 11. Invariants

INV-4 (local truth — remote never writes local directly; it applies through the store),
INV-6 (mutations keyed on sync_id), INV-2 (both transports dialect-neutral).

## 12. Known limits (accepted, documented)

- **LWW, no CRDT.** Concurrent edits to the same `sync_id`: the later-applied wins.
  No field-level merge.
- **Not distributed atomicity.** Consolidation, in particular, commits locally but a
  receiver may defer the survivor. LWW can resurrect a consolidated source after a
  later peer edit.
- **Mixed-version payloads:** older clients can strip unknown fields when relaying, so
  mixed-version provenance preservation is not unconditional (Engram's note).

## 13. Open items

None — OPEN-4 (PG reachability) resolved in `001-storage` §6; the protocol is complete.
