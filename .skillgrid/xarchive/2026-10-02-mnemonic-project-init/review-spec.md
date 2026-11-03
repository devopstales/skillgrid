# Spec Review — mnemonic-project-init

> Change: `.skillgrid/specs/2026-10-02-mnemonic-project-init/`
> Generated: 2026-10-03T07:48:37Z (spec-reviewer)
> Diff range: `f739d371..17652346`
> Spec sources: `acceptance.feature`, `blueprint.md`, `briefing.md`
> Independence: **A** (fresh Spec axis; no Standards/Security merge; no implementer verdict consumed)

## Scenarios

| Scenario | Result | Evidence |
|---|---|---|
| happy path init writes boot file and reports counts | PASS | `TestInitWritesBootFileAndReportsCounts`; `printInitResult` always emits indexed/ingested (test asserts boot path only) |
| init help lists force and docs flags | PASS | Production `newInitFlagSet` registers `--force`/`--docs`; `TestInitHelpListsFlags` mirrors usage text (does not call `runProjectInit -h`) |
| boot file write failure is the only fatal error | PASS | `TestInitBootFileWriteFailureIsFatal`; `runProjectInit` exits before `printInitResult` on error |
| happy path init upserts preamble and sentinel | PASS | `TestInitUpsertsPreambleAndSentinel` + `writeBootFile` |
| second init merges without duplicating the sentinel | PASS | Covered inside `TestInitForceRewritesPreamble` (force=false re-run keeps `USER KEEP`/`USER BELOW`, one sentinel) |
| force rewrites the preamble only | PASS | `TestInitForceRewritesPreamble` (force=true drops `USER KEEP`, keeps `USER BELOW`) |
| happy path init indexes the project | PASS | `TestInitIndexesTheProject` (`Indexed >= 1` on `.go` fixture) |
| init reuses the existing indexer | PASS | Code path: `initRunIndex` → `svc.ResolveProject` + `svc.RunCodeIndex` on the injected `svc`; no second `service.New` inside `projectInit` (inspection; no dedicated assertion) |
| index failure after boot file still exits 0 | PASS | `TestInitIndexFailureIsNonFatal` stubs `initRunIndex`; `err == nil`, Errors contain `index failed`, boot exists |
| happy path init ingests default paths | PASS | `TestInitIngestsDefaultPaths` → `init/docs/README.md`, `init/docs/docs/guide.md` |
| missing docs directory is skipped | PASS | `TestInitSkipsMissingDocs`; code skips missing default roots via `skipLabel` |
| second init upserts the same topic keys | PASS | Second `projectInit` in `TestInitIngestsDefaultPaths` keeps `countTopic == 1` (does not assert content update) |
| happy path init ingests extra --docs path | PASS | `TestInitIngestsExtraDocs` → `init/docs/README.sh` |
| repeated --docs flags ingest each path | PASS | `TestInitIngestsRepeatedDocsFlags` (added 17652346 after feature code) |
| missing extra docs path is listed and non-fatal | PASS | `TestInitMissingExtraDocsIsNonFatal` |
| happy path onboarding skill calls skillgrid init | PASS | `TestOnboardingSkillCallsSkillgridInit` + SKILL.md Step 9 |
| skill passes extra docs through | PASS | `TestOnboardingSkillPassesDocsThrough` (added 17652346 after skill edit) |
| skill does not contain a second ingest procedure | PASS | `TestOnboardingSkillHasNoSecondIngest` |

**Scenario coverage (behavior):** 18/18 PASS against Given/When/Then code paths.

## TDD RED→GREEN

| Task / commits | Claimed RED | Git-visible RED | Verdict |
|---|---|---|---|
| T01 `b5239413` | “undefined projectInit” | Same commit adds tests + `init_cmd.go` | **MISSING_RED** |
| T02 `9dd840f5` | “empty AGENTS” | Same commit adds tests + `init_boot.go` | **MISSING_RED** |
| T03 `03439ce7` | “Indexed=0 / undefined initRunIndex” | Same commit adds tests + index wire | **MISSING_RED** |
| T04 `8763f9b6` | “Ingested 0” | Same commit adds tests + `init_ingest.go` | **MISSING_RED** |
| T05 `2324c38e` | “skillgrid init absent” | Same commit adds SKILL.md + onboarding tests | **MISSING_RED** |
| Gap fill `17652346` | n/a | Tests only, after GREEN for repeated `--docs` / skill `--docs` | post-hoc coverage, not RED→GREEN |

