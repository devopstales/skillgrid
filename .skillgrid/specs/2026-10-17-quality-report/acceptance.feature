# Quality Report — acceptance contract
#
# Source: .skillgrid/specs/2026-10-17-quality-report/
# Trace: change briefing ## Requirements (R1–R10); blueprint tasks SATISFIES lines.
#
# Format rules (acceptance-test-authoring):
# - Steps in the ```gherkin fence are indented 6 spaces (verbatim — the extractor
#   does NOT re-indent)
# - No Gherkin tags in the spec — selection/trace is via the SATISFIES field in
#   blueprint/tasks
# - One blank line between the heading and the fence is fine
# - Gates: each requirement carries a `#### Gates` block — the runnable shadow of
#   its happy-path and failure scenarios. Author it BEFORE implementing.
#
# Scope note: the report is a VIEW over the existing findings store (ADR-0030) +
# a new coverage table (ADR-0038). It is fail-open (ADR-0016): a source that
# errors is present with status=error/unavailable, never a failed report.

## Requirements

### Requirement: Unified report aggregation (R1)

The report service SHALL aggregate all configured sources into one report on
demand: static sources always, DAST sources when `config.yaml` `dast.url` is
set. A source that failed is present with `status=error` (not absent), and the
report still emits the other sources.

#### Scenario: report-aggregates-static-sources

```gherkin
      Given a Go fixture project with a known vulnerable dependency
      When I collect the report with no dast.url configured
      Then the report contains the static sources trivy, semgrep, jscpd, debt, and coverage
      And the report contains no dynamic (DAST) section
      And the report generated_at timestamp is set
```

#### Scenario: report-source-error-is-fail-open

```gherkin
      Given a Go fixture project
      And the jscpd binary errors on this project
      When I collect the report
      Then the report still returns the trivy, semgrep, debt, and coverage sources
      And the jscpd source is present with status "error"
      And no error is propagated to the caller
```

#### Gates
# G1: full static set present, no DAST (mirrors report-aggregates-static-sources)
#   CHECK: go test ./mnemonic/internal/report/ -run TestCollect_FullStaticSet -count=1
#   EXPECT: PASS (trivy, semgrep, jscpd, debt, coverage all in rep.Static; rep.Dynamic nil)
#   EVIDENCE: pending
# G2: a failing source is fail-open (mirrors report-source-error-is-fail-open)
#   CHECK: go test ./mnemonic/internal/report/ -run TestCollect_SourceError_FailOpen -count=1
#   EXPECT: PASS (failed source status=error; other sources present; no returned error)
#   EVIDENCE: pending

### Requirement: JUnit XML output (R2)

The report SHALL emit JUnit XML consumable by CI: one `<testsuite name="<tool>">`
per source that ran, each finding a `<testcase>` whose `<failure message>` carries
the normalized severity. Zero-finding sources emit an empty testsuite, not an
omitted one.

#### Scenario: junit-xml-per-tool-testsuite

```gherkin
      Given a Go fixture project with a known CVE and one debt marker
      When I emit the report as JUnit XML
      Then the XML is well-formed and parses as JUnit
      And it contains a testsuite named "trivy"
      And it contains a testsuite named "debt"
      And the trivy testcase name contains the CVE id
      And the trivy failure message begins with the normalized severity
```

#### Scenario: junit-xml-zero-finding-source-is-empty-testsuite

```gherkin
      Given a report whose jscpd source has zero findings
      When I emit the report as JUnit XML
      Then the XML contains a testsuite named "jscpd" with tests="0" and failures="0"
      And the jscpd testsuite is not omitted
```

#### Gates
# G1: JUnit XML per-tool testsuite + CVE in name (mirrors junit-xml-per-tool-testsuite)
#   CHECK: go test ./mnemonic/internal/report/emit/ -run TestJUnit_PerToolTestSuite -count=1
#   EXPECT: PASS (xml.Unmarshal ok; trivy + debt testsuites; CVE id in a testcase name)
#   EVIDENCE: pending
# G2: zero-finding source is an empty testsuite (mirrors junit-xml-zero-finding-source-is-empty-testsuite)
#   CHECK: go test ./mnemonic/internal/report/emit/ -run TestJUnit_ZeroFindingSource_EmptyTestSuite -count=1
#   EXPECT: PASS (contains <testsuite name="jscpd" tests="0" failures="0">)
#   EVIDENCE: pending

### Requirement: JSON output (R3)

The report SHALL emit the rich JSON the dashboard consumes: a section per source
that ran (tool, status, findings with severity/title/cve_id/rule_id/file/line)
plus the coverage rows. The dashboard renders it with no client transform.

#### Scenario: json-report-shape

