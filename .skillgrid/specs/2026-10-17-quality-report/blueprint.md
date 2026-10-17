# Quality Report Implementation Blueprint (steps, files, verification)

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T2

**Build shape:** Tracer thread — Task 1 wires one real source (trivy, already parsed) through collect → emit (JUnit + JSON) → HTTP route → a minimal dashboard panel, so the end-to-end path is real and connected before the other six sources thicken it. Integration risk (the dispatch + emitter + dashboard wiring) is the unknown; get it connected early.

**Goal:** One command produces a unified quality report (CVE, SAST, DAST, duplication, code debt, test coverage) for any project skillgrid is installed in, normalized into the existing `findings` + a new `coverage` table, emitted as JUnit XML + JSON, and visualized on a `/reports` dashboard page.

**Architecture:** A new `mnemonic/internal/report` package orchestrates the existing scan pipeline (reuses `scan.Service.Start`/`Parse`/`StoreFindings`/`findings` for the sources that already have parsers), adds two parsers (`ParseJscpd`, `ParseAkca`) **and their `scannerCommands`/`ScannerBin` arms** (so `Start` can run them), a grep debt source, and a detected-dispatch coverage source (line %, five languages). Emitters (`emit.JUnit`, `emit.JSON`) read the aggregated `Report`. New HTTP routes (`GET /report/junit.xml`, `GET /report/json`, write-auth wrapped) and a `scan_report` MCP tool serve it; a new `/reports` dashboard page renders the JSON in six panels. `dast.url` is read from the downstream project's `.skillgrid/config.yaml` via a `reportConfigFrom` loader.

> **Note on jscpd install:** jscpd is **already** installed by `GlobalTools()` (`skillgrid-cli/internal/install/config.go:91`). No installer change is needed — it is *not* added to `SecurityTools()` (that set is for `uv`/`go`-managed scanners; jscpd is `npm`-managed and already present). This removes a one-way-door the first draft carried.

**Tech Stack:** Go 1.22+ (no new Go deps — `encoding/xml` for JUnit, `os/exec` for the scanner CLIs); SQLite (modernc.org/sqlite, migration 060); React + d3 (already a UI dep) for the `/reports` page. jscpd (already in `GlobalTools()`) + akca (in `SecurityTools()`) + the five coverage tools are external CLIs invoked via `exec`, installed on the downstream machine.

**Spec:** `.skillgrid/specs/2026-10-17-quality-report/briefing.md`

**Findings:** None (this change ran no research/prototype/sketch; the design rests on the in-force ADR-0030 scan-findings store and ADR-0016 fail-open, both read directly from the code).

## Terms

- `.skillgrid/artifacts/02-technical-terms.md`: **Scan Finding** (already defined) is the unit the report aggregates — no new term. This plan introduces two Go types, `report.Report` and `report.SourceResult`, which are *aliases* over existing `scan.Finding`; if the terms file gains a type-level index, add one line each. No term is redefined here.

## Hypothesis

```
**Claim:** A report service that orchestrates the existing scan pipeline
  (reusing findings) + two new parsers + a grep debt source + a
  detected-dispatch coverage source produces a correct unified JUnit + JSON
  report for a multi-language project, and the dashboard renders it.
**Right condition:** On a Go fixture project, `GET /report/junit.xml` parses
  as JUnit with a `trivy` + `debt` + `coverage` testsuite; `GET /report/json`
  renders all static panels; an unreachable `dast.url` yields DAST
  "unavailable" with HTTP 200.
**Wrong condition:** The report requires a parallel store, a new Go
  dependency, or fails the whole report when one source errors (violating
  fail-open) — i.e., the "reuse the pipeline" premise does not hold.
**Thinnest MVP:** Task 1 — trivy (already parsed) → collect → JUnit+JSON →
  one HTTP route → one dashboard panel. If this end-to-end path does not work,
  the orchestration-over-findings approach is wrong.
**Door check:** Task 1, Step 4 — `GET /report/json` on the Go fixture returns
  a `trivy` section with the fixture CVE and HTTP 200. If it fails, stop and
  revisit the collect-over-findings design.
```

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):
- `GET /report/junit.xml` returns well-formed JUnit XML with one `<testsuite name="<tool>">` per source that ran; each finding is a `<testcase>` whose `<failure message="<severity>: <title>">` carries the normalized severity. (R2, `backstop` — JUnit validity is only observable by parsing the emitted XML.)
- `GET /report/json` returns a JSON object with a section per source that ran (tool, status, findings, coverage rows), shaped for the dashboard with no client transform. (R3)
- On a Go project with no `dast.url`, the report contains static sources only (trivy, semgrep, jscpd, debt, coverage) and no DAST section. (R1, R6)
- With `dast.url` set and reachable, the report includes wapiti + nuclei + akca DAST findings; with the URL unreachable, the DAST sources are present with `status="unavailable"` and the report still returns HTTP 200. (R6, `backstop` — fail-open DAST is observable only at runtime against a live URL.)
- For each of Go, TS/JS, Python, Rust, Java, a fixture project yields per-file line % in the `coverage` table and the coverage panel; a project kind with no coverage tool on PATH is marked `unavailable`, not an error. (R9)
- A source that errors (timeout or exec failure) is present in the report with `status="error"`, and the report still emits the other sources. (R1, ADR-0016)
- `ParseJscpd` and `ParseAkca` map their fixture outputs into `scan.Finding` with the correct field mapping (rule_id/file/line/severity). (R6, R7)

**Artifacts** (files that must exist with real implementation, not stubs):
- `mnemonic/internal/report/service.go` — `report.Service` + `Collect` + `Report`/`SourceResult` types.
- `mnemonic/internal/report/collect.go` — source orchestration + detected-dispatch coverage.
- `mnemonic/internal/report/emit/junit.go` — `emit.JUnit(*Report) ([]byte, error)`.
- `mnemonic/internal/report/emit/json.go` — `emit.JSON(*Report) ([]byte, error)`.
- `mnemonic/internal/scan/parse.go` — `+ ParseJscpd`, `+ ParseAkca` (+ `Parse` dispatch arms).
- `mnemonic/internal/scan/severity.go` — `+ "jscpd"`, `+ "akca"` entries in `toolSeverityMaps`.
- `mnemonic/internal/store/migrations/060_coverage.sql` — `coverage` table + index.
- `skillgrid-cli/internal/install/config.go` — `SecurityTools() + jscpd`.
- `mnemonic/internal/mcp/tools_report.go` — `scan_report` MCP tool + `registerReportTools`.
- `mnemonic/internal/http/server.go` — `GET /report/junit.xml`, `GET /report/json`.
- `skillgrid-ui/src/features/reports/ReportsPage.tsx` + six panel components.
- `mnemonic/internal/report/*_test.go`, `mnemonic/internal/scan/parse_test.go` (extended), `skillgrid-ui/src/features/reports/ReportsPage.test.tsx` — passing tests for the truths above.

**Key links** (critical connections between artifacts that must work together):
- `report.Collect` must call `scan.Service.Start` + `scan.Parse` + `scan.Service.StoreFindings` for trivy/semgrep/jscpd/wapiti/nuclei/akca (reuse, not re-implement), and must upsert coverage rows into the `coverage` table.
- `emit.JUnit` and `emit.JSON` must read the same `report.Report` shape; the HTTP routes and the `scan_report` MCP tool must both call `Collect` then the matching emitter (no divergent report paths).
- The dashboard `ReportsPage` must render from `GET /report/json` with no transform; the DAST panel must be conditional on the presence of a `dast` section.
- `scan_report` (MCP) and `GET /report/{fmt}` (HTTP) must produce byte-identical output for the same `format` arg (same emitter, same collect).

**One-way-door decisions** (hard to reverse — flag for explicit user approval before implementing):
- ⚠ **Migration 060 adds the `coverage` table** — an additive SQLite migration; additive (no existing column changes) so low risk, but it is a schema change that, once applied to a live project DB, is not trivially removed. Tagged on Task 4.
- *(The first draft also flagged `SecurityTools() + jscpd`; that is dropped — jscpd is already installed by `GlobalTools()`, so there is no installer contract change. See the Architecture note.)*