QA `report.md` TDD audit cites “report RED” without SHAs — circular; not accepted under Independence A.

## Spec zone before code

- Base `f739d371` = `chore(state): claim … for execution` — **no** briefing/acceptance/blueprint/tasks in tree.
- First code commit `b5239413` lands CLI without any prior committed spec-zone files for this change.
- At HEAD `17652346`, committed under the change folder: **only** `report.md`.
- Working tree still has **untracked**: `acceptance.feature`, `briefing.md`, `blueprint.md`, `tasks.md`, `adr.md`.
- Locked constraint / briefing Context: “Spec-zone changes commit before code-zone changes” — **violated**.

## Missing or Partial

- Briefing req 1 Then “reports indexed and ingested counts”: fields are printed, but `TestInitWritesBootFileAndReportsCounts` never asserts `Indexed`/`Ingested` (or printed lines).
- Scenario “missing docs directory is skipped” also requires `init/docs/README.md` observation — code does; `TestInitSkipsMissingDocs` does not assert it.
- Scenario “second init upserts…” Then “existing observation is updated” — only duplicate-count asserted, not content refresh.
- Scenario “init reuses the existing indexer” — satisfied by code inspection only; no test that fails if a second store is opened.
- `acceptance.feature` gates still say `EVIDENCE: pending` (file never committed/updated).

## Scope Creep

- Path jail (`TestInitRejectsDocsOutsideProject` / `docs outside project`) — **in scope** (blueprint threat matrix / Global Constraints).
- CLAUDE.md pointer when AGENTS is chosen — **in scope** (briefing req 2 / blueprint Task 2).
- Reject empty ingest files (`empty content`) — small extra vs scenarios; behavior still non-fatal via Errors. Treat as Minor noise, not Important creep.
- No KBs, no new MCP tool, no `--json`, no second store — out-of-scope items correctly absent.

## Wrong Implementation

- None for SATISFIES Given/When/Then vs production code paths.
- Note (not Wrong): `TestInitHelpListsFlags` rebuilds a local FlagSet instead of exercising `runProjectInit`/`newInitFlagSet` — gate is decoupled from production help.

## Findings (Spec axis)

### Critical

1. **MISSING_RED (all T01–T05 SATISFIES scenarios)** — Every feat commit lands failing tests and implementation together; no git-visible RED SHA. Commit-message / QA “report RED” claims are not evidence. Per Spec Critical bar: a green scenario without real RED→GREEN is Critical.

### Important

1. **Spec zone after (and still without) code commits** — `acceptance.feature` / `briefing.md` / `blueprint.md` / `tasks.md` / `adr.md` were never committed before `b5239413..17652346` code; still untracked at review time. Violates locked “spec-zone before code-zone” and briefing Context.

### Minor

1. `TestInitHelpListsFlags` does not invoke production help path.
2. `TestInitWritesBootFileAndReportsCounts` omits count assertions.
3. `TestInitSkipsMissingDocs` omits README observation assertion.
4. Empty-file ingest → Errors is slightly beyond scenario text.

## Assessment

**Spec met?** Partially

**Reasoning:** All 18 acceptance scenarios have matching production behavior and green focused tests, with no Wrong Implementation of Given/When/Then. The Spec axis still fails the Critical RED-evidence bar and the Important spec-zone-before-code constraint, so the axis verdict is **not met** until those process gaps are remediated (commit/spec-zone first going forward; RED evidence cannot be invented retroactively without an accepted waiver).

### Spec axis summary

- **Worst issue:** MISSING_RED for T01–T05 (no git-visible failing test before code)
- **Findings:** 1 Critical / 1 Important / 4 Minor
- **Verdict:** not met
- **Independence:** A
