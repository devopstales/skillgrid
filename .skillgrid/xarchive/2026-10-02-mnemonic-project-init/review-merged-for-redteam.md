# review-standards
# Standards Review — mnemonic-project-init

**Range:** `f739d371`..`17652346`  
**Axis:** Standards (glossary + named ADRs + code-standards smell baseline)  
**Independence:** Grade A — reviewed from `review-range.diff` + standards sources only; no other axis reports consulted.  
**Spec completeness:** not judged.

### Strengths

- **ADR-0012 honored.** Ingest goes through `svc.OpenForDirectory` → `memory.Save` with stable `TopicKey` values (`init/docs/<relpath>`). No Claude OS “four KBs”, no second markdown knowledge store, no parallel SQLite. Matches glossary *Project Init* / *Topic Key* and ADR-0012’s one-store rule.
- **ADR-0016 honored.** Change is a DX CLI sequencer (`projectInit` → boot → existing `RunCodeIndex` → ingest). It does not add a new brain engine, MCP tool, or retrieval path.
- **ADR-0019 / ADR-0020 honored.** No new decision body inlined into `ASSUMPTIONS.md`; no teams/ledger ownership move. QA report lives under the SDD ledger path; state only flips phase.
- **Ubiquitous language.** `projectInit` / `skillgrid init`, onboarding step titled “Project init”, `*service.ProjectHandle`, and `TopicKey` stay inside `.skillgrid/artifacts/02-technical-terms.md`. Rejected “Knowledge Base” / four-KB shape does not appear in code or skill prose.
- **Module split.** `init_boot.go` / `init_cmd.go` / `init_ingest.go` each have one nameable reason to change (boot file, orchestration, ingest). Entry `runProjectInit` stays thin.
- **Error taxonomy at the seam.** Boot failure is fatal; index/open/ingest failures accumulate into `initResult.Errors` and still exit success — recoverable vs fatal is visible to the caller.
- **Behavior tests.** Upsert/force/sentinel-once, path jail, missing defaults skipped, non-fatal index/extra-docs, and onboarding handoff assertions exercise real side effects rather than pure mocks.

### Issues

#### Critical (Must Fix)

_(none)_

#### Important (Should Fix)

1. **Hard standard — agent-config placeholder contract clobbered on every init**
   - File: `skillgrid-cli/cmd/skillgrid/init_boot.go` (`renderSentinel`, ~lines 148–154 in the new file)
   - Standard: `.agents/skills/_shared/agent-config/block.md` — `{tracker_line}` and `{memory_line}` must be filled from the active tracker and mnemonic status; the block is the SSOT for sentinel content.
   - Issue: `renderSentinel` always substitutes `defaultTracker` (`None — work local-only from tasks.md.`) and `enabledMemoryLine`. `upsertSentinel` then replaces the entire sentinel region. Re-running `skillgrid init` on a Backlog/GitHub/GitLab project (or mnemonic-disabled project) silently rewrites correct lines.
   - Why it matters: the boot upsert is destructive relative to the documented fill contract; agents then see a wrong tracker/memory spine on every prompt.
   - Fix: Read tracker + mnemonic from `.skillgrid/config.yaml` (same mapping table as `block.md`); preserve existing placeholder lines when already present if detection is deferred.

2. **Hard standard — boot-path selection contradicts agent-config SSOT**
   - File: `skillgrid-cli/cmd/skillgrid/init_boot.go` (`resolveBootPath`, ~lines 102–123)
   - Standard: `block.md` § Idempotent upsert / Gotchas — if `AGENTS.md` is absent, **create `AGENTS.md`**; only add a CLAUDE pointer when both exist. Never treat sole `CLAUDE.md` as the full-block target.
   - Issue: When only `CLAUDE.md` exists, code returns `CLAUDE.md` as the boot path and writes the full sentinel there.
   - Why it matters: two-source drift risk the shared agent-config rules explicitly forbid; comment claims the template is “from block.md” while the path rule diverges.
   - Fix: Always prefer/create `AGENTS.md`; if `CLAUDE.md` exists alongside, append the one-line pointer only (`ensureClaudePointer` already covers the dual-file case).