## Global Constraints

- Go 1.22+ to build (per ADR-0031). No Go version below 1.22.
- No new Go dependencies without an ADR (per AGENTS.md § Rules). This change adds **zero** Go deps: JUnit is `encoding/xml`, scanners are `exec`. jscpd is a new *installed CLI* (downstream machine), recorded in ADR-0038.
- Trivy findings are advisory-only (per AGENTS.md § Rules) — the report surfaces them; it does not hard-gate on them.
- Fail-open (per ADR-0016): a source failure yields `status="error"`/`"unavailable"`, never a propagated error that kills the report. Only a failed report-row write returns an error.
- Conventional commits only; no AI-attribution trailers (per AGENTS.md § Rules). Spec-zone changes commit before code-zone changes (per AGENTS.md § Rules).
- Coverage is **line % only** (ADR-0038) — no branch/function granularity stored.
- Debt is **markers only** in v1 (TODO/FIXME/XXX); no import cycles / god-nodes (ADR-0038).

## Threat Matrix

Touches subprocesses (scanner CLIs) + a Mnemonic tool contract (new `scan_report`) + DAST URL reachability.

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| Subprocess (running trivy/semgrep/jscpd/coverage/debt grep cmds) | Applicable | Fail-open per ADR-0016: a scanner that errors/times out yields `SourceResult{Status:"error"}`; the report still emits. Per-tool context timeout. | `TestCollect_SourceError_FailOpen` — stub the runner to error on one tool; assert the report still contains the other sources and the failed source has `status="error"`. |
| Mnemonic tool surface (new `scan_report`) | Applicable | New tool; no changes to existing `scan_*` contracts. | `TestScanReportTool_Registered` — assert `scan_report` is registered and its handler returns the emitter output for a fixture report. |
| Git repo / project selection (report runs on the cwd project root) | Applicable | The report targets the current project root, resolved the same way `scan_start` resolves its target (`h.Root()`); no new selector. | `TestCollect_UsesProjectRoot` — assert the coverage dispatch and debt grep run against `h.Root()`, not a hardcoded path. |
| DAST URL reachability | Applicable | Fail-open: unreachable `dast.url` → DAST sources `status="unavailable"`, static report still emits, HTTP 200. | `TestCollect_DASTUnreachable_FailOpen` — `dast.url` set to a dead port; assert DAST sections present with `status="unavailable"` and the report is non-empty. |
| Coverage tool missing from PATH | Applicable | `CmdFor` returns an error for a kind with no tool; collect marks that source `unavailable`, not error. | `TestCollect_CoverageToolMissing_Unavailable` — stub `exec.LookPath` to miss; assert coverage `status="unavailable"`. |

## File Structure

- `mnemonic/internal/report/service.go` — `Service` struct, `Report`/`SourceResult`/`CoverageRow` types, `New`, `Collect`, `reportConfigFrom` (reads `dast.url` from the downstream `.skillgrid/config.yaml`).
- `mnemonic/internal/report/collect.go` — source orchestration (static always, DAST when `dast.url`), debt grep, detected-dispatch coverage, `CmdFor`.
- `mnemonic/internal/report/emit/junit.go` — `JUnit(*Report) ([]byte, error)`; one `<testsuite>` per source (zero-finding tools emit `<testsuite tests="0">`, not omitted).
- `mnemonic/internal/report/emit/json.go` — `JSON(*Report) ([]byte, error)`.
- `mnemonic/internal/report/service_test.go`, `collect_test.go`, `coverage_test.go`, `dast_test.go`, `debt_test.go`, `emit/junit_test.go`, `emit/json_test.go` — tests.
- `mnemonic/internal/report/fixtures/` — `report.json` (a fixture `Report` for emitter tests), `cover.{go,ts,py,rs,java}.json` (coverage parse fixtures).
- `mnemonic/internal/scan/parse.go` — `+ parseJscpd`, `+ parseAkca` (+ `Parse` arms).
- `mnemonic/internal/scan/service.go` — `+ scannerCommands`/`ScannerBin` arms for `jscpd` + `akca`; `+` exported `ExecRunner`; `+ FindingsByScan(ctx, scanID) ([]Finding, error)`.
- `mnemonic/internal/scan/severity.go` — `+ akca` severity map (jscpd hardcodes `MEDIUM`, no map entry).
- `mnemonic/internal/scan/parse_test.go` — `+ TestParseJscpd`, `+ TestParseAkca`, `+` dispatch arms.
- `mnemonic/internal/scan/service_test.go` — `+` `FindingsByScan` test.
- `mnemonic/internal/scan/fixtures/jscpd.json`, `fixtures/akca.json` — parser fixtures.
- `mnemonic/internal/store/migrations/060_coverage.sql` — `coverage` table + index.
- `mnemonic/internal/mcp/tools_report.go` — `scan_report` tool + `registerReportTools` + handler.
- `mnemonic/internal/mcp/tools_report_test.go` — registration + handler + parity test.
- `mnemonic/internal/mcp/server.go` — call `registerReportTools` in **both** `Start()` and `NewServer()`.
- `mnemonic/internal/http/server.go` — `+` two routes + `handleReport` (write-auth wrapped).
- `mnemonic/internal/http/report_test.go` — httptest round-trip.
- `skillgrid-ui/src/features/reports/ReportsPage.tsx` — page shell, fetches `/report/json`.
- `skillgrid-ui/src/features/reports/panels/` — `CvePanel`, `SastPanel`, `DastPanel`, `DuplicationPanel`, `DebtPanel`, `CoveragePanel`.
- `skillgrid-ui/src/features/reports/api.ts` — `fetchReport()` typed client.
- `skillgrid-ui/src/features/reports/ReportsPage.test.tsx` — render from a fixture JSON.
- `skillgrid-ui/src/app.tsx` (or the route table) — `+` the `/reports` route.

---

### Task 1: Report service skeleton + trivy tracer end-to-end (door check)

> Tracer thread: this task wires ONE real source (trivy, already parsed) through collect → emit (JUnit + JSON) → one HTTP route → one dashboard panel. If the end-to-end path does not work, the orchestration-over-findings approach is wrong (see Hypothesis door check).

**Files:**
- Create: `mnemonic/internal/report/service.go`
- Create: `mnemonic/internal/report/collect.go`
- Create: `mnemonic/internal/report/emit/junit.go`
- Create: `mnemonic/internal/report/emit/json.go`
- Create: `mnemonic/internal/report/service_test.go`
- Create: `mnemonic/internal/report/collect_test.go`
- Create: `mnemonic/internal/report/emit/junit_test.go`
- Create: `mnemonic/internal/report/emit/json_test.go`
- Modify: `mnemonic/internal/scan/service.go` (`+` exported `ExecRunner`, `+ FindingsByScan`)
- Create: `mnemonic/internal/scan/service_test.go` additions (`+ TestFindingsByScan`)
- Create: `mnemonic/internal/http/report.go`
- Create: `mnemonic/internal/http/report_test.go`
- Modify: `mnemonic/internal/http/server.go` (register the two `/report/{fmt}` routes, **write-auth wrapped**, in `registerRoutes`)
- Create: `skillgrid-ui/src/features/reports/ReportsPage.tsx`
- Create: `skillgrid-ui/src/features/reports/api.ts`
- Create: `skillgrid-ui/src/features/reports/ReportsPage.test.tsx`
- Modify: `skillgrid-ui/src/app.tsx` (add the `/reports` route)

> **Test helper note:** there is no `store.OpenForTest`. Use the `scan` package's pattern — `store.Open(t.TempDir(), "report-test")` (see `scan/service_test.go` `newTestService`). A `readFixture` helper does not exist cross-package; use the `scan` package's `fixtureBytes(t, name)` and, for the one cross-package fixture in `collect_test.go`, inline the trivy JSON as a `[]byte` constant instead of a path read.

