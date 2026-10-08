# Mnemonic Scan Findings + Dependency Graph — Implementation Blueprint (steps, files, verification)

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T2 (new tables + new MCP tool surface; no new trust boundary, no new dependencies)

**Build shape:** Tracer thread (one trivy scan ingests end-to-end before the other scanners + dep graph thicken)

**Goal:** Give scanner output (trivy/vuln+sbom, wapiti, nuclei, semgrep) a durable, queryable home in the per-project SQLite store — findings, dependency graph, and the declared-vs-imported runtime import tree become first-class, searchable, diffable memory.

**Architecture:** Additive migration 050 adds `scans`, `findings` (+ FTS5), `dependencies` (PK `purl`), `dep_edges`. A `scan` package shells out to scanner CLIs (trivy may run via the already-configured MCP server), writes raw JSON to `<data_dir>/.skillgrid/cache/scans/<scan_id>.json`, and upserts normalized findings with a stable `dedup_hash`. A `dep` package ingests SBOM/lockfile into the graph and answers `affected` (reverse `depends_on` walk) + `runtime` (declared-vs-imported via the code index `edges` table). New `registerScanTools`/`registerDepTools` in `mcp/server.go`. Fail-open throughout.

**Tech Stack:** Go 1.22+ (locked), stdlib `encoding/json` + existing `modernc.org/sqlite` (no new deps), the already-configured trivy MCP server (or `trivy` CLI) for trivy, `uv tool install semgrep` for the new scanner.

**Spec:** `.skillgrid/specs/2026-10-05-mnemonic-scan-findings/briefing.md`

**Findings:** 2026-10-05 plan-mode research — webcache raw+FTS pattern (`webcache/service.go`), migration list ends at `049_session_checkpoint.sql` (next = `050`), `SecurityTools()` at `install/config.go:117` (semgrep absent), code index `edges` carries `kind IN ('imports','dynamic_import')` (`001_schema.sql:564`, `graph_enrichment_035_test.go`).

## Terms

- **Scan** — one scanner invocation against one target. (New.)
- **Finding** — one normalized scanner result row (CVE / rule / vuln) under a scan. (New.)
- **`dedup_hash`** — `sha256(tool, cve|rule_id, pkg, version, file, line)`, the stable identity used for cross-scan diff. (New.)
- **Dependency graph** — `dependencies` (upserted by `purl`) + `dep_edges` (`depends_on`). (New.)
- **Soft-retire** — a dep absent from a newer ingest gets `retired=1`, never deleted. (New.)
- **Runtime import tree** — declared (manifest) vs imported (code index `edges`) for a changed file. (New concept; reuses existing `edges`.)
- **`purl`** — package URL, the dependency identity (e.g. `pkg:pypi/flask@3.0.2`).
- Existing terms (code index, edges, findings, web cache, session) — see `.skillgrid/artifacts/01-business-terms.md` / `02-technical-terms.md`.

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):
- `scan_start` (trivy) on a fixture with a known CVE produces a `scans` row + `findings` rows with stable `dedup_hash`; raw JSON at `<data_dir>/.skillgrid/cache/scans/<scan_id>.json`. `backstop` — held-out test: Task 2 end-to-end ingest test with a fixture artifact.
- Re-running the same scan on unchanged code → `scan_diff --latest` reports **zero** added (hashes identical). `backstop` — held-out test: Task 4 diff test.
- `dep_ingest` soft-retires a purl absent from a newer SBOM (row survives, `retired=1`). `backstop` — held-out test: Task 5 retire test.
- `dep_runtime <file>` reuses `edges` (kind imports/dynamic_import) — no new extraction pass, no reindex. `backstop` — held-out test: Task 6 runtime test against the code index.
- A scanner failure records `status=error` + `error` text and never fails the caller (fail-open). `backstop` — held-out test: Task 2 error-path test.
- With no scanner binaries present, `go test ./...` is green (tests use fixture JSON, not live scans).

**Artifacts** (files that must exist with real implementation, not stubs):
- `skillgrid-cli/internal/mnemonic/store/migrations/050_scan_findings.sql` — `scans`, `findings` (+ `findings_fts` + triggers), `dependencies`, `dep_edges`.
- `skillgrid-cli/internal/mnemonic/scan/service.go` — `Start`, `StoreFindings`, `Diff`, `Status`, parsers.
- `skillgrid-cli/internal/mnemonic/scan/parse.go` — per-scanner pure parsers (trivy, wapiti, nuclei, semgrep).
- `skillgrid-cli/internal/mnemonic/scan/severity.go` — the severity normalization map.
- `skillgrid-cli/internal/mnemonic/scan/raw.go` — raw-artifact write/read under `<data_dir>/.skillgrid/cache/scans/`.
- `skillgrid-cli/internal/mnemonic/dep/service.go` — `Ingest`, `Affected`, `Runtime`, `Graph`, `List`, `Get`.
- `skillgrid-cli/internal/mnemonic/mcp/tools_scan.go` — `registerScanTools`.
- `skillgrid-cli/internal/mnemonic/mcp/tools_dep.go` — `registerDepTools`.
- `skillgrid-cli/internal/install/config.go` — semgrep added to `SecurityTools()`.
- `~/.agents/skills/verification/semgrep/SKILL.md` — new skill.
- `acceptance-tests/features/scan-findings.feature` (mirror of `acceptance.feature`).

