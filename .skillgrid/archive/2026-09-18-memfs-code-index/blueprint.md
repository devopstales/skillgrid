# MemFS over Code Index (OpenViking-style) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Re-point `mem fs ls/tree/find` to walk the **code index** (`files`/`symbols`/`chunks`) as an OpenViking-style virtual filesystem, replacing the current memory-observation view.

**Architecture:** The existing `memfs` package is a virtual FS over the `observations` table, keyed by `topic_key` scope URIs (`mem://project/{id}/{memType}/`). We keep its package shape and `ls`/`tree`/`find` verb API but change the data source from `observations` to the code-index tables, and change the namespace from "scope → memory_type → observation" to "project → repo path → file → symbols → chunks". A new path resolver (`CodePath`) replaces the scope resolver; `List`/`Tree`/`Find` query `files`/`symbols`/`chunks` instead of `observations`. A `cat` verb returns a node's source (symbol span → chunk text; file → concatenated chunks). The CLI `mem fs` subcommand group is extended with `cat` and re-documented; its `ls`/`tree`/`find` keep the same signatures but now resolve code paths. Memory browsing is removed from `mem fs` (the observation store stays intact and is still reachable via `mem search` and the MCP `mem_*` tools — only the `mem fs` *frontend* changes).

**Tech Stack:** Go 1.25.5, `database/sql` + SQLite (`modernc.org/sqlite`), `path.Match` for globbing, existing `store.Store` (migrations 005/011 already create `files`/`symbols`/`chunks`).

**Spec:** n/a — this is a direct user request (OpenViking parity for `mem fs`). Design decisions are recorded inline below.

## Hypothesis

**Claim:** Re-pointing `mem fs` from observations to the code-index tables yields an OpenViking-style browsable code tree (`ls` → directory/symbol listing, `tree` → repo→file→symbol hierarchy, `find` → glob over paths/symbol names, `cat` → source), without a schema migration.

**Right condition:** `mem fs tree memfs://project/{id}/` renders the indexed repo tree; `mem fs ls` on a directory lists subdirs + files; `ls` on a file lists its symbols (name/kind/signature/line range); `cat` on a symbol returns its source; all existing `mem search` / MCP `mem_*` behavior unchanged.

**Wrong condition:** any of: the code tables are empty on a fresh/never-indexed store and the CLI errors instead of reporting "no index"; `ls`/`tree`/`find` still return observations after the change; `cat` returns empty for a symbol whose chunk rows are missing; existing `mem search` / `mem save` tests break.

**Thinnest MVP:** Task 1 only — `List` returns files+symbols for a code path on an indexed store, with a RED test against a real store that has files/symbols/chunks inserted. If `List` can't resolve a code path to rows, the whole approach is wrong (stop).

**Door check:** Task 1, Step 2 (the `List` RED test fails before implementation).

## Design Decisions (recorded)

- **Namespace (OpenViking `resources/<proj>/src/...` parity).** A code path is the repo-relative file path, rooted at the project:
  - `memfs://project/{id}/` (or bare `{id}/`, or `.` / empty) → repo root (all indexed files).
  - `memfs://project/{id}/src/` → directory `src/`.
  - `memfs://project/{id}/src/auth/login.go` → a specific file.
  - `memfs://project/{id}/src/auth/login.go::SymbolName` → a specific symbol in that file.
  The project id segment is validated against `MemFS.projectID` (mismatch → error). No `memType` directory exists in the code namespace (that was the memory layout).
- **Node model (2 levels: files → symbols; OpenViking L0/L1/L2 collapsed onto code).**
  - A **directory** node → `ls` lists subdirectories (as `name/`) and files (as `name`) under that path prefix.
  - A **file** node → `ls` lists its symbols: `kind  name  signature  [start-end]`.
  - A **symbol** node (`file::name`) → `cat` returns the source = the `chunks` whose `[start_line..end_line]` intersect the symbol's `[start_line..end_line]` (concatenated in line order).
  - **`tree`** renders the directory tree down to files, and annotates each file with its symbol count (not every symbol — keeps the tree readable; `ls <file>` drills into symbols).
  - **`find`** globs over file paths and symbol names within scope (scope = path prefix).
  This is the most useful code-nav mapping of OpenViking's resource model (nodes are the indexed content; reading a node returns its content), collapsed from 3 OpenViking tiers to 2 code levels.