**Interfaces:**
- Consumes: `scan.New(*store.Store, dataDir) *scan.Service`; `(*scan.Service).Start(ctx, tool, target) (*scan.Scan, error)`; `scan.Parse(tool string, data []byte) ([]scan.Finding, error)`; `(*scan.Service).StoreFindings(ctx, scanID) (int, error)`; `(*scan.Service).FindingsByScan(ctx, scanID) ([]scan.Finding, error)` (**added this task**); `store.Store.DB *sql.DB`; `service.ProjectHandle.Root() string` / `.Store() *store.Store` / `.DataDir() string`.
- Produces:
  - `type Report struct { Static []SourceResult; Dynamic []SourceResult; Coverage []CoverageRow; GeneratedAt time.Time }`
  - `type SourceResult struct { Tool string; Status string; Findings []scan.Finding }`
  - `type CoverageRow struct { File string; Pkg string; PctLine float64 }`
  - `func New(st *store.Store, root, dataDir string, cfg Config) *Service` where `type Config struct { DastURL string; CoverageMin float64 }`
  - `func reportConfigFrom(h *service.ProjectHandle) Config` — reads the downstream project's `.skillgrid/config.yaml` (the `dast:` / `dast.url` key) and returns `Config`; missing file/key → zero `Config` (no DAST).
  - `func (s *Service) Collect(ctx context.Context) (*Report, error)` — Task 1 implements only the trivy source + the fail-open shell; later tasks add sources.
  - `func JUnit(r *Report) ([]byte, error)` and `func JSON(r *Report) ([]byte, error)` in package `emit`. **Zero-finding sources emit `<testsuite name="<tool>" tests="0" failures="0">`, not an omitted testsuite.**
  - In `scan`: `var ExecRunner RunnerFunc` (exported form of the existing unexported `execRunner`) and `func (s *Service) FindingsByScan(ctx context.Context, scanID string) ([]Finding, error)` — SELECT findings WHERE scan_id = ?, ordered by id. **On a scan row with `status='error'` it returns `nil, nil`** (callers map that to `Status:"error"`, never a silent `Status:"ok"` with no findings).
  - HTTP: `GET /report/junit.xml` and `GET /report/json` → `s.requireWriteAuth(s.handleReport)` (the report launches scanner subprocesses; wrap like the other project-scoped write routes).
- Seam: the `report.Service` is the seam — `Collect` can be altered (add sources, change dispatch) without touching the emitters or the HTTP routes. Adapters: prod `Collect` + a stubbed-runner test double (≥2, seam justified).
- Deletion test: deleting `report` reappears the "run seven scanners by hand and eyeball them" complexity across the operator, CI, and dashboard. Not a pass-through.

**SATISFIES:** `happy path report-aggregates-static-sources`, `edge report-source-error-is-fail-open`, `happy path junit-xml-per-tool-testsuite`, `happy path junit-xml-zero-finding-source-is-empty-testsuite`, `happy path json-report-shape`, `happy path cve-trivy-report`, `happy path report-http-endpoints`, `happy path dashboard-reports-page`

- [ ] **Step 1: Write the failing test** (collector: trivy source, fail-open shell)

```go
// mnemonic/internal/report/collect_test.go
var trivyFixture = []byte(`{"Results":[{"Target":"go.mod","Vulnerabilities":[
  {"VulnerabilityID":"CVE-2024-0001","PkgName":"example.com/lib","InstalledVersion":"1.0.0",
   "FixedVersion":"1.0.1","Severity":"HIGH","Title":"test vuln","PrimaryURL":"https://x"}]}]}`)

func TestCollect_TrvySource_FailOpen(t *testing.T) {
    st, _ := store.Open(t.TempDir(), "report-test") // no OpenForTest; mirror scan/service_test.go
    defer st.Close()
    // Seed a trivy scan row + findings via the real scan service with a stubbed runner.
    scanSvc := scan.New(st, t.TempDir())
    scanSvc.Run = func(ctx context.Context, bin string, args ...string) ([]byte, error) {
        if bin != "trivy" { t.Fatalf("unexpected bin %s", bin) }
        return trivyFixture, nil
    }
    sc, err := scanSvc.Start(ctx, "trivy", ".")
    require.NoError(t, err)
    _, err = scanSvc.StoreFindings(ctx, sc.ID)
    require.NoError(t, err)

    svc := report.New(st, t.TempDir(), t.TempDir(), report.Config{})
    rep, err := svc.Collect(ctx)
    require.NoError(t, err)
    require.Len(t, rep.Static, 1)
    assert.Equal(t, "trivy", rep.Static[0].Tool)
    assert.Equal(t, "ok", rep.Static[0].Status)
    assert.NotEmpty(t, rep.Static[0].Findings)
}
```

And the zero-finding JUnit assertion (in `emit/junit_test.go`):
```go
func TestJUnit_ZeroFindingSource_EmptyTestSuite(t *testing.T) {
    out, err := emit.JUnit(&report.Report{
        Static: []report.SourceResult{{Tool: "jscpd", Status: "ok", Findings: nil}},
        GeneratedAt: time.Now(),
    })
    require.NoError(t, err)
    s := string(out)
    assert.Contains(t, s, `<testsuite name="jscpd" tests="0" failures="0">`)
    assert.NotContains(t, s, "omitted")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./mnemonic/internal/report/ -run TestCollect_TrvySource_FailOpen -v`
Expected: FAIL — `package report` does not exist yet.
(BDD: the `report-aggregates-static-sources` scenario in `acceptance.feature` is RED — `npx cucumber-js --dry-run` shows it pending.)

- [ ] **Step 3: Write minimal implementation**

```go
// mnemonic/internal/report/service.go
package report

type SourceResult struct {
    Tool     string
    Status   string // "ok" | "error" | "unavailable"
    Findings []scan.Finding
}
type CoverageRow struct {
    File    string
    Pkg     string
    PctLine float64
}
type Report struct {
    Static      []SourceResult
    Dynamic     []SourceResult
    Coverage    []CoverageRow
    GeneratedAt time.Time
}
type Config struct {
    DastURL     string
    CoverageMin float64
}
type Service struct {
    store   *store.Store
    root    string
    dataDir string
    cfg     Config
    run     scan.RunnerFunc // defaults to exec; tests stub
}
func New(st *store.Store, root, dataDir string, cfg Config) *Service {
    return &Service{store: st, root: root, dataDir: dataDir, cfg: cfg, run: scan.ExecRunner}
}
```

```go
// mnemonic/internal/report/collect.go  (Task 1: trivy only)
func (s *Service) Collect(ctx context.Context) (*Report, error) {
    rep := &Report{GeneratedAt: time.Now().UTC()}
    rep.Static = append(rep.Static, s.collectTrivy(ctx))
    return rep, nil
}
func (s *Service) collectTrivy(ctx context.Context) SourceResult {
    scanSvc := scan.New(s.store, s.dataDir)
    scanSvc.Run = s.run
    sc, err := scanSvc.Start(ctx, "trivy", s.root)
    if err != nil { return SourceResult{Tool: "trivy", Status: "error"} }
    if sc.Status == "error" { return SourceResult{Tool: "trivy", Status: "error"} }
    if _, err := scanSvc.StoreFindings(ctx, sc.ID); err != nil {
        return SourceResult{Tool: "trivy", Status: "error"}
    }
    fs, err := scanSvc.FindingsByScan(ctx, sc.ID) // read back normalized findings
    if err != nil { return SourceResult{Tool: "trivy", Status: "error"} }
    return SourceResult{Tool: "trivy", Status: "ok", Findings: fs}
}
```

> Note: `scan.FindingsByScan` (SELECT findings WHERE scan_id = ?, ordered by id) and `scan.ExecRunner` (exported `execRunner`) are added to `scan/service.go` **in this task** (see Files). `FindingsByScan` returns `nil, nil` for an error-row scan; the `sc.Status == "error"` check above is the primary guard so a failed scan is never reported `Status:"ok"` with no findings.

```go
// mnemonic/internal/report/emit/junit.go
func JUnit(r *Report) ([]byte, error) {
    // build <testsuites><testsuite name="trivy" tests=N failures=N>
    //   <testcase name="<cve|rule>:<file>:<line>" classname="<tool>">
    //     <failure message="<severity>: <title>">…</failure>
    //   </testcase>…</testsuite>…</testsuites> via encoding/xml
}
```

