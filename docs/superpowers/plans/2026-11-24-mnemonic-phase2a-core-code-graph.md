# Mnemonic Phase 2A: Core Code Graph, Incremental Sync & Hybrid RWR Retrieval Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver the complete Phase 2A `mnemonic code` subsystem (`index`, `status`, `explore`, `read`, `search`, `impact`, `affected`, `neighbors`, `rename`, `watch`, `backfill`, `mcp`) inside `~/.skillgrid/mnemonic/<repo>.db` with pure-Go `gotreesitter` extraction, incremental cross-file resurrection, FTS5 + camelCase segment vocabulary + ONNX `vec0` embeddings, CSR Personalized PageRank (`<=24 KiB` budgeted `code_explore`), and `observation_code_links` joining Phase 1 memories to live code.

**Architecture:** `internal/platform/{apperr,project,sqlitepool,registry}` gains Phase 2 error codes, git worktree root detection, two-lane priority `writeQueue` (`Write` vs `WriteBatch`), and `WalCheckpointValve` so bulk indexing never starves `mem_save`. `internal/code/{api,store,scan,extract,resolve,csr,search,rename,engine,client,resolver,mcp,http,cli,tui}` implements the `code` subsystem strictly on `<repo>.db` (`AppliesToGlobalDB() = false`), keeping SQL inside `internal/code/store` and every Go file `<600` LOC.

**Tech Stack:** Go 1.26 (`CGO_ENABLED=0`), `modernc.org/sqlite` + `modernc.org/sqlite/vec` (`vec0`), `github.com/odvcencio/gotreesitter` (pure-Go tree-sitter + grammars), `github.com/fsnotify/fsnotify`, `github.com/mark3labs/mcp-go`, `charmbracelet/bubbletea`.

## Global Constraints

- Go module path: `mnemonic`; zero CGo (`CGO_ENABLED=0 go build ./...` and `CGO_ENABLED=0 go test -race ./...` must pass).
- Every source file stays `<600` LOC; SQL lives strictly in `internal/code/store` and `internal/platform/{migrate,sqlitepool}`.
- `subsystem = "code"` migrations apply **only** to `~/.skillgrid/mnemonic/<repo>.db` (`global.db` never receives `code_*` tables).
- FTS5 tables (`code_symbols_fts`, `code_chunks_fts`) are **standalone** (no `content='...'`), matching Phase 1's verified SQLite `AFTER DELETE` trigger invariant.
- Every file emits a synthetic `kind = 'file'` root row in `code_symbols`; numeric `code_symbols.id` FKs bind strictly post-insert via deterministic `uid = sha256(path || '\0' || kind || '\0' || qualified_name || '\0' || sig_hash)`.
- Process `os.Getwd()` is forbidden inside `internal/code/*`; all filesystem reads confine paths via `filepath.EvalSymlinks` against `code_index_meta.active_root` (`PATH_OUTSIDE_ROOT` = exit `9` / HTTP `403`).
- Additional error codes: `PATH_OUTSIDE_ROOT` (`9` / `403`), `INDEX_BUSY` (`10` / `429`), `LSP_UNAVAILABLE` (`11` / `503`); `code_rename` drift returns `FINGERPRINT_CONFLICT` (`5` / `409`).
- Zero stub tools: Phase 2A registers **only** Phase 2A's 11 implemented MCP tools (`6` Eager Core + `3` Deferred Extended + `2` Admin).

---

## File Structure (Phase 2A)

```text
internal/platform/
├── apperr/{apperr.go,apperr_test.go}                   # Modify: add OutsideRoot, IndexBusy, LSPUnavailable
├── project/{root.go,root_test.go}                      # Create: git worktree root + HEAD SHA + symlink confinement
├── sqlitepool/{priority.go,wal_valve.go,valve_test.go} # Create: two-lane Write/WriteBatch + WalCheckpointValve
└── registry/{registry.go,registry_test.go}             # Modify/Create: AppliesToGlobalDB() bool on Subsystem
internal/code/
├── api/{types.go,service.go}                           # Domain DTOs, capability flags, Service interface
├── store/
│   ├── migrations/001_code_core.sql                    # Core code graph, standalone FTS5, vec0, observation_code_links
│   ├── store.go                                        # Init, Meta, SymbolUID, Segmentize, Querier helpers
│   ├── batch.go                                        # Resurrection, UpsertFileBundle, ReopenFailedRefs
│   ├── graph_io.go                                     # Edges, UnresolvedRefs, CSR bulk loader, Neighbors
│   ├── search_io.go                                    # FTS5 + segment + vec0 queries, ObservationCodeLinks
│   └── store_test.go                                   # Schema, FTS triggers, resurrection & link round-trip tests
├── scan/
│   ├── ignore.go                                       # .gitignore / .git/info/exclude / .mnemonicignore matcher
│   ├── scanner.go                                      # Delta walk, generated/test classifier, binary/size guards
│   ├── watcher.go                                      # Recursive fsnotify directory watcher with dynamic subdir add
│   └── scan_test.go                                    # Ignore precedence, CWD independence, recursive watch tests
├── extract/
│   ├── languages.go                                    # 30+ language detection, Tier 1/2/3 map, Stamp()
│   ├── pool.go                                         # Panic-recovering gotreesitter worker pool + circuit breaker
│   ├── extractor.go                                    # Span stack, file root symbol, call/import/type extraction
│   ├── chunker.go                                      # AST callable/skeleton/window chunker + embedding projection
│   └── extract_test.go                                 # Real gotreesitter Go/TS/Py/Rust/Java/C# + panic recovery tests
├── resolve/
│   ├── manifests.go                                    # go.mod/go.work, tsconfig/package.json, pyproject, Cargo, pom, csproj
│   ├── resolver.go                                     # Pending ref resolution across 6 Tier-1 ecosystems
│   └── resolve_test.go                                 # Incremental edit/barrel/manifest resurrection isomorphism test
├── csr/
│   ├── graph.go                                        # Generation-fenced forward/reverse CSR adjacency cache
│   ├── pagerank.go                                     # 25-step Personalized PageRank + Reverse BFS Impact/Affected
│   └── csr_test.go                                     # PageRank convergence, generated/test demotion & impact tests
├── search/
│   ├── explore.go                                      # 4-signal RRF + CSR RWR + <=24 KiB whole-symbol cluster allocator
│   ├── verify.go                                       # validateAnswerFiles staleness check, secret redactor, AST grep
│   └── search_test.go                                  # Budget cap, whole-function invariant, staleness & memory links
├── rename/
│   ├── rename.go                                       # 2-step dry_run preview + plan_token atomic file applier
│   └── rename_test.go                                  # Preview diff, atomic apply, 409 stale-file abort tests
├── engine/
│   ├── local.go                                        # Cooperative Index DAG, Explore, Read, Impact, Affected, Watch
│   └── local_test.go                                   # Polyglot real-parser compliance + concurrent mem_save SLA test
└── {client,resolver,mcp,http,cli,tui}/                 # Daemon IPC, 11 zero-stub MCP tools, CLI subcommands, [2: Code] TUI
```

---

### Task 1: Platform Extensions (`apperr`, `project.ConfinePath`, Priority `WriteBatch` & `WalCheckpointValve`)

**Files:**
- Modify: `internal/platform/apperr/apperr.go`, `internal/platform/apperr/apperr_test.go`
- Create: `internal/platform/project/root.go`, `internal/platform/project/root_test.go`
- Modify: `internal/platform/sqlitepool/pool.go`
- Create: `internal/platform/sqlitepool/wal_valve.go`, `internal/platform/sqlitepool/valve_test.go`

**Interfaces:**
- Consumes: `sqlitepool.DB` (`Path`, `W`, `R`).
- Produces:
  - `apperr.OutsideRoot` (`exit 9`, `HTTP 403`), `apperr.IndexBusy` (`exit 10`, `HTTP 429`), `apperr.LSPUnavailable` (`exit 11`, `HTTP 503`).
  - `project.DetectWorktree(dir string) (root string, headSHA string, err error)` and `project.ConfinePath(activeRoot, relOrAbs string) (absPath string, repoRel string, err error)`.
  - `(d *sqlitepool.DB).WriteBatch(ctx context.Context, fn func(tx *sql.Tx) error) error` (yields to queued `Write` calls) and `sqlitepool.NewWalValve(db *sqlitepool.DB, softBytes, hardBytes, capBytes int64) *WalValve`.

- [ ] **Step 1: Write failing tests for error codes, symlink confinement, and priority writer fairness**

`internal/platform/project/root_test.go`:

```go
package project

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"mnemonic/internal/platform/apperr"
)

func TestConfinePathRejectsTraversalAndEscapingSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	insideFile := filepath.Join(root, "pkg", "a.go")
	if err := os.MkdirAll(filepath.Dir(insideFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(insideFile, []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(outside, "secret.txt")
	_ = os.WriteFile(secret, []byte("nope"), 0o644)
	_ = os.Symlink(secret, filepath.Join(root, "escape.go"))

	abs, rel, err := ConfinePath(root, "pkg/a.go")
	if err != nil || rel != "pkg/a.go" || abs == "" {
		t.Fatalf("valid path failed: abs=%q rel=%q err=%v", abs, rel, err)
	}
	for _, bad := range []string{"../secret.txt", "/etc/passwd", "escape.go"} {
		if _, _, err := ConfinePath(root, bad); apperr.CodeOf(err) != apperr.OutsideRoot {
			t.Fatalf("ConfinePath(%q) code = %q, want OutsideRoot", bad, apperr.CodeOf(err))
		}
	}
}

func TestDetectWorktreeFindsGitRootAndHead(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644)
	run("add", "main.go")
	run("commit", "-m", "init")

	sub := filepath.Join(dir, "sub", "nested")
	_ = os.MkdirAll(sub, 0o755)
	gotRoot, head, err := DetectWorktree(sub)
	if err != nil || len(head) < 7 {
		t.Fatalf("DetectWorktree = (%q, %q, %v)", gotRoot, head, err)
	}
	wantRoot, _ := filepath.EvalSymlinks(dir)
	if gotRoot != wantRoot {
		t.Fatalf("root = %q, want %q", gotRoot, wantRoot)
	}
}
```

`internal/platform/sqlitepool/valve_test.go`:

```go
package sqlitepool

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestHighPriorityWritePreemptsBatchQueue(t *testing.T) {
	p := NewPool()
	defer p.Close()
	d, err := p.Get(filepath.Join(t.TempDir(), "prio.db"), func(db *DB) error {
		_, err := db.W.Exec(`CREATE TABLE kv(k TEXT, v INTEGER)`)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	valve := NewWalValve(d, 4<<20, 8<<20, 32<<20)
	if err := valve.Begin(ctx); err != nil {
		t.Fatal(err)
	}
	defer valve.Finish(ctx)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			_ = d.WriteBatch(ctx, func(tx *sql.Tx) error {
				_, err := tx.Exec(`INSERT INTO kv(k,v) VALUES ('batch', ?)`, i)
				time.Sleep(2 * time.Millisecond)
				return err
			})
			_ = valve.Step(ctx)
		}
	}()

	time.Sleep(5 * time.Millisecond)
	start := time.Now()
	if err := d.Write(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec(`INSERT INTO kv(k,v) VALUES ('mem_save', 1)`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("PriorityHigh Write took %v; expected <50ms preemption", elapsed)
	}
	wg.Wait()
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `CGO_ENABLED=0 go test ./internal/platform/apperr/... ./internal/platform/project/... ./internal/platform/sqlitepool/... -v`
Expected: FAIL (`undefined: apperr.OutsideRoot`, `undefined: ConfinePath`, `undefined: NewWalValve`).

- [ ] **Step 3: Implement `apperr` additions, `project/root.go`, priority `WriteBatch`, and `WalValve`**

Update `internal/platform/apperr/apperr.go` constants and maps:

```go
const (
	Validation     Code = "VALIDATION_ERROR"
	Unauthorized   Code = "UNAUTHORIZED"
	NotFound       Code = "NOT_FOUND"
	Conflict       Code = "FINGERPRINT_CONFLICT"
	Gone           Code = "TOMBSTONE_GONE"
	UnsupportedFS  Code = "UNSUPPORTED_FS"
	Corrupt        Code = "STORAGE_CORRUPT"
	OutsideRoot    Code = "PATH_OUTSIDE_ROOT"
	IndexBusy      Code = "INDEX_BUSY"
	LSPUnavailable Code = "LSP_UNAVAILABLE"
	Internal       Code = "INTERNAL"
)