```gherkin
      Given a Go fixture project with findings and coverage data
      When I emit the report as JSON
      Then the JSON has a "static" array with a section per static source
      And each section has tool, status, and findings
      And each finding has severity, title, file, and line
      And the JSON has a "coverage" array of file/pct_line rows
      And the JSON has a "generated_at" field
```

#### Gates
# G1: JSON shape for the dashboard (mirrors json-report-shape)
#   CHECK: go test ./mnemonic/internal/report/emit/ -run TestJSON_ReportShape -count=1
#   EXPECT: PASS (unmarshal into ReportJSON; static[].tool/status/findings; coverage[].pct_line; generated_at)
#   EVIDENCE: pending

### Requirement: CVE via Trivy (R4)

The report SHALL include Trivy CVE findings (already normalized: cve_id,
package, fixed_version) from the findings store, multi-language.

#### Scenario: cve-trivy-report

```gherkin
      Given a project whose dependency manifest declares a vulnerable package
      When I collect the report
      Then the trivy source has status "ok"
      And the trivy findings include the known CVE id
      And the finding carries the package and fixed_version
      And the JUnit trivy testsuite contains a testcase named with the CVE id
```

#### Gates
# G1: CVE present in the report (mirrors cve-trivy-report)
#   CHECK: go test ./mnemonic/internal/report/ -run TestCollect_TrvySource_FailOpen -count=1
#   EXPECT: PASS (trivy source ok; finding carries CVE id + package + fixed_version)
#   EVIDENCE: pending

### Requirement: SAST via Semgrep (R5)

The report SHALL include Semgrep code-level rule findings (normalized: rule_id,
file/line) from the findings store. Default registry rules apply; a project's own
`.semgrep.yml` is picked up automatically if present.

#### Scenario: sast-semgrep-report

```gherkin
      Given a project source file that trips a default semgrep rule
      When I collect the report
      Then the semgrep source has status "ok"
      And the semgrep findings include the rule id
      And the finding carries the file and line
      And the JUnit semgrep testsuite contains a testcase for the rule
```

#### Gates
# G1: SAST rule finding present (mirrors sast-semgrep-report)
#   CHECK: go test ./mnemonic/internal/report/ -run TestCollect_FullStaticSet -count=1
#   EXPECT: PASS (semgrep in rep.Static with a rule_id finding)
#   EVIDENCE: pending

### Requirement: DAST via Wapiti + Nuclei + AKCA (R6)

When `config.yaml` `dast.url` is set, the report SHALL run wapiti, nuclei, and
akca against that URL and include their findings. AKCA output is parsed by a new
`ParseAkca`. When the URL is unreachable, the DAST sources are marked
`unavailable` (fail-open) and the static report still emits.

#### Scenario: dast-auto-from-config

```gherkin
      Given a config.yaml with dast.url set to a reachable test app
      When I collect the report
      Then the report contains a dynamic (DAST) section
      And the DAST section includes wapiti, nuclei, and akca sources
      And each reachable DAST source has status "ok" with findings
      And akca findings are parsed with rule id and file
```

#### Scenario: dast-unreachable-fail-open

```gherkin
      Given a config.yaml with dast.url set to an unreachable port
      When I collect the report
      Then the report still returns the static sources
      And the DAST section is present
      And the wapiti, nuclei, and akca sources each have status "unavailable"
      And the HTTP report endpoint returns 200, not an error
```

#### Scenario: dast-unset-is-absent

```gherkin
      Given a config.yaml with no dast.url
      When I collect the report
      Then the report contains no dynamic (DAST) section
```

#### Gates
# G1: DAST auto-runs all three when reachable (mirrors dast-auto-from-config)
#   CHECK: go test ./mnemonic/internal/report/ -run TestCollectDAST_Reachable_RunsThree -count=1
#   EXPECT: PASS (3 DAST sources ok; akca findings parsed)
#   EVIDENCE: pending
# G2: unreachable DAST is fail-open (mirrors dast-unreachable-fail-open)
#   CHECK: go test ./mnemonic/internal/report/ -run TestCollectDAST_Unreachable_FailOpen -count=1
#   EXPECT: PASS (3 DAST sources unavailable; static sources present)
#   EVIDENCE: pending
# G3: akca parser maps the fixture (part of dast-auto-from-config)
#   CHECK: go test ./mnemonic/internal/scan/ -run TestParseAkca -count=1
#   EXPECT: PASS (3 findings; rule id + file + normalized severity)
#   EVIDENCE: pending

### Requirement: Duplication via jscpd (R7)

