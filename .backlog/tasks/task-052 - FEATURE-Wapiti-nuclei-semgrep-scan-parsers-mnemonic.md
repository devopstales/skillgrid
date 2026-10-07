---
id: TASK-052
title: '[FEATURE] Wapiti + nuclei + semgrep scan parsers (mnemonic)'
status: done
assignee: []
created_date: '2026-10-07 11:30'
updated_date: '2026-10-07 13:33'
labels: []
milestone: m-7
dependencies:
  - TASK-051
references:
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/briefing.md
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/blueprint.md
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/tasks.md
documentation:
  - .skillgrid/specs/2026-10-05-mnemonic-scan-findings/briefing.md
  - .skillgrid/artifacts/04-adr-0030-scan-findings-structured-store.md
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Current State: only the trivy parser arm exists (TASK-051).\n\nExpected State: parse.go gains ParseWapiti (report[] type/info/url), ParseNuclei (JSONL template-id, info.severity, matcher-name, host), ParseSemgrep (results[] check_id->RuleID, extra.severity, path, start.line, extra.message); Parse(tool,data) dispatch over all four; fixtures wapiti.json/nuclei.jsonl/semgrep.json; Start routes to the right scanner command/args per tool.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go test ./internal/mnemonic/scan/ -run TestParseDispatchAllTools passes: trivy=1, wapiti=2, nuclei=3, semgrep=4 findings with correct field mapping
- [ ] #2 go build ./... clean in the mnemonic module
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 All parsers pure (bytes -> []Finding) and fixture-driven, no live scanner binaries in unit tests
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Follow blueprint Task 3 (tasks.md TICKET-03); work from mnemonic/ module
2. TDD: extend parse_test.go with TestParseDispatchAllTools (1/2/3/4) RED first; author fixtures wapiti.json, nuclei.jsonl, semgrep.json
3. Implement ParseWapiti/ParseNuclei/ParseSemgrep in parse.go + Parse(tool,data) dispatch; Start routes scanner command/args per tool
4. go test ./internal/mnemonic/scan/ -v + go build ./... -> green
5. Commit: feat(mnemonic): wapiti/nuclei/semgrep scan parsers (fixture-driven) + Refs: TASK-052
<!-- SECTION:PLAN:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
Remaining scanner parsers (wapiti, nuclei, semgrep) + 4-way dispatch.

## What
- `parse.go`: `ParseWapiti` (report[] type/info/url), `ParseNuclei` (JSONL template-id/info.severity/matcher-name/host), `ParseSemgrep` (results[] check_id->rule/extra.severity/path/start.line/extra.message); `Parse(tool,data)` dispatch over all four tools.
- `service.go`: `scannerCommands` map += wapiti/nuclei/semgrep (Start routes per tool; target appended last).
- Fixtures wapiti.json (2), nuclei.jsonl (3), semgrep.json (4) + tests.

## Why
TASK-052 — thicken the scan tracer from trivy-only to all four scanners. SATISFIES `happy path trivy scan ingests findings with stable hash` (extended to all four tools).

## Where
- mnemonic/internal/scan/{parse.go,parse_test.go,service.go} + fixtures/{wapiti.json,nuclei.jsonl,semgrep.json}

## Verified
- `go test ./internal/scan/ -v -count=1` → 12/12 PASS (TestParseDispatchAllTools 1/2/3/4 + per-parser field-mapping + nuclei bad-line + all pre-existing T051 tests).
- `go build ./...` + `go vet ./internal/scan/` clean.

## Key Learnings
- Finding struct fields: Tool, RuleID, Severity, Title, CVEID, Package, Version, FixedVersion, File, Line, Message, Links. Parsers store RAW severity (the trivy arm doesn't call NormalizeSeverity either — StoreFindings normalizes after DedupHash, and the hash excludes severity so a finding survives reclassification). The brief's "ERROR->HIGH" is NormalizeSeverity behavior, not a parser duty.
- scannerCommands map: `full := append(append([]string{}, args...), target)` — target appended last. wapiti needs the URL before `-o json`, so it uses a `-` placeholder (`wapiti --url - -o json <target>`); real URL substitution is a Start-side concern for TICKET-04.
- nuclei JSONL: one object per line, blank lines tolerated via dec.More()+io.EOF break. semgrep start.line is 1-based, passed through.
<!-- SECTION:FINAL_SUMMARY:END -->