```go
// mnemonic/internal/http/report.go
func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
    s.withProjectHandle(w, r, func(h *service.ProjectHandle, projectID string) {
        fmt := r.PathValue("fmt") // "junit.xml" | "json"
        svc := report.New(h.Store(), h.Root(), scanDataDir(h), reportConfigFrom(h))
        rep, err := svc.Collect(r.Context())
        if err != nil { writeError(w, 500, err.Error()); return }
        var (out []byte; ct string; e error)
        if fmt == "junit.xml" { out, e = emit.JUnit(rep); ct = "application/xml" } else {
            out, e = emit.JSON(rep); ct = "application/json"
        }
        if e != nil { writeError(w, 500, e.Error()); return }
        w.Header().Set("Content-Type", ct)
        w.Write(out)
    })
}

// reportConfigFrom reads the downstream project's .skillgrid/config.yaml.
// The report needs only dast.url (and the optional coverage threshold). A
// missing file or key yields a zero Config (no DAST), fail-open.
func reportConfigFrom(h *service.ProjectHandle) report.Config {
    path := filepath.Join(h.Root(), ".skillgrid", "config.yaml")
    data, err := os.ReadFile(path)
    if err != nil { return report.Config{} }
    var c struct {
        Dast struct{ URL string `yaml:"url"` } `yaml:"dast"`
        Quality struct{ CoverageMin int `yaml:"coverage_min"` } `yaml:"quality"`
    }
    if yaml.Unmarshal(data, &c) != nil { return report.Config{} }
    return report.Config{DastURL: c.Dast.URL, CoverageMin: float64(c.Quality.CoverageMin)}
}
```

Register in `server.go` `registerRoutes` (**write-auth wrapped** — the report launches scanner subprocesses):
```go
s.mux.HandleFunc("GET /report/{fmt}", s.requireWriteAuth(s.handleReport))
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./mnemonic/internal/report/ ./mnemonic/internal/http/ -run 'TestCollect|TestHandleReport' -v`
Expected: PASS. **Door check:** `GET /report/json` on the Go fixture returns a `trivy` section with the fixture CVE and HTTP 200. If it fails, stop and revisit the collect-over-findings design.

- [ ] **Step 5: Build the minimal dashboard panel + route**

```tsx
// skillgrid-ui/src/features/reports/api.ts
export interface SourceResult { tool: string; status: string; findings: Finding[] }
export interface ReportJSON {
  static: SourceResult[]; dynamic: SourceResult[];
  coverage: { file: string; pkg: string; pct_line: number }[];
  generated_at: string
}
export async function fetchReport(): Promise<ReportJSON> {
  return apiFetch<ReportJSON>('/report/json')
}
```

```tsx
// skillgrid-ui/src/features/reports/ReportsPage.tsx  (Task 1: trivy/CVE panel only)
export function ReportsPage() {
  const [rep, setRep] = useState<ReportJSON | null>(null)
  const [err, setErr] = useState('')
  useEffect(() => { fetchReport().then(setRep).catch(e => setErr(String(e))) }, [])
  if (err) return <ErrorState message={err} />
  if (!rep) return <LoadingState />
  return <><PageHeader title="Quality Report" />
    <CvePanel findings={rep.static.find(s => s.tool === 'trivy')?.findings ?? []} />
  </>
}
```

Add the route in `app.tsx` (the existing route table) and a `ReportsPage.test.tsx` that renders from a fixture JSON (model on `SecurityPage.test.tsx`).

- [ ] **Step 6: Commit**

```bash
git add mnemonic/internal/report/ mnemonic/internal/http/report*.go mnemonic/internal/http/server.go skillgrid-ui/src/features/reports/ skillgrid-ui/src/app.tsx
git commit -m "feat(report): trivy tracer end-to-end — collect over findings, JUnit+JSON emit, /report route, /reports page"
```

### Task 2: jscpd parser + `scannerCommands` arm (duplication source)

> jscpd is **already installed** by `GlobalTools()` (`skillgrid-cli/internal/install/config.go:91`) — no installer change. The gap is that `scan.Start`'s `scannerCommands` has no `jscpd` arm, so `Start` returns `status=error "unsupported scanner tool"` and the source is permanently `error`. This task adds the parser **and** the `scannerCommands`/`ScannerBin` arms so the collector (Task 8) can run it via the reused `Start`.

**Files:**
- Modify: `mnemonic/internal/scan/parse.go` (`+ parseJscpd`, `+` `Parse` arm)
- Modify: `mnemonic/internal/scan/service.go` (`+ scannerCommands["jscpd"]`; `ScannerBin` needs no change — bin == tool key)
- Create: `mnemonic/internal/scan/fixtures/jscpd.json`
- Modify: `mnemonic/internal/scan/parse_test.go` (`+ TestParseJscpd`, `+` dispatch arm)

**Interfaces:**
- Consumes: `scan.Finding` (existing); the `Parse` switch + `scannerCommands` map (existing).
- Produces: `parseJscpd(data []byte) ([]scan.Finding, error)`; `Parse("jscpd", data)` arm; `scannerCommands["jscpd"] = []string{"--reporters", "json"}`.
- Seam: none new (extends the existing `Parse` switch + `scannerCommands` map).
- Deletion test: pass-through — the parser is a leaf; the `scannerCommands` entry is data.

**SATISFIES:** `happy path duplication-jscpd-report`

- [ ] **Step 1: Write the failing test**

```go
// mnemonic/internal/scan/parse_test.go
func TestParseJscpd(t *testing.T) {
    fs, err := Parse("jscpd", fixtureBytes(t, "jscpd.json"))
    require.NoError(t, err)
    require.Len(t, fs, 2) // fixture has 2 clones
    assert.Equal(t, "duplication", fs[0].RuleID)
    assert.Equal(t, "src/a.ts", fs[0].File)
    assert.Equal(t, "MEDIUM", fs[0].Severity) // parser hardcodes MEDIUM (no severity map entry)
}
func TestParseDispatchIncludesJscpd(t *testing.T) {
    for _, tool := range []string{"trivy","wapiti","nuclei","semgrep","jscpd"} {
        _, err := Parse(tool, []byte("{}"))
        if err == nil { continue }               // valid {} may parse to zero findings
        assert.NotContains(t, err.Error(), "unsupported")
    }
}
func TestScannerCommandsIncludesJscpd(t *testing.T) {
    _, ok := scannerCommands["jscpd"]
    assert.True(t, ok, "jscpd must be runnable via Start")
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./mnemonic/internal/scan/ -run 'TestParseJscpd|TestParseDispatch|TestScannerCommands' -v`
Expected: FAIL — `parse: unsupported tool "jscpd"`; `scannerCommands` has no `jscpd`.

- [ ] **Step 3: Write the fixture + implementation**

`fixtures/jscpd.json` (real jscpd `--reporters json` shape — two clones):
```json
{ "name": "jscpd", "version": "4.0.0", "duplication": {
  "totalTokens": 100, "loc": 20, "minimumLength": 50,
  "clones": [
    { "offsets": [ {"name":"src/a.ts","start":{"line":10,"column":1},"end":{"line":20,"column":1}},
                    {"name":"src/b.ts","start":{"line":5,"column":1},"end":{"line":15,"column":1}} ],
      "percent": 100, "lines": 10, "tokens": 50 },
    { "offsets": [ {"name":"src/c.ts","start":{"line":1,"column":1},"end":{"line":8,"column":1}},
                    {"name":"src/d.ts","start":{"line":2,"column":1},"end":{"line":9,"column":1}} ],
      "percent": 90, "lines": 7, "tokens": 45 }
  ]
}}
```

