# ADR Review Manifest

- Status: completed
- Review date: 2026-08-27

## Review Summary

ADR review completed for this change.

## In-Force ADRs Reviewed

- None - `adr/` has no in-force ADRs.

## New Durable ADRs Created

- None - no major durable architectural decisions were introduced. The MemIndex architecture (SQLite + FTS5, two transports over one service layer) is a significant subsystem but its patterns (MCP stdio, HTTP REST, incremental indexing) are well-established. Future ADRs may be warranted if hybrid embeddings (v2) or bi-temporal memory (v1.1) are implemented.
