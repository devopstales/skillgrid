# Task 21 — Handoff: Review (FRESH reviewer)

**Verdict:** `PASS WITH WARNINGS`
**Approved to merge:** yes

Scope judged: single step-21 commit `90a6253` (5 files, +786/−1). Diff, brief,
report, and on-disk code all agree. All 4 tests re-run green locally.

---

## Findings

### 1. [major] `SaveHandoff` double-generates (and double-advances the cursor)

`SaveHandoff` calls `GenerateHandoff` internally, but the CLI (`runMemHandoff`)
calls `GenerateHandoff` first to build the summary, then calls `SaveHandoff`
which calls `GenerateHandoff` again.

- `mem.go:895` `artifact, err := h.Memory().GenerateHandoff(hCtx(), cwd)`
- `mem.go:904` `if err := h.Memory().SaveHandoff(hCtx(), cwd, outDir); err != nil {`
- `handoff.go:202-203` `func (s *Service) SaveHandoff(...) { h, err := s.GenerateHandoff(...) }`

Consequences:
- **Cursor advanced twice.** Each `GenerateHandoff` writes a new cursor
  (`handoff.go:194`). Two generations in one CLI run advance the cursor by the
  gap between the two `time.Now()` calls. Any file edited in that tiny window is
  missed by the *next* handoff's delta. Negligible in practice, but the cursor
  no longer means "time of this handoff" — it means "time of the last of two
  invisible generations."
- **The printed summary and the written file can diverge.** The summary
  (`--json` excluded) is built from the *first* generation's `artifact`, but the
  file on disk is the *second* generation's. A file edited between the two calls
  changes the delta → the stdout counts and the file's `changed_files` no longer
  match.

The `--json` path avoids the second generation (returns after `printJSON`).
Only the default summary path double-generates.

**Fix (pick one):** have the CLI call a save-that-reuses-the-already-generated
artifact, e.g. add `SaveHandoffArtifact(ctx, h, outDir) error` and call it; or
make `runMemHandoff` call only `SaveHandoff` and derive the summary from the
returned `*Handoff`. This also removes the wasted second on-disk hub/stub read.

### 2. [major] First handoff's delta is always empty (no baseline)

`handoff.go:141-148`:
```go
changed := map[string]bool{}
if !cursor.IsZero() {
    for _, f := range files {
        if isHubFile(f.path) || fileModifiedSince(...) {
            changed[f.path] = true
        }
    }
}
```
When no cursor exists (first handoff for a project), `cursor.IsZero()` is true
→ `changed` stays empty → **delta has no changed files and no risk files.** The
first handoff's delta carries only recent events + working set. So a brand-new
agent receiving the very first handoff sees a completely empty change-set with
no "baseline state." This is consistent (documented in the report) but weaker
than the brief's intent ("delta contains changed file stubs ... computed at
handoff time"). On the first run there is nothing "since the last handoff," but
a baseline of hub files + a note would be more useful. Minor design gap; not a
defect. Note: hub files are still surfaced in the *prefix*, so they are not
invisible — just absent from `delta.changed_files`/`risk_files` on run 1.

### 3. [major] Hub files are *always* in the delta (not only "risk when changed")

`handoff.go:144` `if isHubFile(f.path) || fileModifiedSince(...)`. Every
generation, every hub file is appended to `changed_files` and flagged in
`risk_files`, regardless of whether it changed since the cursor. This is stated
as intentional ("a hub change is always visible"), and it is what makes
`TestHandoffDeltaDynamic`'s third-generation assertion hold (only the hub
remains). But it means `risk_files` is a standing list of all hub files, not a
signal that a hub file *changed*. For a repo with several hub files the delta
will perpetually list them, diluting the "risk" meaning. Acceptable given the
design choice, but the brief's wording ("risk files (hub files among changes)")
implies risk = hub files *that changed*. The implementation is hub files *period*.

### 4. [minor] Stale comment: `kv_meta` vs actual `handoff_meta`

`handoff.go:25` and `handoff_test.go:43` say the cursor is persisted "into the
store's kv_meta table," but the actual table is `handoff_meta`
(`031_handoff.sql`, `handoff.go:307`/`328`). Cosmetic — the code is correct, the
prose references the wrong table name.

### 5. [minor] `isHubFile` basename/suffix semantics are looser than the report claims