The report SHALL include jscpd code-clone findings. jscpd is already installed by
`GlobalTools()`; the gap closed here is the `ParseJscpd` parser and the
`scannerCommands` arm so `scan.Start` can run it. Clones map to findings with
rule_id=duplication and the clone file/line.

#### Scenario: duplication-jscpd-report

```gherkin
      Given a project with two source files sharing a duplicated block
      When I collect the report
      Then the jscpd source has status "ok"
      And the jscpd findings include rule_id "duplication"
      And the finding carries the first clone's file and line
      And the JUnit jscpd testsuite contains a testcase for the clone
```

#### Gates
# G1: jscpd parser maps the fixture (part of duplication-jscpd-report)
#   CHECK: go test ./mnemonic/internal/scan/ -run TestParseJscpd -count=1
#   EXPECT: PASS (2 findings; rule_id=duplication; file + line; severity MEDIUM)
#   EVIDENCE: pending
# G2: jscpd is runnable via Start (part of duplication-jscpd-report)
#   CHECK: go test ./mnemonic/internal/scan/ -run TestScannerCommandsIncludesJscpd -count=1
#   EXPECT: PASS (scannerCommands has a jscpd arm)
#   EVIDENCE: pending

### Requirement: Code debt via markers (R8)

The report SHALL grep the project source for `TODO|FIXME|XXX` and record each as
a debt finding (rule_id=`debt:<marker>`, file/line). Debt is markers-only in v1;
import cycles and god-nodes are out of scope.

#### Scenario: debt-markers-report

```gherkin
      Given a project source file containing a TODO, a FIXME, and an XXX marker
      When I collect the report
      Then the debt source has status "ok"
      And the debt findings include rule_id "debt:TODO"
      And the debt findings include rule_id "debt:FIXME"
      And the debt findings include rule_id "debt:XXX"
      And each finding carries its file and line
```

#### Gates
# G1: debt markers found (mirrors debt-markers-report)
#   CHECK: go test ./mnemonic/internal/report/ -run TestCollectDebt -count=1
#   EXPECT: PASS (3 findings; rule ids debt:TODO / debt:FIXME / debt:XXX; file + line)
#   EVIDENCE: pending

### Requirement: Test coverage via detected dispatch (R9)

The report SHALL detect the project kind (reusing onboarding detection) and run
the native coverage tool, storing per-file line % (line % only) in the coverage
table. Supported kinds: Go, TS/JS, Python, Rust, Java. A kind with no coverage
tool on PATH is marked `unavailable`, not an error.

#### Scenario: coverage-dispatch-go

```gherkin
      Given a Go fixture project (go.mod present) with tests
      When I collect the report
      Then the coverage source detects kind "go"
      And the coverage source runs the native go coverage command
      And per-file line % rows are stored in the coverage table
      And the coverage JSON rows carry file and pct_line
```

#### Scenario: coverage-dispatch-ts

```gherkin
      Given a TS fixture project (package.json present) with an istanbul coverage report
      When I collect the report
      Then the coverage source detects kind "ts"
      And per-file line % rows are parsed from the istanbul JSON
      And the coverage JSON rows carry file and pct_line
```

#### Scenario: coverage-dispatch-py

```gherkin
      Given a Python fixture project (pyproject.toml present) with a coverage.py report
      When I collect the report
      Then the coverage source detects kind "py"
      And per-file line % rows are parsed from the coverage.py JSON
      And the coverage JSON rows carry file and pct_line
```

#### Scenario: coverage-dispatch-rs

```gherkin
      Given a Rust fixture project (Cargo.toml present) with a cargo-llvm-cov report
      When I collect the report
      Then the coverage source detects kind "rust"
      And per-file line % rows are parsed from the llvm-cov JSON
      And the coverage JSON rows carry file and pct_line
```

#### Scenario: coverage-dispatch-java

```gherkin
      Given a Java fixture project (pom.xml present) with a jacoco report
      When I collect the report
      Then the coverage source detects kind "java"
      And per-file line % rows are parsed from the jacoco JSON
      And the coverage JSON rows carry file and pct_line
```

#### Scenario: coverage-tool-missing-unavailable

```gherkin
      Given a Go fixture project whose coverage tool is absent from PATH
      When I collect the report
      Then the coverage source has status "unavailable"
      And the report still returns the other static sources
      And no error is propagated to the caller
```

