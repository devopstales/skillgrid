# Task 25 — memfs Virtual Filesystem — Report

## Sub-task Implementation

### 25.1 [RED] `mem ls` — scope-based directory listing
- **Test**: `TestMemLSScopeListing` in `memfs_test.go`
- **Implementation**: `ls.go` — `MemFS.List(ctx, scope)` resolves the scope via `ResolveURI`, builds a SQL `WHERE topic_key = ? OR topic_key LIKE ?` query, returns `[]Observation`.
- **Scope storage choice**: `topic_key` column (no new migration needed). Pattern: `{kind}/{id}/{memory_type}/...` — consistent with step 19's directory retrieval.
- **RED evidence**: First run failed with `expected 2, got 0` (LIKE missing `%` wildcard). Fixed → PASS.

### 25.2 [RED] `mem tree` — hierarchical tree view
- **Test**: `TestMemTreeHierarchical` in `memfs_test.go`
- **Implementation**: `tree.go` — `MemFS.Tree(ctx, scope)` queries all topic_keys under the prefix, builds a `TreeNode` hierarchy, renders with `├──`/`└──` indentation. First-level children (memory_type dirs) always render with trailing `/`.
- **RED evidence**: First run failed with `tree missing entities/ directory` (leaf node rendered as title instead of dir name). Fixed → PASS.

### 25.3 [RED] `mem find` — glob pattern search
- **Test**: `TestMemFindPatternMatching` in `memfs_test.go`
- **Implementation**: `find.go` — `MemFS.Find(ctx, pattern, scope)` queries observations (optionally scoped), matches glob pattern against topic_key basename and title using `path.Match`.
- **RED evidence**: Compiled and passed on first run (implementation was straightforward).

### 25.4 [RED] `mem://` URI resolution
- **Test**: `TestMemURIResolution` in `memfs_test.go`
- **Implementation**: `scope.go` — `ResolveURI(uri)` parses `mem://` URIs (or bare paths) into `ScopeFilter{Kind, ID, MemoryType}`. Pure string operation, no DB call. Validates kind (project|user), requires ID, errors on malformed input.
- **RED evidence**: Compiled and passed on first run.

### 25.5 [AFK] `mem fs` CLI group coexists with `mem list`
- **Test**: `TestMemLSCoexistsWithMemList` in `mem_fs_test.go`
- **Implementation**: `mem.go` — added `case "fs"` to the `runMem` switch, added `runMemFS` function with `ls`/`tree`/`find` subcommands. Updated usage text. `mem fs ls` uses `printJSON` (same as `mem list`), `mem fs tree` prints raw text, `mem fs find` uses `printJSON`.
- **RED evidence**: First run failed with `unknown mem command "fs"` (exit 2). Added the case → PASS.

## ScopeFilter Struct + ResolveURI

```go
type ScopeFilter struct {
    Kind       string // "project" or "user"
    ID         string // project or user identifier
    MemoryType string // optional directory; "" = all
}
```

`ResolveURI` accepts:
- `mem://project/{id}/` → `{Kind:"project", ID:id}`
- `mem://project/{id}/{memType}` → `{Kind:"project", ID:id, MemoryType:memType}`
- `mem://user/{id}/` → `{Kind:"user", ID:id}`
- Bare paths (no scheme): `project/A/preferences`, `user/B/`
- Errors: empty path, unknown kind, missing ID

## Scope Storage Choice

**`topic_key`** — no new migration. The existing `topic_key` column already carries the slash-separated path hierarchy. The memfs scope pattern is `{kind}/{id}/{memory_type}/...` which is consistent with step 19's directory retrieval. No `035_scope.sql` migration needed.

## List/Tree/Find Logic

- **List**: `ResolveURI(scope)` → `Prefix()` → SQL `WHERE topic_key = prefix OR topic_key LIKE prefix/%`
- **Tree**: Query all topic_keys under prefix → build `TreeNode` hierarchy → render with `├──`/`└──`
- **Find**: Query observations (optionally scoped) → `path.Match(pattern, basename(topic_key)) || path.Match(pattern, title)`

## `mem fs` CLI Group

```
skillgrid mem fs <ls|tree|find> [args]
  ls <scope>               list observations in a scope
  tree <scope>             hierarchical tree view
  find <pattern> [scope]   glob pattern search (* and ? supported)
```

## Migration

None. The memfs uses the existing `topic_key` column for scope association.

## RED→GREEN Evidence

| Sub-task | RED | GREEN |
|----------|-----|-------|
| 25.1 | `expected 2, got 0` (LIKE missing `%`) | PASS |
| 25.2 | `tree missing entities/ directory` | PASS |
| 25.3 | PASS (straightforward) | PASS |
| 25.4 | PASS (straightforward) | PASS |
| 25.5 | `unknown mem command "fs"` (exit 2) | PASS |

## Test Output

### `go test ./internal/mnemonic/memfs/ -count=1`
```
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memfs	1.168s
```

### `go test ./cmd/skillgrid/ -count=1`
```
ok  	github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid	105.638s
```

### Baselines
```
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store	9.676s
FAIL	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service	59.005s
  --- TestReindexStructuralIsEmbedderFree: pre-existing "database is closed" (fails on parent commit too)
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory	32.803s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory/layer	5.591s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex	22.424s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/route	10.461s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/affected	17.806s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/community	5.380s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg	9.127s
```

## Commits

| SHA | Subject |
|-----|---------|
| `8a2d275` | feat(mnemonic): memfs virtual filesystem with ls, tree, find, and URI resolution |
| `aaf780f` | feat(mnemonic): mem fs CLI group with ls, tree, find subcommands |

## Concerns

1. **Pre-existing test failure**: `TestReindexStructuralIsEmbedderFree` in `service` fails with "database is closed" — a store pooling issue unrelated to memfs. Fails on parent commit `aaf780f~1` as well.
2. **No migration needed**: The `topic_key` column suffices for scope association. No `035_scope.sql` was created.
3. **`mem fs tree` output is raw text** (not JSON) while `mem fs ls` and `mem fs find` output JSON. This matches the brief's description of tree as a "hierarchical tree view" (human-readable).