- **Data source.** `files(id, path, mtime_ns, size, content_hash, indexed_at)`, `symbols(id, file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)`, `chunks(id, file_id, start_line, end_line, text, content_hash)`. `symbols.file_id`/`chunks.file_id` FK → `files.id` (ON DELETE CASCADE). No migration — all tables exist (005/011).
- **Empty-store behavior.** `ls`/`tree`/`find` on a store with zero `files` rows return an empty listing with a single stderr note `mem fs: no code index for project {id} (run skillgrid index)` and exit 0 — not an error. `cat` on an unknown file/symbol returns exit 1 with a clear error.
- **One-way door.** Removing the memory-observation view from `mem fs` changes the observable output of a shipped subcommand. It is low-risk (no data loss — observations are untouched; `mem search`/MCP still expose them) but it IS a behavior change to a documented command, so Task 4 (CLI re-point + help) carries a `> ⚠ one-way` stop.

## Global Constraints

- Go 1.25.5; build with `go build ./...`; test with `go test ./...` (config `testing.runner`).
- No new dependencies. No schema migration (code tables already exist).
- Follow existing `memfs` patterns: `MemFS` struct bound to `*store.Store` + `projectID`; `context.Context` first arg on query methods; errors wrapped `fmt.Errorf("memfs <verb>: %w", err)`; 200-row `LIMIT` on list/find queries.
- Conventional commits; per-work-unit commit with the `[skillgrid-context]` block (skillgrid:work-unit-commits).
- The `observations` table, `memory.Service`, and all `mem search` / MCP `mem_*` tools MUST remain byte-for-byte behaviorally unchanged.

## Must-Haves (Goal-Backward Verification)

**Truths:**
- T1: `mem fs ls memfs://project/{id}/src/` lists subdirs and files under `src/` (from `files.path`), sorted, no observations. `backstop` (needs an indexed store; held-out test in Task 1).
- T2: `mem fs ls <file-path>` lists that file's symbols with kind, name, signature, and line range (from `symbols`), not chunks.
- T3: `mem fs tree memfs://project/{id}/` renders the repo directory tree with files as leaves annotated by symbol count.
- T4: `mem fs find *.go` (and `find <glob> <scope>`) matches file paths and symbol names; returns the matched nodes.
- T5: `mem fs cat <file>::<symbol>` returns the symbol's source from `chunks` intersecting its line span.
- T6: `mem fs` on a never-indexed store prints the "no code index" note and exits 0 (no panic, no error exit).
- T7: Existing `mem search`, `mem save`, and MCP `mem_*` behavior is unchanged (regression).

**Artifacts:**
- `internal/mnemonic/memfs/codepath.go` — `CodePath` type + `ResolveCodePath`.
- `internal/mnemonic/memfs/ls.go` — re-pointed to code index (files+symbols).
- `internal/mnemonic/memfs/tree.go` — code-index tree renderer.
- `internal/mnemonic/memfs/find.go` — code-index glob search.
- `internal/mnemonic/memfs/cat.go` — node source reader.
- `cmd/skillgrid/mem.go` — `mem fs` re-pointed + `cat` verb + updated help.

**Key links:**
- L1: `ResolveCodePath` output (kind=dir/file/symbol + resolved path + symbol name) is consumed by `List`/`Tree`/`Find`/`Cat` to pick the query shape.
- L2: `List`/`Find`/`Cat` JOIN `files`↔`symbols`↔`chunks` on `file_id` so a path resolves to the right rows.
- L3: CLI `runMemFS` passes the resolved project id + code path into `memfs.MemFS` methods (same handle open as today).

**One-way doors:**
- `> ⚠ one-way: mem fs no longer lists memory observations (Task 4)` — behavior change to a shipped subcommand; observations remain reachable via `mem search`/MCP.

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| SQL injection / path interpolation | N/A: all queries use `?` placeholders; path matching is done in Go `path.Match` / SQL `LIKE ?` with a user-supplied prefix, never string-concatenated SQL. | `ResolveCodePath` returns plain strings; queries bind them. | (covered by T1/T4 parameterized queries) |
| Unindexed / empty store | Applicable: a fresh store has 0 `files` rows. | `List`/`Tree`/`Find` return empty + "no code index" note; `cat` errors clearly. | T6 held-out test (Task 1 Step 6). |
| Dangling symbol/chunk refs (pruned symbols) | N/A: we only SELECT and JOIN; ON DELETE CASCADE already removed orphans at the FK level, and a JOIN simply omits unmatched rows. | JOIN (not LEFT JOIN needed for display); a file with no symbols lists as a bare file. | T2 edge: file with 0 symbols (Task 1). |
| Project-id mismatch | Applicable: `memfs://project/A/...` while `MemFS.projectID` is B. | `ResolveCodePath` validates the id equals `MemFS.projectID`; mismatch → `ErrScopeMismatch`. | T1 negative case (Task 1). |
| Large repos (many files) | N/A: `LIMIT 200` retained on `ls`/`find`; `tree` is inherently full but bounded by indexed file count (same as before). | Keep `LIMIT 200`; document truncation in help. | (informational) |

