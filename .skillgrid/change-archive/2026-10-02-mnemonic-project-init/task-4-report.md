# Task 4 report — TICKET-04 Default and extra ingest

## Status

**DONE**

## What was implemented

After code index, `projectInit` opens `OpenForDirectory`, then `ingestPaths`:

- Defaults: `README.md`, `docs/`, `.skillgrid/ASSUMPTIONS.md`, `.skillgrid/ARCHITECTURE.md`, `.skillgrid/artifacts/` — missing roots → `Skipped`; existing files/`WalkDir` regular files → `memory.Save`.
- Topic key `init/docs/<relpath>` (slash-separated); `CapturePrompt: false`; `ToolName: skillgrid-init`; `SessionStart(..., "skillgrid-init")`.
- Types: `architecture` for ASSUMPTIONS/ARCHITECTURE/artifacts; `discovery` otherwise.
- `--docs`: jailed via `filepath.Rel` + `..` prefix → `Errors`; missing extras → `Errors`; non-fatal.
- Second init upserts same topic keys (no duplicate rows). No new store/KB (ADR-0012).

## SATISFIES

- happy path init ingests default paths
- missing docs directory is skipped
- second init upserts the same topic keys
- happy path init ingests extra --docs path
- missing extra docs path is listed and non-fatal

Acceptance: `TestInitIngestsDefaultPaths`, `TestInitSkipsMissingDocs`, `TestInitIngestsExtraDocs`, `TestInitMissingExtraDocsIsNonFatal`, `TestInitRejectsDocsOutsideProject`.

## TDD Evidence

**RED** (tests only, no `ingestPaths`):

```bash
cd skillgrid-cli && go test ./cmd/skillgrid -count=1 -run 'TestInitIngests|TestInitSkips|TestInitMissing|TestInitRejects'
```

```
--- FAIL: TestInitIngestsDefaultPaths … README observation
--- FAIL: TestInitSkipsMissingDocs … skipped = []
--- FAIL: TestInitIngestsExtraDocs … extra docs
--- FAIL: TestInitMissingExtraDocsIsNonFatal … Errors:[]
--- FAIL: TestInitRejectsDocsOutsideProject … expected jail error
FAIL
```

**GREEN** (after `init_ingest.go` + wire):

```bash
cd skillgrid-cli && go test ./cmd/skillgrid -count=1 -run 'TestInit'
```

```
ok  	github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid	18.255s
```

## Gates

| Gate | Command | Exit | EXPECT |
|------|---------|------|--------|
| ingest defaults | `go test ./cmd/skillgrid -count=1 -run TestInitIngestsDefaultPaths` | 0 | PASS |
| skip missing docs | `go test ./cmd/skillgrid -count=1 -run TestInitSkipsMissingDocs` | 0 | PASS |
| extra --docs | `go test ./cmd/skillgrid -count=1 -run TestInitIngestsExtraDocs` | 0 | PASS |
| missing extra non-fatal | `go test ./cmd/skillgrid -count=1 -run TestInitMissingExtraDocsIsNonFatal` | 0 | PASS |
| jail outside | `go test ./cmd/skillgrid -count=1 -run TestInitRejectsDocsOutsideProject` | 0 | PASS |
| all TestInit* | `go test ./cmd/skillgrid -count=1 -run 'TestInit'` | 0 | PASS |

(Run from `skillgrid-cli/`.)

## Files changed

- `skillgrid-cli/cmd/skillgrid/init_ingest.go` — **created** `ingestPaths`, jail, Save
- `skillgrid-cli/cmd/skillgrid/init_cmd.go` — wire OpenForDirectory + ingest after index
- `skillgrid-cli/cmd/skillgrid/init_cmd_test.go` — five ingest tests + second-init upsert

## Commit / Checkpoint

`8763f9b633f6c0e87a046e04927f621732cb3821` — `feat(cli): ingest project docs from skillgrid init`

(On protected `release/2`; `GIT_DIR`/`GIT_WORK_TREE` unset. Subject clean; no AI attribution trailers.)

```
[skillgrid-context]
Task: TICKET-04 — default + jailed --docs ingest via memory.Save (init/docs/<relpath>).
Decisions: OpenForDirectory after index; SessionStart skillgrid-init; missing
defaults→Skipped; bad/missing extras→Errors; architecture vs discovery types;
countTopic via Recent (FTS omits topic_key).
Remaining: T05 onboarding skill calls skillgrid init.
Tried: TDD RED (Ingested 0 / missing observations) → ingestPaths GREEN; all TestInit* PASS.
```

## Concerns

- `countTopic` uses `Recent` instead of brief's `Search("init")` because `observations_fts` does not index `topic_key`.
- Second-init assertion re-opens the handle after the first close (shared DB closed by prior `OpenForDirectory` cleanup).