3. **Hard standard — duplicated sentinel body vs documented SSOT**
   - File: `skillgrid-cli/cmd/skillgrid/init_boot.go` (`sentinelTemplate`, ~lines 23–69)
   - Standard: `block.md` opening line — “Single source of truth for the `## Skillgrid` block… must not drift.” Code-standards smell: **Duplicated Code**.
   - Issue: Full sentinel markdown is copy-pasted into a Go string constant. No parity test against `block.md`.
   - Why it matters: next edit to the lean AGENTS spine will update one copy and leave the other; every-prompt context then disagrees with the skill docs.
   - Fix: Load/parse `block.md` at build or runtime, or add a golden test that diffs the rendered template against the markdown fenced block.

4. **Hard standard — package-level mutable seam**
   - File: `skillgrid-cli/cmd/skillgrid/init_cmd.go` (`var initRunIndex = func…`, ~lines 90–98)
   - Standard: `.agents/skills/_shared/rules/code-standards.md` — “No global mutable state”; checklist “No shared mutable state without a single owner.”
   - Issue: Tests reassign `initRunIndex`. Production and test share a mutable package var.
   - Why it matters: parallel tests or future callers can race; the seam is not in the function signature (Explicit over implicit).
   - Fix: Pass an `indexFn` (or small interface) into `projectInit`; keep the default adapter local to `runProjectInit`.

5. **Hard standard — swallowed walk errors on `--docs` directories**
   - File: `skillgrid-cli/cmd/skillgrid/init_ingest.go` (`ingestExtra` WalkDir callback, ~lines 133–145)
   - Standard: code-standards Errors — “Never swallow.”
   - Issue: Default-root walk appends `walkErr` into `errs`. Extra-docs walk does `if walkErr != nil || d.IsDir() || !… { return nil }` and drops the error.
   - Why it matters: partial `--docs` failures look like success (`Errors` empty, `Ingested` under-count); inconsistent with the default-path path.
   - Fix: Mirror the default walk’s `errs = append(errs, …)` handling (and don’t overwrite a single `errMsg` for multi-file failures).

#### Minor (Nice to Have)

1. **Optional: Duplicated Code — directory walk shape**
   - File: `init_ingest.go` default-root walk vs `ingestExtra` directory branch


# review-spec
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

1. **MISSING_RED (all T01–T05 SATISFIES scenarios)** — Every 

# review-edge
# Edge-case hunter — mnemonic-project-init

**Range:** `f739d371..17652346`  
**Lens:** writeBootFile, ingestPaths jail, symlink/--docs, empty files, chmod, second-init merges, index failure paths  
**Grade:** A (exhaustive path walk; only unhandled branches reported)

