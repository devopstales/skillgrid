# Mnemonic Scan Findings + Dependency Graph (structured scanner store) — Design Briefing (requirements & intent)

> **STATUS:** `accepted` (2026-10-05)

**Topic:** 2026-10-05-mnemonic-scan-findings
**Date:** 2026-10-05
**Classification:** medium (new tables + new MCP tool surface; no new trust boundary, no new dependencies) — T2
**Build shape:** Tracer thread (one trivy scan ingests end-to-end before the other scanners thickening)
**Queued behind:** 2026-10-05-mnemonic-central-pg
**Reference design:** `skillgrid-cli/internal/mnemonic/webcache/service.go` (FTS + raw-file store + `DefaultDataDir()`), `internal/mnemonic/store/migrations/` embedded-migration discipline

## Problem / Intent

Security scanners (trivy, wapiti, nuclei, semgrep) run ad-hoc and their results evaporate: an agent re-runs the scan next session, nobody can ask "did this CVE get fixed since last week?", "what does this library update touch?", or "is this dep declared but never imported?". The code index already models *where* code is; the memory store already models *what we know*. This change gives scanner output a durable, queryable home in the same per-project SQLite store — so findings, dependency graphs, and the declared-vs-imported runtime import tree are first-class, searchable, diffable memory.

The shape mirrors the web-cache: structured rows + FTS5 for search, raw artifact files on disk (out of repo, under the mnemonic data dir), and a set of read tools. Scanners are the *writers*; mnemonic is the *store + query surface*.

## Locked decisions (user, 2026-10-05)