var exitCodes = map[Code]int{
	Validation: 2, Unauthorized: 3, NotFound: 4, Conflict: 5,
	Gone: 6, UnsupportedFS: 7, Corrupt: 8, OutsideRoot: 9,
	IndexBusy: 10, LSPUnavailable: 11, Internal: 1,
}

var httpStatuses = map[Code]int{
	Validation: 400, Unauthorized: 401, NotFound: 404, Conflict: 409,
	Gone: 410, UnsupportedFS: 500, Corrupt: 500, OutsideRoot: 403,
	IndexBusy: 429, LSPUnavailable: 503, Internal: 500,
}
```

Create `internal/platform/project/root.go`:

```go
package project

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"mnemonic/internal/platform/apperr"
)

// DetectWorktree resolves the canonical realpath of the git worktree (or dir if
// not a git repo) and current HEAD SHA without ever calling os.Getwd().
func DetectWorktree(dir string) (string, string, error) {
	if dir == "" {
		return "", "", apperr.New(apperr.Validation, "worktree directory is required")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", "", err
	}
	realDir, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", "", err
	}
	topCmd := exec.Command("git", "-C", realDir, "rev-parse", "--show-toplevel")
	if out, err := topCmd.Output(); err == nil {
		if top, err := filepath.EvalSymlinks(strings.TrimSpace(string(out))); err == nil {
			realDir = top
		}
	}
	var head string
	headCmd := exec.Command("git", "-C", realDir, "rev-parse", "HEAD")
	if out, err := headCmd.Output(); err == nil {
		head = strings.TrimSpace(string(out))
	}
	return realDir, head, nil
}

// ConfinePath resolves candidate relative to activeRoot and guarantees the
// symlink-evaluated target remains strictly inside activeRoot.
func ConfinePath(activeRoot, candidate string) (string, string, error) {
	if activeRoot == "" || candidate == "" {
		return "", "", apperr.New(apperr.Validation, "activeRoot and path are required")
	}
	realRoot, err := filepath.EvalSymlinks(activeRoot)
	if err != nil {
		return "", "", err
	}
	joined := candidate
	if !filepath.IsAbs(candidate) {
		joined = filepath.Join(realRoot, candidate)
	}
	cleaned := filepath.Clean(joined)
	// Evaluate symlinks on the deepest existing ancestor so new temp files also confine safely.
	evalTarget, err := evalExisting(cleaned)
	if err != nil {
		return "", "", apperr.Newf(apperr.OutsideRoot, "resolve %q: %v", candidate, err)
	}
	rel, err := filepath.Rel(realRoot, evalTarget)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", apperr.Newf(apperr.OutsideRoot, "path %q escapes root %q", candidate, realRoot)
	}
	return evalTarget, filepath.ToSlash(rel), nil
}

func evalExisting(p string) (string, error) {
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		return resolved, nil
	}
	parent := filepath.Dir(p)
	if parent == p {
		return p, os.ErrNotExist
	}
	resolvedParent, err := evalExisting(parent)
	if err != nil {
		return "", err
	}
	return filepath.Join(resolvedParent, filepath.Base(p)), nil
}
```

Create `internal/platform/sqlitepool/wal_valve.go` and add `highWait atomic.Int32` + `WriteBatch` on `sqlitepool.DB`:

```go
package sqlitepool

import (
	"context"
	"database/sql"
	"os"
	"sync/atomic"
	"time"

	"mnemonic/internal/platform/apperr"
)

// WriteBatch runs a background/bulk write transaction, yielding first to any
// pending high-priority interactive Write callers (e.g. mem_save).
func (d *DB) WriteBatch(ctx context.Context, fn func(tx *sql.Tx) error) error {
	for d.highWait.Load() > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Millisecond):
		}
	}
	return d.writeInternal(ctx, false, fn)
}

func (d *DB) writeInternal(ctx context.Context, high bool, fn func(tx *sql.Tx) error) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultWriteTTL)
		defer cancel()
	}
	if high {
		d.highWait.Add(1)
		defer d.highWait.Add(-1)
	}
	select {
	case d.sem <- struct{}{}:
		defer func() { <-d.sem }()
	case <-ctx.Done():
		return ctx.Err()
	}
	tx, err := d.W.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// WalValve bounds WAL growth during bulk indexing without blocking readers.
type WalValve struct {
	db       *sqlitepool.DB // replaced by *DB in package sqlitepool
	soft     int64
	hard     int64
	capBytes int64
	baseline int64
	running  atomic.Bool
}
```

Note: inside `package sqlitepool`, `WalValve` references `db *DB` directly:

```go
type WalValve struct {
	db       *DB
	soft     int64
	hard     int64
	capBytes int64
	baseline atomic.Int64
}

func NewWalValve(db *DB, soft, hard, capBytes int64) *WalValve {
	if soft <= 0 {
		soft = 256 << 20
	}
	if hard <= 0 {
		hard = 512 << 20
	}
	if capBytes <= 0 {
		capBytes = 1024 << 20
	}
	return &WalValve{db: db, soft: soft, hard: hard, capBytes: capBytes}
}

func (v *WalValve) walSize() int64 {
	st, err := os.Stat(v.db.Path + "-wal")
	if err != nil {
		return 0
	}
	return st.Size()
}

func (v *WalValve) Begin(ctx context.Context) error {
	v.baseline.Store(v.walSize())
	_, err := v.db.W.ExecContext(ctx, `PRAGMA wal_autocheckpoint = 0`)
	return err
}

func (v *WalValve) Step(ctx context.Context) error {
	cur := v.walSize()
	if cur >= v.capBytes {
		return apperr.Newf(apperr.Internal, "WAL size %d exceeded safety cap %d", cur, v.capBytes)
	}
	growth := cur - v.baseline.Load()
	if growth >= v.hard {
		_, err := v.db.W.ExecContext(ctx, `PRAGMA wal_checkpoint(RESTART)`)
		v.baseline.Store(v.walSize())
		return err
	}
	if growth >= v.soft {
		_, _ = v.db.W.ExecContext(ctx, `PRAGMA wal_checkpoint(PASSIVE)`)
		v.baseline.Store(v.walSize())
	}
	return nil
}

func (v *WalValve) Finish(ctx context.Context) error {
	_, _ = v.db.W.ExecContext(ctx, `PRAGMA wal_autocheckpoint = 1000`)
	_, err := v.db.W.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`)
	return err
}
```

And update `(d *DB) Write` in `internal/platform/sqlitepool/pool.go` to delegate to `d.writeInternal(ctx, true, fn)` and add `highWait atomic.Int32` to `type DB struct`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `CGO_ENABLED=0 go test -race ./internal/platform/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/platform/apperr internal/platform/project internal/platform/sqlitepool
git commit -m "feat(platform): add Phase 2 error codes, root confinement, priority WriteBatch and WalValve"
```

---

### Task 2: Code Subsystem API Types & Core SQLite Store (`001_code_core.sql`)

**Files:**
- Create: `internal/code/api/types.go`, `internal/code/api/service.go`
- Create: `internal/code/store/migrations/001_code_core.sql`
- Create: `internal/code/store/store.go`, `internal/code/store/batch.go`, `internal/code/store/graph_io.go`, `internal/code/store/search_io.go`
- Test: `internal/code/store/store_test.go`

**Interfaces:**
- Consumes: `sqlitepool.DB`, `migrate.Apply`, `apperr`.
- Produces:
  - `api.Service`, `api.Symbol`, `api.FileRecord`, `api.Chunk`, `api.Edge`, `api.UnresolvedRef`, `api.ExploreRequest`, `api.ExploreResult`, `api.ReadRequest`, `api.ImpactRequest`, `api.AffectedRequest`, `api.RenameRequest`.
  - `store.Init(ctx context.Context, db *sqlitepool.DB, isGlobal bool) error` (no-op when `isGlobal == true`).
  - `store.SymbolUID(repoRelPath, kind, qualifiedName, sig string) string` and `store.SegmentizeIdentifier(name string) []string`.
  - `store.ResurrectIncomingEdges(ctx, tx, paths []string) error`, `store.DeleteFiles(ctx, tx, paths []string) error`, `store.UpsertFileBundle(ctx, tx, bundle FileBundle) error`, `store.ReopenFailedRefs(ctx, tx, exportedNames []string, manifestChanged bool) error`.

- [ ] **Step 1: Write failing store & schema tests**

`internal/code/store/store_test.go`:

```go
package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"reflect"
	"testing"

	"mnemonic/internal/code/api"
	"mnemonic/internal/platform/migrate"
	"mnemonic/internal/platform/sqlitepool"
)

func openTestRepoDB(t *testing.T) *sqlitepool.DB {
	t.Helper()
	p := sqlitepool.NewPool()
	t.Cleanup(func() { _ = p.Close() })
	d, err := p.Get(filepath.Join(t.TempDir(), "repo.db"), func(db *sqlitepool.DB) error {
		if _, err := db.W.Exec(`CREATE TABLE observations(uid TEXT PRIMARY KEY, type TEXT, title TEXT, deleted_at TEXT)`); err != nil {
			return err
		}
		return Init(context.Background(), db, false)
	})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestGlobalDBSkipsCodeMigrations(t *testing.T) {
	p := sqlitepool.NewPool()
	defer p.Close()
	g, err := p.Get(filepath.Join(t.TempDir(), "global.db"), func(db *sqlitepool.DB) error {
		if err := migrate.Apply(context.Background(), db, "platform", migrate.PlatformFS()); err != nil {
			return err
		}
		return Init(context.Background(), db, true)
	})
	if err != nil {
		t.Fatal(err)
	}
	var n int
	_ = g.R.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name='code_symbols'`).Scan(&n)
	if n != 0 {
		t.Fatalf("global.db must not contain code_symbols, got %d", n)
	}
}

func TestSegmentizeAndSymbolUIDCollisionProof(t *testing.T) {
	got := SegmentizeIdentifier("parseHTTPRequestURL_v2")
	want := []string{"http", "parse", "request", "url", "v2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SegmentizeIdentifier = %v, want %v", got, want)
	}
	u1 := SymbolUID("a/handler.go", "function", "pkg.Handle", "func()")
	u2 := SymbolUID("b/handler.go", "function", "pkg.Handle", "func()")
	if u1 == u2 {
		t.Fatal("identical symbol names in different files must never collide on UID")
	}
}

