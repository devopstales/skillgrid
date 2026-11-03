# Task 1 review — TICKET-01 Init command door check

**Reviewed:** `f739d371` → `b5239413`  
**Diff:** `task-1.diff` (verified; report not trusted blindly)  
**Scope:** Task-scoped gate only

---

## 1. Spec compliance: PASS

Matches Task 1 brief / TICKET-01 acceptance for Wave 1 door check.

| Requirement | Evidence |
|---|---|
| Files: `init_cmd.go`, `init_cmd_test.go`, `main.go` dispatch + usage line | Diff: 3 files, +184, usage line next to `index` |
| `projectInit` / `initResult` / `runProjectInit` | Present; Indexed/Ingested default 0 |
| Happy path writes boot file | Creates empty `AGENTS.md` when missing; returns `BootFile` |
| Help names `--force` and `--docs` | `newInitFlagSet` + Usage; `-h`/`--help` short-circuit |
| Non-directory / write failure fatal | `Stat` + `!IsDir` error; write error returned; `runProjectInit` exits 1 |
| Wave 1 placeholders OK | Empty file; `Preamble`/`Sentinel` strings; no index/ingest |
| No new deps; ADR-0012 untouched | CLI-only; reuses `service.Service` / `stringSliceFlag` |
| Spec zone already committed; code-only | Diff is `skillgrid-cli/cmd/skillgrid` only |

### Findings

**Minor**
- `TestInitWritesBootFileAndReportsCounts` asserts boot path only, not `Indexed`/`Ingested` (or printed counts). Matches the brief’s prescribed skeleton; `printInitResult` does emit zeros. Gap vs SATISFIES wording “reports counts,” not vs Step 1 tests.
- `TestInitBootFileWriteFailureIsFatal` fails on non-directory before a write attempt. Matches brief + ticket (“Non-directory target returns error”); true permission-denied write path is untested (acceptable for this door check).
- Existing `AGENTS.md` is left alone but result still reports `Preamble: "written"` — placeholder semantics, deferred to later tickets.

**Critical / Important:** none for task scope.

---

## 2. Code quality: approved

Clean skeleton aligned with existing CLI (`newMnemonicService`, `stringSliceFlag`, `envOr`, `case "index"`-style dispatch). Scope held to Wave 1; ignored `svc`/`force`/`extraDocs` are explicit seams for T02–T04.

### Findings

**Important**
- `TestInitHelpListsFlags` re-defines flags/`Usage` instead of calling `newInitFlagSet`. Production uses `Var` + `stringSliceFlag`; the test uses `String("docs", ...)`. Help can drift from production without failing the test. Implementer noted this; not a blocker for the door check (brief prescribed that skeleton), but should be tightened when touching help next.

**Minor**
- `printInitResult` takes `*os.File` rather than `io.Writer` (harder to unit-test output shape).
- Redundant `fs.SetOutput(os.Stderr)` before `newInitFlagSet` (which also sets it).
- Human stdout lines are provisional vs any later JSON field contract (noted in report; fine for T01).

No overbuild, no stray deps, no merge-scope creep.

---

## 3. TDD evidence check

| Claim | Assessment |
|---|---|
| RED: `undefined: projectInit` | Credible — tests call `projectInit` before implementation; matches brief Expected FAIL |
| GREEN: three named tests PASS | Parent re-verified; report’s package path matches module layout |
| Tests follow brief Step 1 | Diff test bodies match the prescribed skeleton |
| Commit message | `feat(cli): add skillgrid init command skeleton` as specified |

**Verdict:** RED→GREEN claim accepted for this gate (not re-run here).

---

## 4. Overall: APPROVED

Door check is in place: dispatch, flags, empty boot file, fatal on bad target, three acceptance tests green, Wave 1 constraints respected. Proceed to TICKET-02+.

Optional follow-ups (non-blocking): wire help test through `newInitFlagSet`; assert zero counts on happy path; prefer `io.Writer` for `printInitResult`.
