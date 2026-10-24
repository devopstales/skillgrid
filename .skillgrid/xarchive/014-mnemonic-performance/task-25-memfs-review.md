# Task 25 — memfs Virtual Filesystem — Review

Reviewed 2 commits (`8a2d275`, `aaf780f`), 8 files. Verified: `go vet` clean,
`go build` clean, all 4 focused memfs tests PASS, `TestMemLSCoexistsWithMemList`
runs the real CLI via `exec.Command` (mem_test.go:49 `runMemCLI`). No `mcp/` or
`.sql` files touched in either commit → additive, no tool-contract or schema change.

## Verdict

**PASS WITH WARNINGS**

## Findings

### Major

1. **`LIMIT 200` silently truncates — `ls.go:34`, `find.go:46`, `find.go:57`**
   All three queries end `LIMIT 200` with no `total` / `truncated` signal.
   `mem ls project/A/` returns up to 200 observations and the caller cannot tell
   whether more exist. For a "directory listing" of a project's entire
   preferences/entities subtree this is a real completeness gap (a large project
   lists silently incomplete). The `count` in the CLI JSON is `len(obs)` — it
   reports what was returned, not what exists, so it understates. Fix: add a
   `SELECT COUNT(*)` companion or a `truncated` flag; at minimum document the
   cap in the CLI usage.

2. **`?` glob wildcard advertised but never tested — `find.go:84` + brief 25.3**
   The brief (line 53) says "support `*` (any chars) and `?` (single char)".
   `matchesGlob` uses `path.Match` (which does honor `?`), so the *code* is
   correct, but `TestMemFindPatternMatching` only exercises `*` patterns
   (`auth*`, `*.go`, `auth*test*`). No test asserts `?` (e.g. `auth_?est.go`
   matches `auth_test.go`). The advertised single-char behavior is unproven by a
   test. Fix: add a `?` case to the find test.

### Minor

3. **`LIKE` `_` wildcard not escaped — `ls.go:33`, `find.go:45`, `tree.go:46`**
   `topic_key LIKE prefix+"/%"` — a `_` in a project ID (e.g. `project/my_proj/`)
   is a single-char wildcard in SQLite `LIKE`, so it would also match
   `project/myXproj/`. Same latent issue exists in step 19's own `retrieval.go:272`
   (`o.topic_key LIKE root+"/%"`), so this is **consistent with the existing
   codebase convention**, not a regression — but it means `mem ls`/`tree`/`find`
   can false-match for IDs containing `_`. Low severity (project/user IDs rarely
   contain `_`), and matching step 19's behavior is defensible. If IDs with `_`
   are expected, escape `_`/`%` in the prefix or use `GLOB` (no `_` wildcard).

4. **Root observations (topic_key == prefix) invisible in `tree` — `tree.go:46`**
   `Tree` queries `topic_key LIKE prefix+"/%"` only, so an observation whose
   topic_key exactly equals the prefix (e.g. `project/A` under `mem tree project/A/`)
   is dropped. This matches step 19's directory-walk semantics (root is not its
   own child) and the brief's nested-scope test, so it is acceptable — flagged for
   awareness only. `List` handles the exact-prefix case via the `OR topic_key = ?`
   clause; `Tree` does not.

5. **CLI joins multi-token scope with spaces — `mem.go:76,93,108`**
   `scope := strings.Join(pos[1:], " ")`. `mem fs ls project/A/ preferences`
   yields the scope string `project/A/ preferences`, and `splitPath`
   (`scope.go:70`) `TrimSpace`s each segment, so it happens to parse. Fragile:
   a scope with an embedded space would break. The test always passes a single
   token so it is latent. Consider documenting that `<scope>` is a single
   argument.

6. **Tree output is raw text while `ls`/`find` are JSON — `mem.go:99`**
   `mem fs tree` prints `fmt.Print(tree)` (unindented human text); `ls` and
   `find` emit JSON. Inconsistent machine-parseability across the `mem fs` group
   (the report already notes this). Cosmetic; acceptable for a "tree view" but
   means the `mem fs` group is not uniformly scriptable.

### Nit

7. **`ResolveURI` drops segments beyond the 3rd — `scope.go:63`**
   `mem://project/A/preferences/sub1` → `{Kind:project, ID:A, MemoryType:preferences}`,
   the `/sub1` is silently dropped. `Prefix()` still returns
   `project/A/preferences` and `List` matches via `prefix+"/%"`, so the result is
   correct — but the parse is lossy (no error on a 4th+ segment). Document it or
   keep deeper segments for future use.