1. **Schema Option A** — new `scan_*` + `dep_*` structured tables with dedicated MCP tools (not a blob table, not per-tool tables).
2. **Day-one scanners:** trivy (vuln + sbom), wapiti, nuclei, semgrep.
3. **Dependency graph included** (package + `depends_on` edges), upserted by `purl`.
4. **Runtime import tree included** (declared-vs-imported) — computed from the existing code index `edges` table (`kind IN ('imports','dynamic_import')`), **no new extraction pass** (WS7 reuses `edges`; resolved 2026-10-05 by reading `001_schema.sql:564` + `graph_enrichment_035_test.go`).
5. **semgrep skill added** + its install step (`uv tool install semgrep`); trivy/nuclei/wapiti already in `SecurityTools()`.
6. **Raw scan artifacts** live at `<mnemonic_data_dir>/.skillgrid/cache/scans/<scan_id>.json` (out of repo; data dir = `~/.skillgrid/mnemonic/<project>/` per `webcache.DefaultDataDir()`).
7. **No scan TTL** — scans are not pruned by time (the user's explicit choice; no expiry column).
8. **Severity normalization map** — trivy identity (`UNKNOWN`→`INFO`); wapiti/nuclei capitalize; semgrep `ERROR`→`HIGH`, `WARNING`→`MEDIUM`, `NOTE`→`INFO`.
9. **`dedup_hash = sha256(tool, cve|rule_id, pkg, version, file, line)`**; `scan_diff` compares hash sets across `scan_id`s.
10. **`scan_diff` supports `--latest`** (compare the two most recent `scan_id`s).

## Data model

**`scans`** — one row per scan invocation. `id` (UUIDv7), `tool`, `target`, `scanners` (e.g. `vuln,sbom`), `status` (ok|error|partial), `started_at`, `finished_at`, `finding_count`, `raw_path`, `error`. No TTL columns.

**`findings`** — one row per normalized finding. `scan_id`, `tool`, `dedup_hash`, `severity` (normalized), `title`, `cve_id`, `rule_id`, `package`, `version`, `fixed_version`, `file`, `line`, `message`, `links` (JSON array). FTS5 index over `title, cve_id, rule_id, package, message, file`.

**`dependencies`** — upserted by `purl`. `purl` (PK), `name`, `version`, `ecosystem`, `manifest` (provenance — which manifest produced it), `retired` (soft-retire flag, never deleted), `last_seen` (scan timestamp). Soft-retire: a dep absent from a newer ingest gets `retired=1`, `last_seen` preserved — never a hard delete.

**`dep_edges`** — `depends_on` relations. `from_purl`, `to_purl`, `version_range`. Replaced per-ingest (delete-and-reinsert within the ingest's transaction), so the graph reflects the latest SBOM.

**Runtime import tree (no table)** — `dep_runtime <file>` queries the existing `edges` table for `kind IN ('imports','dynamic_import')` where `file_id` is the changed source file. Declared = parsed from lockfile/manifest at ingest (carried on `dependencies.manifest`); imported = the `edges` query. The declared-vs-imported diff is computed in the tool layer, not the schema.

## Tool surface

**Scan tools:** `scan_start` (runs a scanner via its CLI/MCP, returns `scan_id`), `scan_store_findings` (ingest a raw artifact into `findings`), `scan_list` (recent scans, filterable by tool/status), `scan_get <scan_id>` (scan row + finding summary), `scan_status` (counts by tool/severity, freshness), `scan_diff [--latest | <old> <new>]` (added/removed `dedup_hash` sets).

**Dependency tools:** `dep_ingest` (parse an SBOM/lockfile into `dependencies` + `dep_edges`, soft-retire absent purls), `dep_list` (current purls, filter retired), `dep_get <purl>` (dep row + edges), `dep_affected <purl>` (reverse `depends_on` walk — what breaks if this changes), `dep_graph` (full `from`→`to` edge set), `dep_runtime <file>` (declared-vs-imported import tree for a changed file).

## Success Criteria

- **Purpose:** a scanner run leaves behind queryable memory — re-scan diff, dependency blast-radius, and declared-vs-imported import tree are all answerable without re-running the scanner.
- **Success criteria (verifiable):**
  1. `scan_start` (trivy vuln) on a fixture with a known CVE produces a `scans` row + `findings` rows with a stable `dedup_hash`; the raw JSON is written to `<data_dir>/.skillgrid/cache/scans/<scan_id>.json`.
  2. Re-running the same scan on unchanged code produces a `scan_diff --latest` with **zero** added findings (hashes identical) — the diff is reproducible.
  3. A `scan_diff` after fixing the CVE reports that `dedup_hash` under **removed** (fixed) and the `findings` row is retained for audit (soft, not deleted).
  4. `scan_list --tool trivy --severity CRITICAL` returns only matching rows; `scan_status` returns correct counts by tool/severity.
  5. `dep_ingest` on a fixture SBOM upserts by `purl`; re-ingest with one package removed soft-retires it (`retired=1`, row survives); `dep_affected` on a leaf package returns its reverse edge set.
  6. `dep_runtime <changed-file>` returns the file's `edges` imports; a package in the manifest but never imported is flagged declared-only, and an imported-but-not-declared symbol is flagged import-only.
  7. `semgrep` is in `SecurityTools()` and installs via `uv tool install semgrep`; `task install-deps` surfaces it alongside trivy/nuclei/wapiti3.
  8. The new migration is additive (new tables only); `go test ./...` is green with no scanner binaries present (tests use fixture JSON, not live scans).

## Out of scope

- Central-PG syncing of scans/deps (the central-pg spec owns the sync layer; scans join it in a later pass if shared memory is wanted).
- Live-scan integration in `go test` (scanners run in CI/functional layer, not unit tests — fixtures are the test doubles).
- Dependency-Track as a live "library impact" source (it is advertised in the MCP toolset but rejected at runtime this session; DT stays an optional external source, not a mnemonic dependency).
- Per-scan retention/TTL (explicitly out — no time-based pruning).
- Storing full scanner stdout/stderr (only the parsed artifact is kept; the artifact is the raw file).

## Requirements

1. **Schema (migration `050_scan_findings.sql`).**
   - Additive: `scans`, `findings` (+ `findings_fts` FTS5 + triggers mirroring the 001/045 FTS pattern), `dependencies` (PK `purl`), `dep_edges` (`UNIQUE(from_purl, to_purl)`).
   - No column changes to existing tables; `edges` (001) is read, not modified.
   - Acceptance scenario: `happy path scan migration applies idempotently`

2. **Ingest service (`scan` package).**
   - `Start(ctx, tool, target, scanners) (scanID string, rawPath string, err error)` — shells out to the scanner CLI (or calls the trivy MCP server for trivy), writes raw JSON to `<data_dir>/.skillgrid/cache/scans/<scanID>.json`, inserts a `scans` row (`status=ok`/`error`).
   - `StoreFindings(ctx, scanID, tool) error` — parses the raw artifact, applies the severity map, computes `dedup_hash`, upserts `findings`, updates `scans.finding_count`/`status`.
   - Per-scanner parsers: `trivy` (vuln list + sbom), `wapiti` (report), `nuclei` (JSONL), `semgrep` (JSON). Each is pure (artifact bytes → `[]Finding`), unit-testable with fixtures.
   - Fail-open: a scanner error records `status=error` + `error` text; it never fails the caller's session (ADR-0016 floors).
   - Acceptance scenario: `happy path trivy scan ingests findings with stable hash`
   - Acceptance scenario: `error path scanner failure records status=error and is fail-open`

3. **Diff.**
   - `Diff(ctx, oldID, newID) (DiffResult, error)` — set difference of `dedup_hash` per `scan_id`: `added`, `removed`, `unchanged`. `--latest` resolves the two most recent `scans` by `started_at`.
   - Removed findings are **not deleted** (audit trail); the diff is a hash-set computation.
   - Acceptance scenario: `happy path unchanged re-scan diffs to zero added`
   - Acceptance scenario: `happy path fixed cve appears as removed in diff`

4. **Dependency graph (`dep` package).**
   - `Ingest(ctx, scanID, sbomPath) error` — parse SBOM/lockfile → upsert `dependencies` by `purl` (set `last_seen`, clear `retired` if present); replace `dep_edges` for the ingested set; soft-retire purls absent from the ingest.
   - `Affected(ctx, purl) ([]string, error)` — reverse `depends_on` walk (BFS, depth-capped at 10), returns the purls that transitively depend on `purl`.
   - `Runtime(ctx, sourceFile) (RuntimeResult, error)` — query `edges` (`kind IN ('imports','dynamic_import') AND file_id = <resolved>`) for the imported set; diff against the manifest's declared set (from `dependencies.manifest` provenance). Returns `declaredOnly`, `importOnly`, `shared`.
   - Acceptance scenario: `happy path dep ingest upserts by purl and soft-retires absent`
   - Acceptance scenario: `happy path dep_affected returns reverse dependency set`
   - Acceptance scenario: `happy path dep_runtime flags declared-only and import-only`

5. **Install: semgrep.**
   - Add to `SecurityTools()` (`install/config.go:117`): `{Name: "semgrep", Manager: "uv", InstallArgs: []string{"tool", "install", "semgrep"}, Bin: "semgrep", Hint: "curl -LsSf https://astral.sh/uv/install.sh | sh"}`.
   - No new install manager — `uv` is already used by `wapiti3`.
   - Acceptance scenario: `happy path semgrep installs via uv`

6. **Skills.**
   - Create `~/.agents/skills/verification/semgrep/SKILL.md` (mirror of `wapiti`/`nuclei` structure).
   - Add a "Store results in mnemonic" section to the `trivy`, `wapiti`, `nuclei` skills: after a scan, call `scan_start`/`scan_store_findings` (and `dep_ingest` for SBOM) so results persist.
   - Acceptance scenario: `happy path semgrep skill exists and documents mnemonic store step`

7. **MCP registration.**
   - `mcp/server.go` — add `registerScanTools(s)` + `registerDepTools(s)` alongside the existing `registerXxxTools` (server.go:42-72).
   - Acceptance scenario: `happy path scan and dep tools are registered`

8. **No behavior change for existing tools.**
   - The new tables and tools are additive; `mem_*`/`code_*`/`web_*` surfaces are untouched.
   - Acceptance scenario: `happy path existing tool surface is unchanged`

## Global constraints (carried into the blueprint)

- Go 1.22+ minimum to build (locked).
- **No new dependencies without an ADR** — this change adds none (parsers use stdlib `encoding/json` + existing `modernc.org/sqlite`; trivy may run via the already-configured MCP server or its CLI).
- Local store unchanged: per-project SQLite, WAL, pure-Go driver, embedded migrations (ADR-0012) — migration 050 is additive tables only.
- Raw artifacts out of repo under `<data_dir>/.skillgrid/cache/scans/` (mirrors `webcache`'s `DefaultDataDir()`).
- Fail-open: a scan/ingest error never fails the calling session (ADR-0016 floors).
- No TTL on scans (user decision).
- `edges` (code index) is read-only here — no reindex, no new extractor (WS7 reuses `kind IN ('imports','dynamic_import')`).
- BDD always on: every requirement maps to a scenario in `acceptance.feature`.
