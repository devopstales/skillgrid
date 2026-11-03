# Task 21 — Handoff: Prefix + Delta Artifacts

## Status

PASS. All four sub-tasks (21.1, 21.2, 21.3, 21.4) implemented and green. One
pre-existing unrelated failure in the `service` package
(`TestReindexStructuralIsEmbedderFree`) is not caused by this change — it fails
on a clean `git stash` of HEAD too.

## Commits

- `90a6253` `feat(mnemonic): prefix + delta handoff artifacts for agent-to-agent context`

## What was implemented per sub-task

### 21.1 [RED] Stable prefix with hub summaries + repo file count
- `TestHandoffPrefixStable`: generate handoff → verify `prefix.hub_files` has
  summaries and `prefix.file_count`; modify a non-hub file → prefix unchanged;
  modify a hub file → prefix reflects the change.
- `GenerateHandoff` builds the prefix from the codeindex `files` table (the
  stable project metadata) plus on-disk reads of the hub files.

### 21.2 [RED] Dynamic delta with changed stubs, risk files, recent events
- `TestHandoffDeltaDynamic`: generate; modify 3 files (1 hub, 2 regular);
  regenerate → delta lists all 3 changed files with stubs; hub file flagged as
  a risk file; recent session events (last 10 of 12 seeded) included; delta
  reflects only changes since the last handoff (a no-change regeneration drops
  the regular files, keeping only the always-present hub file).

### 21.3 [AFK] Handoff saved to `handoff.latest.json`
- `TestHandoffSavedToFile`: `SaveHandoff` writes `handoff.latest.json` into the
  target dir; JSON is valid and contains `prefix` + `delta` + `generated_at`; a
  second generation overwrites (still a single valid JSON object, not appended).

### 21.4 [AFK] `mem handoff` CLI
- `TestMemHandoffCLI`: `mem handoff` prints a summary (project id + artifact
  path) and writes `handoff.latest.json`; `mem handoff --json` prints the full
  JSON (both `prefix` and `delta` sections).

## The Handoff struct

```go
type Handoff struct {
    Prefix      HandoffPrefix `json:"prefix"`
    Delta       HandoffDelta  `json:"delta"`
    GeneratedAt time.Time     `json:"generated_at"`
    ProjectID   string        `json:"project_id"`
}

type HandoffPrefix struct {
    HubFiles  []HandoffHubFile `json:"hub_files"`
    FileCount int              `json:"file_count"`
}

type HandoffHubFile struct {
    Path    string `json:"path"`
    Summary string `json:"summary"`
}

type HandoffDelta struct {
    ChangedFiles []ChangedFileStub `json:"changed_files"`
    RiskFiles    []string          `json:"risk_files"`
    RecentEvents []RecentEvent     `json:"recent_events"`
    WorkingSet   WorkingSetSummary `json:"working_set"`
}

type ChangedFileStub struct {
    Path string `json:"path"`
    Stub string `json:"stub"`
}

type RecentEvent struct {
    ID        int64  `json:"id"`
    Title     string `json:"title"`
    CreatedAt string `json:"created_at"`
}

type WorkingSetSummary struct {
    ChangedCount int    `json:"changed_count"`
    RiskCount    int    `json:"risk_count"`
    Summary      string `json:"summary"`
}
```

## How the stable prefix is computed

- **Hub file list**: a fixed pattern list `hubFilePatterns`
  (`AGENTS.md`, `CLAUDE.md`, `go.mod`, `package.json`, `main.go`,
  `cmd/main.go`, `internal/main.go`). A codeindex path is a hub file when its
  slash-normalized path equals a pattern or has it as a `/`-suffixed tail.
- **Hub file summaries**: for each hub file found in the codeindex `files`
  table, the first `hubSummaryLines` (20) lines of the on-disk file are
  captured (joined with newlines). A missing/unreadable file yields `""`
  (best-effort; the handoff never fails on one file).
- **Repo file count**: `COUNT` of codeindex `files` rows (excluding the
  `session:<id>` pseudo-files that carry no on-disk content).
- **Stability**: the prefix is derived only from the codeindex file population
  and hub file contents. It changes only when a hub file changes or the file
  count changes — a non-hub file edit leaves it byte-identical (verified by
  `TestHandoffPrefixStable`).

## How the dynamic delta is computed

- **Changed-file detection** (the tricky part, documented): the memory package
  cannot depend on the service layer (import cycle), and a fresh CLI process
  cannot rely on in-process state. So a **last-handoff cursor** (RFC3339Nano)
  is persisted in a new additive `handoff_meta` table (migration
  `031_handoff.sql`, key `handoff:cursor`) on every generation. The delta
  includes a codeindex file when **it is a hub file OR its on-disk mtime is
  strictly after the cursor** (`os.Stat(...).ModTime().After(cursor)`).
  - **Why on-disk mtime, not `files.mtime_ns`**: the codeindex table only
    advances when the indexer runs; an agent editing files does not re-index.
    Real filesystem mtime reflects edits immediately, with no re-index.
  - **Why the cursor is RFC3339Nano**: RFC3339 (second precision) truncates
    sub-second time, so an unchanged file written in the same second as the
    generation could falsely appear modified. Nano precision fixes it.
  - **First handoff**: no cursor → empty delta (nothing "changed since never").
  - **Hub files are always included** in the delta (even if their mtime
    predates the cursor) so a hub change is always visible to the next agent.