```go
// parse.go
func parseJscpd(data []byte) ([]Finding, error) {
    var r struct {
        Duplication struct {
            Clones []struct {
                Offsets []struct{ Name string; Start struct{ Line int } } `json:"offsets"`
            } `json:"clones"`
        } `json:"duplication"`
    }
    if err := json.Unmarshal(data, &r); err != nil { return nil, fmt.Errorf("parse jscpd: %w", err) }
    var fs []Finding
    for _, c := range r.Duplication.Clones {
        if len(c.Offsets) < 2 { continue }
        fs = append(fs, Finding{
            Tool: "jscpd", RuleID: "duplication", Severity: "MEDIUM",
            Title: fmt.Sprintf("duplicate block %s↔%s", c.Offsets[0].Name, c.Offsets[1].Name),
            File: c.Offsets[0].Name, Line: c.Offsets[0].Start.Line,
        })
    }
    return fs, nil
}
// in Parse(): case "jscpd": return parseJscpd(data)
```

```go
// service.go — scannerCommands +=
"jscpd": {"--reporters", "json"},
```

> No `severity.go` change for jscpd: the parser hardcodes `MEDIUM` (already normalized), so a `toolSeverityMaps["jscpd"]` entry would be dead code. (akca *does* get a map in Task 3, because its raw severities are `ERROR`/`WARNING`.)

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./mnemonic/internal/scan/ -run 'TestParseJscpd|TestParseDispatch|TestScannerCommands' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/scan/parse.go mnemonic/internal/scan/service.go mnemonic/internal/scan/parse_test.go mnemonic/internal/scan/fixtures/jscpd.json
git commit -m "feat(scan): jscpd parser + scannerCommands arm (duplication source)"
```

### Task 3: akca parser + `scannerCommands` arm

**Files:**
- Modify: `mnemonic/internal/scan/parse.go` (`+ parseAkca`, `+` `Parse` arm)
- Modify: `mnemonic/internal/scan/severity.go` (`+ "akca"` map — its raw severities are `ERROR`/`WARNING`)
- Modify: `mnemonic/internal/scan/service.go` (`+ scannerCommands["akca"]`)
- Create: `mnemonic/internal/scan/fixtures/akca.json`
- Modify: `mnemonic/internal/scan/parse_test.go` (`+ TestParseAkca`)

**Interfaces:**
- Consumes: `scan.Finding`; the `Parse` switch; `scannerCommands` map.
- Produces: `parseAkca(data []byte) ([]scan.Finding, error)`; `Parse("akca", data)` arm; `scannerCommands["akca"] = []string{"scan"}`.
- Seam / Deletion test: leaf parser; pass-through.

**SATISFIES:** `happy path dast-auto-from-config` (akca leg)

- [ ] **Step 1: Write the failing test**

```go
func TestParseAkca(t *testing.T) {
    fs, err := Parse("akca", fixtureBytes(t, "akca.json"))
    require.NoError(t, err)
    require.Len(t, fs, 3)
    assert.Equal(t, "sqli", fs[0].RuleID)
    assert.Equal(t, "/search", fs[0].File)
    assert.Equal(t, "HIGH", scan.NormalizeSeverity("akca", fs[0].Severity))
}
```

- [ ] **Step 2: Run to verify it fails** — `go test ./mnemonic/internal/scan/ -run TestParseAkca -v` → FAIL (unsupported tool).
- [ ] **Step 3: Fixture + implementation**

`fixtures/akca.json` (AKCA evidence-driven shape — 3 findings with proof):
```json
{ "findings": [
    { "ruleId": "sqli", "severity": "ERROR", "url": "/search", "method": "GET",
      "proof": { "payload": "' OR 1=1--", "status": 200 }, "line": 0 }
] }
```
```go
func parseAkca(data []byte) ([]Finding, error) {
    var r struct{ Findings []struct {
        RuleID string `json:"ruleId"`; Severity string `json:"severity"`
        URL string `json:"url"`; Proof struct{ Payload string; Status int } `json:"proof"`
    } } `json:"findings"`
    if err := json.Unmarshal(data, &r); err != nil { return nil, fmt.Errorf("parse akca: %w", err) }
    var fs []Finding
    for _, f := range r.Findings {
        fs = append(fs, Finding{ Tool:"akca", RuleID:f.RuleID, Severity:f.Severity,
            Title: fmt.Sprintf("%s %s (proof: %s→%d)", f.RuleID, f.URL, f.Proof.Payload, f.Proof.Status),
            File: f.URL, Links: []string{f.URL} })
    }
    return fs, nil
}
// in Parse(): case "akca": return parseAkca(data)
// severity.go: "akca": { "ERROR":"HIGH", "WARNING":"MEDIUM", "INFO":"INFO" }
// service.go scannerCommands += "akca": {"scan"},
```

- [ ] **Step 4: Run to verify it passes** — `go test ./mnemonic/internal/scan/ -run TestParseAkca -v` → PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat(scan): akca parser (DAST evidence source)"`

### Task 4: Coverage table (migration 060) + detected dispatch (5 langs, line %)

> ⚠ one-way: migration 060 adds the `coverage` table (additive SQLite schema change). STOP for explicit user approval before Step 5 (applying the migration to a live project DB).

**Files:**
- Create: `mnemonic/internal/store/migrations/060_coverage.sql`
- Create: `mnemonic/internal/report/coverage.go`
- Create: `mnemonic/internal/report/coverage_test.go`

**Interfaces:**
- Consumes: `store.Store.DB *sql.DB`; onboarding project-kind detection (manifest presence: `go.mod`, `package.json`, `pyproject.toml`, `Cargo.toml`, `pom.xml`/`build.gradle`).
- Produces:
  - `type Kind string` — `KindGo|KindTS|KindPy|KindRust|KindJava|KindUnknown`.
  - `func DetectKind(root string) Kind` — returns the first matching manifest.
  - `func CmdFor(k Kind) (bin string, args []string, err error)` — the native coverage command per kind.
  - `func StoreCoverage(ctx context.Context, db *sql.DB, project string, rows []CoverageRow, now string) error` — upserts into `coverage` (one signature; `now` is the `scanned_at` stamp).
- Seam: `CmdFor` is the seam — a new language is a new arm, not a new collector. Adapters: five prod arms + a stubbed-`exec` test (≥2).
- Deletion test: deleting coverage reappears the "run five coverage tools by hand" complexity across the operator + CI. Not a pass-through.

**SATISFIES:** `happy path coverage-dispatch-go/ts/py/rs/java`

- [ ] **Step 1: Write the failing test**

```sql
-- migrations/060_coverage.sql
CREATE TABLE IF NOT EXISTS coverage (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project TEXT NOT NULL,
    file TEXT NOT NULL,
    package TEXT NOT NULL DEFAULT '',
    pct_line REAL NOT NULL,
    scanned_at TEXT NOT NULL,
    UNIQUE (project, file)
);
CREATE INDEX IF NOT EXISTS idx_coverage_project ON coverage(project);
```

```go
// coverage_test.go
func TestDetectKind(t *testing.T) {
    mk := func(files ...string) string {
        d := t.TempDir()
        for _, f := range files { os.WriteFile(filepath.Join(d, f), []byte("{}"), 0o644) }
        return d
    }
    assert.Equal(t, KindGo, DetectKind(mk("go.mod")))
    assert.Equal(t, KindTS, DetectKind(mk("package.json")))
    assert.Equal(t, KindPy, DetectKind(mk("pyproject.toml")))
    assert.Equal(t, KindRust, DetectKind(mk("Cargo.toml")))
    assert.Equal(t, KindJava, DetectKind(mk("pom.xml")))
    assert.Equal(t, KindUnknown, DetectKind(mk("README.md")))
}
func TestCmdFor_AllFive(t *testing.T) {
    bin, _, err := CmdFor(KindGo); require.NoError(t, err)
    assert.Equal(t, "go", bin)
    _, _, err = CmdFor(KindTS); require.NoError(t, err)
    _, _, err = CmdFor(KindUnknown); require.Error(t, err) // no tool → unavailable
}
func TestStoreCoverage_Upsert(t *testing.T) {
    st, _ := store.Open(t.TempDir(), "report-test"); defer st.Close()
    now := time.Now().UTC().Format(time.RFC3339)
    require.NoError(t, StoreCoverage(ctx, st.DB, "p", []CoverageRow{{File:"a.go",Pkg:"x",PctLine:85}}, now))
    require.NoError(t, StoreCoverage(ctx, st.DB, "p", []CoverageRow{{File:"a.go",Pkg:"x",PctLine:90}}, now))
    // upsert, not duplicate
    var n int
    st.DB.QueryRow(`SELECT COUNT(*) FROM coverage WHERE project='p' AND file='a.go'`).Scan(&n)
    assert.Equal(t, 1, n)
    var pct float64
    st.DB.QueryRow(`SELECT pct_line FROM coverage WHERE project='p'`).Scan(&pct)
    assert.Equal(t, 90.0, pct)
}
```

