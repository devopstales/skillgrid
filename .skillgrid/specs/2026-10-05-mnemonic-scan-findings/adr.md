# ADR Review Manifest

- **Change:** `2026-10-05-mnemonic-scan-findings`
- **Status:** completed
- **Review date:** 2026-10-05

## In-Force ADRs Reviewed

- `.skillgrid/artifacts/04-adr-0012-sqlite-as-second-brain.md` — per-project SQLite store, additive migrations only; migration 050 adds four new tables.
- `.skillgrid/artifacts/04-adr-0016-second-brain-capability-layer.md` — fail-open floors: a scan/ingest error never blocks or fails the calling session.
- `.skillgrid/artifacts/04-adr-0024-central-pg-shared-memory-state.md` — these tables are future central-PG sync candidates (outbox pattern, no schema change).

## New Durable ADRs Created

- `.skillgrid/artifacts/04-adr-0030-scan-findings-structured-store.md` — scanner output lives in structured `scan_*`/`dep_*` tables + FTS with dedicated MCP tools, not blobs or per-tool tables.

## Supersessions

- None.
