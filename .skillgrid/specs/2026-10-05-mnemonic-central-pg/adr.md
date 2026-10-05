# ADR Review Manifest

- **Change:** `2026-10-05-mnemonic-central-pg`
- **Status:** in review
- **Review date:** 2026-10-05

## In-Force ADRs Reviewed

- `.skillgrid/artifacts/04-adr-0012-sqlite-as-second-brain.md` — SQLite stays the per-machine source of truth; this change adds a sync layer, not a storage swap. Its revisit trigger (1) — "multiple agents/sessions concurrently with access control … a concurrency/authorization change, not a storage swap" — is exactly the condition this change now satisfies. Local store untouched.
- `.skillgrid/artifacts/04-adr-0011-observations-are-bitemporal.md` — the local AUDN classifier stays the single writer of `valid_at/invalid_at/superseded_by`; sync transports the resulting rows and never re-classifies.
- `.skillgrid/artifacts/04-adr-0016-second-brain-capability-layer.md` — fail-open floors hold: sync never blocks or fails a local write.
- `.skillgrid/artifacts/04-adr-0005-trust-boundary-mcp-spawn-model.md` — the MCP-spawned process gains one new external dependency (the PG over the network); the token file (0600) is the credential at that boundary.
- `.skillgrid/artifacts/04-adr-0009-vector-search-in-sql-latency-viant-deferred.md` — embeddings/code index remain local-only; nothing in this change touches the vector path.
- Locked constraint "No new dependencies without an ADR" — satisfied by ADR-0024 naming `pgx/v5`.

## New Durable ADRs Created

- `.skillgrid/artifacts/04-adr-0024-central-pg-shared-memory-state.md` — team-shared memory state on a VM PostgreSQL: local SQLite + outbox journal, LWW + tombstones, tokens-as-roles, schema-per-project, all scopes synced, code index excluded.

## Supersessions

- None. ADR-0012 remains in force (local engine unchanged); ADR-0024 adds the team-sharing layer above it.