8. **`memfs.Observation` is a reduced view of `memory.Observation` — `memfs.go:455`**
   memfs defines its own `Observation` (ID/Title/Content/TopicKey/MemoryType/
   CreatedAt) rather than reusing `memory.Observation`. This is the *right* call
   for package isolation (avoids a `memfs`→`memory` import that could risk a
   cycle, since `memfs` only needs `store`), but it means `mem fs ls` JSON drops
   governance fields (owner/visibility/pinned) that `mem list` carries. Two
   "observation" shapes now exist in the CLI surface. Fine for step 25; worth a
   note so a future step that unifies them doesn't surprise consumers.

## Evidence checked

| # | Check | Result |
|---|-------|--------|
| 1 | **Scope parsing (25.1, CRITICAL)** | **PASS.** `ResolveURI("mem://project/A/preferences")` → `{project, A, preferences}`, `Prefix()="project/A/preferences"` (scope.go:43, verified by `TestMemURIResolution`). Validation rejects: empty (`ErrEmptyScope`), scheme-only, `len(segs)<2`, unknown kind (`mem://org/X/`), missing ID (`mem://project/`). Bare paths (`project/A/preferences`) accepted. `List` filters via `topic_key = ? OR topic_key LIKE prefix/%` (ls.go:33) → `project/A/preferences` returns only the 2 preferences obs, `project/A/` returns all 4, `user/B/` returns only B. `TestMemLSScopeListing` is REAL and asserts exact counts (2/4/1) — not a smoke test. |
| 2 | **Scope storage / `LIKE`** | **PASS (with minor #3).** Uses `topic_key` prefix, consistent with step 19 `retrieval.go:272` (`o.topic_key LIKE root+"/%"`). Filtering is prefix-match. `_` in LIKE is NOT escaped → latent false-match for IDs with `_` (minor, consistent with step 19). |
| 3 | **Tree building (25.2)** | **PASS.** Builds `TreeNode` hierarchy from topic_key segments, sorts children alphabetically (`sortChildren`, deterministic), renders with `├──`/`└──` + `│` indent, memory_type dirs always get trailing `/`, deep hierarchies recurse. `TestMemTreeHierarchical` verifies `preferences/`, `entities/` siblings + `sub1`/`sub2` nested. |
| 4 | **Glob matching (25.3)** | **PASS (with major #2).** `Find` uses `path.Match` against topic_key basename AND title (find.go:88,94) — proper glob, supports `*`/`?`/`[`. `auth*`→2, `*.go`→4, `auth*test*`→1 all verified by test. Invalid/empty pattern → error (find.go:28). `?` untested (major #2). Note: `path.Match` doesn't match `*` across `/`, but matching is on basename/title (single segment) so that's correct here. |
| 5 | **CLI (25.4)** | **PASS.** `mem fs` group wired into `runMem` switch (mem.go:17), `runMemFS` with `ls`/`tree`/`find`. Errors: missing args → exit 2, open/list/tree/find errors → exit 1, unknown subcommand → exit 2, `help`/empty → usage. `TestMemLSCoexistsWithMemList` exercises the real binary via `exec.Command`. |
| 6 | **Additive / tool contract** | **PASS.** Neither commit touches `internal/mnemonic/mcp/` or any `.sql`. memfs is a new package + new CLI subcommand; no `mem_*`/`code_*`/`web_*` signature changed. Purely additive. |
| 7 | **TDD discipline** | **PASS.** Tests cover (a) ls scope listing 3 scopes (b) tree hierarchical (c) find pattern (d) CLI coexistence. Implementer report shows genuine RED evidence for 25.1/25.2/25.5; 25.3/25.4 "passed first run" (acceptable for straightforward logic, but weakens the RED claim slightly). |
| 8 | **Package isolation** | **PASS.** `memfs` imports only `store` (for `*store.Store`→`*sql.DB`) and `database/sql`. No circular import. Defines its own `Observation` (see nit #8) rather than importing `memory`. |

## DoD gap (not a code defect)

The brief's Definition of Done requires "All `@step-25` scenarios in
`acceptance.feature` pass". **No `acceptance.feature` exists** anywhere under
`.skillgrid/sdd/014-mnemonic-performance/`, and there are no `@step-25` tagged
scenarios in any `.feature` file in the change. Unit tests fully cover the
behavior (a/b/c/d above), so functionally the step is complete, but the DoD's
BDD line is unmet — either the feature file was never created for this change,
or the brief's DoD line is stale. Flagging so the orchestrator can close it
intentionally (waiver or add scenarios) before archive.

## Approved to merge

**Yes** — approve with the two Major findings (LIMIT truncation, untested `?`)
tracked as follow-ups. Neither blocks the additive CLI from merging; the
`LIMIT 200` cap is the one most worth addressing before the next step builds on
`mem ls` for larger trees.
