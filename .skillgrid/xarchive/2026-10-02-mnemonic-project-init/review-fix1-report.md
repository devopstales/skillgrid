# Review fix wave 1 — mnemonic-project-init

> Change: `.skillgrid/specs/2026-10-02-mnemonic-project-init/`
> Date: 2026-10-03
> Source: parallel review BACK-TO-APPLY (`review.md` D1–D7, D16)

## Status

**FIXED** — Critical/High symlink jail + ingest bounds + help/walkErr/cwd findings.

**Commit:** `cb077edfb795b3150593ef2ddb8ecfecb0a116a3`  
`fix(cli): harden skillgrid init path jail and ingest bounds`

## Fixes

| ID | Finding | Fix |
|----|---------|-----|
| D1 | Symlink escapes `--docs`/default jail | Defaults use `Lstat` + regular-file check; `--docs` joins under `absDir`, `EvalSymlinks` both roots, Rel jail, reject `.` |
| D2 | Symlink boot overwrites host | `writeBootFile` `Lstat` refuses symlink targets before `WriteFile` |
| D3 | Unbounded / binary ingest | `saveIngestFile` rejects >512KiB or NUL in first 512 bytes |
| D4 | Help test hand-built FlagSet | `TestInitHelpListsFlags` calls `newInitFlagSet` + `Usage` |
| D5 | `walkErr` swallowed in extras | Append walk errors into `errMsg` like defaults |
| D6 | Relative `--docs` Abs vs cwd | Non-absolute extras join under project `absDir` |
| D7 | `--docs` project root | Reject `rel == "."` |
| D16 | `..foo` false reject | Outside check is `rel == ".." \|\| HasPrefix(rel, "../")` (via `outsideJail`) |

## Tests added

- `TestInitRejectsDocsSymlinkEscape`
- `TestInitRejectsSymlinkBootFile`
- `TestInitRejectsDocsProjectRoot`
- `TestInitHelpListsFlags` (production FlagSet)

## Verification

```
cd skillgrid-cli && go test ./cmd/skillgrid -count=1 -run 'TestInit|TestOnboardingSkill'
ok  github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid  23.137s
```

## Deferred (unchanged)

D8–D15 from review coordinator (test seam, sentinel SSOT, double Open, warm indexed:0, etc.).
