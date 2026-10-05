# ADR — Mnemonic Scan Findings Store: structured tables, not blobs

**Date:** 2026-10-05
**Status:** accepted
**Deciders:** paladm
**Related:** ADR-0012 (SQLite as second brain), ADR-0016 (second-brain capability layer), ADR-0024 (central PG — future sync of these tables)

## Context

Scanner output (trivy/vuln+sbom, wapiti, nuclei, semgrep) needs a durable home so an agent can answer "did this CVE get fixed?", "what does this dep update touch?", and "is this dep declared but never imported?" without re-running the scanner. Three shapes were on the table.

## Decision

**Option A — structured `scan_*`/`dep_*` tables + dedicated MCP tools.**

- `scans` (one row per invocation), `findings` (+ FTS5) per finding, `dependencies` upserted by `purl`, `dep_edges` for `depends_on`, and the runtime import tree reusing the code index `edges` table (no new extraction pass).
- Raw artifacts stored as files under `<data_dir>/.skillgrid/cache/scans/<scan_id>.json`, mirroring `webcache`.

### Why not the alternatives

- **Blob table (one JSON column per scan):** cheapest to build, but every query (diff, severity filter, dependency walk) re-parses the blob in Go. No FTS. The diff is the *point* of this feature and needs a stable hash column, not a re-parse each time. Rejected.
- **Per-tool tables (`trivy_findings`, `nuclei_results`, …):** natural to each tool's shape, but the diff and the cross-tool `scan_status` want a common column vocabulary (`severity`, `file`, `line`, `dedup_hash`). N tools × 4 scanners = N duplicated FTS indexes and N duplicated diff paths. Rejected.
- **Dependency-Track as the dependency store:** DT is advertised in the MCP toolset but rejected at runtime this session (KubeDash BOM not loaded); making mnemonic depend on a live DT server couples a local-first store to network availability, contrary to ADR-0012. DT stays an optional external source; `purl`-keyed local tables are the source of truth.

## Consequences

- **Good:** one diff path across all tools; FTS over findings; dependency blast-radius + declared-vs-imported are first-class queries; no new Go dependencies; `edges` reuse means no reindex and no new extractor for the runtime tree.
- **Cost:** migration 050 adds four tables to every store on next open (additive, ADR-0012-safe). The runtime-tree diff is computed in the tool layer (declared from manifest provenance, imported from `edges`), so that logic must stay correct as `edges` schema evolves.
- **Future:** these tables are natural central-PG sync candidates (ADR-0024); if shared memory is wanted later, they join the outbox pattern without schema change.
- **Open (accepted):** no scan TTL (user decision) — scans accumulate; pruning is a later operational concern, not a schema one.
