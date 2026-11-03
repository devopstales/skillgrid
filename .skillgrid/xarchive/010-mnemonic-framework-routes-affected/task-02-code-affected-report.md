# 02-code-affected — implementation report

## Per sub-task

### 02.1 `code_affected` traversal (RED → GREEN)
- Created `skillgrid-cli/internal/mnemonic/affected/affected.go`: `Affected(ctx, db, Options)` — depth-capped reverse BFS from the changed files' symbols over the stored edges (`calls`, `imports`, `reference`, `route`, `extends`, `implements`, `references`, `navigates`, `tests_for`, `tested_by`). Only **stored** edges are walked (step-01 drop policy: a dropped reference has no edge, so it contributes nothing). Reports `TestFiles` (deduped, sorted), `Relationships` (each hop: from/to/kind/confidence/depth/dep path), `Depth`, `Filter`. Empty changed set → empty result + clear message (not an error). A changed set whose paths match no indexed symbol → clear "not indexed" message (no invented edges).
- `--depth` (default 5), `--filter` (substring/segment match on reported test files).
- `affected_test.go`: `TestCodeAffected` (transitive chain, depth cap, filter), `TestCodeAffectedEmpty`, `TestCodeAffectedNoInventedEdges`.
- RED: `go test ./skillgrid-cli/internal/mnemonic/affected/... -run CodeAffected -count=1` → `undefined: Affected`, `undefined: Options` (compile FAIL). GREEN: PASS.

### 02.2 `--stdin` (RED → GREEN)
- `AffectedFromStdin` + `ParseFileList` in `affected.go` (newline file list, CRLF/blank tolerant; empty list = empty result + message, not an error).
- CLI: `cmd/skillgrid/search.go` dispatches `search affected` / `search rename` BEFORE the shared search flag-set parses (so affected's own flags don't leak); new `cmd/skillgrid/search_affected_cli.go` — `runSearchAffected` with `--stdin` / `FILES...` / `--base` / `--depth` / `--filter` / `--json` / `--quiet`.
- `cmd/skillgrid/search_affected_test.go`: `TestCodeAffectedStdinCLI` (piped diff list → affected test file; empty stdin → clear message).
- RED: `... -run CodeAffectedStdin` → `undefined: AffectedFromStdin` + `undefined: runSearchAffected` (compile FAIL). GREEN: PASS.

### 02.3 `--base <ref>` areas + owners (RED → GREEN)
- `affected/base.go`: `AffectedBase(ctx, db, repo, baseRef, opts)` — `git merge-base <base> HEAD` + `git diff --name-only <mb> HEAD` (no `--stdin` plumbing) derives the changed set; runs the same traversal; groups into `Area` (per-symbol hop detail collapsed under each area: `Name`, `Changed`, `TestFiles`, `Hops`, `Owners`); owners via `git blame HEAD -- <path>` (author parsed from the blame metadata group; `git log --diff-filter=A` fallback for files absent on HEAD). Unknown ref → clear error. Empty diff → empty result + message.
- Hermetic git fixture (`base_test.go:gitFixtureRepo`): temp repo, `init -b main`, base commit (`src/base.go` + `src/base_test.go`), `checkout -b feat`, feature commit (modify `src/base.go`, add `src/mid.go`). `GIT_CONFIG_GLOBAL`/`GIT_CONFIG_SYSTEM` point at an empty file so ambient config can't color blame. Never touches the real repo's history.
- `--base` wired into `runSearchAffected` (prints merge-base, areas + "tag: …" owners; `--json`/`--quiet` shape it for a PR comment/CI gate).
- Tests: `TestCodeAffectedBase` (changed set, areas, owners = fixture committer), `TestCodeAffectedBaseNoChanges`, `TestCodeAffectedBaseBadRef`.
- RED: `... -run CodeAffectedBase` → `undefined: AffectedBase`, `undefined: Area` (compile FAIL). GREEN: PASS.

