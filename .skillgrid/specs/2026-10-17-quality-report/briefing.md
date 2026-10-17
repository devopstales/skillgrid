# Quality Report — Design Briefing (requirements & intent)

> **STATUS:** `draft` (2026-10-17)

## Problem / Intent

A project created with skillgrid can be Go, TS, Python, Rust, or Java, but today the only quality signal it surfaces is a live Trivy CVE count on the SecurityPage. An operator who wants to know "is this project healthy?" has to run CVE, SAST, DAST, duplication, debt, and coverage scans by hand across up to seven tools, in up to five languages, and eyeball the outputs. This feature gives skillgrid a single unified quality report — one command, one durable store, one CI-consumable JUnit XML, one dashboard page — for **any** project it is installed in.

## Purpose & Success Criteria

- **Purpose:** Produce a unified, multi-language quality report (CVE, SAST, DAST, duplication, code debt, test coverage) for the project skillgrid is installed in, normalized into the existing findings store, emitted as JUnit XML + JSON, and visualized on a `/reports` dashboard page.
- **Success criteria:**
  - `GET /report/junit.xml` returns valid JUnit XML with one `<testsuite>` per source that ran.
  - `GET /report/json` returns the rich JSON the dashboard consumes.
  - A Go project with no `dast.url` gets a static-only report (CVE, SAST, duplication, debt, coverage panels; DAST absent).
  - A project with `dast.url` set gets DAST findings; an unreachable URL marks DAST *unavailable* (fail-open), not an error.
  - Coverage is measured for Go, TS/JS, Python, Rust, and Java via detected dispatch (native tool per language, line % only).
  - `jscpd` and `akca` are installed by the installer and their output is parsed into the findings store.