`handoff.go:252-260`:
```go
if rel == pat || strings.HasSuffix(rel, "/"+pat) {
```
The comment (`handoff.go:36`, report) says "matches by basename or path suffix,"
and the report's own prose says "equals a pattern or has it as a `/`-suffixed
tail." These agree with the code. However, a file like `cmd/main.go` is a hub
(file under `cmd/`), while `cmd/main.go.bak` is not, and `sub/main.go` *is* a
hub even though it is not the root entrypoint (the `main.go` pattern has no
leading `/`, so any `.../main.go` matches). So *every* `main.go` in the tree is
treated as a hub — in a repo with generated or vendored `main.go` files this
over-includes. Deterministic, but broader than "the entry point." Document the
intention or tighten to anchored roots.

### 6. [minor] Prefix `FileCount` includes non-hub files → any add/delete changes the prefix

`handoff.go:127` `prefix := HandoffPrefix{FileCount: len(files)}`. The brief's
"STABLE (only changes when hub files change)" is qualified in the implementation
to "hub files **or the file count**." Adding or deleting *any* non-hub file
changes `FileCount` and thus the prefix. This is defensible (the repo
population is legitimately part of "stable metadata") and is exactly what
`TestHandoffPrefixStable` asserts (it edits a non-hub file's *content* without
changing the count, so the prefix holds). But the stability guarantee is
"stable across non-hub *edits*," not "stable across non-hub *additions*." Worth
one line in the brief/report to avoid over-reading the guarantee.

### 7. [nit] `readHandoffCursor` swallows a parse error as "no cursor"

`handoff.go:317-321` returns `(time.Time{}, nil)` on a malformed timestamp. A
corrupt cursor silently degrades to "first handoff" (empty delta) rather than
surfacing an error. Acceptable for a best-effort artifact, but a log would help
debugging.

### 8. [nit] Clock-skew window acknowledged, not guarded

