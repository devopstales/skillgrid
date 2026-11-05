---
status: accepted
supersedes: none
date: 2026-11-01
---

# ADR-0041: Mnemonic v2 Postgres driver — lib/pq


## Context

Mnemonic v2's `Store` interface (spec `001-storage` §7) is dialect-agnostic: SQLite (default, local truth) and Postgres (replication target or direct-PG dialect render). The SQLite leg uses `modernc.org/sqlite` (pure-Go, cgo-free — already in v1's go.mod). The Postgres leg needs a `database/sql` driver.

Per AGENTS.md § Rules: "No new dependencies without an ADR."

## Decision

Use `github.com/lib/pq` as the Postgres driver. It is a pure-Go `database/sql` driver (no CGo), the most widely-used Postgres driver for Go, and it supports everything the spec requires: `READ COMMITTED` isolation, `BIGSERIAL`, `TIMESTAMPTZ`, `JSONB`, `tsvector`, `vector` (pgvector via raw SQL), and connection pooling via `database/sql`.

The driver is confined to `internal/core/store/postgres.go` — the Go driver surface is `database/sql`, so a future driver swap (e.g. `pgx`) is a one-file change.

## Alternatives considered

- **`pgx`** — pure-Go, higher performance, native `vector` type support. Rejected for v2: the spec's PG surface is SQL-transaction-based (push/pull batches), not a high-throughput query workload; `lib/pq` is sufficient and more familiar. Revisit if profiling shows `database/sql` overhead.
- **`jackc/pgx` stdlib-compatible mode** — same surface as `lib/pq` via `database/sql`. Rejected: no advantage over `lib/pq` for this workload.

## Consequences

- **Positive:** minimal new dependency surface (one module); the cgo-free invariant holds (`lib/pq` is pure-Go); the driver is swappable behind `database/sql`.
- **Negative:** one more module in `go.sum`; the PG tests need a reachable Postgres instance (`MNEMONIC_PG_TEST_DSN`) — CI must provide one or the tests skip.
- **One-way door:** the `lib/pq` import in `postgres.go` is the only place the driver appears. Swapping to `pgx` later is a contained change but requires re-running the full PG parity suite (Task 18's tests).
