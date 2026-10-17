# Unified quality report: report service over the findings store

---
status: "accepted"
supersedes: none
date: 2026-10-17
---

**Deciders:** paladm
**Related:** ADR-0030 (scan findings store), ADR-0016 (fail-open), ADR-0031 (Go 1.22+ floor)

## Context

A project created with skillgrid can be Go, TS, Python, Rust, or Java, but the only quality signal surfaced today is a live Trivy CVE count on the SecurityPage. An operator who wants "is this project healthy?" must run up to seven scanners (trivy, semgrep, wapiti, nuclei, akca, jscpd, coverage) across up to five languages by hand and eyeball the outputs. The scan pipeline (ADR-0030) already normalizes trivy/semgrep/wapiti/nuclei into the `findings` store; jscpd and akca are missing parsers, coverage has no integration, and there is no report output or dashboard view of the aggregate.

## Decision

**A `report` service orchestrates the existing scan pipeline** (reusing `scan_start`/`scan_store_findings`/`findings`) and adds the missing sources, then emits one report in two formats and one dashboard page.

- **Sources, by kind:**
  - *Static (always, any project):* trivy (CVE), semgrep (SAST), jscpd (duplication), grep `TODO|FIXME|XXX` (debt markers), coverage (detected dispatch).
  - *Dynamic (when `config.yaml` `dast.url` is set):* wapiti, nuclei, akca.
- **Two new parsers:** `ParseJscpd` and `ParseAkca` (mirror `ParseSemgrep`/`ParseNuclei`), so all seven sources land in the existing `findings` table. jscpd is added to `SecurityTools()` (installed on downstream machines).
- **Coverage = detected dispatch, line % only, five languages:** detect project kind (reusing onboarding detection) and run the native tool — Go `go test -coverprofile` + `go tool cover -json`; TS/JS `nyc report --reporter=json`; Python `pytest --cov --cov-report=json`; Rust `cargo-llvm-cov --json`; Java `jacoco` → `jacoco.json`. Per-file line % stored in a new `coverage` table (migration 051), **not** as findings (coverage is a per-file metric, not a severity finding).
- **Code debt = markers only (v1):** grep `TODO|FIXME|XXX` (universal). Import cycles + god-nodes are deferred — the code indexer is Go-only; they join when the indexer is multi-language.
- **DAST = auto + fail-open:** when `dast.url` is set, the report runs the DAST tools against it. An unreachable URL marks DAST `unavailable` (fail-open, ADR-0016); the static report still emits. No `dast.url` → DAST sources absent.
- **Two emit formats:** JUnit XML (one `<testsuite>` per source; each finding a `<testcase>` with `<failure message="<severity>: <title>">`) for CI, and JSON (shaped for the dashboard) for the page. No custom HTML generator.
- **Dashboard:** a new `/reports` page with six panels (CVE, SAST, DAST, duplication, debt, coverage), fed by the JSON, reusing the SecurityPage severity badges and the existing d3 dependency. DAST panel conditional on `dast.url`.

### Why not the alternatives

- **A new report pipeline + store:** duplicates the `findings` model and the scan pipeline ADR-0030 just locked. The report is a *view* over existing findings + a new `coverage` table, not a parallel system. Rejected.
- **Per-tool HTML reports (each scanner's own output):** no unified view, no CI format, no dashboard. The JUnit + JSON + page triple covers report + visualize without a new generator. Rejected.
- **Super-Linter / MegaLinter as the report source:** they emit lint feedback, not normalized findings; they are linter *orchestrators* (a CI concern), not findings *emitters* (a scan concern); and they add a container dependency to a single-binary product. The per-language linter belongs in the *project's* own config, not skillgrid's scanner set. Rejected.
- **Branch/function coverage:** the five native tools emit it, but line % is what a dashboard surfaces and what a threshold gates on. Storing all three triples the schema and parser work for a metric the report doesn't display. Line % only; revisit if a project asks for branch coverage.

## Consequences

- **Positive:** one command → one durable report for any project, any language; CI gets a native JUnit file; the dashboard shows six capabilities from one JSON; the findings store stays the single source of truth (ADR-0030).
- **Negative / cost:** a new `report` package + migration (051) + two parsers + a dashboard page; coverage dispatch must track five native tools' output shapes (fixture-driven, so no live-tool coupling in tests).
- **No new Go dependencies:** jscpd/akca/coverage tools are external CLIs invoked via `exec` (installed on the downstream machine), not Go imports; JUnit XML is built with `encoding/xml`. jscpd becoming an installed tool is recorded here (the "no new dependencies without an ADR" rule, AGENTS.md § Rules).
- **Fail-open surface grows:** each new source can fail; the report must emit even when a source errors (per ADR-0016). The threat matrix covers the subprocess + DAST-URL boundaries.
- **Deferred:** import cycles/god-nodes (Go-only indexer), per-project custom semgrep rule seeding, branch/function coverage, non-Go debt structure.