**Key links** (critical connections that must work together):
- `scan.Start` ⇒ shell to scanner CLI / trivy MCP ⇒ raw JSON written to `<data_dir>/.skillgrid/cache/scans/<id>.json` ⇒ `scans` row inserted (`status=ok|error`).
- `scan.StoreFindings` ⇒ read raw file ⇒ `parse.go` (pure, fixture-testable) ⇒ severity map ⇒ `dedup_hash` ⇒ `findings` upsert ⇒ `scans.finding_count` updated.
- `scan.Diff` ⇒ set difference of `dedup_hash` per `scan_id` ⇒ `added`/`removed`/`unchanged`.
- `dep.Ingest` ⇒ SBOM parse ⇒ `dependencies` upsert by `purl` ⇒ `dep_edges` replace ⇒ absent purls soft-retired.
- `dep.Runtime` ⇒ resolve `file_id` for the source file ⇒ `SELECT ... FROM edges WHERE kind IN ('imports','dynamic_import') AND file_id=?` ⇒ diff against manifest declared set.
- `mcp/server.go` ⇒ `registerScanTools(s)` + `registerDepTools(s)` (server.go:42-72 pattern).

**One-way-door decisions** (hard to reverse — explicit user approval required before implementing):
1. ⚠ **Migration 050 adds `scans`, `findings`, `findings_fts`, `dependencies`, `dep_edges`** to every existing store on next open — additive (new tables only), no column changes to existing tables (per ADR-0012).
2. ⚠ **`dedup_hash` formula is the cross-scan contract** — changing it invalidates every prior diff; locked now (sha256 over tool, cve|rule_id, pkg, version, file, line).
3. ⚠ **Raw artifacts live under `<data_dir>/.skillgrid/cache/scans/`** (out of repo) — the path is the durable contract for "where is my scan's raw file"; renaming requires a `raw.go` migration story.

## Global Constraints

- Go 1.22+ minimum to build (locked).
- **No new dependencies without an ADR** — this change adds none (stdlib `encoding/json` + existing `modernc.org/sqlite`; trivy via the already-configured MCP server or CLI).
- Local store unchanged: per-project SQLite, WAL, pure-Go driver, embedded migrations (ADR-0012). Migration 050 is additive tables only.
- `edges` (code index, ADR-0009/0016 per-machine derived state) is **read-only** here — no reindex, no new extractor (the runtime tree reuses `kind IN ('imports','dynamic_import')`).
- Fail-open: a scan/ingest error never blocks or fails the calling session (ADR-0016 floors).
- No scan TTL (user decision) — no expiry column, no time-based prune.
- Raw artifacts out of repo under `<data_dir>/.skillgrid/cache/scans/` (mirrors `webcache.DefaultDataDir()`).
- Tests use fixture artifacts (JSON files), never live scanner binaries, so `go test ./...` is green without trivy/wapiti/nuclei/semgrep installed.
- BDD always on: each task's SATISFIES names its scenario in `acceptance.feature`.

## File Structure

```
skillgrid-cli/
  internal/mnemonic/
    store/migrations/
      050_scan_findings.sql   # scans, findings (+ findfts + triggers), dependencies, dep_edges (additive)
    scan/
      service.go              # Start, StoreFindings, Diff, Status, List, Get — thin SQL + orchestration
      service_test.go
      parse.go                # pure per-scanner parsers: ParseTrivy/ParseWapiti/ParseNuclei/ParseSemgrep ([]byte -> []Finding)
      parse_test.go           # fixture-driven, table-driven
      severity.go             # Normalize(tool, raw) -> normalized severity (the map)
      severity_test.go
      raw.go                  # WriteRaw/ReadRaw under <data_dir>/.skillgrid/cache/scans/<id>.json
      raw_test.go
      fixtures/               # trivy.json, wapiti.json, nuclei.jsonl, semgrep.json (test doubles)
    dep/
      service.go              # Ingest, Affected, Runtime, Graph, List, Get — thin SQL + orchestration
      service_test.go
      sbom.go                 # pure SBOM/lockfile parse -> []Package + []Edge
      sbom_test.go
    mcp/
      server.go               # MODIFY: registerScanTools(s) + registerDepTools(s) (server.go:42-72 pattern)
      tools_scan.go           # scan_start, scan_store_findings, scan_list, scan_get, scan_status, scan_diff
      tools_scan_test.go
      tools_dep.go            # dep_ingest, dep_list, dep_get, dep_affected, dep_graph, dep_runtime
      tools_dep_test.go
  internal/install/
    config.go                 # MODIFY: SecurityTools() += semgrep (uv tool install semgrep)
    install_test.go           # MODIFY: assert semgrep present with manager=uv
acceptance-tests/features/scan-findings.feature
```

## Tasks

