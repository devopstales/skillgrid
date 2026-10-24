# Task 2 report — TICKET-02 Preamble and sentinel upsert

## Status

**DONE**

## What was implemented

`writeBootFile(dir, force)` in `init_boot.go`, wired from `projectInit`:

- Boot target: `AGENTS.md` if it exists or `CLAUDE.md` is absent; else `CLAUDE.md`. If both exist, write AGENTS and ensure CLAUDE has the one-line pointer.
- Preamble region `<!-- skillgrid-preamble:start -->`…`end`: insert when missing (`written`), keep when present and `force=false` (`kept`), replace on `--force` (`forced`).
- Sentinel `<!-- skillgrid:start -->`…`end`: replace-in-place from canonical `block.md` template (local-only tracker, enabled mnemonic memory line, locked-constraints bullets or default); never duplicate.
- No `RunCodeIndex` / ingest (T03/T04).

## SATISFIES

- happy path init upserts preamble and sentinel
- second init merges without duplicating the sentinel
- force rewrites the preamble only

Acceptance scenarios map to `TestInitUpsertsPreambleAndSentinel` and `TestInitForceRewritesPreamble`. `npx cucumber-js --dry-run` could not confirm pending/pass (local cucumber install missing module); Go tests are the gate oracles.

## TDD Evidence

**RED** (before `writeBootFile`):

```bash
cd skillgrid-cli && go test ./cmd/skillgrid -count=1 -run 'TestInitUpsertsPreambleAndSentinel|TestInitForceRewritesPreamble'
```

```
--- FAIL: TestInitUpsertsPreambleAndSentinel — missing preamble: (empty AGENTS)
--- FAIL: TestInitForceRewritesPreamble — merge dropped user text
```

Expected: skeleton still wrote empty AGENTS.

**GREEN** (after implementation):

```bash
cd skillgrid-cli && go test ./cmd/skillgrid -count=1 -run 'TestInitUpsertsPreambleAndSentinel|TestInitForceRewritesPreamble|TestInitHelpListsFlags|TestInitWritesBootFileAndReportsCounts|TestInitBootFileWriteFailureIsFatal'
```

```
=== RUN   TestInitHelpListsFlags — PASS
=== RUN   TestInitWritesBootFileAndReportsCounts — PASS
=== RUN   TestInitBootFileWriteFailureIsFatal — PASS
=== RUN   TestInitUpsertsPreambleAndSentinel — PASS
=== RUN   TestInitForceRewritesPreamble — PASS
ok  	github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid
```

## Gates

| Gate | Command | Exit | EXPECT |
|------|---------|------|--------|
| G3 | `go test ./cmd/skillgrid -count=1 -run TestInitUpsertsPreambleAndSentinel` | 0 | PASS |
| G4 | `go test ./cmd/skillgrid -count=1 -run TestInitForceRewritesPreamble` | 0 | PASS |

(Run from `skillgrid-cli/`.)

## Files changed

- `skillgrid-cli/cmd/skillgrid/init_boot.go` (new) — `writeBootFile`, preamble/sentinel upsert
- `skillgrid-cli/cmd/skillgrid/init_cmd.go` — `projectInit` → `writeBootFile`
- `skillgrid-cli/cmd/skillgrid/init_cmd_test.go` — two acceptance tests

## Commit / Checkpoint

`9dd840f5159ea4d3685db873aaa8606e06f4c6d5` — `feat(cli): upsert AGENTS preamble and Skillgrid sentinel`

```
[skillgrid-context]
Task: TICKET-02 — preamble and sentinel upsert (writeBootFile).
Decisions: AGENTS vs CLAUDE boot target; preamble markers kept unless
--force; sentinel replace-in-place from block.md template; no index/ingest.
Remaining: T03 RunCodeIndex; T04 ingestPaths; T05 onboarding handoff.
Tried: TDD RED (empty AGENTS) then init_boot.go + projectInit wire-up;
focused TestInit* PASS.
```

(Rewritten with `git commit-tree` + message file after ANSI leaked into `-m` HEREDOC.)

## Refinement (4-pass)

1. **Scope:** Only boot-file upsert; no index/ingest; unused `filepath`/`projectBootFile` removed from `init_cmd.go`.
2. **Expert:** Names match blueprint (`writeBootFile`, preamble states); sentinel payload mirrors `block.md` placeholders.
3. **Defect hunt:** Fixed ANSI-tainted commit message; simplified newline suffix helper; re-ran focused tests after cleanup — clean.
4. **Polish:** gofmt; final pass 2→4 found nothing new.

## Lazy ladder

Built the blueprint-specified full `writeBootFile` (stdlib only, no new deps). Nothing skipped within T02 scope; index/ingest deferred by ticket.

## Concerns

- Sentinel template is duplicated as a Go string (copied from `block.md`); drift if onboarding skill and CLI diverge — acceptable per blueprint “copied from existing block rules.”
- `TestInitForceRewritesPreamble` covers merge + force in one test (matches blueprint); second-init merge scenario has no separate named test but is asserted inside that test.
- Cucumber dry-run unavailable in this environment (missing `cucumber` module).

## Self-review findings

None outstanding after ANSI commit rewrite and helper cleanup.