`fileModifiedSince` uses `info.ModTime().After(cursor)` (strict `After`). A file
whose mtime equals the cursor exactly is not flagged. The report correctly calls
this out (concern #2) and notes nano precision makes the window negligible. No
code guard (epsilon) — acceptable.

---

## Evidence checked

**1. Stable prefix (21.1) — CRITICAL: HOLD.**
- Prefix computed from `HandoffPrefix{FileCount: len(files)}` + per-hub
  `readHeadSummary` (`handoff.go:127-137`). Inputs: the codeindex `files` row
  set (count) and the on-disk head of each hub file. No mtime, no cursor, no
  time — so it is stable across non-hub *edits*.
- **Hub definition:** fixed pattern list `hubFilePatterns` (`handoff.go:37-45`),
  matched by `isHubFile` (exact or `/`-suffixed tail). Not codeindex importance.
  Deterministic.
- **Test is real:** `TestHandoffPrefixStable` writes a hub + non-hub file,
  generates (`h1`), edits the *non-hub* file, regenerates (`h2`), asserts
  `FileCount` and every hub `Path`/`Summary` are identical; then edits the *hub*
  file, regenerates (`h3`), asserts the hub `Summary` changed. Genuine
  modify-non-hub→unchanged / modify-hub→changed flow. PASS.

**2. Changed-file detection (21.2) — CRITICAL: HOLD (with design caveats #2/#3).**
- Cursor persisted in `handoff_meta` (key `handoff:cursor`) and advanced on
  every `GenerateHandoff` (`handoff.go:324-332`, called at `:194`).
- Delta = hub files ∪ files with `os.Stat(...).ModTime().After(cursor)`
  (`handoff.go:141-148`, `fileModifiedSince` `:340-346`). Real FS mtime, not the
  codeindex `mtime_ns` (documented rationale: an edit doesn't re-index).
- **Test is real:** `TestHandoffDeltaDynamic` seeds 4 files + 12 events; first
  handoff → empty delta (no cursor); sleeps; modifies 1 hub + 2 regular, leaves
  1 unchanged; second handoff → delta lists all 3 with non-empty stubs, excludes
  the unchanged file, hub flagged in `risk_files`, `recent_events == 10` (last 10
  of 12), working-set summary non-empty; third handoff with no further change →
  only the hub remains. PASS.
- **Clock skew / past mtime:** strict `After` comparison → a file whose mtime is
  *before* the cursor is not flagged (correct); an mtime *in the past relative to
  the cursor* is also excluded (a re-touched file with backdated mtime would be
  missed — inherent to mtime, documented). Nano precision makes the
  same-second false-positive negligible.

**3. Delta freshness (21.2): HOLD.** Computed at generation time (re-queries DB +
re-reads disk each `GenerateHandoff`), not cached. Contains all four required
parts: changed stubs (path + first 50 lines via `readHeadSummary(...,stubLines)`
`stubLines=50`), risk files, last-10 recent session events
(`recentSessionEvents`, `LIMIT 10`, `deleted_at IS NULL`), working-set summary.

**4. `031_handoff.sql` migration: HOLD.** Purely additive —
`CREATE TABLE IF NOT EXISTS handoff_meta (key TEXT PRIMARY KEY, value TEXT NOT
NULL);`. Two columns, no `updated_at` (the cursor value itself is the timestamp;
no separate `updated_at` needed and none is required by the brief). No existing
schema touched.

**5. File output (21.3): HOLD.** `SaveHandoff` writes
`filepath.Join(outDir, "handoff.latest.json")` (`handoff.go:214`), `MkdirAll`
first, `WriteFile` (truncate/overwrite, 0o644), 2-space-indented JSON. Default
`outDir` = the project's mnemonic data dir (report: `~/.skillgrid/mnemonic` via
`service.DefaultDataDir`; brief's `.skillgrid/handoff.latest.json` is the same
concept). Path printed in the CLI summary (`mem.go:910`
`memory.HandoffPath(outDir)`). `TestHandoffSavedToFile` confirms valid JSON with
`prefix`/`delta`/`generated_at` and overwrite-not-append. PASS.

**6. CLI (21.4): HOLD.** `mem handoff` → `runMemHandoff` (`mem.go:891`) opens the
project, generates, writes the artifact, prints a JSON summary (project, path,
generated_at, prefix counts, delta counts). `--json` prints the full `Handoff`.
Errors (open fail, GenerateHandoff fail, SaveHandoff fail) → stderr + `os.Exit(1)`.
"no project" is handled by `svc.Open` failing → exit 1. `TestMemHandoffCLI`
exercises default + `--json`, asserts path + project id + `prefix`/`delta` in
output. PASS. (Note finding #1: the default path double-generates.)

**7. Additive-ness / tool contract: HOLD.** `git show 90a6253` on `mem.go` =
73 changed lines, exactly **1 removal**, and that removal is a
1-space doc-whitespace reflow in `printMemUsage`
(`newest-first (id, state_hash, created_at)` indentation) — no behavior change.
No `mem_*`/`code_*`/`web_*` MCP tool signature changed; `GenerateHandoff`/
`SaveHandoff`/`HandoffPath` are new symbols, `handoff` is a new subcommand, and
`--json`/`--handoff-dir` did not pre-exist (verified against `90a6253~1`). New
file, new test file, new table, additive subcommand + two new flags. No CGO
(`modernc.org/sqlite` unchanged).

**8. TDD discipline: HOLD (with the one gap noted in #2).** Four tests map to
the four sub-tasks: (a) `TestHandoffPrefixStable` → stable prefix, (b)
`TestHandoffDeltaDynamic` → dynamic delta, (c) `TestHandoffSavedToFile` → file
output, (d) `TestMemHandoffCLI` → CLI. All four re-run green:
```
--- PASS: TestHandoffPrefixStable (0.28s)
--- PASS: TestHandoffDeltaDynamic (0.23s)
--- PASS: TestHandoffSavedToFile (0.22s)
--- PASS: TestMemHandoffCLI (2.42s)
```
RED evidence is in the report (build-fail before `handoff.go` existed). The
`TestHandoffDeltaDynamic` third-generation assertion depends on the
always-include-hub behavior (finding #3) — it tests the design as built, which is
fine, but it encodes the standing-hub-list assumption.

---

## Bottom line

Both CRITICAL guarantees hold as implemented and are genuinely tested:
- **Stable prefix** — derived only from file count + hub-file heads; a non-hub
  content edit leaves it byte-identical, a hub edit changes it. The one
  qualification (file *count* changes the prefix too) is by design.
- **Changed-file detection** — on-disk mtime vs a persisted RFC3339Nano cursor
  advanced each generation; first handoff is empty (baseline gap), hub files are
  always included. Correct and cross-process.

Merge blockers: none. The warnings to carry forward are finding #1 (double
generation in the default CLI path — a real, if small, cursor/consistency
inconsistency worth a follow-up task) and the design caveats #2/#3 (empty
baseline delta; risk = all hub files, not hub files changed). Recommend
documenting #6's stability qualification in the brief and fixing the stale
`kv_meta` comment (#4).
