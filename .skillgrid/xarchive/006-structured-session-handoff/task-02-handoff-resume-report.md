# Task 02 — handoff-resume: Implementation Report

**Status:** Implemented. All 02.x sub-tasks done; focused + baseline suites green.

## What was implemented per sub-task

### 02.1 `[RED]` Mnemonic tool surface (session_handoff/session_resume + mem_* intact)
- **RED test** `skillgrid-cli/internal/mnemonic/mcp/tools_session_handoff_test.go`:
  - `TestSessionHandoffTools` — asserts `session_handoff` + `session_resume` are
    registered, the tool surface grows additively **78 → 80**, and every 005
    `mem_*` tool keeps its name + required params (reuses the existing
    `expectedMemToolSurface` baseline lock). Also asserts bad-arg validation
    (no `progress` on handoff, no `handoff_id` on resume).
  - `TestSessionToolsDispatch` — end-to-end: real `mem_session_start` +
    `mem_save` handlers dispatch, then a real `session_handoff` MCP call writes
    the three `.cleave/` files + a row, and a follow-up `session_resume` MCP
    call returns the stored `NEXT_PROMPT`.
  - `TestMemSaveDispatch` — direct proof `mem_save` still dispatches and
    returns an id once the session tools are registered.
- **Implementation** `tools_session_handoff.go`:
  - `registerSessionTools(s)` — the **registrar hook** (step 03 extends this
    list without re-editing `server.go`'s chain). Registered in both `Start()`
    and `NewServer()` in `server.go`.
  - `handleSessionHandoff` → opens the CWD project handle once (`openService`),
    builds a `relay.Bundle`, calls `relay.Handoff`, returns `{handoff_id, paths}`.
  - `handleSessionResume` → calls `relay.Resume`, returns
    `{prompt, handoff_id, archive_id?}` (archive_id only when `archive=true`).
- **Surface lock updated 78 → 80** (additive) in the six existing lock tests:
  `server_test.go` (TestAllToolsRegistered `want` list), governance,
  code_knowledge, code_affected, memory_layer, memory_layer_args,
  memory_retrieval. `mem_*` names + required params unchanged.

**RED→GREEN evidence (02.1):**
- With the registration removed (temporarily), the focused set FAILED:
  ```
  --- FAIL: TestSessionHandoffTools
      tools_session_handoff_test.go:39: expected tool "session_handoff" to be registered
      tools_session_handoff_test.go:39: expected tool "session_resume" to be registered
      tools_session_handoff_test.go:45: expected 80 tools (78 baseline + 2 session), got 78
  ```
- With the registration present, the same focused set PASSES:
  ```
  go test ./internal/mnemonic/mcp/ -run 'SessionHandoff|SessionTools|MemSave' -count=1
  ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp
  ```

### 02.2 `[AFK]` cleave.go — WriteBundle / ReadBundle
- `skillgrid-cli/internal/mnemonic/relay/cleave.go`:
  - `Bundle` struct: `Progress` / `Knowledge` / `NextPrompt` (+ optional
    `SourceSession`, `ContextSummary`).
  - `WriteBundle(projectRoot, sessionID, b)` — writes the three
    `.skillgrid/.cleave/{PROGRESS,KNOWLEDGE,NEXT_PROMPT}.md` files and returns
    the three absolute paths. **Fail closed:** a write failure removes any
    already-written files (no half-bundle).
  - `ReadBundle(projectRoot)` — reads the three files; a missing `.cleave/` or
    any missing file returns a clear error (no invented content).
  - `BundleDir(projectRoot)` — absolute `.skillgrid/.cleave` path.

### 02.3 `[AFK]` relay.go — Handoff / Resume (fail closed, no orphan rows)
- `skillgrid-cli/internal/mnemonic/relay/relay.go`:
  - `Handoff(ctx, db, projectID, handoffID, projectRoot, b) (handoffID, paths, error)` —
    **writes the cleave files FIRST, then the `session_handoffs` row.** If the
    file write fails, no row is inserted (and WriteBundle rolled back the
    half-bundle). If the row insert fails after the files were written, the
    files are removed (covers the other orphan direction). Blank `handoffID`
    is generated from source session + timestamp. Empty `next_prompt` fails
    before any file/row (resume must not invent a prompt).
  - `Resume(ctx, db, projectID, handoffID, projectRoot, archive) (prompt, id, archiveID, error)` —
    looks up the handoff row (unknown id → clear "unknown handoff id" error),
    reads the cleave bundle (missing/incomplete → clear error, no invented
    prompt), returns `NEXT_PROMPT`. With `archive=true` it inserts a
    `session_archives` row (FK-safe: `source_session` scanned as `sql.NullString`)
    and flips `session_handoffs.status` to `archived`, returning the archive id.
  - `HandoffCount(ctx, db, projectID)` — for step 03 `session_status`.
  - `Store` interface (`ExecContext`/`QueryRowContext`) — `*sql.DB` satisfies it;
    the relay only touches the two 019 tables, never sessions/observations.
- `service.go`: added `ProjectHandle.Root()` accessor (the workspace root that
  owns `.skillgrid/`) so the MCP handler can locate the cleave tree.

### 02.4 `[AFK]` .gitignore
- `.gitignore`: added `.skillgrid/.cleave/` (gitignored by default, like 003 L0).

### 02.5 `[AFK]` happy path
- `relay_test.go`: `TestHandoff` (files + row + round-trip), `TestResume`
  (returns NEXT_PROMPT; archive path records `session_archives` + flips status).
- MCP `TestSessionToolsDispatch` (end-to-end happy path at the tool boundary).

### 02.6 `[AFK]` edge
- `relay_test.go`: `TestResumeMissing` (known id, deleted `.cleave/` → clear
  error, prompt empty), `TestResumeUnknown` (never-recorded id → "unknown
  handoff id" error), `TestHandoffFailClosedNoOrphan` (read-only `.skillgrid`
  dir → file write fails → **0 rows**), `TestHandoffRequiresNextPrompt`,
  `TestReadBundleMissing`, `TestSoftOptionalL0` (handoff succeeds with no L0
  tree; mirrors into `workspace/sessions/{id}/` when present).

## Fail-closed no-orphan invariant

Enforced by **file-write-before-row**: `relay.Handoff` calls `WriteBundle`
first; only after it returns the three paths does it `INSERT` the
`session_handoffs` row. A file-write failure returns before the INSERT, so no
row exists. The reverse direction (row written, then crash) is also covered: a
row-insert failure removes the just-written files. **Proof test:**
`TestHandoffFailClosedNoOrphan` makes `.skillgrid` read-only (0555) so the
`.cleave` mkdir/write fails, asserts `Handoff` returns an error **and** that
`SELECT COUNT(*) FROM session_handoffs` is `0`.

## How Resume errors on missing .cleave/ / unknown id (no invented prompt)

- Unknown id: `Resume`'s `SELECT ... WHERE project=? AND handoff_id=?`
  returns `sql.ErrNoRows` → `relay: unknown handoff id %q for project %q`.
- Missing bundle: `ReadBundle` returns `relay: missing cleave file %s (no .cleave/ bundle)`.
- In both cases the returned prompt is `""`. **Proof:** `TestResumeMissing`
  deletes the `.cleave/` dir, asserts error + empty prompt + "cleave" in the
  message; `TestResumeUnknown` asserts the "unknown handoff id" message.

## How soft-optional L0 degrades

`.skillgrid/workspace/sessions/{id}/` is read-only-folded in
`cleave.mirrorL0`: `os.Stat` the dir; if absent → no-op (handoff succeeds).
If present → best-effort copy of the three files; any mirror error is
swallowed. **Proof:** `TestSoftOptionalL0` — handoff with no L0 tree succeeds;
handoff with the L0 tree present mirrors all three files into it.

## Full relevant suite + output

```
$ go test ./internal/mnemonic/relay/ ./internal/mnemonic/mcp/ -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/relay   2.078s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp    35.477s
```

### Baselines (005/008/010/011/013) still pass
```
$ go test ./internal/mnemonic/memory/... ./internal/mnemonic/route/... ./internal/mnemonic/affected/... ./internal/mnemonic/community/... ./internal/mnemonic/pdg/... ./internal/mnemonic/codeindex/... -count=1
ok  .../mnemonic/memory        9.521s
ok  .../mnemonic/memory/layer  3.268s
ok  .../mnemonic/route         1.469s
ok  .../mnemonic/affected     11.908s
ok  .../mnemonic/community     3.062s
ok  .../mnemonic/pdg           4.255s
ok  .../mnemonic/codeindex    16.011s
```
Plus `store` + `service` (schema migration + handle Root accessor):
```
ok  .../mnemonic/store         3.542s
ok  .../mnemonic/service      21.690s
```

## Commits

- `29feb6e` feat(mnemonic): add relay cleave bundle and handoff/resume (fail closed)
- `b72ecb8` feat(mnemonic): register session_handoff and session_resume (78→80 surface)
- `174eeb4` chore(sdd): gitignore .skillgrid/.cleave/ by default

## Concerns / decisions

1. **Resume archives (does it, not step 04).** `Resume` accepts `archive`
   (default false) and records the `session_archives` row + flips status when
   true. Step 04 (CLI) can drive the same flag. If the team prefers resume to
   NEVER archive and leave it to an explicit archive step, flip the default to
   false (already false) and drop the archive call site — the prompt return is
   independent of archiving either way.
2. **Cleave file formats** are plain markdown: `PROGRESS.md` (what was done),
   `KNOWLEDGE.md` (decisions/learnings), `NEXT_PROMPT.md` (the resume seed).
   No header/templating beyond the raw caller-provided body — `NEXT_PROMPT.md`
   is returned verbatim on resume. Step 03 `knowledge_compact` will refresh
   `KNOWLEDGE.md`.
3. **cleave_path is stored repo-relative** as `.skillgrid/.cleave` (a constant),
   not per-handoff. `handoff_id` is the resume key; the bundle is a single
   shared `.cleave/` dir per project (latest handoff wins on disk, matching
   "latest or named handoff" resume). Step 03 status/compact can extend this.
4. **FK on source_session:** `session_handoffs.source_session` references
   `sessions(id) ON DELETE SET NULL`, so a handoff's `source_session` must be
   an existing session id (or blank → NULL). Tests seed the session row.
5. Pre-existing uncommitted change `docs/skillgrid/archive/013-.../tasks.md`
   (013 archive state bump) was left untouched — not part of step 02.

## Review finding fix — archive status-update no longer swallowed

**Finding (MINOR but real):** In the `Resume(archive=true)` path, the follow-up
`UPDATE session_handoffs SET status='archived'` was swallowed with `_, _ =`. A
transient failure there left a `session_archives` row written but the handoff
still `pending` — the reverse of the fail-closed orphan this change cares about.

**Fix (consistent fail-closed):** `relay.go` now surfaces a failed status flip
instead of swallowing it. After the `session_archives` INSERT succeeds, the
`UPDATE ... SET status='archived'` error is checked; on failure `Resume` returns
`relay: archived but failed to mark handoff archived: <err>` (the archive row
was written but the handoff status flip failed). The caller sees the incomplete
archive rather than a silently-inconsistent state. No schema/tool-surface
change; `archive` still defaults to false.

**Tests:**
- `TestResume` (existing, now the must-have happy-path assertion) — already
  asserts `Resume(archive=true)` flips the handoff status to `archived` AND the
  `session_archives` row exists. Confirmed green.
- `TestResumeArchiveStatusFlipFails` (new) — drives `Resume(archive=true)`
  through the real path with a store wrapper (`failOnUpdate`) that fails only
  the `UPDATE session_handoffs SET status` query. Asserts: `Resume` returns an
  error mentioning "failed to mark handoff archived"; the `session_archives`
  row was written (count 1, before the flip); and the handoff is still
  `pending` — proving the inconsistency is surfaced, not hidden by `_, _ =`.

**Relevant suite + output:**
```
$ go test ./internal/mnemonic/relay/ ./internal/mnemonic/mcp/ -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/relay   2.031s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp    39.967s
```
**Baselines (005/008/010/011/013) still pass:**
```
$ go test ./internal/mnemonic/memory/... ./internal/mnemonic/route/... ./internal/mnemonic/affected/... ./internal/mnemonic/community/... ./internal/mnemonic/pdg/... ./internal/mnemonic/codeindex/... -count=1
ok  .../mnemonic/memory        12.555s
ok  .../mnemonic/memory/layer   1.876s
ok  .../mnemonic/route          2.258s
ok  .../mnemonic/affected       9.797s
ok  .../mnemonic/community      3.120s
ok  .../mnemonic/pdg            4.678s
ok  .../mnemonic/codeindex     16.224s
```

**Commit:** see step 02 fix commit (fail-closed archive status flip).