```json
[
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:113-154",
    "trigger_condition": "--docs path is in-tree symlink to outside file",
    "guard_snippet": "real, err := filepath.EvalSymlinks(absPath); jail Rel(absDir, real)",
    "potential_consequence": "Jail bypass; outside file bytes saved into project memory",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:118-122",
    "trigger_condition": "--docs path Rel is '.' (project root)",
    "guard_snippet": "if rel == \".\" || rel == \"\" { return jail-or-reject error }",
    "potential_consequence": "Walk ingests entire tree including secrets and .git",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:119",
    "trigger_condition": "In-project file/dir name starts with '..' e.g. ..foo",
    "guard_snippet": "outside := rel == \"..\" || strings.HasPrefix(rel, \"../\")",
    "potential_consequence": "Valid in-project --docs path falsely rejected as outside",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:131-145",
    "trigger_condition": "--docs targets symlink directory (Stat follows)",
    "guard_snippet": "info, err := os.Lstat(absPath); reject ModeSymlink or EvalSymlinks+re-jail",
    "potential_consequence": "WalkDir skips symlink root; silent zero ingest, no Errors",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:79-81",
    "trigger_condition": "Default root README.md is symlink or non-regular",
    "guard_snippet": "errs = append(errs, root+\": not a regular file\")",
    "potential_consequence": "Silent skip: neither Skipped nor Errors lists the path",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:63-65",
    "trigger_condition": "Symlink file under docs/ or artifacts walk",
    "guard_snippet": "if d.Type()&fs.ModeSymlink!=0 { errs=append(...); return nil }",
    "potential_consequence": "Symlinked docs silently omitted from ingest and report",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:133-141",
    "trigger_condition": "WalkDir walkErr or multiple saveErr under --docs dir",
    "guard_snippet": "if walkErr!=nil { errs=append(...)} ; collect all saveErrs",
    "potential_consequence": "Walk failures dropped; only last saveErr kept in errMsg",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:167-170",
    "trigger_condition": "File was ingested then becomes empty/whitespace on re-init",
    "guard_snippet": "on empty: mem delete-or-supersede topicKey, or Skipped+note",
    "potential_consequence": "Second-init leaves stale observation; Errors only, no upsert",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_boot.go:261-268",
    "trigger_condition": "AGENTS.md has skillgrid:start without matching :end",
    "guard_snippet": "if start>=0 && end<=start { return error mismatched sentinel markers }",
    "potential_consequence": "Every re-init appends another sentinel; duplicates accumulate",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_boot.go:239-258",
    "trigger_condition": "Preamble start marker present without valid end after it",
    "guard_snippet": "if start>=0 && end<=start { return error mismatched preamble markers }",
    "potential_consequence": "Second-init inserts another preamble region; markers multiply",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_boot.go:91-96",
    "trigger_condition": "AGENTS.md WriteFile OK then ensureClaudePointer WriteFile fails",
    "guard_snippet": "write CLAUDE pointer first, or rollback AGENTS on pointer error",
    "potential_consequence": "Fatal exit after AGENTS already mutated; partial boot state",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_boot.go:102-122",
    "trigger_condition": "AGENTS.md exists as symlink to path outside project",
    "guard_snippet": "if fi.Mode()&os.ModeSymlink!=0 { return err refuse symlink boot }",
    "potential_consequence": "WriteFile follows link and overwrites outside boot target",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_boot.go:157-161",
    "trigger_condition": "ASSUMPTIONS.md exists but chmod makes it unreadable",
    "guard_snippet": "if !os.IsNotExist(err) { return \"\", err } // surface EACCES",
    "potential_consequence": "Silent default Rules; locked constraints omitted from sentinel",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_cmd.go:120-125",
    "trigger_condition": "RunCodeIndex returns non-nil err with FilesIndexed > 0",
    "guard_snippet": "res.Indexed = indexed; if indexErr != nil { append Errors }",
    "potential_consequence": "Partial index progress reported as Indexed 0",
    "kind": "edge"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:33-36",
    "trigger_condition": "ingestPaths SessionStart succeeds; never SessionEnd",
    "guard_snippet": "defer mem.SessionEnd(ctx, sid, \"skillgrid-init\")",
    "potential_consequence": "Each init leaves an open dangling session row",
    "kind": "edge"
  }
]
```