- **Out of scope:**
  - Import cycles + god-nodes (the code indexer is Go-only; debt is markers-only in v1).
  - Per-project custom semgrep rule *seeding* (default registry rules + auto-pickup of a project's own `.semgrep.yml` only).
  - Super-Linter / MegaLinter (linter orchestrators, not findings emitters — wrong category).
  - A custom HTML report generator (JUnit XML + the dashboard page cover "report" + "visualize").
  - Branch / function coverage granularity (line % only).

## Context

- **Scan pipeline (exists, reused):** `mnemonic/internal/scan/` — `scan_start` MCP tool runs a scanner, persists raw output, records a `scans` row; `scan_store_findings` parses raw output into normalized `findings`. Parsers exist for trivy, wapiti, nuclei, semgrep (`parse.go`). AKCA is installed and runnable but has **no parser yet**.
- **Store (exists, extended):** migration `050_scan_findings.sql` — `scans`, `findings` (has both `cve_id` and `rule_id` columns, FTS5), `dependencies` tables. This change adds migration `051` for a `coverage` table.
- **Installer (exists, extended):** `skillgrid-cli/internal/install/config.go` `SecurityTools()` returns the scanners installed on a downstream machine (wapiti3, semgrep, akca, nuclei, trivy). This change adds jscpd.
- **Dashboard (exists, extended):** `skillgrid-ui/src/features/security/SecurityPage.tsx` renders a live Trivy result with severity badges; `d3` is already a UI dependency. This change adds a new `features/reports/` page that reuses the severity-badge components and d3.
- **Coverage dispatch (new):** onboarding (`lifecycle/onboarding/references/process.md`) already detects project kind from manifests (`go.mod`, `package.json`, `pyproject.toml`, `Cargo.toml`, `pom.xml`/`build.gradle`). The report service reuses that detection to choose the native coverage command.
- **Constraints:** Go 1.22+ to build (ADR-0031); no new Go dependencies without an ADR (AGENTS.md § Rules); Trivy findings are advisory-only (AGENTS.md § Rules); fail-open per ADR-0016 (a scanner failure yields a `status=error` row, not a propagated error).
- **Fits existing patterns:** yes — the report service mirrors `scan/` (a package with a Service, parsers, fixtures, and MCP tool registration); the coverage table mirrors the `findings`/`dependencies` tables; the dashboard page mirrors `SecurityPage`.

## Approaches Considered

- **Chosen:** A new `report` service that *orchestrates the existing scan pipeline* (reuses `scan_start`/`scan_store_findings`/`findings` for the sources that already have parsers) plus two new parsers (jscpd, akca), a new grep-based debt source, a detected-dispatch coverage source, a `coverage` table, JUnit+JSON emitters, and a dashboard page. Rationale: reuses the durable store and the normalized-findings model (ADR-0030) instead of a parallel pipeline; one report, one format, one page.
- **Rejected:** A standalone report tool (new binary, new store) — duplicates the findings model and the scan pipeline; a per-tool HTML report (each scanner's own output) — no unified view, no CI format, no dashboard; Super-Linter/MegaLinter as the report source — they emit lint feedback, not normalized findings, and add a container dependency to a single-binary product.

## Requirements

1. **R1 — Unified report command:** The report service aggregates all configured sources into one report on demand.
   - **Current:** No aggregation; each scanner is run and stored independently via `scan_*` MCP tools; no report output.
   - **Target:** A `report` service (and `scan_report` MCP tool + `GET /report/{fmt}` HTTP routes) that collects the static sources always, the DAST sources when `config.yaml` `dast.url` is set, normalizes them into the `findings` + `coverage` tables, and emits the report.
   - **Acceptance:** `scan_report` / `GET /report/json` returns a report containing every source that ran; a source that failed is present with `status=error`, not absent.
   - **Acceptance scenario:** `happy path report-aggregates-static-sources` → `acceptance.feature`

2. **R2 — JUnit XML output:** The report emits JUnit XML consumable by CI.
   - **Current:** No JUnit output anywhere in the repo.
   - **Target:** `GET /report/junit.xml` returns well-formed JUnit XML with one `<testsuite name="<tool>">` per source that ran; each finding is a `<testcase>` whose `<failure message="<severity>: <title>">` carries the normalized severity.
   - **Acceptance:** the emitted XML parses as JUnit; a Go project with a known vulnerable dependency yields a `trivy` `<testsuite>` with a `<testcase>` whose name contains the CVE id.
   - **Acceptance scenario:** `happy path junit-xml-per-tool-testsuite` → `acceptance.feature`

3. **R3 — JSON output:** The report emits the rich JSON the dashboard consumes.
   - **Current:** No report JSON.
   - **Target:** `GET /report/json` returns per-source sections (tool, status, findings with severity/title/cve_id/rule_id/file/line, coverage rows) shaped for the dashboard panels.
   - **Acceptance:** the JSON contains a section per source that ran; the dashboard renders it without a transform.
   - **Acceptance scenario:** `happy path json-report-shape` → `acceptance.feature`

4. **R4 — CVE (Trivy):** CVEs are reported, multi-language.
   - **Current:** Trivy is installed, parsed, and stored; the SecurityPage shows a *live* trivy run only, not the store.
   - **Target:** The report includes Trivy CVE findings from the store (already normalized: `cve_id`, `package`, `fixed_version`).
   - **Acceptance:** a project with a vulnerable dependency shows the CVE in the report's CVE panel and JUnit `trivy` testsuite.
   - **Acceptance scenario:** `happy path cve-trivy-report` → `acceptance.feature`

5. **R5 — SAST (Semgrep):** Code-level rule findings are reported, multi-language.
   - **Current:** Semgrep is installed, parsed, and stored.
   - **Target:** The report includes Semgrep SAST findings from the store (normalized: `rule_id`, file/line). Default registry rules; a project's own `.semgrep.yml` is picked up automatically if present.
   - **Acceptance:** a project with a code-level rule hit shows it in the SAST panel and JUnit `semgrep` testsuite.
   - **Acceptance scenario:** `happy path sast-semgrep-report` → `acceptance.feature`

6. **R6 — DAST (Wapiti + Nuclei + AKCA):** Live-app findings are reported when `dast.url` is set.
   - **Current:** Wapiti + Nuclei are installed, parsed, and stored. AKCA is installed and runnable but has no parser.
   - **Target:** When `config.yaml` `dast.url` is set, the report runs wapiti, nuclei, and akca against that URL and includes their findings. `ParseAkca` is added. When the URL is unreachable, the DAST sources are marked *unavailable* (fail-open) and the static report still emits.
   - **Acceptance:** with `dast.url` set and reachable, the report includes DAST findings from all three tools; with the URL unreachable, the DAST panel shows "unavailable: <reason>" and the report still returns 200.
   - **Acceptance scenario:** `happy path dast-auto-from-config` / `edge dast-unreachable-fail-open` → `acceptance.feature`

7. **R7 — Duplication (jscpd):** Code clones are reported, multi-language.
   - **Current:** jscpd has a skill but is not in `SecurityTools()` and has no parser.
   - **Target:** jscpd is added to `SecurityTools()` (installed on downstream machines) and `ParseJscpd` maps its `--reporters json` clones into `findings` (rule_id=`duplication`, file/line from the clone).
   - **Acceptance:** a project with a duplicated block shows it in the duplication panel and JUnit `jscpd` testsuite; `TestSecurityToolsIncludeJscpd` passes.
   - **Acceptance scenario:** `happy path duplication-jscpd-report` → `acceptance.feature`

8. **R8 — Code debt (markers):** TODO/FIXME/XXX markers are reported, universal.
   - **Current:** No debt source in the report.
   - **Target:** The report greps the project source for `TODO|FIXME|XXX` and records each as a debt finding (rule_id=`debt:<marker>`, file/line).
   - **Acceptance:** a project containing a `TODO:` comment shows it in the debt panel and JUnit `debt` testsuite.
   - **Acceptance scenario:** `happy path debt-markers-report` → `acceptance.feature`

9. **R9 — Test coverage (detected dispatch):** Line % coverage is measured per language.
   - **Current:** No coverage integration. `graph.FairCoverage` is graph fair-coverage (edges table), not test coverage.
   - **Target:** The report detects the project kind (reusing onboarding detection) and runs the native coverage tool: Go `go test -coverprofile` + `go tool cover -json`; TS/JS `nyc report --reporter=json`; Python `pytest --cov --cov-report=json`; Rust `cargo-llvm-cov --json`; Java `jacoco` report → `jacoco.json`. Per-file line % is stored in a new `coverage` table (migration 051). Line % only.
   - **Acceptance:** for each of the five languages, a fixture project yields per-file line % in the `coverage` table and the coverage panel; a project kind with no configured coverage tool is marked unavailable (fail-open), not an error.
   - **Acceptance scenario:** `happy path coverage-dispatch-go` / `happy path coverage-dispatch-ts` / `happy path coverage-dispatch-py` / `happy path coverage-dispatch-rs` / `happy path coverage-dispatch-java` → `acceptance.feature`

10. **R10 — Dashboard `/reports` page:** All six capabilities are visualized.
    - **Current:** SecurityPage shows live Trivy only; no report page.
    - **Target:** A new `/reports` page with six panels — CVE, SAST, DAST, duplication, debt, coverage — fed by `GET /report/json`. Reuses the SecurityPage severity badges and d3. The DAST panel is conditional on `dast.url` (absent when unset; "unavailable" when unreachable).
    - **Acceptance:** the page renders all six panels from the JSON; with no `dast.url` the DAST panel is absent; with an unreachable `dast.url` it shows unavailable.
    - **Acceptance scenario:** `happy path dashboard-reports-page` / `edge dashboard-dast-conditional` → `acceptance.feature`

## Implementation Decisions

- **Modules to build/modify:**
  - NEW `mnemonic/internal/report/` — `collect` (orchestrator + detected-dispatch coverage), `emit/junit.go`, `emit/json.go`, `service.go`.
  - EXTEND `mnemonic/internal/scan/parse.go` — `+ ParseJscpd`, `+ ParseAkca` (mirror the existing `ParseSemgrep`/`ParseNuclei` shape); `+ fixtures/jscpd.json`, `+ fixtures/akca.json`.
  - EXTEND `skillgrid-cli/internal/install/config.go` — `SecurityTools() + jscpd` (manager `uv` or `npm`, `Bin: "jscpd"`).
  - NEW migration `mnemonic/internal/store/migrations/051_coverage.sql` — `coverage(id, project, file, package, pct_line, scanned_at)` table + index.
  - NEW `mnemonic/internal/mcp/tools_report.go` — `scan_report` tool.
  - EXTEND `mnemonic/internal/http/server.go` — `GET /report/junit.xml`, `GET /report/json`.
  - NEW `skillgrid-ui/src/features/reports/` — `ReportsPage.tsx` + six panel components.
  - EXTEND `.skillgrid/config.yaml` (the *downstream* project's config, via onboarding) — `dast.url` key.
- **Interfaces:**
  - `report.Service.Collect(ctx, projectRoot, cfg) (*Report, error)` where `Report` = `{ Static []SourceResult; Dynamic []SourceResult; Coverage []CoverageRow; GeneratedAt time.Time }` and `SourceResult` = `{ Tool string; Status string; Findings []Finding }` (reuses the scan `Finding` shape).
  - `emit.JUnit(*Report) ([]byte, error)` — one `<testsuite>` per `SourceResult`.
  - `emit.JSON(*Report) ([]byte, error)`.
  - `scan_report` MCP tool: args `{ format: "junit" | "json" }`, returns the serialized report.
  - Coverage dispatch: `coverage.CmdFor(kind) (string, []string, error)` returning the command + args per detected kind.
- **Data flow:** `scan_report` / `GET /report/{fmt}` → `report.Service.Collect` → for each source: run (reuse `scan.Service.Start` for tools that have it; direct exec for grep/coverage) → parse (reuse `scan.Parse` or new parsers) → upsert into `findings` / `coverage` → `emit.JUnit` / `emit.JSON` → response. Raw outputs persist under the data dir keyed by `scan_id` (existing pattern).
- **Error handling:** fail-open per ADR-0016 — a source that errors yields `SourceResult{Status:"error"}` and the report still emits. DAST unreachable → `Status:"unavailable"`. Coverage tool absent for a kind → that source `unavailable`. Only a failed report-row write returns an error.
- **Dependencies:** no new Go dependencies (jscpd/akca are external CLIs invoked via `exec`, not Go imports; JUnit XML is built with `encoding/xml`, not a new library). jscpd is a new *installed tool* (downstream machine), not a Go dep — record in the ADR.

## Testing Decisions

- **What makes a good test:** test the report's observable output (JUnit XML shape, JSON shape, findings in the store), not the collector's internals. Parsers are tested against fixture files (the existing `parse_test.go` pattern), never live scanner binaries.
- **Modules to test:** `report/collect` (dispatch + aggregation, with a stubbed runner), `report/emit/junit` + `report/emit/json` (golden-output), `scan/parse` `+ ParseJscpd`/`ParseAkca` (fixtures), `install` `TestSecurityToolsIncludeJscpd`, `coverage.CmdFor` (per-kind), the HTTP routes (httptest round-trip), the dashboard page (component render from a fixture JSON).
- **Prior art:** `mnemonic/internal/scan/parse_test.go` (fixture-driven parsers), `mnemonic/internal/mcp/tools_scan_test.go` (MCP tool registration + handler), `skillgrid-ui/src/features/security/SecurityPage.test.tsx` (panel render from a fixture response).
- **Edge cases:** a source that times out (per-tool timeout, fail-open); a project with no source files (empty report, not an error); a mixed-language project (coverage dispatch picks one kind); DAST set but URL down (unavailable, not error); a coverage tool missing from PATH (unavailable, not error); jscpd/akca output with zero findings (empty testsuite, not omitted).

## Impact on Global Docs

- `.skillgrid/artifacts/00-prd.md`: add the unified quality report as a feature (one line in the relevant capability section) — *only if the PRD has a quality/security capability section; otherwise None*.
- `.skillgrid/ASSUMPTIONS.md`: add ADR-0038 row (unified quality report: report service over the findings store + JUnit/JSON emit + `/reports` dashboard; jscpd + akca as installed scanners; coverage as detected dispatch; DAST auto from `dast.url`, fail-open).
- `.skillgrid/ARCHITECTURE.md`: add the `report` component (a service under `mnemonic/internal/`), the `coverage` table (051), the `scan_report` MCP tool, the `/report/{fmt}` HTTP routes, and the `/reports` dashboard page.

## Clarity Report

| Dimension           | Score | Min  | Status | Notes                              |
|---------------------|-------|------|--------|------------------------------------|
| Goal Clarity        | 0.95  | 0.75 | ok     | One report, six capabilities, any project. |
| Boundary Clarity    | 0.90  | 0.70 | ok     | In/out of scope explicit; Go-only debt deferred. |
| Constraint Clarity  | 0.90  | 0.65 | ok     | No new Go deps; fail-open; advisory Trivy. |
| Acceptance Criteria | 0.95  | 0.70 | ok     | Every requirement has Current/Target/Acceptance + scenario. |
| **Clarity**         | 0.05  | ≤0.20| pass   | |

**Interview log:**

| Round | Question summary         | Decision locked                    |
|-------|-------------------------|------------------------------------|
| 1     | Coverage v1 languages? | All five: Go, TS/JS, Python, Rust, Java (detected dispatch). |
| 1     | DAST explicit or auto? | Auto from `config.yaml` `dast.url`. |
| 1     | AKCA now or defer? | In v1 — add `ParseAkca`. |
| 1     | Code debt structure? | Markers only (TODO/FIXME/XXX); cycles/god-nodes deferred (Go-only indexer). |
| 2     | Coverage granularity? | Line % only. |
| 2     | DAST unreachable? | Fail-open, mark DAST unavailable, never block the report. |

## Open Questions & Assumptions

- **Assumption:** jscpd is installed via `npm` (it is an npm package, `kucherenko/jscpd`); the installer's `SecurityTools()` currently uses `uv`/`go` managers, so a `npm` manager arm is needed — or jscpd is run via `npx` (no install). The blueprint settles this; the ADR records the choice.
- **Assumption:** the downstream project's root is resolved the same way `scan_start` resolves its target (cwd / project root). No new repo-selection logic.
- **Assumption:** JUnit XML severity mapping reuses the existing `NormalizeSeverity` (`ERROR`→`HIGH`, etc.). No new severity model.

## Decisions (ADR)

- `.skillgrid/artifacts/04-adr-0038-unified-quality-report.md` — the unified quality report is a report service over the existing findings store (not a new pipeline), emitting JUnit XML + JSON, with a `/reports` dashboard page; jscpd + akca become installed scanners; coverage is a detected dispatch (line %, five languages); DAST is auto from `dast.url` and fail-open.

## Terms

- `.skillgrid/artifacts/02-technical-terms.md`: **Scan Finding** (already defined — the report reuses it) is the unit the report aggregates. No new terms introduced; the report is a *view* over existing findings + a new `coverage` row. If the blueprint introduces a `Report`/`SourceResult` type, add it to the terms file as an alias note.