---

### Task 1: CodePath resolver + re-point `List` to the code index

> Door check task. If `List` cannot resolve a code path to files/symbols rows, STOP.

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/memfs/codepath.go`
- Modify: `skillgrid-cli/internal/mnemonic/memfs/ls.go` (replace observation query with files/symbols query)
- Modify: `skillgrid-cli/internal/mnemonic/memfs/memfs.go` (add `ErrScopeMismatch`; update doc comment)
- Test: `skillgrid-cli/internal/mnemonic/memfs/codepath_test.go`
- Test: `skillgrid-cli/internal/mnemonic/memfs/ls_code_test.go`

**Interfaces:**
- Consumes: `store.Store` (via `MemFS.db`), code tables `files`/`symbols`/`chunks` (005/011).
- Produces:
  - `type CodePath struct { Kind string /* "dir"|"file"|"symbol" */; Dir string; File string; Symbol string }`
  - `func ResolveCodePath(memfsProjectID, raw string) (CodePath, error)` — raw is `memfs://project/{id}/...`, bare `{id}/...`, `.`/empty (repo root), `{dir}/`, `{dir}/{file}`, or `{dir}/{file}::{symbol}`.
  - `var ErrScopeMismatch = errors.New("memfs: project id mismatch")`
  - Re-pointed `func (fs *MemFS) List(ctx context.Context, path string) ([]Entry, error)` where `type Entry struct { Name string; Kind string /* "dir"|"file"|"symbol" */; Signature string; StartLine int; EndLine int; SymbolID int64 }`.
  - The old `Observation` struct and its usage are removed from `ls.go` (memory view gone). `Tree`/`Find` are re-pointed in Tasks 2–3 (they may be left temporarily returning observations only if they'd otherwise break compilation — but they are re-pointed before the package is committed green in Task 3, so Task 1 must keep the package compiling: `Tree`/`Find` are re-pointed in the same commit batch as `List` if needed to compile; otherwise Task 1 stubs their return to the new `Entry` shape with a `TODO` guarded by a test added in Task 2/3). **Decision:** to keep each task independently green, Task 1 re-points `List` AND minimally re-points `Tree`/`Find` to the new `Entry` shape (full logic in Tasks 2/3). This is a 1-commit vertical slice.

**SATISFIES:** scenario `code-index-listing` (acceptance.feature)

- [ ] **Step 1: Write the failing tests**

`codepath_test.go` — table test for `ResolveCodePath`:

```go
func TestResolveCodePath(t *testing.T) {
	cases := []struct {
		raw    string
		kind   string
		dir    string
		file   string
		symbol string
		want   error
	}{
		{raw: "", kind: "dir", dir: ""},
		{raw: ".", kind: "dir", dir: ""},
		{raw: "memfs://project/A/src/", kind: "dir", dir: "src"},
		{raw: "A/src/", kind: "dir", dir: "src"},
		{raw: "memfs://project/A/src/auth/login.go", kind: "file", dir: "src/auth", file: "login.go"},
		{raw: "A/src/auth/login.go::Handler", kind: "symbol", dir: "src/auth", file: "login.go", symbol: "Handler"},
		{raw: "memfs://project/B/src/", want: ErrScopeMismatch}, // id B != A
	}
	for _, c := range cases {
		got, err := ResolveCodePath("A", c.raw)
		if (err != nil) != (c.want != nil) {
			t.Fatalf("ResolveCodePath(%q) err=%v, want err=%v", c.raw, err, c.want)
		}
		if c.want != nil {
			if got != c.want {
				t.Fatalf("ResolveCodePath(%q) = %v, want %v", c.raw, err, c.want)
			}
			continue
		}
		if got.Kind != c.kind || got.Dir != c.dir || got.File != c.file || got.Symbol != c.symbol {
			t.Errorf("ResolveCodePath(%q) = %+v, want kind=%s dir=%q file=%q sym=%q",
				c.raw, got, c.kind, c.dir, c.file, c.symbol)
		}
	}
}
```

`ls_code_test.go` — seeds a real store with files/symbols/chunks, then checks `List`:

```go
func seedCodeIndex(t *testing.T, st *store.Store) {
	t.Helper()
	db := st.DB
	// Two files under src/, one symbol each, one chunk each.
	if _, err := db.Exec(`INSERT INTO files (path, size, content_hash, indexed_at) VALUES (?, 100, 'h1', '2026-01-01T00:00:00Z')`, "src/auth/login.go"); err != nil {
		t.Fatalf("insert file login: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO files (path, size, content_hash, indexed_at) VALUES (?, 50, 'h2', '2026-01-01T00:00:00Z')`, "src/util.go"); err != nil {
		t.Fatalf("insert file util: %v", err)
	}
	var loginID int64
	if err := db.QueryRow(`SELECT id FROM files WHERE path = 'src/auth/login.go'`).Scan(&loginID); err != nil {
		t.Fatalf("login id: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO symbols (file_id, name, kind, language, signature, start_line, end_line, content_hash, uid) VALUES (?, 'Handler', 'function', 'go', 'func Handler() error', 1, 10, 'c1', 'uid-login-handler')`, loginID); err != nil {
		t.Fatalf("insert symbol: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO chunks (file_id, start_line, end_line, text, content_hash) VALUES (?, 1, 10, 'func Handler() error { return nil }', 'ck1')`, loginID); err != nil {
		t.Fatalf("insert chunk: %v", err)
	}
}

func TestListRepoRootListsFiles(t *testing.T) {
	st, _ := store.Open(t.TempDir(), "code-ls")
	t.Cleanup(func() { st.Close() })
	seedCodeIndex(t, st)
	fs := New(st, "code-ls")

	entries, err := fs.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List(root): %v", err)
	}
	// Root: src/ dir + util.go file (src/util.go) ... actually both are under src/;
	// root listing shows "src/" only. Assert src/ present and no observation fields.
	names := map[string]bool{}
	for _, e := range entries {
		names[e.Name] = true
	}
	if !names["src"] {
		t.Errorf("root listing missing dir 'src': %+v", entries)
	}
}

