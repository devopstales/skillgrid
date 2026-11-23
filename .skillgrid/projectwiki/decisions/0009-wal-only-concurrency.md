# Concurrency: WAL + busy_timeout only (no leases)

---
status: accepted
date: 2026-11-23
supersedes: none
---

## Context and Problem Statement

[[entities/mnemonic]] is a local file that several processes can touch: the
MCP server (one per agent) plus CLI invocations, sometimes concurrently,
against one project's `.sqlite`. The predecessor carries a sync-lease table
(`004_sync_lease.sql`) — but that lease exists for **cloud sync** (two
devices, one brain), a concern v1 drops.

## Considered Options

- **WAL mode + `busy_timeout=5000` + `synchronous=NORMAL`**: SQLite's native
  multi-connection story — many readers, one writer, writers wait out short
  contention. Zero application code.
- **Application-level leases** (inherited table): a lease row per writer
  with expiry — correct for cross-device sync, heavy for local multi-
  process writes.
- **A queue/daemon owning all writes**: serialization by architecture —
  the very daemon v1 has no [[decisions/0008-memory-core-only-v1]].

## Decision Outcome

Chosen option: **WAL + busy_timeout only.** Every store open sets
`PRAGMA journal_mode=WAL`, `busy_timeout=5000`, `synchronous=NORMAL`. No
lease table, no writer queue, no daemon. Concurrency that outgrows this (a
burst of >5 s writer contention) is a local anomaly to be measured, not
pre-solved.

### Consequences

- Good: zero application concurrency code in v1; the whole story is three
  PRAGMAs + a documented assumption.
- Good: a `SQLITE_BUSY` after the timeout is a loud, debuggable error, not a
  silent lease-expiry race.
- Bad: two long writers can still block each other up to 5 s — accepted at
  agent-write cadence (saves are sentence-sized).
- Bad: the assumption must be re-checked when the deferred `code` tool
  lands (bulk indexing is a long writer) — revisit trigger recorded.

## Not adopted

- **Lease table** — a cloud-sync artifact (two devices) imported into a
  single-machine problem; YAGNI.
- **Write queue/daemon** — an architecture for a problem the busy timeout
  already handles; the daemon is otherwise deferred.

## Related

- Supersedes: none.
- Depends on: [[decisions/0003-per-project-sqlite-file]] (per-project
  files make the contention domain small), [[concepts/degradation-floor]]
  (loud-failure culture).
- Consumed by: [[architecture/mnemonic-memory-core]] (store layer),
  [[syntheses/mnemonic-core]].