# review-verif
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
- Disposition: patch (open index after init and assert file/c

# review-security
# Security Review — mnemonic-project-init

> Specialist: Security (OWASP + parallel-code-review/security.md)
> Range: `f739d371..17652346`
> Diff: `.skillgrid/sdd/2026-10-02-mnemonic-project-init/review-range.diff`
> Focus: path traversal / `--docs` jail, AGENTS.md writes, `memory.Save` ingest, symlink escapes
> Mode: read-only evaluation

Security Review Progress:
- [x] Step 1: Map entry points and trust boundaries
- [x] Step 2: Load the references this code needs
- [x] Step 3: Sweep for candidate issues
- [x] Step 4: Triage every candidate
- [x] Step 5: Report findings

## Entry points and trust boundaries

| Entry | Control | Sink |
|-------|---------|------|
| CLI `skillgrid init [dir]` | operator / agent cwd | `writeBootFile`, `RunCodeIndex`, `ingestPaths` |
| `--docs path` (repeatable) | operator / onboarding agent | `ingestExtra` → `os.ReadFile` → `memory.Save` |
| `--force` | operator | preamble rewrite in boot file |
| Project tree contents | **untrusted if repo is malicious** | default ingest roots + boot-file path selection |
| Onboarding SKILL.md | agent reads user-named docs paths | forwards as `--docs` |

Trust boundary the jail claims to enforce: **no read/write outside the project directory**. Lexical `filepath.Abs` + `filepath.Rel` + `HasPrefix(..)` is insufficient when the kernel follows symlinks on `Stat`/`ReadFile`/`WriteFile`.

Threat model (reachable): clone or open a crafted project, run `skillgrid init` (or let onboarding do it). No network auth surface in this diff.

## What was checked (and cleared)

| Candidate | Verdict |
|-----------|---------|
| Lexical `--docs /etc/passwd` / `../outside` | Jail rejects (`TestInitRejectsDocsOutsideProject`) |
| Nested symlink **files** under walked dirs (`docs/`, artifacts, `--docs <dir>`) | `WalkDir` + `d.Type().IsRegular()` skips symlink entries |
| Symlink **directory** as `--docs` root | `Stat` sees dir; `WalkDir` `Lstat`s root and does not descend — no leak (n=0) |
| Command/SQL injection via ingest | Reads bytes into `Save` only; no exec |
| New network/auth/CORS surface | None |
| Secrets committed in diff | None |

QA’s “path jail covered” claim is true for **lexical** outside paths only; it does not cover symlink escapes.

## Findings (highest impact first)

### 1. `--docs` jail bypass via in-project file symlink (read)

**[High] Symlink escapes lexical docs jail into host files (CWE-59, OWASP A01:2025 Broken Access Control, ASI02 Tool Misuse)**

- **Location:** `skillgrid-cli/cmd/skillgrid/init_ingest.go:113-154` (jail at `:118-122`; read at `:150` → `saveIngestFile` `:163`)
- **Path:** `--docs <symlink-inside-project>` → `filepath.Abs` (no `EvalSymlinks`) → `Rel` stays in-tree → `os.Stat`/`os.ReadFile` follow link → `memory.Save`
- **Impact:** Malicious repo plants `leak.md` → `/etc/passwd` (or SSH keys, cloud creds). Init persists host file contents into the project memory store under `init/docs/leak.md`.
- **Evidence:** Lexical Rel of an in-project symlink is `leak.md` (jail allows); `ReadFile` returns outside content. No `Lstat` / `EvalSymlinks` / `O_NOFOLLOW`.
- **Confidence:** Confirmed
- **Fix:** Resolve both project root and candidate with `filepath.EvalSymlinks` (or open with `O_NOFOLLOW`), re-run `Rel` on real paths, reject `ModeSymlink` before read; add `TestInitRejectsDocsSymlinkEscape`.

### 2. Default single-file ingest follows symlinks (read)

**[High] README/ASSUMPTIONS/ARCHITECTURE symlink reads host files (CWE-59, A01:2025, LLM02 Sensitive Information Disclosure)**

- **Location:** `skillgrid-cli/cmd/skillgrid/init_ingest.go:79-87` → `saveIngestFile:163`
- **Path:** default root `README.md` (etc.) is a symlink → `os.Stat` reports regular (target mode) → `ReadFile` follows → `memory.Save`
- **Impact:** Same as (1) without needing `--docs`. Walked trees are safer; these three single-file roots are not.
- **Confidence:** Confirmed
- **Fix:** Same confined-open helper for all ingest reads; reject non-regular after `Lstat`.

### 3. Boot-file write follows symlinks (write)

**[High] AGENTS.md/CLAUDE.md symlink overwrite outside project (CWE-59, A01:2025, ASI02)**

- **Location:** `skillgrid-cli/cmd/skillgrid/init_boot.go:91` (`WriteFile` boot path); `:145` (`ensureClaudePointer` writes `CLAUDE.md`)
- **Path:** crafted `AGENTS.md` → sensitive host path → `resolveBootPath` treats it as existing → `os.WriteFile` follows symlink → host file replaced with preamble/sentinel
- **Impact:** Destructive write outside the project tree when the operator runs init on a malicious checkout (classic malicious-repo local-tool attack).
- **Confidence:** Confirmed (WriteFile-follows-symlink reproduced)
- **Fix:** `Lstat` boot targets; refuse symlinks (or create exclusively with `O_EXCL`/`O_NOFOLLOW`); never follow links when writing AGENTS/CLAUDE.

### 4. In-project `--docs` can dump arbitrary files into persistent memory

**[Medium] No sensitive-path filter before memory.Save (CWE-200, LLM02, ASI06 Memory & Context Poisoning)**

- **Location:** `skillgrid-cli/cmd/skillgrid/init_ingest.go:90-97`, `:131-145`, `:157-185`
- **Path:** operator/agent `--docs .env` / `--docs .` / `--docs secrets/` → walk regular files → full content `Save` as `init/docs/<relpath>`
- **Impact:** Secrets already in the repo (or broad trees) become durable agent-retrievable memory. Onboarding explicitly forwards user-named paths as `--docs`. Jail only stops *outside* the project.
- **Confidence:** Confirmed (by design breadth; exposure is real)
- **Fix:** Allowlist doc-like extensions/roots for extras; deny `.env*`, `*.pem`, `*credentials*`, `.git/`, etc.; cap bytes per file; document that `--docs` is not a secret vault.

### 5. Unbounded `ReadFile` into Save

**[Low] Unbounded ingest read enables local memory pressure (CWE-400, LLM06 Unbounded Consumption)**

- **Location:** `skillgrid-cli/cmd/skillgrid/init_ingest.go:163-167`
- **Path:** huge regular file under default/`--docs` tree → full `[]byte` in process + store
- **Impact:** Local DoS / st

# review-perf
# Performance review — mnemonic-project-init

> Specialist: performance · Range `f739d371..17652346` · Diff: `review-range.diff`
> Focus: WalkDir over docs/artifacts, per-file `memory.Save`, `RunCodeIndex` cost, double `OpenForDirectory`

## Findings (JSON)

```json
[
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:55",
    "issue": "WalkDir ingests every regular file with no type or size filter",
    "cost_and_scale": "Binary assets under docs/ (≈3MB PNGs here) become SQLite observation blobs per init",
    "fix": "Limit to text extensions and enforce a MaxFileSize skip like codeindex"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_ingest.go:172",
    "issue": "Each file calls memory.Save with full AUDN and candidate queries",
    "cost_and_scale": "About 4–6 SQLite round-trips per file; bites at tens to hundreds of docs",
    "fix": "Wrap ingest in one transaction or a bulk topic_key upsert path"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_cmd.go:120",
    "issue": "RunCodeIndex opens the store then OpenForDirectory opens it again",
    "cost_and_scale": "Two store.Open/migrate cycles per init once the pool refcount hits zero",
    "fix": "Open once; run codeindex.New(handle.Store()).Run then ingest on same handle"
  },
  {
    "location": "skillgrid-cli/cmd/skillgrid/init_cmd.go:96",
    "issue": "init always runs full codeindex Scan even on a warm index",
    "cost_and_scale": "Full-tree walk each init; large repos pay seconds even when FilesIndexed is 0",
    "fix": "Reuse one open with ingest; skip Scan only if product allows warm short-circuit"
  }
]
```

## Notes (for coordinator)

- Spec requires calling `RunCodeIndex` on init; do not drop the call. Finding 4 is about residual Scan cost and sharing the open (finding 3), not removing the index step.
- Evidence for finding 1: this repo’s `docs/assets/` holds ~1MB PNG ×3 plus JPEGs; WalkDir has no `.md` filter or size cap (unlike `codeindex.MaxFileSize` 500KiB).
- `ResolveProject` in `initRunIndex` (line 93) is a cheap extra resolve before `RunCodeIndex` re-resolves; not filed separately.


