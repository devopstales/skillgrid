# Re-review — project-init fix wave 1 (D1–D7 / D16)

> Change: `.skillgrid/specs/2026-10-02-mnemonic-project-init/`
> Base: `62600901` · Head: `c32ef909` (fix: `cb077edf`)
> Diff: `.skillgrid/sdd/2026-10-02-mnemonic-project-init/review-fix1.diff`
> Prior: `.skillgrid/specs/2026-10-02-mnemonic-project-init/review.md`
> Fix report: `.skillgrid/sdd/2026-10-02-mnemonic-project-init/review-fix1-report.md`
> Parent suite: `TestInit*|TestOnboardingSkill` PASS (not re-run here)

## Overall: STILL_OPEN

Code closes the D1–D7/D16 defects. Two medium items lack regression tests that exercise the fixed behavior, so they remain open under the “addressed with tests” bar.

## Finding verdicts

| ID | Verdict | Evidence |
|----|---------|----------|
| D1 | **FIXED** | `ingestExtra` EvalSymlinks both roots + Rel jail (`init_ingest.go:131-145`). `TestInitRejectsDocsSymlinkEscape` asserts error + topic `init/docs/leak.md` count 0 (`init_cmd_test.go:170-196`). Defaults also `Lstat` so symlink roots are not followed as regular files (`init_ingest.go:48`). |
| D2 | **FIXED** | `writeBootFile` refuses symlink targets via `Lstat` before `WriteFile` (`init_boot.go:77-83`). `TestInitRejectsSymlinkBootFile` asserts fatal error + host body unchanged (`init_cmd_test.go:198-218`). |
| D3 | **STILL_OPEN** | Bounds exist: `maxIngestBytes` 512KiB + NUL probe in first 512 (`init_ingest.go:26`, `212-226`). **No test** creates an oversized or NUL-bearing file and asserts non-ingest / error. Fix report lists no covering test. |
| D4 | **FIXED** | `TestInitHelpListsFlags` calls `newInitFlagSet` then `Usage` (`init_cmd_test.go:232-241`); no hand-built FlagSet/Usage. |
| D5 | **FIXED** | Extra walk captures callback `walkErr`, accumulates save errors, and appends outer `WalkDir` error (`init_ingest.go:157-185`). Defaults already append walk errors (`init_ingest.go:58-78`). Defect (swallow) gone. No dedicated walkErr fixture (hard without chmod); not reopened. |
| D6 | **STILL_OPEN** | Relative extras join under project `absDir` before `Abs` (`init_ingest.go:122-125`). **No test** uses a relative `--docs` with cwd ≠ project dir. Existing extras tests pass absolute paths only (`TestInitIngestsExtraDocs`, `TestInitIngestsRepeatedDocsFlags`). |
| D7 | **FIXED** | `outsideJail` rejects `rel == "."` (`init_ingest.go:116-118`). `TestInitRejectsDocsProjectRoot` passes project dir as `--docs` and expects errors (`init_cmd_test.go:220-230`). |
| D16 | **FIXED** | Jail uses `rel == ".."` \|\| `HasPrefix(rel, "../")`, not `HasPrefix(rel, "..")` (`init_ingest.go:116-118`). False reject of `..foo` removed. Covered indirectly by outside/`../` paths; no positive `..foo` allow test — residual only, not reopened. |

## New breakage in the fix diff

None Critical/Important.

- **Minor:** Default roots that are symlinks are now silently skipped (`Lstat` + non-regular `continue`) rather than followed. Intentional jail hardening; not a D1–D7 regression.

## Out-of-scope observations

- `ensureClaudePointer` still `WriteFile`s `CLAUDE.md` without a symlink refuse (`init_boot.go:133-153`). Untouched by this fix; adjacent to D2 class. Ledger for later if desired.
- Deferred D8–D20 / D21 from `review.md` unchanged; not in this wave.

## Open items for next fix round

1. **D3** — add tests: file >512KiB rejected; file with NUL in first 512 rejected; neither creates `init/docs/…` observation.
2. **D6** — add test: `os.Chdir` away from project, pass relative docs path, assert ingest under project (or equivalent without relying on abs paths).
