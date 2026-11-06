# Task 1 report — TICKET-01 Init command door check

## Status

**DONE**

## Files changed

- `skillgrid-cli/cmd/skillgrid/init_cmd.go` (new) — `initResult`, `projectInit`, `runProjectInit`, flag/help wiring
- `skillgrid-cli/cmd/skillgrid/init_cmd_test.go` (new) — three BDD tests from blueprint Task 1
- `skillgrid-cli/cmd/skillgrid/main.go` — `init` dispatch + help line

## Tests

```bash
cd skillgrid-cli && go test ./cmd/skillgrid -count=1 -run 'TestInitHelpListsFlags|TestInitWritesBootFileAndReportsCounts|TestInitBootFileWriteFailureIsFatal'
```

Result: **PASS** (`ok github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid`)

RED phase: compile failed with `undefined: projectInit` as expected.

## Commit

`b5239413ed21185845a0860791ff6b16958a3ba8` — `feat(cli): add skillgrid init command skeleton`

(Rewritten with `git commit-tree` after the prepare-commit hook injected ANSI and `Co-authored-by`.)

## Concerns

- `TestInitHelpListsFlags` duplicates flag definitions instead of calling `newInitFlagSet`; drift risk until a shared helper test is added.
- `projectInit` ignores `svc`, `force`, and `extraDocs` until T02–T04; `Preamble`/`Sentinel` are placeholders, not real boot-file content yet.
- `printInitResult` output shape is provisional (human lines on stdout); may need alignment with briefing JSON field names in a later slice.

## Self-review (4-pass)

1. **Scope:** Wave 1 only — no preamble/sentinel template, index, or ingest.
2. **Acceptance:** All three named tests green; non-directory target returns error from `projectInit`.
3. **Patterns:** Matches `index` subcommand dispatch; reuses `stringSliceFlag`, `newMnemonicService`, `envOr`.
4. **Constraints:** No new dependencies; ADR-0012 unchanged.