- [ ] **Step 2: Run to verify it fails** — `go test ./mnemonic/internal/report/ -run 'TestDetectKind|TestCmdFor|TestStoreCoverage' -v` → FAIL (no coverage.go).
- [ ] **Step 3: Implementation**

```go
// coverage.go
type Kind string
const ( KindGo Kind="go"; KindTS Kind="ts"; KindPy Kind="py"; KindRust Kind="rust"; KindJava Kind="java"; KindUnknown Kind="" )
func DetectKind(root string) Kind {
    for _, f := range []struct{f Kind; m string}{{KindGo,"go.mod"},{KindTS,"package.json"},
        {KindPy,"pyproject.toml"},{KindRust,"Cargo.toml"},{KindJava,"pom.xml"}} {
        if _, err := os.Stat(filepath.Join(root, f.m)); err == nil { return f.f }
    }
    if _, err := os.Stat(filepath.Join(root, "build.gradle")); err == nil { return KindJava }
    return KindUnknown
}
func CmdFor(k Kind) (string, []string, error) {
    switch k {
    case KindGo:   return "go", []string{"test","-coverprofile=cover.out","./..."}, nil
    case KindTS:   return "nyc", []string{"report","--reporter=json"}, nil
    case KindPy:   return "pytest", []string{"--cov","--cov-report=json"}, nil
    case KindRust: return "cargo-llvm-cov", []string{"--json"}, nil
    case KindJava: return "jacoco", []string{"report"}, nil
    default: return "", nil, fmt.Errorf("no coverage tool for kind %q", k)
    }
}
func StoreCoverage(ctx context.Context, db *sql.DB, project string, rows []CoverageRow, now string) error {
    for _, r := range rows {
        if _, err := db.ExecContext(ctx,
            `INSERT INTO coverage(project,file,package,pct_line,scanned_at) VALUES(?,?,?,?,?)
             ON CONFLICT(project,file) DO UPDATE SET pct_line=excluded.pct_line, scanned_at=excluded.scanned_at`,
            project, r.File, r.Pkg, r.PctLine, now); err != nil { return err }
    }
    return nil
}
```

> **Signature is single-sourced:** `StoreCoverage(ctx, db, project, rows, now)` — the `now` parameter is the `scanned_at` stamp. The Interfaces block, test, and implementation all use this exact form.

> Each kind's JSON output shape differs (Go `cover` JSON, Istanbul `coverage-summary`, coverage.py JSON, llvm-cov JSON, JaCoCo JSON). A per-kind `parseCover<Kind>(data []byte) []CoverageRow` extracts `file → pct_line`. Task 4 implements `parseCoverGo` (the door-check language); the other four arms are added in Task 5. The parse functions are fixture-driven.

- [ ] **Step 4: Run to verify it passes** — `go test ./mnemonic/internal/report/ ./mnemonic/internal/store/ -run 'TestDetectKind|TestCmdFor|TestStoreCoverage' -v` → PASS (migration applies in the store test suite).
- [ ] **Step 5: Commit**

```bash
git add mnemonic/internal/store/migrations/060_coverage.sql mnemonic/internal/report/coverage.go mnemonic/internal/report/coverage_test.go
git commit -m "feat(report): coverage table (060) + detected dispatch (kind detect + cmd-for + store)"
```

### Task 5: Coverage parse arms (TS/Py/Rust/Java) + wire coverage into Collect

**Files:**
- Modify: `mnemonic/internal/report/coverage.go` (`+ parseCoverTS/Py/Rust/Java`)
- Create: `mnemonic/internal/report/fixtures/cover.{go,ts,py,rs,java}.json`
- Modify: `mnemonic/internal/report/coverage_test.go` (`+` per-kind parse tests)
- Modify: `mnemonic/internal/report/collect.go` (`+ collectCoverage` + call in `Collect`)

**Interfaces:**
- Consumes: `DetectKind`, `CmdFor`, `StoreCoverage` (Task 4); `CoverageRow`.
- Produces: `parseCover<Kind>(data []byte) ([]CoverageRow, error)` for the four remaining kinds; `collectCoverage(ctx) (SourceResult, []CoverageRow)`.
- Seam / Deletion test: per-kind parsers are leaves; `collectCoverage` is a collector arm.

**SATISFIES:** `happy path coverage-dispatch-ts/py/rs/java`, `edge coverage-tool-missing-unavailable`

- [ ] **Step 1: Write the failing test**

```go
// parseCover dispatches on kind (the kind arg is required — each language's
// JSON shape differs). Task 4's parseCoverGo is the first arm.
func parseCover(k Kind, data []byte) ([]CoverageRow, error) {
    switch k {
    case KindGo:   return parseCoverGo(data)
    case KindTS:   return parseCoverTS(data)
    case KindPy:   return parseCoverPy(data)
    case KindRust: return parseCoverRust(data)
    case KindJava: return parseCoverJava(data)
    default: return nil, fmt.Errorf("parseCover: unknown kind %q", k)
    }
}

func TestParseCover_AllKinds(t *testing.T) {
    cases := []struct{ kind Kind; file string; want []CoverageRow }{
        {KindTS, "cover.ts.json", []CoverageRow{{File:"src/a.ts", Pkg:"x", PctLine:75}}},
        {KindPy, "cover.py.json", []CoverageRow{{File:"a.py", Pkg:"", PctLine:90}}},
        {KindRust, "cover.rs.json", []CoverageRow{{File:"lib.rs", Pkg:"crate", PctLine:60}}},
        {KindJava, "cover.java.json", []CoverageRow{{File:"Main.java", Pkg:"com.x", PctLine:80}}},
    }
    for _, c := range cases {
        rows, err := parseCover(c.kind, fixtureBytes(t, c.file))
        require.NoError(t, err)
        assert.Equal(t, c.want, rows)
    }
}
// Name matches the threat-matrix row exactly.
func TestCollect_CoverageToolMissing_Unavailable(t *testing.T) {
    // stub exec.LookPath to miss for the detected kind's bin;
    // a real assertion: collectCoverage returns SourceResult{Tool:"coverage", Status:"unavailable"} + nil rows.
    res, rows := svc.collectCoverage(ctx)
    assert.Equal(t, "unavailable", res.Status)
    assert.Empty(t, rows)
}
```

- [ ] **Step 2: Run to verify it fails** — `go test ./mnemonic/internal/report/ -run 'TestParseCover_AllKinds|TestCollectCoverage' -v` → FAIL.
- [ ] **Step 3: Fixtures + per-kind parsers + wire into Collect**

Each fixture is the real native tool's JSON shape for that language; each `parseCover<Kind>` maps to `[]CoverageRow{File, Pkg, PctLine}` (line % only). The `parseCover(k, data)` dispatcher (Step 1) routes to the per-kind arm. `collectCoverage(ctx) (SourceResult, []CoverageRow)` = `DetectKind` → `CmdFor` → run (stubbed in test) → `parseCover(k, out)` → `StoreCoverage`; a `CmdFor`/`LookPath` miss returns `SourceResult{Tool:"coverage", Status:"unavailable"}` + nil rows. Add `rep.Coverage = covRows` and the coverage `SourceResult` to `rep.Static` in `Collect`.

- [ ] **Step 4: Run to verify it passes** — `go test ./mnemonic/internal/report/ -run 'TestParseCover_AllKinds|TestCollectCoverage' -v` → PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat(report): coverage parse arms (ts/py/rs/java) + wire coverage into collect"`