### Task 1: Migration 050 — additive scan + dep tables

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/store/migrations/050_scan_findings.sql`
- Create: `skillgrid-cli/internal/mnemonic/store/scan_tables_test.go`

**Interfaces:**
- Consumes: `*store.Store` (existing; `store.Open` at store/store.go:104), the embedded-migration runner (migrations are applied by file name in numeric order — 050 runs after 049).
- Produces: the four tables + FTS; no Go symbols.
- SATISFIES: `happy path scan migration applies idempotently`

- [ ] **Step 1: Write the failing test** — `store/scan_tables_test.go`:

```go
func TestScanTablesMigrateIdempotent(t *testing.T) {
	s := testStore(t) // existing helper: opens temp store + runs all migrations
	for _, tbl := range []string{"scans", "findings", "findings_fts", "dependencies", "dep_edges"} {
		var n int
		if err := s.DB.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type IN ('table','view') AND name = ?`, tbl).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			t.Fatalf("table/view %s missing after migration", tbl)
		}
	}
	// idempotency: re-opening the same file re-runs the runner without error (no-op on existing tables)
	s2 := reopenStore(t, s)
	_ = s2
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/store/ -run TestScanTablesMigrateIdempotent -v`
Expected: FAIL — `no such table: scans`.

- [ ] **Step 3: Write the migration**

`store/migrations/050_scan_findings.sql` (additive; FTS + triggers mirror the 001 `symbol_fts` pattern):

```sql
-- 050_scan_findings: structured scanner results + dependency graph (additive; T2, no new deps)
CREATE TABLE IF NOT EXISTS scans (
	id TEXT PRIMARY KEY,                         -- UUIDv7
	tool TEXT NOT NULL,                          -- trivy | wapiti | nuclei | semgrep
	target TEXT NOT NULL,
	scanners TEXT NOT NULL DEFAULT '',           -- e.g. vuln,sbom
	status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','ok','error','partial')),
	started_at TEXT NOT NULL,
	finished_at TEXT,
	finding_count INTEGER NOT NULL DEFAULT 0,
	raw_path TEXT,
	error TEXT
);
CREATE INDEX IF NOT EXISTS idx_scans_started ON scans (started_at);
CREATE INDEX IF NOT EXISTS idx_scans_tool ON scans (tool, started_at);

CREATE TABLE IF NOT EXISTS findings (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	scan_id TEXT NOT NULL REFERENCES scans(id) ON DELETE CASCADE,
	tool TEXT NOT NULL,
	dedup_hash TEXT NOT NULL,                    -- sha256(tool, cve|rule_id, pkg, version, file, line)
	severity TEXT NOT NULL,                      -- normalized: CRITICAL|HIGH|MEDIUM|LOW|INFO
	title TEXT NOT NULL,
	cve_id TEXT,
	rule_id TEXT,
	package TEXT,
	version TEXT,
	fixed_version TEXT,
	file TEXT,
	line INTEGER,
	message TEXT,
	links TEXT NOT NULL DEFAULT '[]',            -- JSON array of URLs
	UNIQUE (scan_id, dedup_hash)
);
CREATE INDEX IF NOT EXISTS idx_findings_scan ON findings (scan_id);
CREATE INDEX IF NOT EXISTS idx_findings_hash ON findings (dedup_hash);
CREATE INDEX IF NOT EXISTS idx_findings_sev ON findings (severity);

CREATE VIRTUAL TABLE IF NOT EXISTS findings_fts USING fts5(
	title, cve_id, rule_id, package, message, file,
	content='findings', content_rowid='id'
);
CREATE TRIGGER IF NOT EXISTS findings_ai AFTER INSERT ON findings BEGIN
	INSERT INTO findings_fts(rowid, title, cve_id, rule_id, package, message, file)
	VALUES (new.id, new.title, new.cve_id, new.rule_id, new.package, new.message, new.file);
END;
CREATE TRIGGER IF NOT EXISTS findings_ad AFTER DELETE ON findings BEGIN
	INSERT INTO findings_fts(findings_fts, rowid, title, cve_id, rule_id, package, message, file)
	VALUES ('delete', old.id, old.title, old.cve_id, old.rule_id, old.package, old.message, old.file);
END;
CREATE TRIGGER IF NOT EXISTS findings_au AFTER UPDATE ON findings BEGIN
	INSERT INTO findings_fts(findings_fts, rowid, title, cve_id, rule_id, package, message, file)
	VALUES ('delete', old.id, old.title, old.cve_id, old.rule_id, old.package, old.message, old.file);
	INSERT INTO findings_fts(rowid, title, cve_id, rule_id, package, message, file)
	VALUES (new.id, new.title, new.cve_id, new.rule_id, new.package, new.message, new.file);
END;

CREATE TABLE IF NOT EXISTS dependencies (
	purl TEXT PRIMARY KEY,                       -- pkg:pypi/flask@3.0.2
	name TEXT NOT NULL,
	version TEXT NOT NULL,
	ecosystem TEXT,
	manifest TEXT,                                -- provenance: which manifest produced it
	retired INTEGER NOT NULL DEFAULT 0,           -- soft-retire (never deleted)
	last_seen TEXT                                 -- last scan that saw this purl
);
CREATE INDEX IF NOT EXISTS idx_deps_retired ON dependencies (retired);

CREATE TABLE IF NOT EXISTS dep_edges (
	from_purl TEXT NOT NULL,
	to_purl TEXT NOT NULL,
	version_range TEXT,
	UNIQUE (from_purl, to_purl)
);
CREATE INDEX IF NOT EXISTS idx_dep_edges_from ON dep_edges (from_purl);
CREATE INDEX IF NOT EXISTS idx_dep_edges_to ON dep_edges (to_purl);
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/store/ -run TestScanTablesMigrateIdempotent -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/store/migrations/050_scan_findings.sql \
  skillgrid-cli/internal/mnemonic/store/scan_tables_test.go
git commit -m "feat(mnemonic): additive scan + dependency graph tables (050)"
```

---

### Task 2: Scan package — Start + StoreFindings (trivy tracer, fail-open)

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/scan/service.go`
- Create: `skillgrid-cli/internal/mnemonic/scan/raw.go`
- Create: `skillgrid-cli/internal/mnemonic/scan/parse.go` (trivy arm only; other scanners in Task 3)
- Create: `skillgrid-cli/internal/mnemonic/scan/severity.go`
- Create: `skillgrid-cli/internal/mnemonic/scan/service_test.go`, `raw_test.go`, `severity_test.go`
- Create: `skillgrid-cli/internal/mnemonic/scan/fixtures/trivy.json` (a trivy vuln artifact with one known CVE)

**Interfaces:**
- Consumes: `*store.Store`, `context.Context`, the trivy MCP server (or `trivy` CLI) for running the scan.
- Produces:
  - `func New(s *store.Store, dataDir string) *Service`
  - `(*Service).Start(ctx, tool, target, scanners string) (scanID, rawPath string, err error)` — shells out (or calls trivy MCP), writes raw JSON to `<dataDir>/.skillgrid/cache/scans/<scanID>.json`, inserts a `scans` row. **Fail-open:** on scanner error, `status=error` + `error` text, no returned error to the caller's session.
  - `(*Service).StoreFindings(ctx, scanID, tool string) (int, error)` — read raw file, `parse.Parse(tool, bytes)`, severity map, `dedup_hash`, upsert `findings`, update `scans.finding_count`/`status`.
  - `func DedupHash(tool, cveID, ruleID, pkg, version, file string, line int) string` — `hex(sha256(...))`.
  - `func NormalizeSeverity(tool, raw string) string` — the map (trivy identity `UNKNOWN`→`INFO`; wapiti/nuclei capitalize; semgrep `ERROR`→`HIGH`, `WARNING`→`MEDIUM`, `NOTE`→`INFO`).
  - `type Finding struct { Severity, Title, CVEID, RuleID, Package, Version, FixedVersion, File string; Line int; Message string; Links []string }`
  - `raw.go`: `func WriteRaw(dataDir, scanID, data []byte) (string, error)` / `func ReadRaw(path string) ([]byte, error)` — path = `<dataDir>/.skillgrid/cache/scans/<scanID>.json` (0644).
- SATISFIES: `happy path trivy scan ingests findings with stable hash`, `error path scanner failure records status=error and is fail-open`

- [ ] **Step 1: Write the failing tests** — `severity_test.go`, `raw_test.go`, and a fixture-driven ingest test in `service_test.go`:

```go
func TestNormalizeSeverityMap(t *testing.T) {
	cases := []struct{ tool, in, want string }{
		{"trivy", "UNKNOWN", "INFO"}, {"trivy", "CRITICAL", "CRITICAL"}, {"trivy", "low", "LOW"},
		{"wapiti", "high", "HIGH"}, {"nuclei", "critical", "CRITICAL"},
		{"semgrep", "ERROR", "HIGH"}, {"semgrep", "WARNING", "MEDIUM"}, {"semgrep", "NOTE", "INFO"},
	}
	for _, c := range cases {
		if got := NormalizeSeverity(c.tool, c.in); got != c.want {
			t.Fatalf("NormalizeSeverity(%s,%s)=%s want %s", c.tool, c.in, got, c.want)
		}
	}
}

func TestWriteRawAndReadBack(t *testing.T) {
	dir := t.TempDir()
	p, err := WriteRaw(dir, "scan-1", []byte(`{"Vulnerabilities":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(p, filepath.Join(".skillgrid", "cache", "scans", "scan-1.json")) {
		t.Fatalf("path = %s", p)
	}
	b, err := ReadRaw(p)
	if err != nil || string(b) != `{"Vulnerabilities":[]}` {
		t.Fatalf("roundtrip: %q err=%v", b, err)
	}
}

func TestStartTrivyIngestsStableHash(t *testing.T) {
	svc := newTestService(t) // store + dataDir; Start uses a stub scanner returning fixtures/trivy.json
	scanID, _, _ := svc.Start(ctx, "trivy", "/repo", "vuln")
	n, err := svc.StoreFindings(ctx, scanID, "trivy")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("finding_count = %d, want 1 (fixture has one CVE)", n)
	}
	var h1 string
	if err := s.DB.QueryRow(`SELECT dedup_hash FROM findings WHERE scan_id=?`, scanID).Scan(&h1); err != nil {
		t.Fatal(err)
	}
	// deterministic: recompute
	want := DedupHash("trivy", "CVE-2024-1234", "", "flask", "2.0.1", "requirements.txt", 0)
	if h1 != want {
		t.Fatalf("hash = %s, want %s", h1, want)
	}
}

func TestStartScannerErrorIsFailOpen(t *testing.T) {
	svc := newTestServiceFailing(t) // stub scanner returns an error
	_, _, err := svc.Start(ctx, "trivy", "/repo", "vuln")
	if err != nil {
		t.Fatalf("Start must be fail-open, got %v", err)
	}
	// a scans row with status=error exists
	var status, errMsg string
	if err := s.DB.QueryRow(`SELECT status, error FROM scans WHERE tool='trivy'`).Scan(&status, &errMsg); err != nil {
		t.Fatal(err)
	}
	if status != "error" || errMsg == "" {
		t.Fatalf("want status=error with message, got status=%q msg=%q", status, errMsg)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/scan/ -run 'TestNormalize|TestWriteRaw|TestStartTrivy|TestStartScannerError' -v`
Expected: FAIL — package `scan` symbols undefined.

- [ ] **Step 3: Implement**

`severity.go` — the map (table-driven, switch on tool). `raw.go` — `os.MkdirAll` the scans dir, `os.WriteFile` 0644, `os.ReadFile`. `parse.go` (trivy arm) — `json.Unmarshal` the trivy `Vulnerabilities` array → `[]Finding` (map `VulnerabilityID`→CVEID, `PkgName`→Package, `InstalledVersion`→Version, `FixedVersion`, `Severity`, `Title`, `PrimaryURL`→Links). `service.go` — `Start` mints a UUIDv7, runs the scanner (stub interface in tests, real exec/MCP in prod), writes raw, inserts `scans` row; `StoreFindings` reads raw → parse → `NormalizeSeverity` → `DedupHash` → upsert `findings` → update `scans`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/scan/ -v && go build ./...`
Expected: PASS + build clean.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/scan/
git commit -m "feat(mnemonic): scan package — start, store findings, severity map, raw store"
```

---

### Task 3: Remaining scanner parsers (wapiti, nuclei, semgrep) + SBOM ingest path

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/scan/parse.go` — add `ParseWapiti`, `ParseNuclei`, `ParseSemgrep`.
- Create: `skillgrid-cli/internal/mnemonic/scan/fixtures/{wapiti.json,nuclei.jsonl,semgrep.json}`.
- Modify: `skillgrid-cli/internal/mnemonic/scan/parse_test.go` — table-driven cases for all four parsers.
- Modify: `skillgrid-cli/internal/mnemonic/scan/service.go` — `Start` routes to the right scanner command/args per tool.

**Interfaces:**
- Consumes: Task 2's `Finding` type + `Parse` dispatch.
- Produces: `func Parse(tool string, data []byte) ([]Finding, error)` — dispatches to the four parsers. Each is pure (bytes → findings), fixture-driven.
- SATISFIES: `happy path trivy scan ingests findings with stable hash` (extended to all four tools)

- [ ] **Step 1: Write the failing test** — extend `parse_test.go`:

```go
func TestParseDispatchAllTools(t *testing.T) {
	cases := []struct{ tool, fixture string; wantN int }{
		{"trivy", "trivy.json", 1},
		{"wapiti", "wapiti.json", 2},
		{"nuclei", "nuclei.jsonl", 3},
		{"semgrep", "semgrep.json", 4},
	}
	for _, c := range cases {
		b, _ := os.ReadFile("fixtures/" + c.fixture)
		got, err := Parse(c.tool, b)
		if err != nil {
			t.Fatalf("Parse(%s): %v", c.tool, err)
		}
		if len(got) != c.wantN {
			t.Fatalf("Parse(%s) = %d findings, want %d", c.tool, len(got), c.wantN)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/scan/ -run TestParseDispatchAllTools -v`
Expected: FAIL — `Parse(wapiti)` etc. not implemented.

- [ ] **Step 3: Implement the three parsers**

`ParseWapiti` (wapiti report JSON: `report[]` with `type`, `info`, `url`), `ParseNuclei` (JSONL: one JSON object per line with `template-id`, `info.severity`, `matcher-name`, `host`), `ParseSemgrep` (JSON `results[]` with `check_id`→RuleID, `extra.severity`, `path`→File, `start.line`, `extra.message`). Each maps into `Finding` with `Links` from `info.reference` / `extra.metadata.references`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/scan/ -v && go build ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/scan/
git commit -m "feat(mnemonic): wapiti/nuclei/semgrep scan parsers (fixture-driven)"
```

---

### Task 4: Diff + Status + List + Get

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/scan/service.go` — `Diff`, `Status`, `List`, `Get`.
- Modify: `skillgrid-cli/internal/mnemonic/scan/service_test.go` — diff, status, list, get tests.

**Interfaces:**
- Consumes: Tasks 1-2 (tables + `findings` rows).
- Produces:
  - `(*Service).Diff(ctx, oldID, newID string) (DiffResult, error)` — `DiffResult{ Added, Removed, Unchanged []string }` (sets of `dedup_hash`). Removed findings are **not deleted** (audit).
  - `(*Service).LatestDiff(ctx) (DiffResult, error)` — resolves the two most recent `scans` by `started_at`.
  - `(*Service).Status(ctx) (StatusResult, error)` — counts by tool + severity + freshness (most-recent scan time per tool).
  - `(*Service).List(ctx, tool, status string, limit int) ([]ScanRow, error)`.
  - `(*Service).Get(ctx, scanID string) (ScanRow, error)`.
- SATISFIES: `happy path unchanged re-scan diffs to zero added`, `happy path fixed cve appears as removed in diff`

- [ ] **Step 1: Write the failing tests** — `service_test.go`:

```go
func TestUnchangedRescanDiffsToZeroAdded(t *testing.T) {
	svc := newTestService(t)
	id1, _ := ingestFixture(t, svc, "trivy")      // first scan
	id2, _ := ingestFixture(t, svc, "trivy")      // same fixture, new scan_id
	res, err := svc.Diff(ctx, id1, id2)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Added) != 0 || len(res.Removed) != 0 {
		t.Fatalf("unchanged rescan: added=%d removed=%d, want 0/0", len(res.Added), len(res.Removed))
	}
}

func TestFixedCveAppearsAsRemoved(t *testing.T) {
	svc := newTestService(t)
	id1, _ := ingestFixture(t, svc, "trivy")      // has CVE-2024-1234
	id2, _ := ingestFixtureWithout(t, svc, "trivy", "CVE-2024-1234") // fixed
	res, _ := svc.Diff(ctx, id1, id2)
	if !contains(res.Removed, DedupHash("trivy", "CVE-2024-1234", "", "flask", "2.0.1", "requirements.txt", 0)) {
		t.Fatalf("fixed CVE not in removed: %v", res.Removed)
	}
	// audit: the original finding row is NOT deleted
	var n int
	if err := s.DB.QueryRow(`SELECT count(*) FROM findings WHERE scan_id=?`, id1).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("audit: original findings should survive, got %d", n)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/scan/ -run 'TestUnchanged|TestFixedCve' -v`
Expected: FAIL — `Diff` undefined.

- [ ] **Step 3: Implement**

`Diff` — `SELECT dedup_hash FROM findings WHERE scan_id=?` for both, set-difference. `LatestDiff` — `SELECT id FROM scans ORDER BY started_at DESC LIMIT 2`. `Status`/`List`/`Get` — thin SQL.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/scan/ -v && go build ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/scan/
git commit -m "feat(mnemonic): scan diff (hash-set), status, list, get"
```

---

### Task 5: Dep package — Ingest + Affected + Graph + List + Get (soft-retire)

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/dep/service.go`
- Create: `skillgrid-cli/internal/mnemonic/dep/sbom.go`
- Create: `skillgrid-cli/internal/mnemonic/dep/service_test.go`, `sbom_test.go`
- Create: `skillgrid-cli/internal/mnemonic/dep/fixtures/sbom.cyclonedx.json`

**Interfaces:**
- Consumes: `*store.Store`, `context.Context`.
- Produces:
  - `func New(s *store.Store) *Service`
  - `(*Service).Ingest(ctx, scanID, sbomPath string) (added, retired int, err error)` — parse SBOM → upsert `dependencies` by `purl` (`last_seen` = now, `retired=0` if present); replace `dep_edges` for the ingested set; **soft-retire** purls absent from the ingest (`retired=1`, row survives).
  - `(*Service).Affected(ctx, purl string) ([]string, error)` — reverse `depends_on` BFS (depth-capped at 10): the purls that transitively depend on `purl`.
  - `(*Service).Graph(ctx) ([]Edge, error)` — full `from`→`to` edge set.
  - `(*Service).List(ctx, retired bool) ([]DepRow, error)`.
  - `(*Service).Get(ctx, purl string) (DepRow, error)`.
  - `sbom.go`: `func ParseSBOM(data []byte) ([]Pkg, []Edge, error)` — CycloneDX + SPDX → `Pkg{PURL, Name, Version, Ecosystem}` + `Edge{From, To, VersionRange}`.
- SATISFIES: `happy path dep ingest upserts by purl and soft-retires absent`, `happy path dep_affected returns reverse dependency set`

- [ ] **Step 1: Write the failing tests** — `service_test.go`:

```go
func TestIngestUpsertsAndSoftRetires(t *testing.T) {
	svc := depNew(t)
	_, _, err := svc.Ingest(ctx, "scan-1", "fixtures/sbom.cyclonedx.json") // flask + its deps
	if err != nil {
		t.Fatal(err)
	}
	var flask int
	s.DB.QueryRow(`SELECT count(*) FROM dependencies WHERE name='flask' AND retired=0`).Scan(&flask)
	// second ingest WITHOUT flask
	_, _, err = svc.Ingest(ctx, "scan-2", "fixtures/sbom-noflask.json")
	if err != nil {
		t.Fatal(err)
	}
	var retired int
	if err := s.DB.QueryRow(`SELECT retired FROM dependencies WHERE name='flask'`).Scan(&retired); err != nil {
		t.Fatal(err) // row must still exist
	}
	if retired != 1 {
		t.Fatalf("flask should be soft-retired, retired=%d", retired)
	}
}

func TestAffectedReturnsReverseSet(t *testing.T) {
	svc := depNew(t)
	// graph: app -> flask -> werkzeug
	_, _, _ = svc.Ingest(ctx, "scan-1", "fixtures/sbom.graph.json")
	got, err := svc.Affected(ctx, purl("werkzeug"))
	if err != nil {
		t.Fatal(err)
	}
	// flask and app depend on werkzeug (transitively)
	if !contains(got, purl("flask")) || !contains(got, purl("app")) {
		t.Fatalf("Affected(werkzeug) = %v, want flask+app", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/dep/ -run 'TestIngest|TestAffected' -v`
Expected: FAIL — package `dep` symbols undefined.

- [ ] **Step 3: Implement**

`sbom.go` — CycloneDX `components[]` (with `purl`) + `dependency` edges (`ref`→`dependsOn`); SPDX as a fallback. `service.go` — `Ingest` in one transaction: upsert `dependencies` (clear `retired` if present, set `last_seen`), delete + reinsert `dep_edges` for the ingested set, then `UPDATE dependencies SET retired=1 WHERE purl NOT IN (<ingested set>)`. `Affected` — BFS over `dep_edges` reverse (`WHERE to_purl = ?`), depth-capped. `Graph`/`List`/`Get` — thin SQL.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/dep/ -v && go build ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/dep/
git commit -m "feat(mnemonic): dep package — SBOM ingest, soft-retire, affected, graph"
```

---

### Task 6: Dep.Runtime — declared-vs-imported via the code index `edges` (no new extractor)

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/dep/service.go` — `Runtime`.
- Modify: `skillgrid-cli/internal/mnemonic/dep/service_test.go` — runtime test against the code index.

**Interfaces:**
- Consumes: `*store.Store` (the code index `edges` table, `001_schema.sql:564` — `kind IN ('imports','dynamic_import')`, `file_id`, `to_name`, `target_path`, `confidence`, `line`), `dependencies.manifest` provenance.
- Produces:
  - `(*Service).Runtime(ctx, sourceFile string) (RuntimeResult, error)` — `RuntimeResult{ DeclaredOnly, ImportOnly, Shared []string }`.
  - **Resolved scope (WS7):** `edges.file_id` points at **source files** (not manifests). The imported set = `SELECT to_name, target_path FROM edges WHERE kind IN ('imports','dynamic_import') AND file_id = (SELECT id FROM files WHERE path = ?)`. The declared set = purls in `dependencies` whose `manifest` maps to this source file's project (carried on `dependencies.manifest` at ingest). The declared-vs-imported diff is computed in the tool layer — **no schema change, no reindex, no new extractor.**
- SATISFIES: `happy path dep_runtime flags declared-only and import-only`

- [ ] **Step 1: Write the failing test** — `service_test.go` (seed a code-index file + import edge, seed a manifest dep, assert the three buckets):

```go
func TestRuntimeFlagsDeclaredAndImportOnly(t *testing.T) {
	svc := depNew(t)
	seedCodeIndexFile(t, "app/main.py")                 // files row
	seedImportEdge(t, "app/main.py", "flask")            // edges row kind='imports'
	seedImportEdge(t, "app/main.py", "os")               // stdlib import, not a manifest dep
	svc.Ingest(ctx, "scan-1", "fixtures/sbom.flask.json") // manifest declares flask + werkzeug
	res, err := svc.Runtime(ctx, "app/main.py")
	if err != nil {
		t.Fatal(err)
	}
	// flask: declared AND imported
	if !contains(res.Shared, "flask") {
		t.Fatalf("flask should be shared, got %v", res.Shared)
	}
	// werkzeug: declared (in manifest) but never imported
	if !contains(res.DeclaredOnly, "werkzeug") {
		t.Fatalf("werkzeug should be declared-only, got %v", res.DeclaredOnly)
	}
	// os: imported but not declared in any manifest
	if !contains(res.ImportOnly, "os") {
		t.Fatalf("os should be import-only, got %v", res.ImportOnly)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/dep/ -run TestRuntimeFlagsDeclaredAndImportOnly -v`
Expected: FAIL — `Runtime` undefined.

- [ ] **Step 3: Implement**

`Runtime` — resolve `file_id` for `sourceFile` (query `files`), `SELECT to_name, target_path FROM edges WHERE kind IN ('imports','dynamic_import') AND file_id=?` for the imported set; declared set = `SELECT name FROM dependencies WHERE retired=0 AND manifest=<project manifest>`; three-way set diff into `DeclaredOnly`/`ImportOnly`/`Shared`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/dep/ -v && go build ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/dep/
git commit -m "feat(mnemonic): dep runtime — declared-vs-imported via code index edges"
```

---

### Task 7: MCP registration — registerScanTools + registerDepTools

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_scan.go`
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_dep.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/server.go` — add the two `register*` calls (server.go:42-72 pattern).
- Create: `skillgrid-cli/internal/mnemonic/mcp/tools_scan_test.go`, `tools_dep_test.go`

**Interfaces:**
- Consumes: `scan.Service`, `dep.Service` (Tasks 2-6), the existing MCP tool-registration pattern.
- Produces:
  - `registerScanTools(s *Server)` — `scan_start`, `scan_store_findings`, `scan_list`, `scan_get`, `scan_status`, `scan_diff` (params: `tool`, `target`, `scanners`, `scan_id`, `old`, `new`, `latest`).
  - `registerDepTools(s *Server)` — `dep_ingest`, `dep_list`, `dep_get`, `dep_affected`, `dep_graph`, `dep_runtime` (params: `purl`, `scan_id`, `sbom`, `source_file`).
- SATISFIES: `happy path scan and dep tools are registered`

- [ ] **Step 1: Write the failing test** — `tools_scan_test.go` / `tools_dep_test.go`:

```go
func TestScanToolsRegistered(t *testing.T) {
	s := newTestServer(t)
	for _, name := range []string{"scan_start", "scan_store_findings", "scan_list", "scan_get", "scan_status", "scan_diff"} {
		if !s.hasTool(name) {
			t.Fatalf("tool %s not registered", name)
		}
	}
}

func TestDepToolsRegistered(t *testing.T) {
	s := newTestServer(t)
	for _, name := range []string{"dep_ingest", "dep_list", "dep_get", "dep_affected", "dep_graph", "dep_runtime"} {
		if !s.hasTool(name) {
			t.Fatalf("tool %s not registered", name)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/mcp/ -run 'TestScanToolsRegistered|TestDepToolsRegistered' -v`
Expected: FAIL — tools not registered.

- [ ] **Step 3: Implement**

`tools_scan.go` + `tools_dep.go` — each tool is a thin adapter: decode params → call the `scan`/`dep` service → encode the result. Wire `registerScanTools(s)` + `registerDepTools(s)` in `server.go` alongside the existing `registerXxxTools` calls. Construct the `scan.Service` (needs `dataDir` from `webcache.DefaultDataDir()`) and `dep.Service` at server boot.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/mcp/ -v && go build ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/mcp/
git commit -m "feat(mnemonic): register scan + dep MCP tools"
```

---

### Task 8: Install (semgrep) + skills

**Files:**
- Modify: `skillgrid-cli/internal/install/config.go` — `SecurityTools()` += semgrep.
- Modify: `skillgrid-cli/internal/install/install_test.go` — assert semgrep present with manager `uv`.
- Create: `~/.agents/skills/verification/semgrep/SKILL.md` (mirror of `wapiti`/`nuclei`).
- Modify: `~/.agents/skills/verification/{trivy,wapiti,nuclei}/SKILL.md` — add a "Store results in mnemonic" section.
- Create: `acceptance-tests/features/scan-findings.feature` (mirror of `acceptance.feature`).

**Interfaces:**
- Consumes: the existing `SecurityTools()` (`install/config.go:117`) + `installSecurityTools` (`install/install.go:440`).
- Produces:
  - semgrep entry: `{Name: "semgrep", Manager: "uv", InstallArgs: []string{"tool", "install", "semgrep"}, Bin: "semgrep", Hint: "curl -LsSf https://astral.sh/uv/install.sh | sh"}`.
  - The semgrep skill with a "Store results in mnemonic" step (call `dep_ingest` for SBOM / `scan_store_findings` for findings).
- SATISFIES: `happy path semgrep installs via uv`, `happy path semgrep skill exists and documents mnemonic store step`

- [ ] **Step 1: Write the failing test** — `install_test.go`:

```go
func TestSecurityToolsIncludeSemgrep(t *testing.T) {
	tools := SecurityTools()
	var found bool
	for _, t := range tools {
		if t.Name == "semgrep" {
			found = true
			if t.Manager != "uv" || t.Bin != "semgrep" {
				t.Fatalf("semgrep config wrong: %+v", t)
			}
		}
	}
	if !found {
		t.Fatal("semgrep missing from SecurityTools()")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd skillgrid-cli && go test ./internal/install/ -run TestSecurityToolsIncludeSemgrep -v`
Expected: FAIL — semgrep absent.

- [ ] **Step 3: Implement**

Add the semgrep entry to `SecurityTools()`. No new install manager — `uv` is already used by `wapiti3`. Create the semgrep skill (copy `wapiti/SKILL.md` structure, replace with semgrep specifics + the mnemonic-store step). Add the "Store results in mnemonic" section to trivy/wapiti/nuclei skills.

- [ ] **Step 4: Write the BDD acceptance feature**

`acceptance-tests/features/scan-findings.feature` — mirror `acceptance.feature`; scenarios map to the SATISFIES lines in this plan (migration idempotency, trivy ingest, fail-open, unchanged-rescan diff, fixed-CVE diff, dep soft-retire, dep_affected, dep_runtime, semgrep install, semgrep skill, tool registration, existing-surface-unchanged).

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd skillgrid-cli && go test ./internal/install/ -run TestSecurityToolsIncludeSemgrep -v && go test ./internal/mnemonic/... -count=1`
Expected: PASS, no regressions.

- [ ] **Step 6: Commit**

```bash
git add skillgrid-cli/internal/install/ skillgrid-cli/acceptance-tests/features/scan-findings.feature
git commit -m "feat(mnemonic): install semgrep, add scan skills + BDD feature"
```

---

### Task 9: Full-suite green + build + the held-out backstop

**Files:** (no new files — verification only)

**Interfaces:**
- Consumes: Tasks 1-8.
- Produces: green `go test ./...`, clean build, the held-out backstop scenarios from "Must-Haves" exercised.
- SATISFIES: `happy path existing tool surface is unchanged`

- [ ] **Step 1: Run the full suite**

Run: `cd skillgrid-cli && go test ./... -count=1 && go build ./...`
Expected: PASS + build clean, with **no scanner binaries present** (tests are fixture-driven).

- [ ] **Step 2: Run the BDD acceptance suite**

Run: `cd skillgrid-cli && cucumber acceptance-tests/features/scan-findings.feature` (per the repo's acceptance runner)
Expected: PASS.

- [ ] **Step 3: Verify the existing surface is unchanged**

Run: `cd skillgrid-cli && go test ./internal/mnemonic/mcp/ -run 'Test.*Registered' -v`
Expected: all pre-existing `register*` tests still pass (the new registrations are additive, not replacing).

- [ ] **Step 4: Commit (if anything was touched)**

```bash
git add -A
git commit -m "test(mnemonic): full-suite green for scan findings + dep graph"
```

**This completes the tracer thread:** one trivy scan ingests end-to-end (start → raw file → findings with stable hash → diff), and the dependency graph + runtime import tree answer the "what breaks / what's declared-but-unused" questions — all fail-open, all fixture-tested, no new dependencies.

## Execution order

1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 (strictly sequential; each task's tests gate the next). Task 1 (schema) and Task 5's SBOM parser are the two that must exist before any MCP tool can be wired (Task 7).