### 02.4 `code_rename` buckets (RED → GREEN)
- `affected/rename.go` + `rename_support.go`: `Rename(ctx, db, RenameOptions)` — resolve `old` via 005-style disambiguation (ambiguous → ranked candidates, most-connected first, no silent pick, zero edits; not-found → clean flag). Plan splits into **graph edits** (high-confidence: definition site + typed references from the edges table, resolved endpoints only) and **text-search edits** (low-confidence: other indexed files holding a symbol named `old`, flagged "review carefully (string match)"). Every edit carries a `confidence` label (`high`/`low`). `DryRun` defaults true → plan only, nothing written.
- Tests: `TestCodeRename` (buckets, per-edit confidence, definition + typed caller in graph bucket, dry_run default), `TestCodeRenameTextSearchBucket` (string-match file lands low-confidence, uid-narrow disambiguates).
- RED: `... -run CodeRename` → `undefined: Rename`, `undefined: RenameOptions` (compile FAIL). GREEN: PASS.

### 02.5 non-dry `code_rename` apply (RED → GREEN)
- Apply path in `rename.go`: with `Apply` it writes **only** the planned files (string replace of old→new on the working-tree file); a planned file absent on disk (indexed but deleted) is skipped, not an error; no `git commit`/`push` anywhere in the path.
- `rename_apply_test.go`: `TestCodeRenameApply` — dry plan leaves files untouched; apply edits exactly `src/base.go` (definition) + `src/mid.go` (typed caller); `src/touch_me.go` (outside plan) untouched; no `.git` side effects.
- RED: `... -run CodeRenameApply` → test file references `RenameOptions.Apply`/`plan.Applied` before the apply path existed (compile FAIL). GREEN: PASS.

### 02.6 / 02.7 / 02.8 (AFK)
- `TestCodeRenameAmbiguous` (02.6): two symbols named `base` → `Ambiguous`, 2 ranked candidates (most-connected first), `TotalEdits == 0` (no silent pick). PASS.
- `TestCodeAffectedEmpty` (02.7): empty changed set → no error, empty result, clear message. PASS.
- `TestBlastRadius` (02.8): a RESOLVED route→handler `references` edge is traversed (appears in relationships); a DROPPED route (node with no stored edge, step-01 drop policy) appears nowhere and adds no test files. PASS.

### 02.9 MCP tool surface (RED → GREEN)
- `mcp/tools_code_affected.go`: `code_affected` (changed/stdin/base/depth/filter/json/quiet; clear validation errors for missing changed set and stdin+base clash) + `code_rename` (old/new required; file/uid/kind narrow; dry_run default true; apply) — distinct `code_*` names, registered via `registerAffectedTools` in `server.go` (both `Start` and `NewServer`).
- CLI parity: `search_rename_cli.go` — `skillgrid search rename OLD NEW [--file F] [--uid U] [--kind K] [--apply] [--json]` (dry-run default; human table of graph vs text-search buckets; ambiguous → candidate list).
- `tools_code_affected_test.go`: `TestAffectedTools` (both tools registered; 005/008/010 tools intact incl. `code_search` schema-stable; tool surface 71 → **73**; affected query surfaces `src/base_test.go`; rename dry_run reports `dry_run:true` + graph bucket) and `TestAffectedToolsBadArgs` (no changed set; rename missing `new` → clear validation errors).
- Baseline count locks updated for the additive +2: `server_test.go` (TestAllToolsRegistered want-list += `code_affected`, `code_rename`) and `tools_code_knowledge_test.go` (71 → 73).
- RED: `go test ./skillgrid-cli/internal/mnemonic/mcp/... -run AffectedTools -count=1` → `undefined: handleCodeAffected`, `undefined: handleCodeRename` (compile FAIL). GREEN: PASS.

