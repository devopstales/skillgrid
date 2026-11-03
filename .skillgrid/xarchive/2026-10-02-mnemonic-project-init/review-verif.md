# Verification-gap review — mnemonic-project-init

**Range:** `f739d371..17652346`
**Lens:** verification-gap (Grade A)
**Inputs:** `review-range.diff`, `acceptance.feature`, `report.md`, `init_cmd_test.go`, `init_cmd.go`, `init_boot.go`, `init_ingest.go`, `main.go`

## Summary

Strong nets exist for boot upsert/force, real ingest topic keys, docs jail, and index-failure-via-stub. Several acceptance-mapped tests are **broken-verification** (do not exercise production, or assert a proxy that survives the named regression). Onboarding coverage is **source-text-only** (QA already notes this as the floor).

## Findings

### Help test never calls production flag/help code
- Changed surface: `newInitFlagSet` + help path — `skillgrid-cli/cmd/skillgrid/init_cmd.go:23-41`
- Impacted consumer or site: `runProjectInit` `-h` / `--help` branch — `init_cmd.go:36-41`; operator running `skillgrid init -h`
- Existing test evidence: `TestInitHelpListsFlags` (`init_cmd_test.go:171-186`) builds a **local** `flag.FlagSet`, hardcodes the usage string, and registers `--docs` as `fs.String` (not `fs.Var` / `stringSliceFlag`). It never calls `newInitFlagSet` or `runProjectInit`. Repo search: `newInitFlagSet` / `runProjectInit` appear only in `init_cmd.go` + `main.go` dispatch — **zero test references**.
- Missing verification: assertion that **production** usage (from `newInitFlagSet`) names `--force` and `--docs`
- Demonstration: delete `out.Var(docs, "docs", …)` from `newInitFlagSet` (or rename the flag). `TestInitHelpListsFlags` still passes; focused suite stays green.
- Consequence: CLI help/flags can drift from what QA mapped as covered; operators lose discoverability of `--docs`/`--force` with no failing named test.
- Disposition: patch (`TestInitHelpListsFlags` → call `newInitFlagSet` + `Usage()`, or capture `runProjectInit([]string{"-h"})` stderr)
- gap_type: broken-verification

### Happy-path “reports counts” does not observe ingested / printed counts
- Changed surface: `initResult.Indexed` / `Ingested` and `printInitResult` — `init_cmd.go:13-21`, `72-88`, `134-137`
- Impacted consumer or site: acceptance scenario “happy path init writes boot file and reports counts” (`acceptance.feature` ~L12–20); stdout contract used by operators
- Existing test evidence: `TestInitWritesBootFileAndReportsCounts` (`init_cmd_test.go:188-203`) asserts only `res.BootFile != ""` + file exists. **No** `res.Ingested` / `res.Indexed` assert in that test. Grep of `init_cmd_test.go`: no `Ingested`, no `printInitResult`. Ingest tests assert `countTopic`, not the result field.
- Missing verification: `res.Ingested` (and printed `indexed:` / `ingested:` lines) on a fixture with README
- Demonstration: leave `res.Ingested = 0` after successful `ingestPaths` saves (drop the assignment). `TestInitIngestsDefaultPaths` still passes via `countTopic`; `TestInitWritesBootFileAndReportsCounts` still passes. Separately, gut `printInitResult` to omit count lines — no unit test fails.
- Consequence: “initialized” result can under-report ingest while memory looks fine in deeper tests; print regressions ship silently.
- Disposition: patch (extend `TestInitWritesBootFileAndReportsCounts` to assert `Ingested`/`Indexed` fields; optional tiny `printInitResult` golden)
- gap_type: broken-verification

