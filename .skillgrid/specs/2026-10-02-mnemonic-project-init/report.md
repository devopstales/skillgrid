# Report — mnemonic-project-init

> Change: `.skillgrid/specs/2026-10-02-mnemonic-project-init/` (moves to `.skillgrid/archive/2026-10-02-mnemonic-project-init/` at ship)
> Generated: 2026-10-03T07:40:00Z (qa)
> Gate: PASS
>
> Two phases, one file: **qa** writes the QA half (the sections above `## Final-State Facts`,
> through `## Gate Decision` + `## Human Override`) into the spec folder. **ship** reads the
> `## Gate Decision` verdict PRE-MOVE, then moves the folder. **reflect** completes the retro
> half (from `## Final-State Facts` onward) IN PLACE in the archive folder.

**Tier:** T2 · **Floor:** L2 (classification standard) · verification scope: COMPLETE (focused `TestInit*|TestOnboardingSkill*` + `go build ./cmd/skillgrid` + `skillgrid init -h` smoke)

## Test Plan

### Risk Ranking

Most likely production break: boot-file write succeeds but ingest/index silently no-ops, leaving a “initialized” repo with empty memory/index. Second: sentinel duplication on re-init polluting every prompt. Tests hit those first.

### Plan

| # | Risk / Behavior | Seam | Layer | Test / Scenario | Priority | Cadence | Status |
|---|-----------------|------|-------|-----------------|----------|---------|--------|
| 1 | Boot file written + counts | CLI `projectInit` | unit | `TestInitWritesBootFileAndReportsCounts` | P0 | pr | covered |
| 2 | Help lists `--force`/`--docs` | CLI flags | unit | `TestInitHelpListsFlags` | P1 | pr | covered |
| 3 | Non-dir boot failure fatal | CLI `projectInit` | unit | `TestInitBootFileWriteFailureIsFatal` | P0 | pr | covered |
| 4 | Preamble + single sentinel | `writeBootFile` | unit | `TestInitUpsertsPreambleAndSentinel` | P0 | pr | covered |
| 5 | `--force` preamble only | `writeBootFile` | unit | `TestInitForceRewritesPreamble` | P0 | pr | covered |
| 6 | Index ≥1 on `.go` fixture | `RunCodeIndex` | unit | `TestInitIndexesTheProject` | P0 | pr | covered |
| 7 | Index failure non-fatal | `initRunIndex` stub | unit | `TestInitIndexFailureIsNonFatal` | P0 | pr | covered |
| 8 | Default path ingest | `ingestPaths` | unit | `TestInitIngestsDefaultPaths` | P0 | pr | covered |
| 9 | Missing `docs/` skipped | `ingestPaths` | unit | `TestInitSkipsMissingDocs` | P1 | pr | covered |
| 10 | Extra `--docs` ingest | `ingestPaths` | unit | `TestInitIngestsExtraDocs` | P0 | pr | covered |
| 11 | Repeated `--docs` | `ingestPaths` | unit | `TestInitIngestsRepeatedDocsFlags` | P1 | pr | covered |
| 12 | Missing extra non-fatal | `ingestPaths` | unit | `TestInitMissingExtraDocsIsNonFatal` | P0 | pr | covered |
| 13 | Docs jail | `ingestPaths` | unit | `TestInitRejectsDocsOutsideProject` | P0 | pr | covered |
| 14 | Onboarding names init | SKILL.md | unit | `TestOnboardingSkillCallsSkillgridInit` | P0 | pr | covered |
| 15 | Skill passes `--docs` | SKILL.md | unit | `TestOnboardingSkillPassesDocsThrough` | P1 | pr | covered |
| 16 | No second ingest in skill | SKILL.md | unit | `TestOnboardingSkillHasNoSecondIngest` | P0 | pr | covered |

### Edge-Case Matrix

| Requirement | Edge / Boundary | Expected Behavior | Test / Scenario | Status |
|-------------|-----------------|-------------------|-----------------|--------|
| Boot file | target path is a file | error, non-zero | `TestInitBootFileWriteFailureIsFatal` | covered |
| Index | pipeline fails after boot | exit 0, Errors listed | `TestInitIndexFailureIsNonFatal` | covered |
| Default ingest | no `docs/` | Skipped, still exit 0 | `TestInitSkipsMissingDocs` | covered |
| Extra docs | path outside project | Errors, non-fatal | `TestInitRejectsDocsOutsideProject` | covered |
| Extra docs | missing path | Errors, boot exists | `TestInitMissingExtraDocsIsNonFatal` | covered |

### Out of Scope

- Live IDE onboarding interview (skill prose only; CLI contract tested)
- Shared LLM provider / KBs (explicitly deferred)
- `--json` result shape (not this slice)

## Goal-Backward Verification

**Stated goal:** Run project init and get a lean boot file, forced code index, and local docs in the existing store.

