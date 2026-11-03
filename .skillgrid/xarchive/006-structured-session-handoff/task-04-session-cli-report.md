# 04-session-cli — Report

## Status: PASS

Step 04 is complete. `skillgrid session handoff|resume|status` is wired, mirrors the
MCP Session Relay on the same project store, and fails closed on bad input / no
usable store. All Session tests pass; the 005/008/010/011/013 baselines still pass.

## What was implemented per sub-task

### 04.1 `[RED]` — CLI mirrors MCP on the same store
- **`cmd/skillgrid/session.go`** (new): `runSession` dispatches `handoff|resume|status`.
  - `openSessionService(dataDir, project)` resolves the SAME project store the MCP
    server uses: explicit `--project` → `service.New(dataDir).Open(project.NormalizeID(project))`;
    otherwise `svc.OpenForCWD()` (CWD-resolved) — exactly the resolution the MCP
    `openService()` performs.
  - `runSessionHandoff` calls `relay.Handoff(ctx, h.Store().DB, h.ProjectID(), handoffID, h.Root(), in)`
    — the SAME function + store the MCP `handleSessionHandoff` calls.
  - `runSessionResume` calls `relay.Resume(...)`.
  - `runSessionStatus` calls `relay.Status(...)` with optional caller-supplied
    `--context-usage-percent` / `--cost-usd` (mirrors the MCP optional stats; the relay
    never invents cost/context it does not have).
- **`cmd/skillgrid/main.go`** (modified): added the `case "session": runSession(...)` dispatch.
- **`internal/mnemonic/mcp/tools_session_export.go`** (new): pure aliases
  `HandleSessionResumeForTest` / `HandleSessionStatusForTest` exposing the two MCP
  handlers to the CLI test package so the mirrors-MCP proof can drive the MCP path
  in-process. No new tool, no behavior change (the 82-tool surface is unchanged).
- **`cmd/skillgrid/session_test.go`** (new): `TestSessionMirrorsMCP` (04.1).

### 04.2 `[AFK]` — Bad flags / missing id
- `TestSessionResumeMissingID`, `TestSessionResumeUnknownID`, `TestSessionHandoffMissingFlag`.
  - Missing id (`session resume` with no positional) → exit 2, stderr names `handoff_id`.
  - Unknown id (`session resume no-such-handoff`) → exit 1, stderr carries the relay's
    `unknown handoff id` message.
  - Missing required flag (`session handoff` without `--next-prompt`) → exit 2, stderr
    names `--next-prompt`; unknown flag (`--bogus-flag`) → exit 2 (flag parse abort).
  - No partial cleave bundle: the required-flag/unknown-flag paths abort before the
    relay runs (no files written); the relay's own `WriteBundle` removes any half-bundle
    on write failure, and the row is only written after the files are on disk.

### 04.3 `[AFK]` — No usable store fails closed
- `TestSessionNoStore`, `TestSessionNoStoreResume`.
  - The data dir is a file (so `store.Open` fails with `create data dir: … not a
    directory`). `openSessionService` returns the error → the CLI prints `error: …` to
    stderr and exits non-zero BEFORE the relay runs → no partial cleave bundle.
  - Handoff, resume, and status all fail closed the same way.

## How the CLI mirrors MCP on the SAME store

The CLI is a new caller of the SAME `relay` Module + SAME project store — it does not
reimplement the relay:

| | Store | relay call |
|---|---|---|
| MCP `session_handoff` | `openService()` → CWD/explicit project handle | `relay.Handoff(ctx, h.Store().DB, h.ProjectID(), id, h.Root(), in)` |
| CLI `session handoff` | `openSessionService()` → `svc.Open(NormalizeID(proj))` / `svc.OpenForCWD()` | `relay.Handoff(ctx, h.Store().DB, h.ProjectID(), id, h.Root(), in)` |

Both resolve to the same SQLite file (`dataDir/<project>.sqlite`) and call the identical
`relay.Handoff/Resume/Status` with the identical arguments (`h.Store().DB`, `h.ProjectID()`,
`h.Root()`).

**How the test proves CLI + MCP agree on the same handoff** (`TestSessionMirrorsMCP`):
1. A fixture opens a store `dataDir/<proj>.sqlite` (proj = CWD-resolved id) — the SAME
   file both paths will use.
2. The REAL CLI subprocess (`go run . session handoff --handoff-id cli-parity-handoff …
   --project <proj> --dir <dataDir>`, CWD pinned to the package dir) writes the cleave
   bundle + the `session_handoffs` row. The test asserts 3 cleave files on disk and
   `COUNT(*) = 1` in the shared store.
3. The MCP `session_resume` handler is driven in-process (`mcp.SetService(service.New(dataDir))`
   + `MNEMONIC_PROJECT` + chdir) — it reads back the CLI-written row + bundle and returns
   the stored `NEXT_PROMPT`. The test asserts the prompt equals the CLI's `--next-prompt`.
4. The CLI `session resume` on the same store returns the SAME prompt + handoff_id as the
   MCP resume.