### Missing-docs skip scenario does not assert README still ingested
- Changed surface: default ingest + skip path — `init_ingest.go:43-53`, `ingestPaths`
- Impacted consumer or site: acceptance “missing docs directory is skipped” (`acceptance.feature` ~L147–154) requiring `init/docs/README.md` **and** `docs/` under skipped
- Existing test evidence: `TestInitSkipsMissingDocs` (`init_cmd_test.go:80-97`) only checks `res.Skipped` contains `docs`/`docs/`. Does **not** open memory or `countTopic` for README. `TestInitIngestsDefaultPaths` always creates `docs/`, so it cannot catch “docs missing ⇒ README also skipped.”
- Missing verification: observation `init/docs/README.md` exists when `docs/` is absent
- Demonstration: when `docs` is missing, also skip/`continue` past `README.md` (or clear all defaults). `TestInitSkipsMissingDocs` still green; default-path test still has `docs/` and stays green.
- Consequence: partial init with empty memory for the one default file that exists — matches QA’s top risk (“boot ok, ingest silently no-ops”) on the skip branch.
- Disposition: patch (add `countTopic(..., "init/docs/README.md") == 1` to `TestInitSkipsMissingDocs`)
- gap_type: broken-verification

### Index happy path asserts reported count, not index contents / shared store
- Changed surface: `initRunIndex` → `svc.RunCodeIndex` — `init_cmd.go:92-98`, `120-125`
- Impacted consumer or site: acceptance “happy path init indexes the project” (“project index lists at least one file”) and “init reuses the existing indexer” / “does not open a second store” (`acceptance.feature` ~L96–111)
- Existing test evidence: `TestInitIndexesTheProject` (`init_cmd_test.go:270-284`) asserts only `res.Indexed >= 1`. It does not query the code-index store for files. It does not assert `RunCodeIndex` was used vs a stubbed count, nor that indexing and ingest share one `svc`/data dir. QA traceability maps “reuses indexer” to this test — that is **diff/code inference**, not an assertion.
- Missing verification: held-out check that the project index actually lists ≥1 file; assertion that init does not `service.New` a second store for indexing
- Demonstration: change production `initRunIndex` body to `return 1, nil` (no `RunCodeIndex`). `TestInitIndexesTheProject` still passes; index store empty. Separately, `service.New` inside indexing still yields `Indexed >= 1` with no failing test for “second store.”
- Consequence: QA’s #1 risk (boot ok, index silently empty) is only guarded if the orchestrator forgets to set `Indexed`; a fake/nonzero report or alternate store slips through. “Same pipeline / one store” is unverified by tests.
- Disposition: patch (open index after init and assert file/chunk count ≥1); defer or separate micro-test for “no second `service.New`” if instrumentation is heavy
- gap_type: broken-verification

### Repeatable `--docs` CLI parsing seam has no named test
- Changed surface: `fs.Var(docs, "docs", …)` + `stringSliceFlag` — `init_cmd.go:26-27`, `64`
- Impacted consumer or site: acceptance “repeated --docs flags ingest each path” (`acceptance.feature` ~L188–194); `runProjectInit` parse path
- Existing test evidence: `TestInitIngestsRepeatedDocsFlags` (`init_cmd_test.go:118-143`) calls `projectInit(..., []string{a,b})` — **bypasses flag parsing**. Combined with the fake help test above, production `--docs` registration is never executed in the focused suite.
- Missing verification: `newInitFlagSet` + `Parse([]string{"--docs", a, "--docs", b})` yields both paths into `projectInit`
- Demonstration: change `docs` registration to a single `String` flag (last-wins). Slice-based `TestInitIngestsRepeatedDocsFlags` still passes; real `skillgrid init --docs a.md --docs b.md` drops `a.md` with no failing named test.
- Consequence: multi-`--docs` CLI contract claimed covered while only the ingest loop is covered.
- Disposition: patch (table test: parse via `newInitFlagSet` then ingest, or exec `runProjectInit` with args)
- gap_type: regression

### `main` dispatch for `init` is outside the focused suite
- Changed surface: `main.go:150-152` `case "init": runProjectInit(...)`
- Impacted consumer or site: binary entrypoint; onboarding skill’s `skillgrid init` invocation
- Existing test evidence: all `TestInit*` call `projectInit` directly. Search: no test references `case "init"` / `runProjectInit`. QA smoke `skillgrid init -h` is manual/out-of-suite, not a named automated test in the focused gate.
- Missing verification: deleting or misrouting the `init` case fails a named test
- Demonstration: remove `case "init"` from `main.go`. `go test … -run 'TestInit|TestOnboardingSkill'` still passes; only a live binary smoke would notice.
- Consequence: command disappears from the binary while the library and skill prose still look correct.
- Disposition: patch (small `TestMainInitDispatch` / exec harness) | defer if T2 explicitly accepts library-only coverage (document in report)
- gap_type: regression