#### Gates
# G1: kind detection (shared by all five dispatch scenarios)
#   CHECK: go test ./mnemonic/internal/report/ -run TestDetectKind -count=1
#   EXPECT: PASS (go.mod→go, package.json→ts, pyproject.toml→py, Cargo.toml→rust, pom.xml→java, else unknown)
#   EVIDENCE: pending
# G2: per-kind parse (mirrors the five coverage-dispatch-* scenarios)
#   CHECK: go test ./mnemonic/internal/report/ -run TestParseCover_AllKinds -count=1
#   EXPECT: PASS (ts/py/rs/java fixtures → correct file + pct_line rows)
#   EVIDENCE: pending
# G3: coverage upsert (shared)
#   CHECK: go test ./mnemonic/internal/report/ -run TestStoreCoverage_Upsert -count=1
#   EXPECT: PASS (upsert, not duplicate; latest pct_line wins)
#   EVIDENCE: pending
# G4: missing tool is fail-open (mirrors coverage-tool-missing-unavailable)
#   CHECK: go test ./mnemonic/internal/report/ -run TestCollect_CoverageToolMissing_Unavailable -count=1
#   EXPECT: PASS (coverage status=unavailable; other sources present; no error)
#   EVIDENCE: pending

### Requirement: Dashboard /reports page (R10)

The dashboard SHALL render a `/reports` page with six panels — CVE, SAST, DAST,
duplication, debt, coverage — fed by `GET /report/json`, reusing the shared
severity-badge component and d3. The DAST panel is conditional on `dast.url`
(absent when unset; "unavailable" when unreachable).

#### Scenario: dashboard-reports-page

```gherkin
      Given the /report/json endpoint returns a full report with static and dynamic sections
      When I open the /reports page
      Then the CVE, SAST, DAST, duplication, debt, and coverage panels are rendered
      And each panel shows its severity counts
      And the coverage panel shows a per-file line % heat bar
```

#### Scenario: dashboard-dast-conditional

```gherkin
      Given the /report/json endpoint returns a report with no dynamic section
      When I open the /reports page
      Then the DAST panel is not rendered
      And the five static panels are rendered
```

#### Scenario: dashboard-dast-unavailable

```gherkin
      Given the /report/json endpoint returns a report whose DAST section has status "unavailable"
      When I open the /reports page
      Then the DAST panel is rendered
      And the DAST panel shows an unavailable state
```

#### Gates
# G1: all six panels render (mirrors dashboard-reports-page)
#   CHECK: cd skillgrid-ui && npx vitest run src/features/reports/ReportsPage.test.tsx -t "renders all six panels"
#   EXPECT: PASS (CVE, SAST, DAST, duplication, debt, coverage panels in the DOM)
#   EVIDENCE: pending
# G2: DAST panel conditional (mirrors dashboard-dast-conditional)
#   CHECK: cd skillgrid-ui && npx vitest run src/features/reports/ReportsPage.test.tsx -t "omits the DAST panel"
#   EXPECT: PASS (DAST absent; five static panels present)
#   EVIDENCE: pending

### Requirement: Report endpoints + MCP tool parity (R1/R2/R3)

The report SHALL be served by `GET /report/junit.xml`, `GET /report/json`
(write-auth wrapped), and the `scan_report` MCP tool. The MCP tool and the HTTP
routes produce byte-identical output for the same format (same emitter, same
report).

#### Scenario: report-http-endpoints

```gherkin
      Given an authenticated session on a Go fixture project
      When I request GET /report/junit.xml
      Then the response is 200 with Content-Type application/xml
      And the body is well-formed JUnit XML
      When I request GET /report/json
      Then the response is 200 with Content-Type application/json
      And the body is the report JSON
```

#### Scenario: report-mcp-tool-registered

```gherkin
      Given the mnemonic MCP server is booted
      When I list the registered tools
      Then the scan_report tool is present
      And scan_report requires a format argument
```

#### Scenario: report-mcp-http-parity

```gherkin
      Given the same project and the same collected report
      When I emit JUnit via the scan_report MCP tool with format "junit"
      And I emit JUnit via GET /report/junit.xml
      Then both outputs are byte-identical
```

#### Gates
# G1: HTTP endpoints (mirrors report-http-endpoints)
#   CHECK: go test ./mnemonic/internal/http/ -run TestHandleReport -count=1
#   EXPECT: PASS (junit.xml → 200 application/xml well-formed; json → 200 application/json)
#   EVIDENCE: pending
# G2: MCP tool registered (mirrors report-mcp-tool-registered)
#   CHECK: go test ./mnemonic/internal/mcp/ -run TestScanReportTool_Registered -count=1
#   EXPECT: PASS (scan_report present; format required)
#   EVIDENCE: pending
# G3: MCP/HTTP parity (mirrors report-mcp-http-parity)
#   CHECK: go test ./mnemonic/internal/mcp/ -run TestScanReport_Parity_SameReportBothEmitters -count=1
#   EXPECT: PASS (same Report → emit.JUnit deterministic; MCP output == emitter output)
#   EVIDENCE: pending