5. The MCP `session_status` and the CLI `session status` both report `handoff_count = 1`
   (the CLI count equals the MCP count).

So one handoff, one store, and both the CLI and MCP agree on the row, the cleave bundle,
the resume prompt, and the status count.

## How bad flags / missing id fail closed
- Flag parse error (unknown flag) → `fs.Parse` returns → exit 2, flag error on stderr,
  relay never runs.
- Missing required flag (`--progress` / `--next-prompt`) / missing resume id → exit 2 with
  a message naming the missing field, relay never runs.
- Unknown handoff id on resume → `relay.Resume` returns `unknown handoff id …` → exit 1,
  message on stderr.
- In every case no partial cleave bundle is left (the relay is not reached, or its
  `WriteBundle` rolls back a half-bundle and the row is only written after the files).

## How no-usable-store fails closed
`openSessionService` runs before any relay call. If the store cannot be resolved/opened
(e.g. `--dir` points at a file, or CWD resolution aborts on an ambiguous parent), it
returns an error → the CLI prints `error: <detail>` to stderr and exits 1. Because the
store never opens, `relay.Handoff` never runs and no cleave files are written (asserted in
`TestSessionNoStore`).

## Verification

### Full relevant suite
```
$ go test ./cmd/skillgrid/ ./internal/mnemonic/relay/ ./internal/mnemonic/mcp/ -count=1
ok  	github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid	34.484s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/relay	2.442s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp	40.163s
```

### 005/008/010/011/013 baselines
```
$ go test ./internal/mnemonic/memory/... ./internal/mnemonic/route/... ./internal/mnemonic/affected/... ./internal/mnemonic/community/... ./internal/mnemonic/pdg/... ./internal/mnemonic/codeindex/... -count=1
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory	9.068s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory/layer	3.478s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/route	3.116s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/affected	11.312s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/community	1.710s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg	4.830s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex	15.693s
```

### Focused (Session)
```
$ go test ./cmd/skillgrid/ -run 'TestSession' -count=1
ok  	github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid	14.188s
```
(All 5: TestSessionMirrorsMCP, TestSessionResumeMissingID, TestSessionResumeUnknownID,
TestSessionHandoffMissingFlag, TestSessionNoStore, TestSessionNoStoreResume.)

### RED→GREEN evidence (04.1)
- **RED** (with `session.go` present but the `main.go` dispatch removed):
  `go test ./cmd/skillgrid/ -run 'TestSessionMirrorsMCP' -count=1` →
  `FAIL: session handoff exited non-zero: 1 — error: unknown command "session" (see --help)`.
- **GREEN** (dispatch added): `go test ./cmd/skillgrid/ -run 'TestSessionMirrorsMCP' -count=1`
  → `PASS` (the CLI handoff/resume/status mirror the MCP handlers on the same store).

## Commits
- `0a86df3` — `feat(cli): session handoff resume status`
  (files: `cmd/skillgrid/session.go`, `cmd/skillgrid/session_test.go`,
  `cmd/skillgrid/main.go`, `internal/mnemonic/mcp/tools_session_export.go`)

## Global constraints held
- **Additive**: the MCP 82-tool surface is unchanged (the mcp package test still asserts
  `len(tools) == 82` and passes); the `relay` module is unchanged (the CLI is a new caller).
- **Session Relay only**: no Fact Memory / 004 / 003 / 002 / 005 / OpenCode plugin touched;
  `mem_session_*` (Engram) is untouched.
- **No watchdog** (step 05) and **no compact CLI subcommand** (MCP `knowledge_compact` only)
  were added.
- **Fail closed** on bad flags / missing id / no usable store (non-zero exit + stderr, no
  partial cleave bundle).

## Concerns
- **Project-store resolution**: the CLI resolves the store the same way MCP does — explicit
  `--project` → `svc.Open(NormalizeID(project))` (config root "."), else `svc.OpenForCWD()`
  (CWD-resolved). This means, without `--project`, the cleave bundle lands under the
  CWD's `.skillgrid/.cleave/` (the CWD is the resolved project root). In the test the CLI is
  run in its package dir, so the bundle lands in `skillgrid-cli/cmd/skillgrid/.skillgrid/.cleave/`
  (the test cleans it up with `t.Cleanup` so it does not leak). This matches the MCP
  behavior exactly (MCP `openService` is also CWD-resolved), so the parity is intentional.
- **`--archive` on resume**: exposed as `session resume <id> --archive`, mirroring the MCP
  `session_resume` `archive` boolean. The MCP path returns `archive_id` when `archive` is
  true; the CLI mirrors that (returns `archive_id` when `--archive` is set). This is
  additive and matches MCP; no new store behavior.
- **Test subprocess cost**: the Session tests shell out to `go run .` (compile + run) per
  invocation, so the `cmd/skillgrid` package takes ~34s. This matches the existing
  `mem`/`trail` CLI-test pattern in this repo, so it is consistent with the codebase.