## The `--base` git fixture (hermetic)
`gitFixtureRepo` (base_test.go) builds everything under `t.TempDir()`: `git init -b main` → commit 1 (`src/base.go`, `src/base_test.go`) → `checkout -b feat` → commit 2 (`src/base.go` modified, `src/mid.go` added). So `merge-base main HEAD` = commit 1 and the diff = exactly `{src/base.go, src/mid.go}`. Every `git` invocation runs with `cmd.Dir = root` and `GIT_CONFIG_GLOBAL`/`GIT_CONFIG_SYSTEM` set to an empty file in the temp dir, so ambient user config (hooks, blame decorations, core settings) cannot leak in. The test store (SQLite) is separate and seeded with matching `files`/`symbols`/`edges` rows; the CLI/MCP git calls run against the temp repo, the traversal against the temp store. No dependency on this repo's history.

## Full suite (final run)
`go test ./internal/mnemonic/affected/... ./internal/mnemonic/mcp/... ./cmd/skillgrid/... -count=1`
```
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/affected   5.920s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp        26.466s
ok  github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid                10.753s
```
005/008/010 baseline:
```
ok  .../internal/mnemonic/graph      0.923s
ok  .../internal/mnemonic/route      1.128s
ok  .../internal/mnemonic/community  1.774s
```
Per-test (affected): TestCodeAffected, TestCodeAffectedStdin, TestCodeAffectedBase, TestCodeAffectedBaseNoChanges, TestCodeAffectedBaseBadRef, TestCodeAffectedEmpty, TestCodeAffectedNoInventedEdges, TestBlastRadius, TestCodeRename, TestCodeRenameTextSearchBucket, TestCodeRenameAmbiguous, TestCodeRenameApply — all PASS. mcp: TestAffectedTools, TestAffectedToolsBadArgs PASS; cmd: TestCodeAffectedStdinCLI, TestCodeAffectedStdin (lib-level) PASS.

## Parallel-session issue (not a step-02 regression)
`internal/mnemonic/setup` (helpers → `internal/install/`, `opencode.go` mid-edit by another session) was reported as breaking `go build` e2e tests (`TestTrail*`, 2 mcp go-build-e2e). In my environment `go build ./...` currently succeeds, so `TestTrail*` and the e2e mcp tests **passed** in the runs above — no failures attributable to it were observed. If a later run of those specific tests fails while `setup`/`install` is mid-edit, it is the pre-existing parallel-session issue, not step 02.

## Commits
| SHA | Subject |
|-----|---------|
| d068db4 | feat(mnemonic): code_affected test-impact traversal |
| 8288d21 | feat(mnemonic): code_affected --stdin for CI and hooks |
| d708a64 | feat(mnemonic): code_affected --base merge-base diff with owners |
| e0a9a57 | feat(mnemonic): code_rename graph and text-search buckets |
| b59c0b1 | feat(mnemonic): code_rename applies only planned files |
| dd593b9 | feat(mnemonic): register code_affected and code_rename tools |

## Concerns
- **Rename apply writes the working tree only**: it string-replaces `old`→`new` in the planned files' on-disk contents (repo-relative to CWD). It is deliberately file-scoped and git-free, but a substring collision (e.g. `base` inside `database`) in a planned file would be renamed too — that residual risk is exactly what the per-edit confidence labels + "review carefully" flag are for; `dry_run` default keeps it opt-in.
- **text-search bucket is graph-anchored**: it covers indexed files that hold a symbol named `old` (outside the graph bucket); a purely textual mention in a file with no such symbol is not planned (the index is the grounding; no unindexed files are invented).
- **Tool count lock 71→73** updated in `server_test.go` + `tools_code_knowledge_test.go` (additive, per the 008 knowledge-tool precedent) — both updated in the 02.9 commit.
- `references` is a reserved SQL word — the traversal kind lists quote all identifiers (affected.go, rename.go).

---

## Review fixes (resume)

### Fix 1 (IMPORTANT): package-level `pathCache` → call-scoped
`affected.go` had `var pathCache = map[int64]string{}` — a package-level mutable map keyed by symbol `id`. Two concurrent `Affected` calls over different stores (or a re-indexed store) could collide on an `id` and report a **stale path** in `Relationships.From`; it also leaked state across test packages.