func TestListFileListsSymbols(t *testing.T) {
	st, _ := store.Open(t.TempDir(), "code-ls2")
	t.Cleanup(func() { st.Close() })
	seedCodeIndex(t, st)
	fs := New(st, "code-ls2")

	entries, err := fs.List(context.Background(), "src/auth/login.go")
	if err != nil {
		t.Fatalf("List(file): %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("List(file) = %d entries, want 1 symbol: %+v", len(entries), entries)
	}
	e := entries[0]
	if e.Kind != "symbol" || e.Name != "Handler" || e.Signature != "func Handler() error" || e.StartLine != 1 || e.EndLine != 10 {
		t.Errorf("symbol entry = %+v", e)
	}
}

func TestListEmptyStoreNotesNoIndex(t *testing.T) {
	st, _ := store.Open(t.TempDir(), "code-ls-empty")
	t.Cleanup(func() { st.Close() })
	fs := New(st, "code-ls-empty")
	entries, err := fs.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List(empty) should not error, got %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("List(empty) = %+v, want 0", entries)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/mnemonic/memfs/ -run 'TestResolveCodePath|TestList' -count=1`
Expected: FAIL — `undefined: ResolveCodePath`, `New` returns old `Observation`-based `List` (compile error or wrong type).

- [ ] **Step 3: Write `codepath.go`**

```go
package memfs

import (
	"fmt"
	"strings"
)

// CodePath is a resolved location in the code-index namespace.
//   - Kind "dir":    Dir is the directory prefix ("" = repo root).
//   - Kind "file":   Dir + File identify one indexed file.
//   - Kind "symbol": Dir + File + Symbol identify one symbol in a file.
type CodePath struct {
	Kind   string // "dir" | "file" | "symbol"
	Dir    string // directory prefix, no leading/trailing slash ("" = root)
	File   string // basename (file/symbol kinds)
	Symbol string // symbol name (symbol kind)
}

// ErrScopeMismatch is returned when the project id in a code URI does not
// equal the MemFS project id.
var ErrScopeMismatch = fmt.Errorf("memfs: project id mismatch")

// ResolveCodePath parses a code-index path into a CodePath. Accepted forms:
//
//	"" or "."                         → repo root
//	"src/"                            → directory src/
//	"src/auth/login.go"               → file
//	"src/auth/login.go::Handler"      → symbol in file
//	"memfs://project/A/src/..."       → same, with scheme + project id
//	"A/src/..."                       → bare project id + path
//
// The project id (when present) must equal memfsProjectID, else ErrScopeMismatch.
func ResolveCodePath(memfsProjectID, raw string) (CodePath, error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "memfs://")
	segs := splitPath(s)
	// If the first segment is the memfs project id, validate and drop it.
	// A bare path that does not start with the project id is treated as a
	// repo-relative path (root/dir/file) — the CLI always passes the same id.
	if len(segs) > 0 && segs[0] == memfsProjectID {
		segs = segs[1:]
	} else if len(segs) > 0 && memfsProjectID != "" && segs[0] != memfsProjectID && strings.HasPrefix(raw, "memfs://project/") {
		// Explicit scheme with a different project id → mismatch.
		return CodePath{}, ErrScopeMismatch
	}
	// Repo root.
	if len(segs) == 0 {
		return CodePath{Kind: "dir", Dir: ""}, nil
	}
	// Symbol: last segment contains "::".
	if i := strings.LastIndex(segs[len(segs)-1], "::"); i >= 0 {
		last := segs[len(segs)-1]
		fileSeg := last[:i]
		sym := last[i+2:]
		if sym == "" {
			return CodePath{}, fmt.Errorf("memfs: empty symbol name in %q", raw)
		}
		dirSegs := segs[:len(segs)-1]
		dir := strings.Join(dirSegs, "/")
		return CodePath{Kind: "symbol", Dir: dir, File: fileSeg, Symbol: sym}, nil
	}
	if len(segs) == 1 {
		// Single segment: a file basename at root, OR a dir if it was a trailing-slash input.
		// Distinguish by whether the raw path ended with "/".
		if strings.HasSuffix(s, "/") {
			return CodePath{Kind: "dir", Dir: segs[0]}, nil
		}
		return CodePath{Kind: "file", Dir: "", File: segs[0]}, nil
	}
	// Two+ segments: all but last = dir, last = file.
	dir := strings.Join(segs[:len(segs)-1], "/")
	file := segs[len(segs)-1]
	return CodePath{Kind: "file", Dir: dir, File: file}, nil
}
```

- [ ] **Step 4: Re-point `List` (and minimally `Tree`/`Find`) in `ls.go`**

Replace the `Observation` struct usage in `ls.go` with:

```go
// Entry is one node in a code-index listing (ls).
type Entry struct {
	Name      string
	Kind      string // "dir" | "file" | "symbol"
	Signature string
	StartLine int
	EndLine   int
	SymbolID  int64
}

// List returns the nodes under the given code path.
//   - dir path  → subdirs (Name, Kind "dir") + files (Name, Kind "file")
//   - file path → symbols (Name, Kind "symbol", Signature, StartLine, EndLine)
// An empty store returns 0 entries, nil error (caller prints the note).
func (fs *MemFS) List(ctx context.Context, path string) ([]Entry, error) {
	if fs == nil || fs.db == nil {
		return nil, fmt.Errorf("memfs: not initialized")
	}
	cp, err := ResolveCodePath(fs.projectID, path)
	if err != nil {
		return nil, err
	}
	switch cp.Kind {
	case "dir":
		return fs.listDir(ctx, cp.Dir)
	case "file", "symbol":
		return fs.listSymbols(ctx, cp.Dir, cp.File, cp.Symbol)
	}
	return nil, fmt.Errorf("memfs list: unknown kind %q", cp.Kind)
}

// listDir lists immediate subdirectories and files under dirPrefix.
func (fs *MemFS) listDir(ctx context.Context, dirPrefix string) ([]Entry, error) {
	var like string
	if dirPrefix == "" {
		like = "%/%" // any file with at least one path separator
	} else {
		like = dirPrefix + "/%"
	}
	rows, err := fs.db.QueryContext(ctx, `
		SELECT path FROM files
		WHERE ? = 0 OR path LIKE ?
		ORDER BY path LIMIT 200`, like, // placeholder guard, see below
	)
	// ... (implement exact prefix logic; see Step 5)
	...
}
```

> Note for the implementer: the `dirPrefix == ""` case must match ALL files (no `LIKE`), and the `dirPrefix != ""` case matches `path LIKE dirPrefix||'/%'`. Build the query dynamically (two fixed queries, not string-concatenated SQL). For each returned `path`, compute the segment immediately after `dirPrefix` (or the top-level segment for root) and emit either a `dir` entry (if more segments remain) or a `file` entry (if it's the leaf). Deduplicate dir names. `listSymbols` queries `symbols s JOIN files f ON f.id = s.file_id WHERE f.path = ?` (and `AND s.name = ?` when a symbol is requested), returning one `Entry` per symbol.

- [ ] **Step 5: Re-point `Tree`/`Find` to the new `Entry` shape (compile + minimal green)**

Re-point `Tree` to render the directory tree from `files.path` (leaf = file + symbol count via `SELECT path, COUNT(*) FROM symbols s JOIN files f ... GROUP BY f.path`) and `Find` to glob `files.path` + `symbols.name` within scope. Full assertions land in Tasks 2/3; Task 1 only needs them compiling and returning a non-error empty/seeded result so the package is green.

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/mnemonic/memfs/ -count=1`
Expected: PASS (ResolveCodePath, List root/file/empty, plus the minimal Tree/Find compile tests).

- [ ] **Step 7: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/memfs/
git commit -m "feat(memfs): resolve code paths + list code index (files/symbols)"
```

---

### Task 2: Re-point `Tree` to the code-index repo tree

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/memfs/tree.go`
- Test: `skillgrid-cli/internal/mnemonic/memfs/tree_code_test.go`

**Interfaces:**
- Consumes: `files.path`, `symbols` (per-file count).
- Produces: `func (fs *MemFS) Tree(ctx context.Context, path string) (string, error)` — renders `├──`/`└──` tree of dirs→files, each file annotated `(N symbols)`. Empty store → the "no code index" note string (or empty + note handled by CLI).

**SATISFIES:** scenario `code-index-tree`

- [ ] **Step 1: Write the failing test**

```go
func TestTreeRepoTreeWithSymbolCounts(t *testing.T) {
	st, _ := store.Open(t.TempDir(), "code-tree")
	t.Cleanup(func() { st.Close() })
	seedCodeIndex(t, st) // src/auth/login.go (1 symbol), src/util.go (0 symbols)
	fs := New(st, "code-tree")

	tree, err := fs.Tree(context.Background(), "")
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	if !strings.Contains(tree, "src") {
		t.Errorf("tree missing 'src': %q", tree)
	}
	if !strings.Contains(tree, "login.go") || !strings.Contains(tree, "1 symbols") {
		t.Errorf("tree missing login.go with symbol count: %q", tree)
	}
	if !strings.Contains(tree, "util.go") || !strings.Contains(tree, "0 symbols") {
		t.Errorf("tree missing util.go 0 symbols: %q", tree)
	}
}

func TestTreeEmptyStoreNotesNoIndex(t *testing.T) {
	st, _ := store.Open(t.TempDir(), "code-tree-empty")
	t.Cleanup(func() { st.Close() })
	fs := New(st, "code-tree-empty")
	tree, err := fs.Tree(context.Background(), "")
	if err != nil {
		t.Fatalf("Tree(empty) should not error, got %v", err)
	}
	if !strings.Contains(tree, "no code index") {
		t.Errorf("Tree(empty) = %q, want 'no code index' note", tree)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/mnemonic/memfs/ -run TestTree -count=1`
Expected: FAIL (tree.go still observation-based / wrong shape).

- [ ] **Step 3: Implement `Tree` over `files` + per-file symbol counts**

Query `SELECT f.path, (SELECT COUNT(*) FROM symbols s WHERE s.file_id = f.id) AS n FROM files f ORDER BY f.path LIMIT 200`. Build the directory tree from `path` segments; leaves are files rendered as `<name> (N symbols)`. Empty result → return `"mem fs: no code index for project "+fs.projectID+" (run skillgrid index)\n"`.

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/mnemonic/memfs/ -run TestTree -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/memfs/
git commit -m "feat(memfs): tree renders code-index repo tree with symbol counts"
```

---

### Task 3: Re-point `Find` to code-index glob search

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/memfs/find.go`
- Test: `skillgrid-cli/internal/mnemonic/memfs/find_code_test.go`

**Interfaces:**
- Consumes: `files.path`, `symbols.name`.
- Produces: `func (fs *MemFS) Find(ctx context.Context, pattern, scope string) ([]Entry, error)` — matches file paths and symbol names via `path.Match`; scope (a code path prefix) narrows the search. Returns `Entry` rows (files with `Kind "file"`, symbols with `Kind "symbol"`).

**SATISFIES:** scenario `code-index-find`

- [ ] **Step 1: Write the failing test**

```go
func TestFindByPathAndSymbolName(t *testing.T) {
	st, _ := store.Open(t.TempDir(), "code-find")
	t.Cleanup(func() { st.Close() })
	seedCodeIndex(t, st)
	fs := New(st, "code-find")

	// *.go matches both files.
	files, err := fs.Find(context.Background(), "*.go", "")
	if err != nil {
		t.Fatalf("Find(*.go): %v", err)
	}
	if len(files) != 2 {
		t.Errorf("Find(*.go) = %d, want 2 files: %+v", len(files), files)
	}
	// Handler matches the symbol.
	syms, err := fs.Find(context.Background(), "Handler", "")
	if err != nil {
		t.Fatalf("Find(Handler): %v", err)
	}
	if len(syms) != 1 || syms[0].Kind != "symbol" || syms[0].Name != "Handler" {
		t.Errorf("Find(Handler) = %+v, want 1 symbol Handler", syms)
	}
}

func TestFindScopedNarrowsToDir(t *testing.T) {
	st, _ := store.Open(t.TempDir(), "code-find2")
	t.Cleanup(func() { st.Close() })
	seedCodeIndex(t, st)
	fs := New(st, "code-find2")
	// Scope src/auth/ → only login.go, not util.go.
	res, err := fs.Find(context.Background(), "*.go", "src/auth/")
	if err != nil {
		t.Fatalf("Find scoped: %v", err)
	}
	if len(res) != 1 || !strings.HasSuffix(res[0].Name, "login.go") {
		t.Errorf("Find scoped = %+v, want only login.go", res)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/mnemonic/memfs/ -run TestFind -count=1`
Expected: FAIL.

- [ ] **Step 3: Implement `Find` over `files` + `symbols`**

Two queries (or one UNION): files whose `path` matches the glob (optionally under the scope prefix), and symbols whose `name` matches the glob (joined to files for scope). `path.Match(pattern, path.Base(f.path))` OR `path.Match(pattern, f.path)` for files; `path.Match(pattern, s.name)` for symbols. Scope narrows via `f.path LIKE scope||'%'`. Dedup, `LIMIT 200`.

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/mnemonic/memfs/ -run TestFind -count=1`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add skillgrid-cli/internal/mnemonic/memfs/
git commit -m "feat(memfs): find globs code-index paths and symbol names"
```

---

### Task 4: Add `cat` (node source) + re-point CLI `mem fs` + help

> ⚠ one-way: `mem fs` no longer lists memory observations. Observations remain reachable via `mem search` and MCP `mem_*`. Confirm before committing.

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/memfs/cat.go`
- Test: `skillgrid-cli/internal/mnemonic/memfs/cat_code_test.go`
- Modify: `skillgrid-cli/cmd/skillgrid/mem.go` (re-point `runMemFS` to code paths; add `cat` verb; update help text)
- Modify: `skillgrid-cli/cmd/skillgrid/mem_fs_test.go` (update the coexist test to code-index semantics)

**Interfaces:**
- Consumes: `files`, `symbols`, `chunks`.
- Produces: `func (fs *MemFS) Cat(ctx context.Context, path string) (string, error)` — for a `file::symbol`, returns the concatenated `chunks.text` whose `[start_line..end_line]` intersects the symbol's span (ordered by `start_line`); for a bare `file`, returns the full file text (all chunks in line order). Unknown file/symbol → error.

**SATISFIES:** scenario `code-index-cat`

- [ ] **Step 1: Write the failing test**

```go
func TestCatSymbolSource(t *testing.T) {
	st, _ := store.Open(t.TempDir(), "code-cat")
	t.Cleanup(func() { st.Close() })
	seedCodeIndex(t, st)
	fs := New(st, "code-cat")
	src, err := fs.Cat(context.Background(), "src/auth/login.go::Handler")
	if err != nil {
		t.Fatalf("Cat(symbol): %v", err)
	}
	if !strings.Contains(src, "func Handler() error") {
		t.Errorf("Cat(symbol) = %q, want symbol source", src)
	}
}

func TestCatFileFullText(t *testing.T) {
	st, _ := store.Open(t.TempDir(), "code-cat2")
	t.Cleanup(func() { st.Close() })
	seedCodeIndex(t, st)
	fs := New(st, "code-cat2")
	src, err := fs.Cat(context.Background(), "src/auth/login.go")
	if err != nil {
		t.Fatalf("Cat(file): %v", err)
	}
	if !strings.Contains(src, "func Handler() error") {
		t.Errorf("Cat(file) = %q", src)
	}
}

func TestCatUnknownSymbolErrors(t *testing.T) {
	st, _ := store.Open(t.TempDir(), "code-cat3")
	t.Cleanup(func() { st.Close() })
	seedCodeIndex(t, st)
	fs := New(st, "code-cat3")
	if _, err := fs.Cat(context.Background(), "src/auth/login.go::Nope"); err == nil {
		t.Errorf("Cat(unknown symbol) should error")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/mnemonic/memfs/ -run TestCat -count=1`
Expected: FAIL (`Cat` undefined).

- [ ] **Step 3: Implement `Cat`**

```go
func (fs *MemFS) Cat(ctx context.Context, path string) (string, error) {
	// resolve file path + optional symbol; query chunks intersecting the span.
}
```
For a symbol: `SELECT c.text FROM chunks c JOIN files f ON f.id=c.file_id JOIN symbols s ON s.file_id=c.file_id WHERE f.path=? AND s.name=? AND c.start_line <= s.end_line AND c.end_line >= s.start_line ORDER BY c.start_line`. For a file: same without the symbol join/filter. Concatenate with `\n`.

- [ ] **Step 4: Re-point CLI `runMemFS` + add `cat` + update help**

In `mem.go`, `runMemFS`:
- `ls <path>` → `fs.List(ctx, path)`, print JSON `{project, path, entries, count}`.
- `tree <path>` → `fs.Tree(ctx, path)`, print text.
- `find <pattern> [scope]` → `fs.Find(ctx, pattern, scope)`, print JSON.
- `cat <path>` → `fs.Cat(ctx, path)`, print source to stdout (exit 1 on error).
- Update the help block to document code paths + the `memfs://project/{id}/` scheme + `file::symbol` syntax + the "no code index" note.
- Keep `projID` resolution exactly as today (`svc.Open(projID)`).

- [ ] **Step 5: Update `mem_fs_test.go`**

The existing `TestMemFS*` coexist test asserts memory-observation listings. Replace its assertions with code-index assertions (seed files/symbols, assert `ls`/`tree`/`find`/`cat`), keeping the test name meaningful (e.g. rename to `TestMemFSCoexistsWithCodeIndex` if it previously verified mem search still works — add a `mem search` assertion to prove memory is untouched).

- [ ] **Step 6: Run the full affected suites**

Run: `go test ./internal/mnemonic/memfs/ ./cmd/skillgrid/ -count=1`
Expected: PASS (memfs + cmd/skillgrid; only the pre-existing `TestSkillgridEvalSelfCorpus` git-hook failure may remain).

- [ ] **Step 7: Build + vet**

Run: `go build ./... && go vet ./internal/mnemonic/memfs/ ./cmd/skillgrid/`
Expected: clean.

- [ ] **Step 8: Commit (after one-way-door confirmation)**

```bash
git add skillgrid-cli/internal/mnemonic/memfs/ skillgrid-cli/cmd/skillgrid/
git commit -m "feat(memfs): cat node source + point mem fs at the code index"
```

---

## Self-Review

1. **Spec coverage:** ls (T1/T2, Task 1), tree (T3, Task 2), find (T4, Task 3), cat (T5, Task 4), empty-store (T6, Task 1), memory-unchanged (T7, Task 5 step) — all covered.
2. **Must-haves coverage:** every truth maps to a task; artifacts all named; key links L1–L3 wired.
3. **One-way-door completeness:** the memory→code re-point is flagged on Task 4 and listed in Must-Haves.
4. **Placeholder scan:** Step 4 Task 1 has an explicit "implement exact prefix logic" note with the two fixed queries spelled out — acceptable (the dynamic-query choice is specified, not left open). No TBD/TODO survives in committed code (the Task 1 Tree/Find minimal re-point must be real enough to compile, guarded by Task 2/3 tests).
5. **Type consistency:** `Entry` is defined once (Task 1) and used by `List`/`Find`/`Cat` consumers; `CodePath` + `ResolveCodePath` consistent across tasks; `Cat` returns `(string, error)`.

## Execution Handoff

4 tasks, 1 one-way-door (Task 4). Recommend **slicing** is NOT needed (already vertical, 4 small tasks). Execution options:
1. Subagent-Driven (recommended) — fresh subagent per task.
2. Inline (simple-execution).

Final review: lightweight two-axis (`requesting-code-review`); escalate to `parallel-code-review` only if the diff grows past ~150 changed lines (it should not).

## Code Review (post-implementation)

- Verdict: Standards "met with fixes" + Spec "met" (two-axis, 1461fe7..7813e57).
- Fixed (commit ce4c03b + glossary 38460bc): glossary drift (#1), bare-vs-explicit
  project-id consistency (#2), DRY query extraction + named caps/sentinel
  (#3/#4/#5/#6), single NoCodeIndexNote home (#7).
- Deferred (real, later — note for a future cleanup ticket):
  - #8 `CodePath.Kind`/`Entry.Kind` stringly-typed → `type Kind string` + consts.
  - #9 `Tree`/`List` accept a file/symbol path and silently return empty
    (asymmetric with `Cat`, which errors on a dir).
  - Spec note: `mem fs find` on an unindexed store prints no "no code index"
    note (only `ls`/`tree` do); the acceptance scenario only requires ls/tree.
- Noise (won't-fix / misread):
  - #10 `seenFile` map in Find is a no-op safety net (paths are unique).
  - Spec note: `listDir` edge where a top-level file is named exactly like a
    dir prefix (out of acceptance scope).
