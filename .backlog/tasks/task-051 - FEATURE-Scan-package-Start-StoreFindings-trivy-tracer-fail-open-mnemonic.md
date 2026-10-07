---
id: TASK-051
title: >-
  [FEATURE] Scan package: Start + StoreFindings trivy tracer, fail-open
  (mnemonic)
status: done
assignee: []
created_date: '2026-10-07 11:30'
updated_date: '2026-10-07 13:16'
labels: []
milestone: m-7
dependencies:
  - TASK-050
references:
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/briefing.md
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/blueprint.md
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/tasks.md
documentation:
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/briefing.md
  - .skillgrid/artifacts/04-adr-0030-scan-findings-structured-store.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Current State: scan package does not exist; trivy runs ad-hoc and results evaporate.\n\nExpected State: scan package with New(s *store.Store, dataDir string) *Service; Start (UUIDv7, stubbable scanner runner, raw JSON to <dataDir>/.skillgrid/cache/scans/<scanID>.json, scans row; error -> status=error fail-open); StoreFindings (raw -> Parse trivy -> NormalizeSeverity -> DedupHash -> upsert findings UNIQUE(scan_id,dedup_hash) -> update scans.finding_count/status); severity.go map; raw.go WriteRaw/ReadRaw; trivy arm of parse.go; fixtures/trivy.json with one known CVE.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go test ./internal/mnemonic/scan/ passes: severity map table (8 cases incl. trivy UNKNOWN->INFO, semgrep ERROR->HIGH/WARNING->MEDIUM/NOTE->INFO), raw roundtrip path contains .skillgrid/cache/scans/<id>.json
- [ ] #2 trivy ingest: finding_count=1 and dedup_hash == DedupHash("trivy","CVE-2024-1234","","flask","2.0.1","requirements.txt",0)
- [ ] #3 failing-stub Start returns no error (fail-open) and a scans row exists with status='error' and non-empty error text
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 Raw artifact written 0644 under <dataDir>/.skillgrid/cache/scans/ (one-way door #3 path contract)
- [ ] #7 Scanner failure never returns an error to the caller (ADR-0016 floors)
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Follow blueprint Task 2 (tasks.md TICKET-02); work from mnemonic/ module
2. TDD: severity_test.go, raw_test.go, service_test.go (TestStartTrivyIngestsStableHash, TestStartScannerErrorIsFailOpen) RED first
3. Implement severity.go (map), raw.go (WriteRaw/ReadRaw 0644 under <dataDir>/.skillgrid/cache/scans/), parse.go trivy arm, fixtures/trivy.json (one known CVE), service.go (Start with stubbable runner + UUIDv7 + fail-open; StoreFindings: parse -> NormalizeSeverity -> DedupHash -> upsert -> update scans)
4. go test ./internal/mnemonic/scan/ -v + go build ./... -> green
5. Commit: feat(mnemonic): scan package — start, store findings, severity map, raw store + Refs: TASK-051
<!-- SECTION:PLAN:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
scan package: Start + StoreFindings trivy tracer, fail-open (ADR-0016).

## What
- `service.go`: `New(s *store.Store, dataDir) *Service`; `Start` (UUIDv7 id, stubbable `Run` runner field, writes raw JSON to `<dataDir>/.skillgrid/cache/scans/<id>.json` 0644, inserts `scans` row; scanner error → status='error' + error text, returns the row — fail-open, never an error); `StoreFindings` (read raw → Parse → NormalizeSeverity → DedupHash → upsert findings on (scan_id, dedup_hash) → update scans.finding_count/status).
- `raw.go`: WriteRaw/ReadRaw. `parse.go`: trivy arm of `Parse` (Results[].Vulnerabilities[]). `severity.go`: `NormalizeSeverity` map (trivy identity + UNKNOWN→INFO; semgrep ERROR→HIGH, WARNING→MEDIUM, NOTE→INFO; wapiti/nuclei capitalize).
- `fixtures/trivy.json` (one known CVE) + tests.

## Why
TASK-051 — tracer thread for the scan package; the trivy end-to-end path lands before other parsers thicken it. SATISFIES `happy path trivy scan ingests findings with stable hash` + `error path scanner failure records status=error and is fail-open`.

## Where
- mnemonic/internal/scan/{service.go, parse.go, severity.go, raw.go} + *_test.go + fixtures/trivy.json

## Verified
- `go test ./internal/scan/ -v -count=1` → 7/7 PASS (severity table 8 cases, raw roundtrip, trivy ingest finding_count=1 + correct dedup_hash, fail-open stub, parse trivy, parse unknown, unknown scan id).
- `go build ./...` + `go vet ./internal/scan/` clean.
- DedupHash excludes severity (the 050 DDL comment says the hash material is tool/cve|rule_id/pkg/version/file/line, not severity).

## Key Learnings
- Store handle: `store.Store.DB` is a public `*sql.DB`; idiom is `s.store.DB.ExecContext/QueryRowContext` with prepared statements.
- UUIDv7: `github.com/google/uuid` v1.6.0 already a dep → `uuid.NewV7()`, no new dep (ADR-gate safe).
- Fail-open seams: Start records status='error' + returns the row (only a failed scans-row write errors); StoreFindings on unparseable raw marks the row 'partial' + returns the error.
<!-- SECTION:FINAL_SUMMARY:END -->