**Before:**
```go
// pathCache memoizes symbol->path lookups for the relationship report.
var pathCache = map[int64]string{}

func curPathCache(ctx context.Context, db *sql.DB, id int64) string {
    if p, ok := pathCache[id]; ok { return p }
    s, err := symbolByID(ctx, db, id)
    if err != nil { pathCache[id] = ""; return "" }
    pathCache[id] = s.Path
    return s.Path
}
```
**After** (cache created in the `Affected` entrypoint, passed to the traversal loop + `curPathCache`; no package global remains):
```go
testSeen := map[string]bool{}
pathCache := map[int64]string{}      // per-call: no cross-call/cross-store id collision
...
From: curPathCache(ctx, db, cur.id, pathCache),
...
func curPathCache(ctx context.Context, db *sql.DB, id int64, pathCache map[int64]string) string { ... }
```
Within-call memoization is preserved (the map is still read/written on every `curPathCache` hit inside one traversal). `rg "var pathCache"` → no match; the only `pathCache` occurrences are the local at `affected.go:175` and the parameter.

### Fix 2 (MINOR): `parseBlameAuthor` truncates author at first digit run
`base.go:parseBlameAuthor` took the inner metadata group and cut at the **first digit run**, so `Dev2` → `Dev`. The e2e `git blame` output on this host is the **regular** format `SHA (author YYYY-MM-DD HH:MM:SS +TZ LINENUM) content` (verified: `git blame -- f.txt` → `(Dev2 2026-09-10 10:21:08 +0200 1) one1`; `--porcelain` is the epoch form `author / author-time <epoch>`). The fix anchors on the **line number** (the final metadata group, always all-digits) and strips fields from the back:

1. Drop the trailing all-digits token (the line number) — only if it is all digits, so a metadata-less group is left intact.
2. Epoch form: drop a trailing 10-digit `isEpoch` token, return the author.
3. Regular form: drop a trailing 5-char `isTz` offset (`+0200`/`-0700`), then drop trailing `isDateOrTime` tokens (`YYYY-MM-DD`, `H:MM[:SS]`).

New helpers `isDigits`, `isEpoch`, `isTz`, `isDateOrTime` (all pure, no side effects).

**Tests added** (`base_test.go`):
- `TestParseBlameAuthor` — table-driven: `Dev2` preserved (regular + epoch forms), multi-word author `John Smith Jr`, `Fixture`, a regular `-0700` offset case, a no-trailing-metadata group, and a no-parens line → `""`.
- `TestCodeAffectedBaseDigitAuthor` — hermetic temp git repo whose author name is `Dev2`; `AffectedBase` end-to-end must report the area owner as `Dev2` (not truncated to `Dev`).

Note: while writing the tests I found a second latent truncation — the line-number drop took the **last space-separated token** unconditionally, so a metadata-less group ending in a non-digit word (e.g. `…here`) lost its last word. `isDigits` on that token is the guard that fixes it (verified by `TestParseBlameAuthor`'s no-trailing-metadata case).

### Full relevant suite (post-fix)
`go test ./internal/mnemonic/affected/... ./internal/mnemonic/mcp/... ./cmd/skillgrid/... -count=1`
```
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/affected   7.492s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp        29.391s
ok  github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid                16.671s
```
Per-test (affected): TestCodeAffected, TestCodeAffectedStdin, TestCodeAffectedBase, TestCodeAffectedBaseNoChanges, TestCodeAffectedBaseBadRef, TestParseBlameAuthor, TestCodeAffectedBaseDigitAuthor, TestCodeAffectedEmpty, TestCodeAffectedNoInventedEdges, TestBlastRadius, TestCodeRename, TestCodeRenameTextSearchBucket, TestCodeRenameAmbiguous, TestCodeRenameApply — all PASS. mcp (incl. TestAffectedTools/BadArgs, 73-tool lock) + cmd (TestCodeAffectedStdinCLI) — all PASS.

### Commits (review fixes)
| SHA | Subject |
|-----|---------|
| (fix-1) | fix(mnemonic): scope code_affected path cache to the call |
| (fix-2) | fix(mnemonic): parse blame author past digit-bearing names |