func TestUpsertFileBundleAndIncomingEdgeResurrection(t *testing.T) {
	d := openTestRepoDB(t)
	ctx := context.Background()

	fileA := FileBundle{
		File: api.FileRecord{Path: "pkg/a.go", Language: "go", Tier: "rich", Size: 64, MtimeNS: 1, ContentHash: "h1", ASTHash: "a1"},
		Symbols: []api.Symbol{
			{UID: SymbolUID("pkg/a.go", "file", "pkg/a.go", ""), Name: "a.go", QualifiedName: "pkg/a.go", Kind: "file", StartLine: 1, EndLine: 5, BodyHash: "b0"},
			{UID: SymbolUID("pkg/a.go", "function", "pkg.Target", "func()"), Name: "Target", QualifiedName: "pkg.Target", Kind: "function", IsExported: true, StartLine: 2, EndLine: 4, BodyHash: "b1"},
		},
	}
	fileB := FileBundle{
		File: api.FileRecord{Path: "pkg/b.go", Language: "go", Tier: "rich", Size: 64, MtimeNS: 1, ContentHash: "h2", ASTHash: "a2"},
		Symbols: []api.Symbol{
			{UID: SymbolUID("pkg/b.go", "file", "pkg/b.go", ""), Name: "b.go", QualifiedName: "pkg/b.go", Kind: "file", StartLine: 1, EndLine: 5, BodyHash: "b0"},
			{UID: SymbolUID("pkg/b.go", "function", "pkg.Caller", "func()"), Name: "Caller", QualifiedName: "pkg.Caller", Kind: "function", IsExported: true, StartLine: 2, EndLine: 4, BodyHash: "b2"},
		},
	}

	if err := d.Write(ctx, func(tx *sql.Tx) error {
		if err := UpsertFileBundle(ctx, tx, fileA); err != nil {
			return err
		}
		if err := UpsertFileBundle(ctx, tx, fileB); err != nil {
			return err
		}
		return InsertResolvedEdgeByUID(ctx, tx, "pkg/b.go", fileB.Symbols[1].UID, fileA.Symbols[1].UID, "calls", "EXTRACTED", 3, 2)
	}); err != nil {
		t.Fatal(err)
	}

	// Simulate modifying pkg/a.go: incoming cross-file edge from Caller -> Target must resurrect into code_unresolved_refs!
	if err := d.Write(ctx, func(tx *sql.Tx) error {
		if err := ResurrectIncomingEdges(ctx, tx, []string{"pkg/a.go"}); err != nil {
			return err
		}
		return DeleteFiles(ctx, tx, []string{"pkg/a.go"})
	}); err != nil {
		t.Fatal(err)
	}

	var pending int
	if err := d.R.QueryRow(`SELECT COUNT(*) FROM code_unresolved_refs WHERE status='pending' AND name_tail='Target'`).Scan(&pending); err != nil || pending != 1 {
		t.Fatalf("pending resurrected refs = %d (err %v), want 1", pending, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/code/store/... -v`
Expected: FAIL (`cannot find package "mnemonic/internal/code/store"`).

- [ ] **Step 3: Create `api/types.go`, `001_code_core.sql` (verbatim from Spec §3.1), and `store/*.go`**

Create `internal/code/api/types.go` & `internal/code/api/service.go`:

```go
package api

import "context"

type FileRecord struct {
	ID          int64  `json:"id"`
	Path        string `json:"path"`
	Language    string `json:"language"`
	Tier        string `json:"tier"`
	Size        int64  `json:"size"`
	MtimeNS     int64  `json:"mtime_ns"`
	ContentHash string `json:"content_hash"`
	ASTHash     string `json:"ast_hash"`
	IsGenerated bool   `json:"is_generated"`
	IsTest      bool   `json:"is_test"`
	ParseErrors int    `json:"parse_errors"`
}

type Symbol struct {
	ID            int64   `json:"id"`
	UID           string  `json:"uid"`
	FileID        int64   `json:"file_id"`
	FilePath      string  `json:"file_path,omitempty"`
	ParentUID     string  `json:"parent_uid,omitempty"`
	Name          string  `json:"name"`
	QualifiedName string  `json:"qualified_name"`
	Kind          string  `json:"kind"`
	Signature     string  `json:"signature"`
	Docstring     string  `json:"docstring,omitempty"`
	ReturnType    string  `json:"return_type,omitempty"`
	Visibility    string  `json:"visibility"`
	IsExported    bool    `json:"is_exported"`
	StartLine     int     `json:"start_line"`
	StartCol      int     `json:"start_col"`
	EndLine       int     `json:"end_line"`
	EndCol        int     `json:"end_col"`
	BodyHash      string  `json:"body_hash"`
	CommunityID   *int64  `json:"community_id,omitempty"`
	PageRank      float64 `json:"pagerank,omitempty"`
}

type Chunk struct {
	SymbolUID   string `json:"symbol_uid,omitempty"`
	Kind        string `json:"kind"` // ast | skeleton | window | doc_section
	StartLine   int    `json:"start_line"`
	EndLine     int    `json:"end_line"`
	ContentHash string `json:"content_hash"`
	Text        string `json:"text"`
}

type RawRef struct {
	FromSymbolUID string `json:"from_symbol_uid"`
	ToSymbolUID   string `json:"to_symbol_uid,omitempty"` // populated for intra-file resolved targets
	RawTarget     string `json:"raw_target"`
	NameTail      string `json:"name_tail"`
	ModuleSpec    string `json:"module_spec,omitempty"`
	ReceiverHint  string `json:"receiver_hint,omitempty"`
	Kind          string `json:"kind"`
	Confidence    string `json:"confidence"`
	Line          int    `json:"line"`
	Col           int    `json:"col"`
}

type LinkedMemory struct {
	ObservationUID string `json:"observation_uid"`
	Type           string `json:"type"`
	Title          string `json:"title"`
	LinkKind       string `json:"link_kind"`
}

type ExploreCluster struct {
	FilePath       string         `json:"file_path"`
	StartLine      int            `json:"start_line"`
	EndLine        int            `json:"end_line"`
	Score          float64        `json:"score"`
	Symbols        []string       `json:"symbols"`
	CallTrail      []string       `json:"call_trail,omitempty"`
	LinkedMemories []LinkedMemory `json:"linked_memories,omitempty"`
	Source         string         `json:"source"`
}

type ExploreRequest struct {
	Project      string `json:"project,omitempty"`
	RootDir      string `json:"root_dir,omitempty"`
	Query        string `json:"query"`
	BudgetBytes  int    `json:"budget_bytes,omitempty"`
	IncludeTests bool   `json:"include_tests,omitempty"`
}

type ExploreResult struct {
	Clusters   []ExploreCluster `json:"clusters"`
	StaleFiles []string         `json:"stale_files,omitempty"`
	Degraded   bool             `json:"degraded"`
	TotalBytes int              `json:"total_bytes"`
}
```

Create `internal/code/api/service.go` defining `IndexRequest`, `IndexResult`, `StatusResult`, `ReadRequest`, `ReadResult`, `SearchRequest`, `SearchHit`, `ImpactRequest`, `ImpactResult`, `AffectedRequest`, `AffectedResult`, `NeighborsRequest`, `RenameRequest`, `RenameResult`, and `Service`:

```go
package api

import "context"

type IndexRequest struct {
	Project string `json:"project,omitempty"`
	RootDir string `json:"root_dir"`
	Full    bool   `json:"full,omitempty"`
	LSP     bool   `json:"lsp,omitempty"`
}

type IndexResult struct {
	FilesIndexed    int   `json:"files_indexed"`
	FilesSkipped    int   `json:"files_skipped"`
	FilesDeleted    int   `json:"files_deleted"`
	SymbolsCount    int   `json:"symbols_count"`
	EdgesCount      int   `json:"edges_count"`
	UnresolvedCount int   `json:"unresolved_count"`
	GraphGeneration int64 `json:"graph_generation"`
}

type StatusResult struct {
	State           string          `json:"state"` // not_indexed | fresh | stale | corrupt
	ActiveRoot      string          `json:"active_root"`
	HeadCommit      string          `json:"head_commit"`
	FilesCount      int             `json:"files_count"`
	SymbolsCount    int             `json:"symbols_count"`
	EdgesCount      int             `json:"edges_count"`
	UnresolvedCount int             `json:"unresolved_count"`
	DirtyFiles      int             `json:"dirty_files"`
	Capabilities    map[string]bool `json:"capabilities"`
}

type ReadRequest struct {
	Project string `json:"project,omitempty"`
	RootDir string `json:"root_dir,omitempty"`
	Path    string `json:"path"`
	Symbol  string `json:"symbol,omitempty"`
	Start   int    `json:"start_line,omitempty"`
	End     int    `json:"end_line,omitempty"`
	Mode    string `json:"mode,omitempty"` // slice | skeleton | symbol
}

type ReadResult struct {
	Path           string         `json:"path"`
	StartLine      int            `json:"start_line"`
	EndLine        int            `json:"end_line"`
	Content        string         `json:"content"`
	Symbols        []Symbol       `json:"symbols,omitempty"`
	LinkedMemories []LinkedMemory `json:"linked_memories,omitempty"`
}

type SearchRequest struct {
	Project  string `json:"project,omitempty"`
	RootDir  string `json:"root_dir,omitempty"`
	Query    string `json:"query"`
	Mode     string `json:"mode,omitempty"` // hybrid | lexical | semantic | structural
	Language string `json:"language,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

type SearchHit struct {
	SymbolUID     string  `json:"symbol_uid"`
	QualifiedName string  `json:"qualified_name"`
	Kind          string  `json:"kind"`
	FilePath      string  `json:"file_path"`
	StartLine     int     `json:"start_line"`
	EndLine       int     `json:"end_line"`
	Signature     string  `json:"signature,omitempty"`
	Score         float64 `json:"score"`
}

type ImpactNode struct {
	SymbolUID     string `json:"symbol_uid"`
	QualifiedName string `json:"qualified_name"`
	FilePath      string `json:"file_path"`
	Line          int    `json:"line"`
	Depth         int    `json:"depth"`
	Confidence    string `json:"confidence"`
}

type ImpactRequest struct {
	Project       string `json:"project,omitempty"`
	Target        string `json:"target"`
	MaxDepth      int    `json:"max_depth,omitempty"`
	MinConfidence string `json:"min_confidence,omitempty"`
}

type ImpactResult struct {
	WillBreak      []ImpactNode `json:"will_break"`
	LikelyAffected []ImpactNode `json:"likely_affected"`
}

type AffectedRequest struct {
	Project  string `json:"project,omitempty"`
	RootDir  string `json:"root_dir"`
	Staged   bool   `json:"staged,omitempty"`
	BaseRef  string `json:"base_ref,omitempty"`
	MaxDepth int    `json:"max_depth,omitempty"`
}

type AffectedResult struct {
	ChangedSymbols []Symbol     `json:"changed_symbols"`
	ImpactedProd   []ImpactNode `json:"impacted_prod"`
	ImpactedTests  []ImpactNode `json:"impacted_tests"`
	AffectedFlows  []string     `json:"affected_flows,omitempty"`
}

type NeighborsRequest struct {
	Project string `json:"project,omitempty"`
	Symbol  string `json:"symbol"`
	Mode    string `json:"mode"` // callers | callees | implementors | hierarchy | tests
}

type RenameRequest struct {
	Project   string `json:"project,omitempty"`
	RootDir   string `json:"root_dir"`
	SymbolUID string `json:"symbol_uid"`
	NewName   string `json:"new_name"`
	DryRun    bool   `json:"dry_run"`
	PlanToken string `json:"plan_token,omitempty"`
}

type RenameResult struct {
	PlanToken            string   `json:"plan_token"`
	FilesTouched         []string `json:"files_touched"`
	EditsCount           int      `json:"edits_count"`
	UnifiedDiff          string   `json:"unified_diff,omitempty"`
	UnverifiedCandidates []string `json:"unverified_candidates,omitempty"`
	Applied              bool     `json:"applied"`
}

type Service interface {
	Index(ctx context.Context, req IndexRequest) (IndexResult, error)
	Status(ctx context.Context, project, rootDir string) (StatusResult, error)
	Explore(ctx context.Context, req ExploreRequest) (ExploreResult, error)
	Read(ctx context.Context, req ReadRequest) (ReadResult, error)
	Search(ctx context.Context, req SearchRequest) ([]SearchHit, bool, error)
	Impact(ctx context.Context, req ImpactRequest) (ImpactResult, error)
	Affected(ctx context.Context, req AffectedRequest) (AffectedResult, error)
	Neighbors(ctx context.Context, req NeighborsRequest) ([]ImpactNode, error)
	Rename(ctx context.Context, req RenameRequest) (RenameResult, error)
}
```

Copy Spec §3.1 DDL into `internal/code/store/migrations/001_code_core.sql` (including `_ "modernc.org/sqlite/vec"` registration in `store.go`). Implement `internal/code/store/store.go` & `internal/code/store/batch.go`:

```go
package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"io/fs"
	"sort"
	"strings"
	"time"
	"unicode"

	_ "modernc.org/sqlite/vec"

	"mnemonic/internal/code/api"
	"mnemonic/internal/platform/migrate"
	"mnemonic/internal/platform/sqlitepool"
)

//go:embed migrations/*.sql
var migrationsEmbed embed.FS

type FileBundle struct {
	File    api.FileRecord
	Symbols []api.Symbol
	Chunks  []api.Chunk
	Refs    []api.RawRef
}

func MigrationsFS() fs.FS {
	sub, err := fs.Sub(migrationsEmbed, "migrations")
	if err != nil {
		panic(err)
	}
	return sub
}

// Init applies code subsystem migrations strictly on <repo>.db (never global.db).
func Init(ctx context.Context, db *sqlitepool.DB, isGlobal bool) error {
	if isGlobal {
		return nil
	}
	return migrate.Apply(ctx, db, "code", MigrationsFS())
}

func SymbolUID(repoRelPath, kind, qualifiedName, sig string) string {
	sigSum := sha256.Sum256([]byte(sig))
	h := sha256.New()
	h.Write([]byte(repoRelPath))
	h.Write([]byte{0})
	h.Write([]byte(kind))
	h.Write([]byte{0})
	h.Write([]byte(qualifiedName))
	h.Write([]byte{0})
	h.Write(sigSum[:8])
	return hex.EncodeToString(h.Sum(nil))
}

// SegmentizeIdentifier splits camelCase, PascalCase, acronyms (HTTPRequest -> http, request),
// and snake_case into sorted unique lowercase tokens of length >= 2.
func SegmentizeIdentifier(ident string) []string {
	rs := []rune(ident)
	var parts []string
	start := 0
	flush := func(end int) {
		if end > start {
			tok := strings.ToLower(string(rs[start:end]))
			if len([]rune(tok)) >= 2 {
				parts = append(parts, tok)
			}
		}
	}
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush(i)
			start = i + 1
			continue
		}
		if i > start {
			prev := rs[i-1]
			if unicode.IsLower(prev) && unicode.IsUpper(r) {
				flush(i)
				start = i
			} else if unicode.IsUpper(prev) && unicode.IsUpper(r) && i+1 < len(rs) && unicode.IsLower(rs[i+1]) {
				flush(i)
				start = i
			}
		}
	}
	flush(len(rs))
	sort.Strings(parts)
	out := parts[:0]
	for _, p := range parts {
		if len(out) == 0 || out[len(out)-1] != p {
			out = append(out, p)
		}
	}
	return out
}

func nowISO() string { return time.Now().UTC().Format(time.RFC3339Nano) }
```

And in `internal/code/store/batch.go`:

```go
package store

import (
	"context"
	"database/sql"
	"strings"
)

// ResurrectIncomingEdges converts cross-file resolved edges targeting symbols in
// modified/deleted paths back into pending code_unresolved_refs before CASCADE delete.
func ResurrectIncomingEdges(ctx context.Context, tx *sql.Tx, paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	ph := strings.TrimRight(strings.Repeat("?,", len(paths)), ",")
	args := make([]any, len(paths))
	for i, p := range paths {
		args[i] = p
	}
	q := `INSERT INTO code_unresolved_refs(file_id, from_symbol_id, raw_target, name_tail, module_spec, receiver_hint, kind, line, col, status)
	      SELECT e.file_id, e.from_symbol_id, dst.qualified_name, dst.name, '', '', e.kind, COALESCE(e.line, 1), COALESCE(e.col, 0), 'pending'
	      FROM code_edges e
	      JOIN code_symbols src ON src.id = e.from_symbol_id
	      JOIN code_symbols dst ON dst.id = e.to_symbol_id
	      JOIN code_files df ON df.id = dst.file_id
	      WHERE df.path IN (` + ph + `) AND src.file_id != dst.file_id`
	_, err := tx.ExecContext(ctx, q, args...)
	return err
}

func DeleteFiles(ctx context.Context, tx *sql.Tx, paths []string) error {
	for _, p := range paths {
		if _, err := tx.ExecContext(ctx, `DELETE FROM code_files WHERE path = ?`, p); err != nil {
			return err
		}
	}
	return nil
}

func UpsertFileBundle(ctx context.Context, tx *sql.Tx, b FileBundle) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM code_files WHERE path = ?`, b.File.Path); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO code_files(path, language, tier, size, mtime_ns, content_hash, ast_hash, is_generated, is_test, parse_errors, indexed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		b.File.Path, b.File.Language, b.File.Tier, b.File.Size, b.File.MtimeNS,
		b.File.ContentHash, b.File.ASTHash, boolToInt(b.File.IsGenerated), boolToInt(b.File.IsTest), b.File.ParseErrors, nowISO(),
	)
	if err != nil {
		return err
	}
	fileID, err := res.LastInsertId()
	if err != nil {
		return err
	}
	uidToID := make(map[string]int64, len(b.Symbols))
	for _, s := range b.Symbols {
		var parentID any
		if s.ParentUID != "" {
			if pid, ok := uidToID[s.ParentUID]; ok {
				parentID = pid
			}
		}
		vis := s.Visibility
		if vis == "" {
			vis = "internal"
		}
		sr, err := tx.ExecContext(ctx, `INSERT INTO code_symbols(uid, file_id, parent_id, name, qualified_name, kind, signature, docstring, return_type, visibility, is_exported, start_line, start_col, end_line, end_col, body_hash)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			s.UID, fileID, parentID, s.Name, s.QualifiedName, s.Kind, s.Signature, s.Docstring,
			s.ReturnType, vis, boolToInt(s.IsExported), s.StartLine, s.StartCol, s.EndLine, s.EndCol, s.BodyHash,
		)
		if err != nil {
			return err
		}
		symID, _ := sr.LastInsertId()
		uidToID[s.UID] = symID
		for _, seg := range SegmentizeIdentifier(s.Name) {
			_, _ = tx.ExecContext(ctx, `INSERT OR IGNORE INTO code_name_segments(segment, symbol_id) VALUES (?, ?)`, seg, symID)
		}
	}
	for _, ch := range b.Chunks {
		var symID any
		if id, ok := uidToID[ch.SymbolUID]; ok {
			symID = id
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO code_chunks(file_id, symbol_id, kind, start_line, end_line, content_hash, text)
			VALUES (?, ?, ?, ?, ?, ?, ?)`, fileID, symID, ch.Kind, ch.StartLine, ch.EndLine, ch.ContentHash, ch.Text); err != nil {
			return err
		}
	}
	for _, ref := range b.Refs {
		fromID, ok := uidToID[ref.FromSymbolUID]
		if !ok {
			continue
		}
		if toID, ok := uidToID[ref.ToSymbolUID]; ok && ref.ToSymbolUID != "" {
			conf := ref.Confidence
			if conf == "" {
				conf = "EXTRACTED"
			}
			_, _ = tx.ExecContext(ctx, `INSERT OR IGNORE INTO code_edges(from_symbol_id, to_symbol_id, file_id, kind, confidence, line, col)
				VALUES (?, ?, ?, ?, ?, ?, ?)`, fromID, toID, fileID, ref.Kind, conf, ref.Line, ref.Col)
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO code_unresolved_refs(file_id, from_symbol_id, raw_target, name_tail, module_spec, receiver_hint, kind, line, col, status)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending')`,
			fileID, fromID, ref.RawTarget, ref.NameTail, ref.ModuleSpec, ref.ReceiverHint, ref.Kind, ref.Line, ref.Col); err != nil {
			return err
		}
	}
	return nil
}

func InsertResolvedEdgeByUID(ctx context.Context, tx *sql.Tx, filePath, fromUID, toUID, kind, conf string, line, col int) error {
	_, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO code_edges(from_symbol_id, to_symbol_id, file_id, kind, confidence, line, col)
		SELECT s1.id, s2.id, f.id, ?, ?, ?, ?
		FROM code_symbols s1, code_symbols s2, code_files f
		WHERE s1.uid = ? AND s2.uid = ? AND f.path = ?`,
		kind, conf, line, col, fromUID, toUID, filePath)
	return err
}

func ReopenFailedRefs(ctx context.Context, tx *sql.Tx, exportedNames []string, manifestChanged bool) error {
	if manifestChanged {
		_, err := tx.ExecContext(ctx, `UPDATE code_unresolved_refs SET status = 'pending' WHERE status = 'failed'`)
		return err
	}
	for _, name := range exportedNames {
		if _, err := tx.ExecContext(ctx, `UPDATE code_unresolved_refs SET status = 'pending' WHERE status = 'failed' AND name_tail = ?`, name); err != nil {
			return err
		}
	}
	return nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `CGO_ENABLED=0 go test -race ./internal/code/store/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/code/api internal/code/store
git commit -m "feat(code): add Phase 2A schema, standalone FTS5, segment vocab, and resurrection store"
```

---

### Task 3: Gitignore-Aware Scanner, Generated/Test Classifier & Recursive Watcher (`internal/code/scan`)

**Files:**
- Create: `internal/code/scan/ignore.go`, `internal/code/scan/scanner.go`, `internal/code/scan/watcher.go`
- Test: `internal/code/scan/scan_test.go`

**Interfaces:**
- Consumes: `project.ConfinePath`, `api.FileRecord`.
- Produces:
  - `scan.DeltaPlan{Modified []DiskFile, Deleted []string, Unchanged int, ManifestHash string}`
  - `scan.ScanDelta(activeRoot string, known map[string]api.FileRecord, forceFull bool) (DeltaPlan, error)`
  - `scan.ClassifyGenerated(relPath string, head []byte) bool` and `scan.ClassifyTest(relPath string) bool`
  - `scan.NewWatcher(activeRoot string, debounce time.Duration, onDirty func(paths []string)) (*Watcher, error)` (recursively subscribes non-ignored subdirectories and dynamically attaches newly created directories).

- [ ] **Step 1: Write failing scanner & recursive watcher tests**

`internal/code/scan/scan_test.go`:

```go
package scan

import (
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"mnemonic/internal/code/api"
)

func TestScanDeltaHonorsNestedGitignoreAndClassifiesFiles(t *testing.T) {
	root := t.TempDir()
	mustWrite := func(rel, body string) {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(".gitignore", "ignored_dir/\n*.secret.go\n")
	mustWrite("sub/.gitignore", "local_gen.ts\n")
	mustWrite("ignored_dir/skip.go", "package skip\n")
	mustWrite("sub/local_gen.ts", "export const x = 1;\n")
	mustWrite("api/user.pb.go", "// Code generated by protoc-gen-go. DO NOT EDIT.\npackage api\n")
	mustWrite("api/user_test.go", "package api\n")
	mustWrite("api/user.go", "package api\nfunc GetUser() {}\n")

	// Move process CWD outside root to enforce CWD independence.
	origCWD, _ := os.Getwd()
	_ = os.Chdir(t.TempDir())
	defer func() { _ = os.Chdir(origCWD) }()

	plan, err := ScanDelta(root, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	byPath := map[string]DiskFile{}
	for _, f := range plan.Modified {
		byPath[f.Path] = f
	}
	if len(byPath) != 3 {
		t.Fatalf("got %d files (%v), want 3", len(byPath), byPath)
	}
	if !byPath["api/user.pb.go"].IsGenerated {
		t.Error("expected api/user.pb.go IsGenerated=true")
	}
	if !byPath["api/user_test.go"].IsTest {
		t.Error("expected api/user_test.go IsTest=true")
	}

	// Second scan with matching size + mtime + hash skips all 3 files.
	known := map[string]api.FileRecord{}
	for k, v := range byPath {
		known[k] = api.FileRecord{Path: k, Size: v.Size, MtimeNS: v.MtimeNS, ContentHash: v.ContentHash}
	}
	plan2, err := ScanDelta(root, known, false)
	if err != nil || len(plan2.Modified) != 0 || plan2.Unchanged != 3 {
		t.Fatalf("second delta = %+v err=%v", plan2, err)
	}
}

func TestRecursiveWatcherDetectsEditsInNewlyCreatedSubdir(t *testing.T) {
	root := t.TempDir()
	var hits atomic.Int32
	w, err := NewWatcher(root, 120*time.Millisecond, func(paths []string) {
		hits.Add(1)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	deep := filepath.Join(root, "pkg", "deep")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	time.Sleep(60 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(deep, "svc.go"), []byte("package deep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && hits.Load() == 0 {
		time.Sleep(30 * time.Millisecond)
	}
	if hits.Load() == 0 {
		t.Fatal("recursive watcher missed file write inside newly created subdirectory")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/code/scan/... -v`
Expected: FAIL (`cannot find package "mnemonic/internal/code/scan"`).

- [ ] **Step 3: Implement `ignore.go`, `scanner.go`, and recursive `watcher.go`**

Create `internal/code/scan/ignore.go`:

```go
package scan

import (
	"os"
	"path"
	"path/filepath"
	"strings"
)

var hardcodedDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "dist": true,
	"build": true, "target": true, ".next": true, ".venv": true, "__pycache__": true,
}

type rule struct {
	baseRel string
	pattern string
	dirOnly bool
}

type IgnoreMatcher struct {
	rules []rule
}

func LoadRootIgnores(root string) *IgnoreMatcher {
	m := &IgnoreMatcher{}
	m.addFile(root, "", filepath.Join(root, ".git", "info", "exclude"))
	m.addFile(root, "", filepath.Join(root, ".gitignore"))
	m.addFile(root, "", filepath.Join(root, ".mnemonicignore"))
	return m
}

func (m *IgnoreMatcher) EnterDir(root, relDir string) {
	if relDir == "" || relDir == "." {
		return
	}
	m.addFile(root, relDir, filepath.Join(root, filepath.FromSlash(relDir), ".gitignore"))
}

func (m *IgnoreMatcher) addFile(_, baseRel, file string) {
	b, err := os.ReadFile(file)
	if err != nil {
		return
	}
	for _, raw := range strings.Split(string(b), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		dirOnly := strings.HasSuffix(line, "/")
		line = strings.Trim(line, "/")
		if line != "" {
			m.rules = append(m.rules, rule{baseRel: baseRel, pattern: line, dirOnly: dirOnly})
		}
	}
}

func (m *IgnoreMatcher) Ignored(relPath string, isDir bool) bool {
	base := path.Base(relPath)
	if isDir && hardcodedDirs[base] {
		return true
	}
	for _, r := range m.rules {
		if r.dirOnly && !isDir {
			continue
		}
		scoped := relPath
		if r.baseRel != "" {
			if !strings.HasPrefix(relPath, r.baseRel+"/") {
				continue
			}
			scoped = strings.TrimPrefix(relPath, r.baseRel+"/")
		}
		if ok, _ := path.Match(r.pattern, scoped); ok {
			return true
		}
		if ok, _ := path.Match(r.pattern, path.Base(scoped)); ok {
			return true
		}
	}
	return false
}
```

Create `internal/code/scan/scanner.go`:

```go
package scan

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"mnemonic/internal/code/api"
	"mnemonic/internal/code/extract"
)

const maxIndexFileBytes = 1 << 20 // 1 MiB

type DiskFile struct {
	Path        string
	AbsPath     string
	Language    string
	Tier        string
	Size        int64
	MtimeNS     int64
	ContentHash string
	IsGenerated bool
	IsTest      bool
}

type DeltaPlan struct {
	Modified     []DiskFile
	Deleted      []string
	Unchanged    int
	ManifestHash string
}

var manifestNames = map[string]bool{
	"go.mod": true, "go.work": true, "tsconfig.json": true, "package.json": true,
	"pyproject.toml": true, "Cargo.toml": true, "pom.xml": true, "build.gradle": true, "build.gradle.kts": true,
}

func ScanDelta(activeRoot string, known map[string]api.FileRecord, forceFull bool) (DeltaPlan, error) {
	realRoot, err := filepath.EvalSymlinks(activeRoot)
	if err != nil {
		return DeltaPlan{}, err
	}
	matcher := LoadRootIgnores(realRoot)
	seen := make(map[string]bool, len(known))
	var plan DeltaPlan
	manifestHasher := sha256.New()

	err = filepath.WalkDir(realRoot, func(cur string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if cur == realRoot {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		relOS, err := filepath.Rel(realRoot, cur)
		if err != nil {
			return nil
		}
		rel := filepath.ToSlash(relOS)
		if d.IsDir() {
			if matcher.Ignored(rel, true) {
				return filepath.SkipDir
			}
			matcher.EnterDir(realRoot, rel)
			return nil
		}
		if matcher.Ignored(rel, false) {
			return nil
		}
		base := path.Base(rel)
		if manifestNames[base] || strings.HasSuffix(base, ".csproj") || strings.HasSuffix(base, ".sln") {
			if b, err := os.ReadFile(cur); err == nil {
				manifestHasher.Write([]byte(rel))
				manifestHasher.Write(b)
			}
		}
		lang, tier, ok := extract.DetectLanguage(rel)
		if !ok {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > maxIndexFileBytes {
			return nil
		}
		seen[rel] = true
		mtime := info.ModTime().UnixNano()
		if prev, exists := known[rel]; exists && !forceFull && prev.Size == info.Size() && prev.MtimeNS == mtime {
			plan.Unchanged++
			return nil
		}
		raw, err := os.ReadFile(cur)
		if err != nil {
			return nil
		}
		probe := raw
		if len(probe) > 8192 {
			probe = probe[:8192]
		}
		if bytes.IndexByte(probe, 0) >= 0 {
			delete(seen, rel)
			return nil
		}
		sum := sha256.Sum256(raw)
		hashHex := hex.EncodeToString(sum[:])
		if prev, exists := known[rel]; exists && !forceFull && prev.ContentHash == hashHex {
			plan.Unchanged++
			return nil
		}
		plan.Modified = append(plan.Modified, DiskFile{
			Path:        rel,
			AbsPath:     cur,
			Language:    lang,
			Tier:        tier,
			Size:        info.Size(),
			MtimeNS:     mtime,
			ContentHash: hashHex,
			IsGenerated: ClassifyGenerated(rel, probe),
			IsTest:      ClassifyTest(rel),
		})
		return nil
	})
	if err != nil {
		return DeltaPlan{}, err
	}
	for k := range known {
		if !seen[k] {
			plan.Deleted = append(plan.Deleted, k)
		}
	}
	sort.Slice(plan.Modified, func(i, j int) bool { return plan.Modified[i].Path < plan.Modified[j].Path })
	sort.Strings(plan.Deleted)
	plan.ManifestHash = hex.EncodeToString(manifestHasher.Sum(nil))
	return plan, nil
}

func ClassifyGenerated(relPath string, head []byte) bool {
	lower := strings.ToLower(relPath)
	for _, suf := range []string{".pb.go", "_gen.go", ".generated.ts", ".min.js", ".g.dart"} {
		if strings.HasSuffix(lower, suf) {
			return true
		}
	}
	lines := strings.SplitN(string(head), "\n", 12)
	if len(lines) > 10 {
		lines = lines[:10]
	}
	header := strings.ToLower(strings.Join(lines, "\n"))
	return strings.Contains(header, "code generated by") || strings.Contains(header, "do not edit") || strings.Contains(header, "@generated")
}

func ClassifyTest(relPath string) bool {
	base := path.Base(relPath)
	if strings.HasSuffix(base, "_test.go") || strings.HasPrefix(base, "test_") ||
		strings.HasSuffix(base, "_test.py") || strings.Contains(base, ".test.") ||
		strings.Contains(base, ".spec.") || strings.HasSuffix(base, "Test.java") ||
		strings.HasSuffix(base, "Tests.cs") {
		return true
	}
	return strings.Contains("/"+relPath, "/__tests__/") || strings.Contains("/"+relPath, "/tests/")
}
```

Create `internal/code/scan/watcher.go` subscribing recursively with dynamic subdirectory attachment:

```go
package scan

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	root     string
	matcher  *IgnoreMatcher
	fsw      *fsnotify.Watcher
	debounce time.Duration
	onDirty  func([]string)
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

func NewWatcher(activeRoot string, debounce time.Duration, onDirty func([]string)) (*Watcher, error) {
	realRoot, err := filepath.EvalSymlinks(activeRoot)
	if err != nil {
		return nil, err
	}
	if debounce < 50*time.Millisecond {
		debounce = 1500 * time.Millisecond
	}
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &Watcher{
		root: realRoot, matcher: LoadRootIgnores(realRoot),
		fsw: fsw, debounce: debounce, onDirty: onDirty, stopCh: make(chan struct{}),
	}
	w.addTree(realRoot)
	w.wg.Add(1)
	go w.loop()
	return w, nil
}

func (w *Watcher) addTree(start string) {
	_ = filepath.WalkDir(start, func(cur string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() || d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if cur != w.root {
			rel, _ := filepath.Rel(w.root, cur)
			rel = filepath.ToSlash(rel)
			if w.matcher.Ignored(rel, true) {
				return filepath.SkipDir
			}
			w.matcher.EnterDir(w.root, rel)
		}
		_ = w.fsw.Add(cur)
		return nil
	})
}

func (w *Watcher) loop() {
	defer w.wg.Done()
	dirty := map[string]struct{}{}
	var timer *time.Timer
	var timerC <-chan time.Time

	flush := func() {
		if len(dirty) == 0 {
			return
		}
		paths := make([]string, 0, len(dirty))
		for p := range dirty {
			paths = append(paths, p)
		}
		clear(dirty)
		sort.Strings(paths)
		if w.onDirty != nil {
			w.onDirty(paths)
		}
	}

	for {
		select {
		case <-w.stopCh:
			if timer != nil {
				timer.Stop()
			}
			return
		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if ev.Has(fsnotify.Create) {
				if st, err := os.Stat(ev.Name); err == nil && st.IsDir() {
					w.addTree(ev.Name)
				}
			}
			if rel, err := filepath.Rel(w.root, ev.Name); err == nil {
				relSlash := filepath.ToSlash(rel)
				if !w.matcher.Ignored(relSlash, false) {
					dirty[relSlash] = struct{}{}
					if timer == nil {
						timer = time.NewTimer(w.debounce)
						timerC = timer.C
					} else {
						timer.Reset(w.debounce)
					}
				}
			}
		case <-w.fsw.Errors:
		case <-timerC:
			timer = nil
			timerC = nil
			flush()
		}
	}
}

func (w *Watcher) Close() error {
	close(w.stopCh)
	err := w.fsw.Close()
	w.wg.Wait()
	return err
}
```

- [ ] **Step 4: Run test after Task 4 scaffolds `extract.DetectLanguage` (or add `extract/languages.go` first)**

Create `internal/code/extract/languages.go` now so `scan` compiles cleanly, then run `CGO_ENABLED=0 go test -race ./internal/code/scan/... -v`.
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/code/scan internal/code/extract/languages.go
git commit -m "feat(code): add gitignore-aware delta scanner and recursive fsnotify watcher"
```

---

### Task 4: 3-Tier `gotreesitter` Extractor, Enclosing Span Stack & AST Chunker (`internal/code/extract`)

**Files:**
- Create: `internal/code/extract/languages.go`, `internal/code/extract/pool.go`, `internal/code/extract/extractor.go`, `internal/code/extract/chunker.go`
- Test: `internal/code/extract/extract_test.go`

**Interfaces:**
- Consumes: `store.SymbolUID`, `store.FileBundle`, `gotreesitter` + `grammars`.
- Produces:
  - `extract.DetectLanguage(relPath string) (lang string, tier string, ok bool)`
  - `extract.Stamp() string`
  - `extract.NewPool(workers int) *Pool`, `(p *Pool).ExtractFile(relPath, lang, tier string, src []byte) store.FileBundle`
  - `extract.FormatEmbeddingText(sym api.Symbol) string` (`"<kind>: <qualified_name>\nSig: <signature>\nDoc: <docstring>\nPath: <tail>"`).

- [ ] **Step 1: Write failing polyglot real-parser & circuit-breaker tests**

`internal/code/extract/extract_test.go`:

```go
package extract

import (
	"strings"
	"testing"
)

func TestPolyglotTier1ExtractionAndCallerAttribution(t *testing.T) {
	p := NewPool(2)
	goSrc := []byte(`package auth
// VerifyToken validates a bearer token.
func VerifyToken(tok string) bool {
	return parseClaims(tok)
}
func parseClaims(raw string) bool { return len(raw) > 0 }
`)
	b := p.ExtractFile("internal/auth/token.go", "go", "rich", goSrc)
	if len(b.Symbols) < 3 { // file root + VerifyToken + parseClaims
		t.Fatalf("got %d symbols, want >=3", len(b.Symbols))
	}
	if b.Symbols[0].Kind != "file" {
		t.Fatalf("first symbol kind = %q, want synthetic 'file' root", b.Symbols[0].Kind)
	}
	var foundIntraCall bool
	for _, r := range b.Refs {
		if r.NameTail == "parseClaims" && r.ToSymbolUID != "" && r.FromSymbolUID != b.Symbols[0].UID {
			foundIntraCall = true
		}
	}
	if !foundIntraCall {
		t.Fatalf("expected VerifyToken -> parseClaims call attributed to VerifyToken UID with resolved intra-file ToSymbolUID; refs=%+v", b.Refs)
	}
	proj := FormatEmbeddingText(b.Symbols[1])
	if !strings.Contains(proj, "VerifyToken") || !strings.Contains(proj, "auth/token.go") {
		t.Fatalf("unexpected embedding text: %s", proj)
	}
}

func TestCircuitBreakerRecoversFromGrammarPanic(t *testing.T) {
	p := NewPool(1)
	p.injectPanicForTest = "rust"
	for i := 0; i < 4; i++ {
		b := p.ExtractFile("src/lib.rs", "rust", "rich", []byte("pub fn compute_hash() {}\n"))
		if len(b.Symbols) < 2 {
			t.Fatalf("iter %d: expected fallback regex extraction after panic, got %d symbols", i, len(b.Symbols))
		}
	}
	if !p.IsTripped("rust") {
		t.Fatal("expected rust circuit breaker to trip after 3 panics")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/code/extract/... -v`
Expected: FAIL (`undefined: NewPool`).

- [ ] **Step 3: Implement `languages.go`, panic-safe `pool.go`, span-stack `extractor.go`, and `chunker.go`**

`internal/code/extract/languages.go` maps all 30+ extensions (`Tier 1 rich`: `.go`, `.ts`, `.tsx`, `.js`, `.jsx`, `.py`, `.rs`, `.java`, `.cs`; `Tier 2 map`: `.c`, `.h`, `.cpp`, `.rb`, `.php`, `.kt`, `.swift`, `.scala`, `.dart`, `.lua`, `.sh`, `.sql`, `.tf`, `.ex`, `.erl`, `.hs`, `.ml`, `.zig`, `.css`, `.scss`, `.html`, `.proto`, `.graphql`; `Tier 3 doc`: `.md`, `.yaml`, `.yml`, `.toml`, `.json`) and computes deterministic `Stamp()`.

`internal/code/extract/extractor.go` & `chunker.go`:
- Always prepend synthetic `api.Symbol{UID: store.SymbolUID(relPath, "file", relPath, ""), Name: path.Base(relPath), QualifiedName: relPath, Kind: "file", StartLine: 1, EndLine: lineCount}` as `Symbols[0]`.
- Walk `gotreesitter` tree maintaining an enclosing `[]api.Symbol` interval stack (`startByte <= node.StartByte() && node.EndByte() <= endByte`); every call/type/heritage/import site receives `FromSymbolUID = enclosingTop.UID` (defaulting to `Symbols[0].UID` for file-scope imports, **never** `0` or `file_id`).
- Intra-file call/type references matching a single symbol `Name` in the same file populate `ToSymbolUID` immediately (`Confidence: "EXTRACTED"`); cross-file or receiver-qualified references leave `ToSymbolUID: ""` so `store.UpsertFileBundle` queues them in `code_unresolved_refs(status='pending')`.
- `chunker.go` emits `ast` chunks (`<= 1,200` chars) for callables, `window` chunks (`80` lines, `10` overlap) for large functions, `skeleton` chunks for classes/structs/interfaces, and `FormatEmbeddingText(sym)` (`"<kind>: <qualified_name>\nSig: <signature>\nDoc: <docstring>\nPath: <last_2_segments>"`).

- [ ] **Step 4: Run test to verify it passes**

Run: `CGO_ENABLED=0 go test -race ./internal/code/extract/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum internal/code/extract
git commit -m "feat(code): add 3-tier gotreesitter extractor, span stack, circuit breaker and chunker"
```

---

### Task 5: 6-Ecosystem Workspace Manifests & Incremental Cross-File Resolver (`internal/code/resolve`)

**Files:**
- Create: `internal/code/resolve/manifests.go`, `internal/code/resolve/resolver.go`
- Test: `internal/code/resolve/resolve_test.go`

**Interfaces:**
- Consumes: `sqlitepool.DB`, `store`, `api`.
- Produces:
  - `resolve.WorkspaceIndex` (parses `go.mod`/`go.work`, `tsconfig.json`/`package.json`, `pyproject.toml`, `Cargo.toml`, `pom.xml`/`build.gradle`, `*.csproj`).
  - `resolve.ResolvePending(ctx context.Context, db *sqlitepool.DB, activeRoot string) (resolved int, failed int, err error)` (chunks 500 `pending` refs per `WriteBatch` transaction, resolving module paths, TS/Python barrel re-exports, Go/Rust/Java/C# receiver & interface edges, and marking remaining refs `failed`).

- [ ] **Step 1: Write failing incremental resurrection isomorphism test**

`internal/code/resolve/resolve_test.go`:

```go
package resolve

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"mnemonic/internal/code/extract"
	"mnemonic/internal/code/scan"
	"mnemonic/internal/code/store"
	"mnemonic/internal/platform/sqlitepool"
)

func TestIncrementalBarrelAndGoModuleResolutionMatchesCleanFullIndex(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		full := filepath.Join(root, filepath.FromSlash(rel))
		_ = os.MkdirAll(filepath.Dir(full), 0o755)
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/app\n\ngo 1.26\n")
	write("internal/svc/svc.go", "package svc\nfunc ExecuteOrder() {}\n")
	write("cmd/api/main.go", "package main\nimport \"example.com/app/internal/svc\"\nfunc main() { svc.ExecuteOrder() }\n")
	write("web/tsconfig.json", `{"compilerOptions":{"baseUrl":".","paths":{"@lib/*":["src/lib/*"]}}}`)
	write("web/src/lib/auth.ts", "export function verifySession(): boolean { return true; }\n")
	write("web/src/lib/index.ts", "export { verifySession } from './auth';\n")
	write("web/src/app.ts", "import { verifySession } from '@lib/index';\nexport function run() { return verifySession(); }\n")

	incDB := runMiniIndex(t, root, nil, true)
	// Rename ExecuteOrder -> SubmitOrder in svc.go; caller initially becomes unresolved ('failed'), then fix caller!
	write("internal/svc/svc.go", "package svc\nfunc SubmitOrder() {}\n")
	_ = runMiniIndex(t, root, incDB, false)
	write("cmd/api/main.go", "package main\nimport \"example.com/app/internal/svc\"\nfunc main() { svc.SubmitOrder() }\n")
	_ = runMiniIndex(t, root, incDB, false)

	cleanDB := runMiniIndex(t, root, nil, true)
	if edgeCount(t, incDB) != edgeCount(t, cleanDB) || edgeCount(t, cleanDB) < 2 {
		t.Fatalf("incremental edges=%d != clean full edges=%d", edgeCount(t, incDB), edgeCount(t, cleanDB))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/code/resolve/... -v`
Expected: FAIL (`undefined: ResolvePending`).

- [ ] **Step 3: Implement `manifests.go` and `resolver.go`**

- `manifests.go` loads Go module prefixes (`module <path>` -> directory), TS `paths`/`baseUrl`/workspace package maps, Python `src/` & package directories, Cargo `[workspace] members` + crate roots, Maven/Gradle `src/main/java` package roots, and `.csproj` root namespaces.
- `resolver.go`:
  1. Loads exported symbol table `(file_path, dir, name, kind, uid, id)` from `db.R`.
  2. Follows `re_exports` (`index.ts`, `__init__.py`, `pub use`) up to 6 hops with cycle visited-set guard.
  3. Resolves `pending` rows in batches of 500 via `db.WriteBatch`:
     - Exact import-path + `name_tail` match -> `EXTRACTED`.
     - Unique receiver-type method match or structural Go/TS/Rust/Java/C# interface match -> `INFERRED`.
     - Global exported `name_tail` fallback (`<= 3` candidates) -> `AMBIGUOUS`; otherwise updates `status = 'failed'`.

- [ ] **Step 4: Run test to verify it passes**

Run: `CGO_ENABLED=0 go test -race ./internal/code/resolve/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/code/resolve
git commit -m "feat(code): add 6-ecosystem manifest resolvers and incremental ref healing"
```

---

### Task 6: Generation-Fenced CSR Graph Projection, PageRank & Reverse BFS Impact (`internal/code/csr`)

**Files:**
- Create: `internal/code/csr/graph.go`, `internal/code/csr/pagerank.go`
- Test: `internal/code/csr/csr_test.go`

**Interfaces:**
- Consumes: `sqlitepool.DB`, `api.ImpactNode`.
- Produces:
  - `csr.Cache` (`Get(ctx context.Context, db *sqlitepool.DB) (*Graph, error)` — reloads from SQLite only when `code_index_meta.graph_generation` advances).
  - `(g *Graph).PersonalizedPageRank(seeds map[uint32]float64, includeTests bool) map[uint32]float64` (`alpha = 0.25`, `25` iterations, `0.30x` generated penalty, `0.50x` test penalty).
  - `(g *Graph).ReverseImpact(targetIDs []int64, maxDepth int, minConf string) (willBreak, likely []api.ImpactNode)`.

- [ ] **Step 1: Write failing CSR PageRank, demotion & reverse-BFS impact tests**

`internal/code/csr/csr_test.go`:

```go
package csr

import "testing"

func TestPersonalizedPageRankDemotesGeneratedAndTests(t *testing.T) {
	g := NewBuilder(4)
	// 0: seed handler, 1: prod service called by 0, 2: generated proto called by 0, 3: test helper calling 0
	g.SetNodeMeta(0, 100, false, false)
	g.SetNodeMeta(1, 101, false, false)
	g.SetNodeMeta(2, 102, true, false)
	g.SetNodeMeta(3, 103, false, true)
	g.AddEdge(0, 1, EdgeCalls, ConfExtracted)
	g.AddEdge(0, 2, EdgeCalls, ConfExtracted)
	g.AddEdge(3, 0, EdgeCalls, ConfExtracted)
	graph := g.Build(1)

	scores := graph.PersonalizedPageRank(map[uint32]float64{0: 1.0}, false)
	if scores[1] <= scores[2]*2.0 {
		t.Fatalf("expected prod service (%f) >> generated code (%f)", scores[1], scores[2])
	}
	if scores[1] <= scores[3] {
		t.Fatalf("expected prod service (%f) > test caller (%f)", scores[1], scores[3])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/code/csr/... -v`
Expected: FAIL (`undefined: NewBuilder`).

- [ ] **Step 3: Implement `csr/graph.go` and `csr/pagerank.go`**

```go
package csr

const (
	ConfAmbiguous   uint8 = 1
	ConfInferred    uint8 = 2
	ConfSynthesized uint8 = 3
	ConfExtracted   uint8 = 4
	ConfLSPResolved uint8 = 5
)

func ConfidenceWeight(c uint8) float64 {
	switch c {
	case ConfLSPResolved:
		return 1.00
	case ConfExtracted:
		return 0.95
	case ConfSynthesized:
		return 0.85
	case ConfInferred:
		return 0.65
	default:
		return 0.30
	}
}

type Graph struct {
	Generation int64
	SymIDs     []int64
	IDToIdx    map[int64]uint32
	Generated  []bool
	TestNode   []bool
	FwdOffsets []uint32
	FwdTargets []uint32
	FwdKinds   []uint8
	FwdConf    []uint8
	RevOffsets []uint32
	RevSources []uint32
	RevKinds   []uint8
	RevConf    []uint8
}
```

`PersonalizedPageRank` runs 25 power iterations with `alpha = 0.25` over undirected structural coupling (`Fwd` + `0.6 * Rev` weighted by `ConfidenceWeight`), multiplying both seed vector and post-iteration mass by `0.30` for `Generated[i]` and `0.50` for `TestNode[i]` when `!includeTests`. `ReverseImpact` runs staged BFS up to `maxDepth` (`<= 10`) over `RevOffsets`/`RevSources` filtered by `minConf`, classifying depth `1` into `WillBreak` and depth `>= 2` into `LikelyAffected`.

- [ ] **Step 4: Run test to verify it passes**

Run: `CGO_ENABLED=0 go test -race ./internal/code/csr/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/code/csr
git commit -m "feat(code): add generation-fenced CSR graph projection, PageRank and ReverseImpact"
```

---

### Task 7: 4-Signal RRF, `<=24 KiB` Cluster Allocator & Live Staleness Verifier (`internal/code/search`)

**Files:**
- Create: `internal/code/search/explore.go`, `internal/code/search/verify.go`
- Test: `internal/code/search/search_test.go`

**Interfaces:**
- Consumes: `sqlitepool.DB`, `csr.Graph`, `project.ConfinePath`, `platform/intelligence.Embedder`.
- Produces:
  - `search.RunExplore(ctx context.Context, db *sqlitepool.DB, g *csr.Graph, emb intelligence.Embedder, activeRoot string, req api.ExploreRequest) (api.ExploreResult, error)`
  - `search.ValidateAnswerFiles(activeRoot string, expected map[string]api.FileRecord, budget time.Duration) []string`
  - `search.RedactSecrets(src string) string`.

- [ ] **Step 1: Write failing budget-cap, whole-symbol cluster, secret-redaction & staleness tests**

`internal/code/search/search_test.go`:

```go
package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"mnemonic/internal/code/api"
)

func TestValidateAnswerFilesFlagsUnsavedDiskEdit(t *testing.T) {
	root := t.TempDir()
	f := filepath.Join(root, "svc.go")
	_ = os.WriteFile(f, []byte("package svc // edited\n"), 0o644)
	stale := ValidateAnswerFiles(root, map[string]api.FileRecord{
		"svc.go": {Path: "svc.go", Size: 10, MtimeNS: 1, ContentHash: "0000"},
	}, 250*time.Millisecond)
	if len(stale) != 1 || stale[0] != "svc.go" {
		t.Fatalf("stale = %v, want [svc.go]", stale)
	}
}

func TestRedactSecretsStripsPrivateTagsAndHighEntropyTokens(t *testing.T) {
	in := "const key = \"sk-proj-1234567890abcdefghijklmnop\"; // <private>my-pin</private>"
	out := RedactSecrets(in)
	if strings.Contains(out, "sk-proj-") || strings.Contains(out, "my-pin") || !strings.Contains(out, "[REDACTED]") {
		t.Fatalf("redaction failed: %s", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/code/search/... -v`
Expected: FAIL (`undefined: ValidateAnswerFiles`).

- [ ] **Step 3: Implement `explore.go` and `verify.go`**

`explore.go`:
1. Runs 4-way RRF ($K = 60$, top 60 seeds):
   - `code_symbols_fts` (`bm25(code_symbols_fts, 20.0, 8.0, 3.0, 1.0)`).
   - `code_name_segments` prose-token intersection weighted by `(matched / n_terms)^2`.
   - `code_chunks_fts` trigram matches (`>= 3` runes) mapped to `symbol_id`.
   - `code_symbols_vec` KNN (`vec0`, $k = 60$) when `emb != nil && emb.Ready()`; sets `Degraded = true` otherwise.
2. Calls `g.PersonalizedPageRank(seeds, req.IncludeTests)` + rescues `type_of`/`returns` struct/interface definitions referenced by the top-5 callables.
3. Applies `15%` relative cliff (max 12 primary symbols), groups by `file_path`, merges spans within `gap <= 8` lines into whole-symbol clusters (switching single functions `> 60%` of file budget to `skeleton` + matched hunk window rather than slicing mid-body), enforces `BudgetBytes` (`default 24576`, max `70%` per single file), reads via `project.ConfinePath(activeRoot, path)`, applies `RedactSecrets`, runs `ValidateAnswerFiles(..., 250*time.Millisecond)`, and attaches active `observation_code_links` rows.

`verify.go`:

```go
package search

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"regexp"
	"sort"
	"time"

	"mnemonic/internal/code/api"
	"mnemonic/internal/platform/project"
)

var (
	privateTagRE = regexp.MustCompile(`(?s)<private>.*?</private>`)
	apiTokenRE   = regexp.MustCompile(`\b(?:sk-[A-Za-z0-9_-]{20,}|ghp_[A-Za-z0-9]{20,}|AIza[0-9A-Za-z_-]{25,})\b`)
)

func RedactSecrets(src string) string {
	s := privateTagRE.ReplaceAllString(src, "[REDACTED]")
	return apiTokenRE.ReplaceAllString(s, "[REDACTED]")
}

func ValidateAnswerFiles(activeRoot string, expected map[string]api.FileRecord, budget time.Duration) []string {
	deadline := time.Now().Add(budget)
	var stale []string
	var bytesRead int64
	count := 0
	for rel, rec := range expected {
		if count >= 200 || bytesRead >= 8<<20 || time.Now().After(deadline) {
			break
		}
		count++
		abs, _, err := project.ConfinePath(activeRoot, rel)
		if err != nil {
			stale = append(stale, rel)
			continue
		}
		st, err := os.Stat(abs)
		if err != nil {
			stale = append(stale, rel)
			continue
		}
		if st.Size() == rec.Size && st.ModTime().UnixNano() == rec.MtimeNS {
			continue
		}
		b, err := os.ReadFile(abs)
		if err != nil {
			stale = append(stale, rel)
			continue
		}
		bytesRead += int64(len(b))
		sum := sha256.Sum256(b)
		if hex.EncodeToString(sum[:]) != rec.ContentHash {
			stale = append(stale, rel)
		}
	}
	sort.Strings(stale)
	return stale
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `CGO_ENABLED=0 go test -race ./internal/code/search/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/code/search
git commit -m "feat(code): add 4-signal RRF, <=24KiB PageRank cluster allocator and staleness verifier"
```

---

### Task 8: Two-Step `plan_token` Safe `code_rename` (`internal/code/rename`)

**Files:**
- Create: `internal/code/rename/rename.go`
- Test: `internal/code/rename/rename_test.go`

**Interfaces:**
- Consumes: `sqlitepool.DB`, `project.ConfinePath`, `apperr.Conflict`.
- Produces: `rename.Execute(ctx context.Context, db *sqlitepool.DB, activeRoot string, req api.RenameRequest) (api.RenameResult, error)`.

- [ ] **Step 1: Write failing 2-step preview, atomic apply & 409 drift rejection tests**

`internal/code/rename/rename_test.go`:

```go
package rename

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mnemonic/internal/code/api"
	"mnemonic/internal/platform/apperr"
)

func TestTwoStepRenamePreviewApplyAndStaleConflict(t *testing.T) {
	root, db, targetUID := seedTwoFileGoFixture(t)
	ctx := context.Background()

	preview, err := Execute(ctx, db, root, api.RenameRequest{
		RootDir: root, SymbolUID: targetUID, NewName: "VerifyAuthToken", DryRun: true,
	})
	if err != nil || preview.PlanToken == "" || preview.Applied || !strings.Contains(preview.UnifiedDiff, "+func VerifyAuthToken") {
		t.Fatalf("preview = %+v err=%v", preview, err)
	}

	// Mutate disk file before apply -> must reject with FINGERPRINT_CONFLICT (409)!
	callerPath := filepath.Join(root, "cmd", "main.go")
	origBytes, _ := os.ReadFile(callerPath)
	_ = os.WriteFile(callerPath, append(origBytes, []byte("// drift\n")...), 0o644)
	if _, err := Execute(ctx, db, root, api.RenameRequest{
		RootDir: root, SymbolUID: targetUID, NewName: "VerifyAuthToken", DryRun: false, PlanToken: preview.PlanToken,
	}); apperr.CodeOf(err) != apperr.Conflict {
		t.Fatalf("expected Conflict on drifted file, got %v", err)
	}

	// Restore original file and apply cleanly.
	_ = os.WriteFile(callerPath, origBytes, 0o644)
	applied, err := Execute(ctx, db, root, api.RenameRequest{
		RootDir: root, SymbolUID: targetUID, NewName: "VerifyAuthToken", DryRun: false, PlanToken: preview.PlanToken,
	})
	if err != nil || !applied.Applied || len(applied.FilesTouched) != 2 {
		t.Fatalf("apply = %+v err=%v", applied, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/code/rename/... -v`
Expected: FAIL (`undefined: Execute`).

- [ ] **Step 3: Implement `rename.go`**

`rename.Execute`:
1. Validates `NewName` is a valid lexical identifier (`^[A-Za-z_][A-Za-z0-9_]*$`, else `apperr.Validation`).
2. Loads target symbol declaration + all incoming `code_edges` where `confidence IN ('LSP_RESOLVED', 'EXTRACTED')`, while collecting `INFERRED` / `AMBIGUOUS` sites into `UnverifiedCandidates` (`file:line`) without mutating them.
3. Confines every participating path via `project.ConfinePath(activeRoot, rel)`, verifies exact identifier token match at `(line, col)`, builds unified diff hunks, and computes `plan_token = sha256(symbolUID + "\x00" + newName + "\x00" + sorted(rel + ":" + sha256(diskBytes)))`.
4. When `!req.DryRun`: compares recomputed token against `req.PlanToken` (returning `apperr.New(apperr.Conflict, "PLAN_STALE: target files modified since preview")` on mismatch), writes `.mnemonic-rename-*` temp files in each target directory, `Sync()`s, and `os.Rename`s over target files atomically.

- [ ] **Step 4: Run test to verify it passes**

Run: `CGO_ENABLED=0 go test -race ./internal/code/rename/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/code/rename
git commit -m "feat(code): add 2-step plan_token guarded atomic symbol rename"
```

---

### Task 9: `engine.Local` Orchestrator, `observation_code_links` & SLA Compliance Suite (`internal/code/engine`)

**Files:**
- Create: `internal/code/engine/local.go`
- Test: `internal/code/engine/local_test.go`

**Interfaces:**
- Consumes: `config.Paths`, `project`, `sqlitepool`, `scan`, `extract`, `resolve`, `csr`, `search`, `rename`, `intelligence.Embedder`.
- Produces:
  - `engine.NewLocal(paths config.Paths, pool *sqlitepool.Pool, emb intelligence.Embedder) *Local` implementing `api.Service`
  - `engine.SyncObservationLinks(ctx context.Context, tx *sql.Tx, obsUID, title, content string) error` (auto-populates `observation_code_links` when `mem_save` mentions repo file paths or exported `qualified_name`s).

- [ ] **Step 1: Write failing polyglot real-parser compliance & concurrent `mem_save` `<25ms` SLA tests**

`internal/code/engine/local_test.go`:

```go
package engine

import (
	"context"
	"database/sql"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"mnemonic/internal/code/api"
	"mnemonic/internal/platform/config"
	"mnemonic/internal/platform/sqlitepool"
)

func TestPolyglotComplianceAndMemoryCrossLinks(t *testing.T) {
	dataDir := t.TempDir()
	repoDir := writeSixLanguageFixtureRepo(t) // Go, TS, Python, Rust, Java, C#
	paths, _ := config.NewPaths(dataDir)
	pool := sqlitepool.NewPool()
	defer pool.Close()

	svc := NewLocal(paths, pool, nil)
	ctx := context.Background()
	idx, err := svc.Index(ctx, api.IndexRequest{Project: "poly-app", RootDir: repoDir, Full: true})
	if err != nil || idx.FilesIndexed < 6 || idx.SymbolsCount < 12 {
		t.Fatalf("Index = %+v err=%v", idx, err)
	}

	// Seed a Phase 1 observation mentioning internal/orders/service.go and verify code_explore surfaces it!
	db, _ := pool.Get(paths.ProjectDB("poly-app"), nil)
	_ = db.Write(ctx, func(tx *sql.Tx) error {
		_, _ = tx.Exec(`INSERT INTO observations(uid, type, title, deleted_at) VALUES ('obs-1', 'bugfix', 'Fixed race in ProcessOrder', NULL)`)
		return SyncObservationLinks(ctx, tx, "obs-1", "Fixed race in ProcessOrder", "Touched internal/orders/service.go ProcessOrder")
	})

	exp, err := svc.Explore(ctx, api.ExploreRequest{Project: "poly-app", RootDir: repoDir, Query: "process order"})
	if err != nil || len(exp.Clusters) == 0 || exp.TotalBytes > 24576 {
		t.Fatalf("Explore = %+v err=%v", exp, err)
	}
	if len(exp.Clusters[0].LinkedMemories) == 0 {
		t.Fatalf("expected linked Phase 1 observation in Explore cluster, got %+v", exp.Clusters[0])
	}
}

func TestConcurrentMemSaveLatencyUnder25msDuringBulkCodeIndex(t *testing.T) {
	dataDir := t.TempDir()
	repoDir := writeSyntheticMultiFileGoRepo(t, 250)
	paths, _ := config.NewPaths(dataDir)
	pool := sqlitepool.NewPool()
	defer pool.Close()

	svc := NewLocal(paths, pool, nil)
	ctx := context.Background()
	db, _ := svc.openRepoDB(ctx, "sla-repo")

	var wg sync.WaitGroup
	var durations []time.Duration
	var mu sync.Mutex
	done := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-done:
				return
			default:
				t0 := time.Now()
				_ = db.Write(ctx, func(tx *sql.Tx) error {
					_, err := tx.Exec(`INSERT OR REPLACE INTO observations(uid, type, title) VALUES ('u1', 'decision', 'fast')`)
					return err
				})
				mu.Lock()
				durations = append(durations, time.Since(t0))
				mu.Unlock()
				time.Sleep(2 * time.Millisecond)
			}
		}
	}()

	if _, err := svc.Index(ctx, api.IndexRequest{Project: "sla-repo", RootDir: repoDir, Full: true}); err != nil {
		t.Fatal(err)
	}
	close(done)
	wg.Wait()

	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	p99 := durations[len(durations)*99/100]
	if p99 > 25*time.Millisecond {
		t.Fatalf("p99 high-priority Write during bulk code index = %v, want <= 25ms", p99)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `CGO_ENABLED=0 go test ./internal/code/engine/... -v`
Expected: FAIL (`undefined: NewLocal`).

- [ ] **Step 3: Implement `engine/local.go`**

`engine.Local`:
- Guards concurrent explicit `Index --full` runs per slug via `sync.Map` (`apperr.IndexBusy`).
- Runs `scan.ScanDelta` -> parallel `extract.Pool` -> chunks `50` files per `db.WriteBatch` transaction wrapped in `sqlitepool.NewWalValve`, calling `store.ResurrectIncomingEdges`, `store.DeleteFiles`, `store.UpsertFileBundle`, and `store.ReopenFailedRefs` -> `resolve.ResolvePending` -> bumps `code_index_meta.graph_generation` -> notifies background symbol embedding backfill worker.
- Implements `Affected` by running `git -C activeRoot diff --unified=0 [base|--cached]` to map changed `(file_path, start_line..end_line)` hunks onto overlapping `code_symbols`, then calling `csr.Graph.ReverseImpact` and splitting `is_test` vs production callers.
- Implements `Read`, `Search` (`hybrid|lexical|semantic|structural`), `Impact`, `Neighbors`, and `Rename` delegating to `store`, `csr`, `search`, and `rename`.

- [ ] **Step 4: Run test to verify it passes**

Run: `CGO_ENABLED=0 go test -race ./internal/code/engine/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/code/engine
git commit -m "feat(code): add engine.Local cooperative DAG, git-hunk affected, and memory cross-links"
```

---

### Task 10: MCP Server (11 Phase-2A Tools), Daemon Routes/Workers, CLI (`mnemonic code …`), TUI Tab & `doctor --repair`

**Files:**
- Create: `internal/code/mcp/server.go`, `internal/code/mcp/server_test.go`
- Create: `internal/code/http/handlers.go`, `internal/code/client/client.go`, `internal/code/resolver/resolver.go`
- Create: `internal/code/cli/cli.go`, `internal/code/cli/cli_test.go`
- Create: `internal/code/tui/tab.go`
- Modify: `cmd/mnemonic/main.go`, `internal/platform/registry/registry.go`

**Interfaces:**
- Consumes: `api.Service`, `engine.Local`, `mcp-go`, `platform/ipc`, `platform/tuishell`.
- Produces:
  - `code/mcp.NewServer(svc api.Service, profile string) *server.MCPServer` registering **only** the 11 implemented Phase 2A tools:
    - **Eager Core (6)**: `code_explore`, `code_read`, `code_impact`, `code_affected`, `code_status`, `code_index`.
    - **Extended (`WithDeferLoading(true)`, 3)**: `code_search`, `code_neighbors`, `code_rename`.
    - **Admin (`WithDeferLoading(true)` on `admin|all`, 2)**: `code_watch`, `code_backfill_embeddings`.
  - `code/cli.Run(ctx context.Context, args []string, io cli.IO, svc api.Service) int` where `mnemonic code watch` blocks until `ctx.Done()` (`SIGINT`/`SIGTERM`), streaming JSON/line sync events.
  - `code.Subsystem` registered in `cmd/mnemonic/main.go` + `mnemonic doctor --repair` rebuilding `code_symbols_fts`, `code_chunks_fts`, `code_name_segments`, and `code_symbols_vec`.

- [ ] **Step 1: Write failing MCP progressive-disclosure & blocking-watch CLI tests**

`internal/code/mcp/server_test.go`:

```go
package mcp

import "testing"

func TestPhase2AMCPToolSurfaceHasZeroStubsAndCorrectDeferFlags(t *testing.T) {
	srv := NewServer(nil, "agent")
	tools := srv.ListTools()
	if len(tools) != 9 { // 6 Eager Core + 3 Extended Agent (Admin excluded in 'agent' profile)
		t.Fatalf("agent profile tool count = %d, want 9", len(tools))
	}
	srvAll := NewServer(nil, "all")
	if len(srvAll.ListTools()) != 11 {
		t.Fatalf("all profile tool count = %d, want 11", len(srvAll.ListTools()))
	}
}
```

`internal/code/cli/cli_test.go`:

```go
package cli

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestWatchCommandBlocksUntilContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Millisecond)
	defer cancel()
	var out bytes.Buffer
	start := time.Now()
	code := Run(ctx, []string{"watch", "--root", t.TempDir()}, IO{Out: &out, Err: &out}, newFakeWatchService())
	if code != 0 || time.Since(start) < 150*time.Millisecond {
		t.Fatalf("watch exited prematurely (code=%d, elapsed=%v)", code, time.Since(start))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `CGO_ENABLED=0 go test ./internal/code/mcp/... ./internal/code/cli/... -v`
Expected: FAIL (`undefined: NewServer`, `undefined: Run`).

- [ ] **Step 3: Implement `mcp`, `http`, `client`, `resolver`, `cli`, `tui`, and register `code` subsystem**

- `internal/code/mcp/server.go`: Attaches strict `draft-07` schemas (`additionalProperties: false`), `readOnlyHint: true` on read tools, and `WithDeferLoading(true)` on `code_search`, `code_neighbors`, `code_rename`, `code_watch`, and `code_backfill_embeddings`.
- `internal/code/http/handlers.go` & `client/client.go`: Mounts `/v1/code/{status,index,explore,read,search,impact,affected,neighbors,rename,watch,backfill}` on the daemon mux and implements `DaemonClient` parity tested via `engine/local_test.go`.
- `internal/code/cli/cli.go`: Implements `index`, `status`, `explore`, `read`, `search`, `impact`, `affected`, `neighbors`, `rename`, `watch` (blocking on `<-ctx.Done()`), `backfill`, and `mcp`.
- `internal/code/tui/tab.go`: Mounts `[2: Code]` tab into `platform/tuishell` with live `Explore` query bar, ranked symbol list, and `Enter` source + linked-memory inspector.

- [ ] **Step 4: Run full workspace build & race test suite**

Run: `CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go test -race ./...`
Expected: PASS (zero CGo, zero data races, zero stub tools).

- [ ] **Step 5: Commit**

```bash
git add cmd/mnemonic internal/platform/registry internal/code
git commit -m "feat(code): wire Phase 2A MCP server, daemon IPC, CLI, TUI tab and doctor repair"
```