| Level | Item | Evidence | Status |
|-------|------|----------|--------|
| Truth | `skillgrid init` sequences boot + index + ingest | focused suite PASS + `init -h` | VERIFIED |
| Truth | Single sentinel; `--force` preamble only | Upsert + Force tests | VERIFIED |
| Truth | Index non-empty on Go fixture; index fail non-fatal | Indexes + IndexFailure tests | VERIFIED |
| Truth | Default/extra ingest with stable topic keys | Ingest* + Skips + Rejects tests | VERIFIED |
| Truth | Onboarding finishes via CLI, no second ingest | OnboardingSkill* tests | VERIFIED |
| Artifact | `init_cmd.go`, `init_boot.go`, `init_ingest.go`, main dispatch | present + wired | VERIFIED |
| Key Link | `projectInit` → writeBootFile → initRunIndex → ingestPaths | integration via TestInit* | VERIFIED |
| Data Flow | files → `memory.Save` topic `init/docs/<relpath>` | IngestsDefault/Extra/Repeated | VERIFIED |

## Traceability Matrix

| Scenario (from acceptance.feature) | Test / Scenario | Ran | Result |
|--------------------------------------|-----------------|-----|--------|
| happy path init writes boot file and reports counts | `TestInitWritesBootFileAndReportsCounts` | yes | pass |
| init help lists force and docs flags | `TestInitHelpListsFlags` | yes | pass |
| boot file write failure is the only fatal error | `TestInitBootFileWriteFailureIsFatal` | yes | pass |
| happy path init upserts preamble and sentinel | `TestInitUpsertsPreambleAndSentinel` | yes | pass |
| second init merges without duplicating the sentinel | `TestInitUpsertsPreambleAndSentinel` (second call) | yes | pass |
| force rewrites the preamble only | `TestInitForceRewritesPreamble` | yes | pass |
| happy path init indexes the project | `TestInitIndexesTheProject` | yes | pass |
| init reuses the existing indexer | `TestInitIndexesTheProject` (`svc.RunCodeIndex`) | yes | pass |
| index failure after boot file still exits 0 | `TestInitIndexFailureIsNonFatal` | yes | pass |
| happy path init ingests default paths | `TestInitIngestsDefaultPaths` | yes | pass |
| missing docs directory is skipped | `TestInitSkipsMissingDocs` | yes | pass |
| second init upserts the same topic keys | `TestInitIngestsDefaultPaths` (re-run count) | yes | pass |
| happy path init ingests extra --docs path | `TestInitIngestsExtraDocs` | yes | pass |
| repeated --docs flags ingest each path | `TestInitIngestsRepeatedDocsFlags` | yes | pass |
| missing extra docs path is listed and non-fatal | `TestInitMissingExtraDocsIsNonFatal` | yes | pass |
| happy path onboarding skill calls skillgrid init | `TestOnboardingSkillCallsSkillgridInit` | yes | pass |
| skill passes extra docs through | `TestOnboardingSkillPassesDocsThrough` | yes | pass |
| skill does not contain a second ingest procedure | `TestOnboardingSkillHasNoSecondIngest` | yes | pass |

**Coverage:** 18/18 scenarios covered by a test that ran and passed.

## Verification-Gap Audit

| Gap | Location | Shape | Evidence | Smallest Regression |
|-----|----------|-------|----------|---------------------|
| _(none open)_ | | | | |

QA added `TestInitIngestsRepeatedDocsFlags` and `TestOnboardingSkillPassesDocsThrough` before the gate to clear two acceptance scenarios that had no dedicated test.

## TDD Evidence Audit

| Task | SATISFIES Scenario | RED Evidence | GREEN Evidence | Verdict |
|------|--------------------|--------------|----------------|---------|
| T01 | boot/help/fatal | report: undefined `projectInit` | b5239413 | OK |
| T02 | preamble/sentinel/force | report RED | 9dd840f5 | OK |
| T03 | index + non-fatal | report RED | 03439ce7 | OK |
| T04 | ingest defaults/extras | report RED ingest | 8763f9b6 | OK |
| T05 | onboarding init | RED skillgrid init absent | 2324c38e | OK |

## Security Audit

No new network/auth surface. Path jail for `--docs` covered by `TestInitRejectsDocsOutsideProject`. Trivy advisory-only (config). No CRITICAL.

## Code-Quality Gates

| Gate | Result |
|------|--------|
| Typecheck / build | PASS — `go build ./cmd/skillgrid` |
| Focused suite | PASS — `go test ./cmd/skillgrid -run 'TestInit\|TestOnboardingSkill'` |
| Coverage / mutation / lint | N/A (thresholds 0 / unset) |
| P0 / P1 pass rate | 100% of planned rows covered |

## Floor

Weakest dimension: skill contract tests are source-text assertions (acceptable for SKILL.md handoff). Does not drop below PASS.

## Gate Decision

**PASS**

**Reasoning:** All acceptance scenarios mapped to green focused tests; goal-backward truths VERIFIED; no CRITICAL/MISSING_RED; ADR-0012 (one store) preserved. Smoke: `skillgrid init -h` names both flags.

## Human Override

_(none)_

## Final-State Facts

<!-- reflect fills below -->