### Task 6: Debt markers (grep TODO/FIXME/XXX)

**Files:**
- Create: `mnemonic/internal/report/debt.go`
- Create: `mnemonic/internal/report/debt_test.go`
- Modify: `mnemonic/internal/report/collect.go` (`+ collectDebt` + call in `Collect`)

**Interfaces:**
- Consumes: `s.root`; `scan.Finding`.
- Produces: `collectDebt(ctx) SourceResult` — walks `s.root` source files, records each `TODO|FIXME|XXX` as a `scan.Finding{Tool:"debt", RuleID:"debt:<marker>", File, Line, Severity:"LOW", Title:"<marker>: <text>"}`.
- Seam / Deletion test: leaf source; pass-through.

**SATISFIES:** `happy path debt-markers-report`

- [ ] **Step 1: Write the failing test**

```go
func TestCollectDebt(t *testing.T) {
    root := t.TempDir()
    os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n// TODO: fix later\n"), 0o644)
    os.WriteFile(filepath.Join(root, "b.go"), []byte("package b\n// FIXME: broken\n// XXX: hack\n"), 0o644)
    st, _ := store.Open(t.TempDir(), "report-test"); defer st.Close()
    svc := report.New(st, root, t.TempDir(), report.Config{})
    res := svc.collectDebt(ctx)
    require.Equal(t, "debt", res.Tool)
    require.Len(t, res.Findings, 3)
    rules := map[string]bool{}
    for _, f := range res.Findings { rules[f.RuleID] = true }
    assert.True(t, rules["debt:TODO"]); assert.True(t, rules["debt:FIXME"]); assert.True(t, rules["debt:XXX"])
}
```

- [ ] **Step 2: Run to verify it fails** — `go test ./mnemonic/internal/report/ -run TestCollectDebt -v` → FAIL (no debt.go).
- [ ] **Step 3: Implementation**

`debt.go`: `filepath.WalkDir` over `s.root` (skip `node_modules`, `.git`, `vendor`), for each text source file read lines, regex `(TODO|FIXME|XXX)[:\s]` → emit a finding per hit. Wire `rep.Static = append(rep.Static, s.collectDebt(ctx))` in `Collect`.

- [ ] **Step 4: Run to verify it passes** — `go test ./mnemonic/internal/report/ -run TestCollectDebt -v` → PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat(report): debt markers source (TODO/FIXME/XXX grep)"`

### Task 7: DAST auto from `dast.url` + fail-open (wapiti/nuclei/akca)

**Files:**
- Modify: `mnemonic/internal/report/collect.go` (`+ collectDAST` + call in `Collect`)
- Create: `mnemonic/internal/report/dast_test.go`
- Modify: `mnemonic/internal/report/service.go` (`Config.DastURL` already defined; use it)

**Interfaces:**
- Consumes: `s.cfg.DastURL`; `scan.Service.Start`/`Parse`/`StoreFindings` (reuse for wapiti/nuclei/akca); `exec` for a reachability pre-check.
- Produces: `collectDAST(ctx) []SourceResult` — if `DastURL` is empty, returns `nil` (no DAST section). If set, a reachability probe (HEAD, 3s timeout); unreachable → all three `SourceResult{Status:"unavailable"}`. Reachable → run wapiti/nuclei/akca via the scan service.
- Seam / Deletion test: collector arm; the reachability probe is the seam (prod HTTP probe + test stub).

**SATISFIES:** `happy path dast-auto-from-config`, `edge dast-unreachable-fail-open`, `happy path dast-unset-is-absent`

- [ ] **Step 1: Write the failing test**

```go
func TestCollectDAST_Unset_Nil(t *testing.T) {
    st, _ := store.Open(t.TempDir(), "report-test"); defer st.Close()
    svc := report.New(st, t.TempDir(), t.TempDir(), report.Config{})
    assert.Nil(t, svc.collectDAST(ctx))
}
func TestCollectDAST_Unreachable_FailOpen(t *testing.T) {
    // DastURL = "http://127.0.0.1:1" (dead port); stub the probe to report unreachable
    svc := report.New(..., report.Config{DastURL: "http://127.0.0.1:1"})
    res := svc.collectDAST(ctx)
    require.Len(t, res, 3) // wapiti, nuclei, akca
    for _, r := range res { assert.Equal(t, "unavailable", r.Status) }
}
func TestCollectDAST_Reachable_RunsThree(t *testing.T) {
    // stub the probe reachable + stub the scan runner to return per-tool fixtures
    // assert three SourceResult{Status:"ok"} with findings
}
```

- [ ] **Step 2: Run to verify it fails** — `go test ./mnemonic/internal/report/ -run TestCollectDAST -v` → FAIL.
- [ ] **Step 3: Implementation**

`collectDAST`: empty `DastURL` → `nil`. Else `probe(ctx, DastURL)` (HEAD, 3s ctx timeout); error → three `unavailable`. Else for each of `wapiti`, `nuclei`, `akca`: `scan.Service.Start(ctx, tool, DastURL)` → `StoreFindings` → read back → `SourceResult{Status:"ok"}`. A per-tool error → that tool `error`, others continue (fail-open). Wire `rep.Dynamic = s.collectDAST(ctx)` in `Collect`.

- [ ] **Step 4: Run to verify it passes** — `go test ./mnemonic/internal/report/ -run TestCollectDAST -v` → PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat(report): DAST auto from dast.url + fail-open (wapiti/nuclei/akca)"`

### Task 8: Wire remaining static sources into Collect (semgrep + jscpd)

**Files:**
- Modify: `mnemonic/internal/report/collect.go` (`+ collectSemgrep`, `+ collectJscpd` + calls in `Collect`)
- Modify: `mnemonic/internal/report/collect_test.go` (`+` assertions the full static set is present)

**Interfaces:**
- Consumes: `scan.Service` (reuse) for semgrep + jscpd; `s.root`.
- Produces: the full `Static` set = `[trivy, semgrep, jscpd, debt, coverage]` in `Collect`.
- Seam / Deletion test: collector arms; pass-through leaves.

**SATISFIES:** `happy path sast-semgrep-report`, `happy path duplication-jscpd-report`

- [ ] **Step 1: Write the failing test**

```go
func TestCollect_FullStaticSet(t *testing.T) {
    // stub the runner to return the right fixture per tool (trivy/semgrep/jscpd)
    st, _ := store.Open(t.TempDir(), "report-test"); defer st.Close()
    svc := report.New(st, goFixtureRoot(t), t.TempDir(), report.Config{})
    rep, err := svc.Collect(ctx)
    require.NoError(t, err)
    tools := map[string]bool{}
    for _, s := range rep.Static { tools[s.Tool] = true }
    for _, want := range []string{"trivy","semgrep","jscpd","debt","coverage"} {
        assert.True(t, tools[want], "missing static source %s", want)
    }
}
```

- [ ] **Step 2: Run to verify it fails** — `go test ./mnemonic/internal/report/ -run TestCollect_FullStaticSet -v` → FAIL (semgrep/jscpd not in Static).
- [ ] **Step 3: Implementation** — `collectSemgrep`/`collectJscpd` mirror `collectTrivy` (Start → StoreFindings → read back). Append both to `Static` in `Collect`.
- [ ] **Step 4: Run to verify it passes** — `go test ./mnemonic/internal/report/ -run TestCollect_FullStaticSet -v` → PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat(report): wire semgrep + jscpd into collect (full static set)"`

### Task 9: `scan_report` MCP tool + `GET /report/{fmt}` parity

**Files:**
- Create: `mnemonic/internal/mcp/tools_report.go`
- Create: `mnemonic/internal/mcp/tools_report_test.go`
- Modify: `mnemonic/internal/mcp/server.go` (call `registerReportTools` in **both** `Start()` and `NewServer()` — `registerScanTools` is called in both, line 72 and line 112)