### Onboarding skill nets are source-text-only and weakly phrased
- Changed surface: onboarding Step 9 — `.agents/skills/lifecycle/onboarding/SKILL.md:233-237`
- Impacted consumer or site: acceptance skill scenarios (`acceptance.feature` ~L220–242); agent readers following onboarding
- Existing test evidence: `TestOnboardingSkillCallsSkillgridInit` / `PassesDocsThrough` / `HasNoSecondIngest` (`init_cmd_test.go:315-343`) only `os.ReadFile` + `bytes.Contains` / exact banned phrase `"walk docs/ and mem_save each file"`. No runtime skill execution (out of scope per report). QA Floor already calls these source-text assertions.
- Missing verification: imperative “run `skillgrid init`” in the finish step; `--docs` tied to that step; ban broader second-ingest procedures (any `mem_save`+docs walk), not one frozen phrase
- Demonstration: keep the substring `skillgrid init` in a “do not run …” note while removing Step 9’s run instruction — Calls test still passes. Replace the banned sentence with “walk each docs file and call mem_save” — NoSecondIngest still passes. Stray `--docs` anywhere satisfies PassesDocsThrough.
- Consequence: skill handoff can regress in wording agents follow while named P0/P1 tests stay green.
- Disposition: patch (anchor asserts to Step 9 block; require “Run \`skillgrid init\`” + “`--docs`”; broaden forbidden ingest patterns) | defer live IDE onboarding (explicitly out of scope)
- gap_type: broken-verification

## Other findings

- `TestInitIndexFailureIsNonFatal` stubs `initRunIndex` **intentionally** to inject failure — not a mock-only happy path; do not treat as a gap.
- Ingest happy paths that open the store and `countTopic` are solid regression nets for topic-key upsert / jail / missing-extra.
- Boot preamble/sentinel/`--force` merge tests observe file contents — solid.

## JSON

```json
[
  {
    "title": "Help test never calls production flag/help code",
    "location": "skillgrid-cli/cmd/skillgrid/init_cmd_test.go:171",
    "changed_surface": "init_cmd.go:23-41 newInitFlagSet / -h path",
    "impacted_consumer": "runProjectInit help; skillgrid init -h",
    "existing_test_evidence": "TestInitHelpListsFlags rebuilds FlagSet+hardcoded usage; never calls newInitFlagSet",
    "missing_verification": "production Usage names --force and --docs",
    "demonstration": "remove docs Var from newInitFlagSet; TestInitHelpListsFlags still passes",
    "consequence": "CLI help/flags drift with green focused suite",
    "disposition": "patch: TestInitHelpListsFlags via newInitFlagSet/Usage",
    "gap_type": "broken-verification",
    "flags": ["source-text-adjacent", "missing-regression-net"]
  },
  {
    "title": "Happy-path reports-counts does not observe ingested/printed counts",
    "location": "skillgrid-cli/cmd/skillgrid/init_cmd_test.go:188",
    "changed_surface": "init_cmd.go:13-21,72-88,134-137 initResult + printInitResult",
    "impacted_consumer": "acceptance happy path reports indexed/ingested counts",
    "existing_test_evidence": "TestInitWritesBootFileAndReportsCounts only checks BootFile; no Ingested/printInitResult asserts in suite",
    "missing_verification": "res.Ingested and printed indexed/ingested lines",
    "demonstration": "drop Ingested assignment after saves; topic tests + WritesBootFile still pass",
    "consequence": "undercounted init result / silent print regressions",
    "disposition": "patch: assert Ingested/Indexed on happy path; optional print golden",
    "gap_type": "broken-verification",
    "flags": ["missing-regression-net"]
  },
  {
    "title": "Missing-docs skip scenario does not assert README still ingested",
    "location": "skillgrid-cli/cmd/skillgrid/init_cmd_test.go:80",
    "changed_surface": "init_ingest.go:43-53 default root skip",
    "impacted_consumer": "acceptance missing docs directory is skipped",
    "existing_test_evidence": "TestInitSkipsMissingDocs only checks Skipped; IngestsDefaultPaths always creates docs/",
    "missing_verification": "countTopic init/docs/README.md == 1 when docs/ absent",
    "demonstration": "also skip README when docs missing; SkipsMissingDocs stays green",
    "consequence": "boot ok with empty memory on skip branch",
    "disposition": "patch: add README countTopic to TestInitSkipsMissingDocs",
    "gap_type": "broken-verification",
    "flags": ["missing-regression-net"]
  },
  {
    "title": "Index happy path asserts reported count not index contents/shared store",
    "location": "skillgrid-cli/cmd/skillgrid/init_cmd_test.go:270",
    "changed_surface": "init_cmd.go:92-98,120-125 initRunIndex",
    "impacted_consumer": "acceptance indexes project + reuses existing indexer / one store",
    "existing_test_evidence": "TestInitIndexesTheProject only res.Indexed>=1; no store query; reuse mapped by QA inference",
    "missing_verification": "index lists >=1 file; no second service.New for index",
    "demonstration": "initRunIndex returns 1,nil without RunCodeIndex; IndexesTheProject still passes",
    "consequence": "empty index with fake Indexed; one-store reuse untested",
    "disposition": "patch: query index after init; optional no-second-store check",
    "gap_type": "broken-verification",
    "flags": ["missing-regression-net"]
  },
  {
    "title": "Repeatable --docs CLI parsing seam has no named test",
    "location": "skillgrid-cli/cmd/skillgrid/init_cmd.go:26",
    "changed_surface": "fs.Var docs stringSliceFlag in newInitFlagSet",
    "impacted_consumer": "acceptance repeated --docs flags ingest each path",
    "existing_test_evidence": "TestInitIngestsRepeatedDocsFlags passes []string to projectInit; bypasses Parse",
    "missing_verification": "Parse --docs a --docs b into ingest",
    "demonstration": "single String flag last-wins; slice test still passes; CLI drops first path",
    "consequence": "multi-docs CLI contract falsely covered",
    "disposition": "patch: parse via newInitFlagSet then ingest",
    "gap_type": "regression",
    "flags": ["missing-regression-net"]
  },
  {
    "title": "main dispatch for init is outside the focused suite",
    "location": "skillgrid-cli/cmd/skillgrid/main.go:150",
    "changed_surface": "case init -> runProjectInit",
    "impacted_consumer": "binary skillgrid init; onboarding CLI handoff",
    "existing_test_evidence": "TestInit* call projectInit only; no runProjectInit/main case test; QA smoke not named in suite",
    "missing_verification": "removing case init fails a named test",
    "demonstration": "delete case init; focused TestInit|TestOnboardingSkill still pass",
    "consequence": "verb missing from binary while library/skill look fine",
    "disposition": "patch: dispatch/exec test | defer if library-only accepted",
    "gap_type": "regression",
    "flags": ["missing-regression-net"]
  },
  {
    "title": "Onboarding skill nets are source-text-only and weakly phrased",
    "location": "skillgrid-cli/cmd/skillgrid/init_cmd_test.go:315",
    "changed_surface": "onboarding/SKILL.md:233-237 Step 9",
    "impacted_consumer": "acceptance skill handoff scenarios",
    "existing_test_evidence": "Contains skillgrid init / --docs; bans only exact walk docs/ and mem_save each file",
    "missing_verification": "imperative Run skillgrid init in Step 9; --docs scoped; broader no-second-ingest ban",
    "demonstration": "substring survives in do-not-run note; alternate mem_save ingest prose bypasses exact ban",
    "consequence": "agent-facing handoff regresses with green P0/P1 skill tests",
    "disposition": "patch: tighten Step 9 asserts; defer live IDE onboarding",
    "gap_type": "broken-verification",
    "flags": ["source-text-only"]
  }
]
```