- **Stubs**: first `stubLines` (50) lines of each changed file's on-disk content.
- **Risk files**: hub files among the changed files (flagged).
- **Recent events**: last 10 `session_log` observations for the project
  (newest first, `deleted_at IS NULL`) — the "recent session events" from step
  09.
- **Working set summary**: counts + a one-line human summary.
- Computed **at handoff time, not cached** — each `GenerateHandoff` re-queries.

## Where the artifact is written

- `SaveHandoff(ctx, repoDir, outDir)` writes
  `filepath.Join(outDir, "handoff.latest.json")` (MkdirAll, overwrite, 0o644,
  2-space-indented JSON). Default `outDir` = the mnemonic data dir
  (`~/.skillgrid/mnemonic` via `service.DefaultDataDir`, or `--dir` /
  `--handoff-dir`). `HandoffPath(outDir)` is exported for callers.

## CLI (21.4)

`mem.go` gained a `handoff` subcommand to the manual dispatcher:
- `mem handoff` → opens the project handle, generates the handoff for the CWD,
  writes `handoff.latest.json` under the data dir, prints a JSON summary
  (project, path, generated_at, prefix `{file_count, hub_files}`, delta
  `{changed_files, risk_files, recent_events, working_set}`).
- `mem handoff --json` → prints the full `Handoff` JSON instead of a summary.
- `mem handoff --handoff-dir DIR` → writes the artifact to DIR instead of the
  data dir.
- Usage text updated. No existing `mem_*` / `code_*` / `web_*` tool contract
  changed; the subcommand is additive.

## Global constraints

- **Additive only**: new `handoff.go`, new `handoff_test.go`, new
  `031_handoff.sql` (a new `handoff_meta` table — no existing schema touched),
  new `mem_handoff_test.go`, additive `mem.go` subcommand + two new flags.
- **No tool contract shape change**: `GenerateHandoff` / `SaveHandoff` are new
  `*memory.Service` methods; `mem.go` routes through `h.Memory()`.
- **No CGO**: still `modernc.org/sqlite` (the store already uses it).

## RED → GREEN evidence

- **RED (21.1/21.2/21.3)**: first run before `handoff.go` existed →
  `[build failed]` (`undefined: Handoff`, `GenerateHandoff undefined`,
  `SaveHandoff undefined`, `undefined: HandoffHubFile`).
- **GREEN**: all three pass after implementation:
  ```
  --- PASS: TestHandoffPrefixStable
  --- PASS: TestHandoffDeltaDynamic
  --- PASS: TestHandoffSavedToFile
  ```
- **RED → GREEN (21.4)**: `TestMemHandoffCLI` compiled against the new
  `runMemHandoff` and passes (`--- PASS: TestMemHandoffCLI`).

## Full relevant suite + output

```
$ go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory	36.059s

$ go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemHandoffCLI' -count=1 -v
--- PASS: TestMemHandoffCLI (4.17s)
ok  	github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid	6.510s
```

### 005/008/010/011/013 baselines

```
$ go test ./skillgrid-cli/internal/mnemonic/store/... ./skillgrid-cli/internal/mnemonic/service/... \
    ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/route/... \
    ./skillgrid-cli/internal/mnemonic/affected/... ./skillgrid-cli/internal/mnemonic/community/... \
    ./skillgrid-cli/internal/mnemonic/pdg/... -count=1
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store		5.977s
--- FAIL: TestReindexStructuralIsEmbedderFree (1.08s)   [PRE-EXISTING — fails on clean HEAD]
FAIL	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service	58.698s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex	16.451s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/route		3.168s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/affected	14.186s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/community	2.547s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg		6.982s
```

The single failure is `TestReindexStructuralIsEmbedderFree`
(`reindex_structural_test.go:107: count gamma: sql: database is closed`). It is
**pre-existing and unrelated**: it reproduces identically after `git stash` of
all step-21 changes (run against clean HEAD). It is a pool/double-open
interleaving issue in the reindex test, not a handoff regression.

## Concerns

- The hub file list is a **fixed pattern set**, not derived from codeindex
  importance data (the brief allowed either; the fixed list was chosen for
  simplicity and determinism). A monorepo with a differently-named entry point
  (e.g. `app.go`, `src/index.ts`) would not be treated as a hub file.
- Changed-file detection uses **on-disk mtime vs the persisted cursor**. A
  file edited within the same sub-second window as the cursor write, or whose
  mtime is clock-skewed, could be missed or double-counted; in practice the
  cursor is written at generation and edits happen between generations, so the
  window is not an issue. If sub-millisecond edits become common, the cursor
  comparison could be tightened with a small epsilon.
- `TestReindexStructuralIsEmbedderFree` (service package) is flaky/failing on
  clean HEAD — worth a separate look, but out of scope for step 21.