**Interfaces:**
- Consumes: `report.New`, `report.Service.Collect`, `emit.JUnit`, `emit.JSON` (Tasks 1, 5, 8).
- Produces: `scan_report` MCP tool — arg `{ format: "junit"|"json" }` — returns the emitter output as text content. `registerReportTools(s *server.MCPServer)`.
- Seam: none new (mirrors `registerScanTools`).
- Deletion test: pass-through (thin MCP wrapper over the same `Collect`+emitter the HTTP route uses).

**SATISFIES:** `happy path report-aggregates-static-sources` (MCP leg), `happy path report-mcp-tool-registered`, `happy path report-mcp-http-parity`

- [ ] **Step 1: Write the failing test**

```go
func TestScanReportTool_Registered(t *testing.T) {
    // build the MCP server, assert scan_report is in the tool list
}
func TestScanReportHandler_JSON(t *testing.T) {
    // with a seeded store, call the handler with {format:"json"}
    // assert the result text is valid JSON with a "static" key
}
func TestScanReport_Parity_SameReportBothEmitters(t *testing.T) {
    // Build ONE Report (GeneratedAt is time.Now(), so two Collects differ —
    // parity is "same Report → both emitters", not two collects).
    rep := loadFixtureReport(t)
    j1, _ := emit.JUnit(rep); j2, _ := emit.JUnit(rep)
    assert.Equal(t, j1, j2) // deterministic per Report
    // and the MCP handler for format=junit returns exactly emit.JUnit(rep)
}
```

- [ ] **Step 2: Run to verify it fails** — `go test ./mnemonic/internal/mcp/ -run TestScanReport -v` → FAIL.
- [ ] **Step 3: Implementation**

```go
func registerReportTools(s *server.MCPServer) {
    s.AddTool(mcplib.NewTool("scan_report",
        mcplib.WithDescription("Collect the unified quality report (CVE/SAST/DAST/dup/debt/coverage) and emit it as JUnit XML or JSON. Fail-open per ADR-0016."),
        mcplib.WithString("format", mcplib.Required(), mcplib.Description("junit | json"))),
        handleScanReport)
}
func handleScanReport(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
    _, h, cleanup, err := openService(); if err != nil { return toolError(err) }
    defer cleanup()
    fmt := stringArg(req, "format")
    svc := report.New(h.Store(), h.Root(), scanDataDir(h), reportConfigFrom(h))
    rep, err := svc.Collect(ctx)
    if err != nil { return toolError(err) }
    var out []byte
    if fmt == "junit" { out, err = emit.JUnit(rep) } else { out, err = emit.JSON(rep) }
    if err != nil { return toolError(err) }
    return &mcplib.CallToolResult{Content: []mcplib.Content{&mcplib.TextContent{Text: string(out)}}}, nil
}
```

- [ ] **Step 4: Run to verify it passes** — `go test ./mnemonic/internal/mcp/ ./mnemonic/internal/http/ -run 'TestScanReport|TestHandleReport' -v` → PASS (parity holds).
- [ ] **Step 5: Commit** — `git commit -m "feat(mcp): scan_report tool + HTTP/MCP report parity"`

### Task 10: Dashboard `/reports` page — six panels

**Files:**
- Create: `skillgrid-ui/src/features/reports/panels/CvePanel.tsx`
- Create: `skillgrid-ui/src/features/reports/panels/SastPanel.tsx`
- Create: `skillgrid-ui/src/features/reports/panels/DastPanel.tsx`
- Create: `skillgrid-ui/src/features/reports/panels/DuplicationPanel.tsx`
- Create: `skillgrid-ui/src/features/reports/panels/DebtPanel.tsx`
- Create: `skillgrid-ui/src/features/reports/panels/CoveragePanel.tsx`
- Modify: `skillgrid-ui/src/features/reports/ReportsPage.tsx` (render all six)
- Modify: `skillgrid-ui/src/features/reports/ReportsPage.test.tsx` (render full fixture + DAST-conditional)

**Interfaces:**
- Consumes: `fetchReport()` + `ReportJSON` (Task 1 `api.ts`); the severity-badge component. **`SevBadge` is a local function in `SecurityPage.tsx`, not exported** — Step 3 extracts it to `skillgrid-ui/src/components/ui/Badges.tsx` (exported) and re-points `SecurityPage` at it, so the six panels share one badge.
- Produces: `ReportsPage` rendering six panels from the JSON; `DastPanel` conditional on `rep.dynamic` presence.
- Seam: each panel is a pure presentational component (props in, no fetch) — testable in isolation.
- Deletion test: each panel reappears the "read raw JSON" complexity for its capability. Not pass-through.

**SATISFIES:** `happy path dashboard-reports-page`, `edge dashboard-dast-conditional`, `happy path dashboard-dast-unavailable`

- [ ] **Step 1: Write the failing test**

```tsx
// ReportsPage.test.tsx
it('renders all six panels from the fixture JSON', async () => {
  vi.mocked(fetchReport).mockResolvedValue(fullFixture) // has static+dynamic+coverage
  render(<ReportsPage />)
  await screen.findByText('CVE'); await screen.findByText('SAST');
  await screen.findByText('DAST'); await screen.findByText('Duplication');
  await screen.findByText('Debt'); await screen.findByText('Coverage');
});
it('omits the DAST panel when no dynamic section', async () => {
  vi.mocked(fetchReport).mockResolvedValue(staticOnlyFixture)
  render(<ReportsPage />)
  expect(await screen.queryByText('DAST')).not.toBeInTheDocument()
});
it('shows DAST unavailable when the section has status=unavailable', async () => {
  vi.mocked(fetchReport).mockResolvedValue(dastUnavailableFixture)
  render(<ReportsPage />)
  expect(await screen.findByText(/unavailable/i)).toBeInTheDocument()
})
```

- [ ] **Step 2: Run to verify it fails** — `cd skillgrid-ui && npx vitest run src/features/reports/ReportsPage.test.tsx` → FAIL.
- [ ] **Step 3: Implementation** — first extract `SevBadge` from `SecurityPage.tsx` into `components/ui/Badges.tsx` (exported; `SecurityPage` imports it back). Then six panels (each: severity counts via the shared `SevBadge`, a finding list with file:line, a coverage heat bar via d3 for `CoveragePanel`); `ReportsPage` renders the static panels always, `DastPanel` only when `rep.dynamic?.length`.
- [ ] **Step 4: Run to verify it passes** — `cd skillgrid-ui && npx vitest run src/features/reports/` → PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat(ui): /reports page with six quality panels (DAST conditional)"`

### Task 11: acceptance.feature + full-suite green + QA gate

**Files:**
- Create: `.skillgrid/specs/2026-10-17-quality-report/acceptance.feature`
- (no code — the BDD scenarios referenced by every task's SATISFIES)

**Interfaces:** none.

**SATISFIES:** all scenarios (this task makes the whole `acceptance.feature` green and confirms the full suite).

- [ ] **Step 1: Author `acceptance.feature`** — already authored (24 scenarios) with every scenario named in the task SATISFIES lines: `report-aggregates-static-sources`, `report-source-error-is-fail-open`, `junit-xml-per-tool-testsuite`, `junit-xml-zero-finding-source-is-empty-testsuite`, `json-report-shape`, `cve-trivy-report`, `sast-semgrep-report`, `duplication-jscpd-report`, `debt-markers-report`, `dast-auto-from-config`, `dast-unreachable-fail-open`, `dast-unset-is-absent`, `coverage-dispatch-{go,ts,py,rs,java}`, `coverage-tool-missing-unavailable`, `dashboard-reports-page`, `dashboard-dast-conditional`, `dashboard-dast-unavailable`, `report-http-endpoints`, `report-mcp-tool-registered`, `report-mcp-http-parity`. Confirm the suite runs them all.
- [ ] **Step 2: Run the BDD dry-run + full suite**

Run: `go test ./...` and `cd skillgrid-ui && npx vitest run`
Expected: PASS (the Go + UI suites green; the BDD scenarios all present and covered).

- [ ] **Step 3: Run the QA gate** (`skillgrid:qa` — L3, per the risky classification): full suite + build + Trivy (advisory) + the threat-matrix RED tests present in the suite.
- [ ] **Step 4: Commit** — `git commit -m "test(report): acceptance.feature + full-suite green"`
