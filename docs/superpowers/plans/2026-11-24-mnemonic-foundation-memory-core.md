# Mnemonic Foundation + Memory Core (Plan 1 of 5) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a working, fully tested, in-process (no daemon) local-first SQLite memory: `mnemonic memory save|search|get` backed by `~/.mnemonic/global.db` + `~/.mnemonic/<repo>.db` with FTS5 lexical search, `topic_key` revisions, idempotent saves and sessions.

**Architecture:** `internal/platform/*` is the reusable kernel (errors, paths, project slug, SQLite pool, migration runner). `internal/memory/{api,store,engine,cli}` is the first subsystem; `engine.Local` implements `api.Service` and is the only code that knows about DB routing. Later plans add vectors, MCP, daemon, TUI behind the same `api.Service`.

**Tech Stack:** Go 1.26, `modernc.org/sqlite` (pure Go, zero CGo), `github.com/google/uuid` (UUIDv7), `golang.org/x/text` (NFC), stdlib `flag`/`testing`.

## Scope of this plan (decomposition of the spec)

The spec (`docs/superpowers/specs/2026-11-24-mnemonic-sqlite-memory-design.md`) covers five independently shippable plans. This is Plan 1. Each later plan gets its own document after the previous one ships.

| Plan | Delivers | Spec sections |
|---|---|---|
| **1 (this)** | platform kernel, schema (non-vector), save pipeline (redact/idempotency/topic_key/dedup), lexical search, sessions, CLI `memory save/search/get/session` | 2.1, 3.1-3.5 (minus vectors), 4.2 (minus LLM/vector/conflicts), 4.3 (lexical leg), 6 |
| 2 | embeddings (ONNX/HTTP), `vec0` tables, RRF hybrid search, conflict candidates, judge/compare/consolidate, context/timeline/lookup/pin/prompts/review | 3.5 vectors, 4.1-4.4 |
| 3 | `mnemonic memory mcp` (progressive disclosure, profiles) | 5.1 |
| 4 | daemon (Unix socket + bearer HTTP), `client.DaemonClient`, `ResolveBackend`, workers | 2.3, 5.2 |
| 5 | Bubbletea TUI, `doctor --repair`, `models`, `.mnemonic.yaml`/alias resolution | 5.4, 6 |

## Global Constraints

- Go module path: `mnemonic`; no CGo (`CGO_ENABLED=0 go build ./...` must pass).
- SQL lives only in `internal/memory/store` and `internal/platform/{migrate,sqlitepool}`; `engine` never builds SQL strings.
- Files stay under ~600 LOC; one aggregate per file.
- All timestamps stored as UTC RFC3339Nano text.
- Observation `type` ∈ `decision architecture bugfix pattern config discovery learning preference convention`.
- Scope ∈ `project user global`; `project` → `<repo>.db`, `user|global` → `global.db`.
- Content cap 32 KiB (truncate at a UTF-8 rune boundary); `<private>…</private>` → `[REDACTED]` before hashing/storage.
- Error codes and exit codes: `VALIDATION_ERROR`=2, `UNAUTHORIZED`=3, `NOT_FOUND`=4, `FINGERPRINT_CONFLICT`=5, `TOMBSTONE_GONE`=6, `UNSUPPORTED_FS`=7, `STORAGE_CORRUPT`=8, anything else 1.
- Reserved DB basenames: `global config models run tmp wal` → slug becomes `repo-<slug>`.

## Spec amendments made by this plan (found while planning)

1. **FTS tables are standalone, not external-content** (`fts5(...)` without `content='observations'`). A plain `DELETE FROM fts WHERE rowid=…` inside an `AFTER DELETE` trigger is only reliable when the FTS table owns its text; external-content tables read the (already deleted) source row. Verified in a scratch program (insert/update/soft-delete all keep the index consistent).
2. **No FK from `observations.session_id` / `user_prompts.session_id` / `session_trails.session_id` to `sessions`.** A `user|global` observation lives in `global.db` while its session lives in the project DB; an FK would fail across databases. `session_id` is a plain indexed `TEXT`.
3. **`FileControlPersistWAL` is not used.** Instead, when the inode fence detects a replaced/deleted DB, the pool closes the old handles and deletes the stale `-wal`/`-shm` files before reopening (prevents replaying an old WAL into a new DB file).

Task 1 applies these to the spec file.

## File Structure

```text
go.mod
.gitignore
cmd/mnemonic/main.go                              # root dispatcher: `memory ...`, `version`
internal/platform/apperr/{apperr.go,apperr_test.go}
internal/platform/config/{paths.go,paths_test.go}
internal/platform/project/{project.go,project_test.go}
internal/platform/sqlitepool/{pool.go,fs.go,fs_darwin.go,fs_linux.go,fs_other.go,pool_test.go}
internal/platform/migrate/{migrate.go,migrate_test.go,sql/001_platform.sql}
internal/memory/api/types.go
internal/memory/store/{store.go,sessions.go,observations.go,operations.go,projects.go,search.go,store_test.go,search_test.go,migrations/001_init.sql}
internal/memory/engine/{local.go,save.go,read.go,session.go,save_test.go,read_test.go,session_test.go,testutil_test.go}
internal/memory/cli/{cli.go,cli_test.go}
```

---

### Task 1: Scaffold, spec amendments, `apperr`

**Files:**
- Create: `go.mod`, `.gitignore`, `internal/platform/apperr/apperr.go`
- Test: `internal/platform/apperr/apperr_test.go`
- Modify: `docs/superpowers/specs/2026-11-24-mnemonic-sqlite-memory-design.md`

**Interfaces:**
- Produces: `apperr.Code`, constants (`Validation`, `Unauthorized`, `NotFound`, `Conflict`, `Gone`, `UnsupportedFS`, `Corrupt`, `Internal`), `apperr.New(code Code, msg string) *Error`, `apperr.Newf(code Code, format string, args ...any) *Error`, `apperr.CodeOf(err error) Code`, `apperr.ExitCode(Code) int`, `apperr.HTTPStatus(Code) int`.

- [ ] **Step 1: Create module and ignore file**

```bash
cd /Users/paladm/git/ai-test/skillgrid-mnamonic
printf 'module mnemonic\n\ngo 1.26\n' > go.mod
printf '/dist/\n/bin/\n*.db\n*.db-wal\n*.db-shm\n.DS_Store\n' > .gitignore
```

- [ ] **Step 2: Write the failing test**

`internal/platform/apperr/apperr_test.go`:

```go
package apperr

import (
	"errors"
	"fmt"
	"testing"
)

func TestCodeOfAndMappings(t *testing.T) {
	cases := []struct {
		code   Code
		exit   int
		status int
	}{
		{Validation, 2, 400},
		{Unauthorized, 3, 401},
		{NotFound, 4, 404},
		{Conflict, 5, 409},
		{Gone, 6, 410},
		{UnsupportedFS, 7, 500},
		{Corrupt, 8, 500},
		{Internal, 1, 500},
	}
	for _, c := range cases {
		err := fmt.Errorf("wrapped: %w", New(c.code, "boom"))
		if got := CodeOf(err); got != c.code {
			t.Errorf("CodeOf = %q, want %q", got, c.code)
		}
		if got := ExitCode(c.code); got != c.exit {
			t.Errorf("ExitCode(%q) = %d, want %d", c.code, got, c.exit)
		}
		if got := HTTPStatus(c.code); got != c.status {
			t.Errorf("HTTPStatus(%q) = %d, want %d", c.code, got, c.status)
		}
	}
	if CodeOf(errors.New("plain")) != Internal {
		t.Error("plain errors must map to Internal")
	}
	if got := Newf(NotFound, "uid %s", "x").Error(); got != "NOT_FOUND: uid x" {
		t.Errorf("Error() = %q", got)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/platform/apperr/ -v`
Expected: FAIL (`undefined: Validation` / package has no non-test files).

- [ ] **Step 4: Write implementation**

`internal/platform/apperr/apperr.go`:

```go
// Package apperr defines the canonical domain error codes shared by the CLI,
// MCP, and daemon transports.
package apperr

import (
	"errors"
	"fmt"
)

type Code string

const (
	Validation    Code = "VALIDATION_ERROR"
	Unauthorized  Code = "UNAUTHORIZED"
	NotFound      Code = "NOT_FOUND"
	Conflict      Code = "FINGERPRINT_CONFLICT"
	Gone          Code = "TOMBSTONE_GONE"
	UnsupportedFS Code = "UNSUPPORTED_FS"
	Corrupt       Code = "STORAGE_CORRUPT"
	Internal      Code = "INTERNAL"
)

// Error is a domain error carrying a canonical Code.
type Error struct {
	Code Code
	Msg  string
}

func (e *Error) Error() string { return string(e.Code) + ": " + e.Msg }

func New(code Code, msg string) *Error { return &Error{Code: code, Msg: msg} }

func Newf(code Code, format string, args ...any) *Error {
	return &Error{Code: code, Msg: fmt.Sprintf(format, args...)}
}

// CodeOf returns the Code of the first *Error in err's chain, else Internal.
func CodeOf(err error) Code {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return Internal
}

var exitCodes = map[Code]int{
	Validation: 2, Unauthorized: 3, NotFound: 4, Conflict: 5,
	Gone: 6, UnsupportedFS: 7, Corrupt: 8, Internal: 1,
}

var httpStatuses = map[Code]int{
	Validation: 400, Unauthorized: 401, NotFound: 404, Conflict: 409,
	Gone: 410, UnsupportedFS: 500, Corrupt: 500, Internal: 500,
}

func ExitCode(c Code) int {
	if v, ok := exitCodes[c]; ok {
		return v
	}
	return 1
}

func HTTPStatus(c Code) int {
	if v, ok := httpStatuses[c]; ok {
		return v
	}
	return 500
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/platform/apperr/ -v`
Expected: PASS

- [ ] **Step 6: Apply the three spec amendments**

In `docs/superpowers/specs/2026-11-24-mnemonic-sqlite-memory-design.md`:

1. In section 3.5 replace the `observations_fts` definition lines `content = 'observations',` and `content_rowid = 'id'` by removing both (leave `tokenize = 'trigram remove_diacritics 1'` as the last option), do the same for `prompts_fts`, and change the `obs_fts_au` trigger header to `AFTER UPDATE OF title, content, topic_key, type, tool_name, deleted_at ON observations`.
2. In the `observations`, `user_prompts`, `session_trails` DDL: remove `REFERENCES sessions(id) ON DELETE SET NULL` / `REFERENCES sessions(id) ON DELETE CASCADE` from `session_id` columns (keep them as plain `TEXT`, `NOT NULL` for `session_trails`).
3. In section 3.4 delete the `FileControlPersistWAL("main", 1)` mention and add: "On generation change the pool closes old handles and removes stale `-wal`/`-shm` files before reopening."

Run: `rg -n "content_rowid|REFERENCES sessions|FileControlPersistWAL" docs/superpowers/specs/`
Expected: no matches.

- [ ] **Step 7: Commit**

```bash
git add go.mod .gitignore internal/platform/apperr docs/superpowers/specs
git commit -m "feat(platform): scaffold module and apperr; amend spec (standalone FTS, no cross-DB session FK)"
```

---

### Task 2: `config` paths

**Files:**
- Create: `internal/platform/config/paths.go`
- Test: `internal/platform/config/paths_test.go`

**Interfaces:**
- Produces: `config.Paths{Dir string}`, `config.NewPaths(flagDir string) (Paths, error)` (resolution order: flag, `MNEMONIC_DATA_DIR`, `~/.mnemonic`; creates dir `0700`), `(Paths).GlobalDB() string`, `(Paths).ProjectDB(slug string) string`, `(Paths).RunDir() string`.

- [ ] **Step 1: Write the failing test**

`internal/platform/config/paths_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewPathsPrecedence(t *testing.T) {
	envDir := filepath.Join(t.TempDir(), "env")
	flagDir := filepath.Join(t.TempDir(), "flag")

	t.Setenv("MNEMONIC_DATA_DIR", envDir)
	p, err := NewPaths("")
	if err != nil || p.Dir != envDir {
		t.Fatalf("env: got %q err %v", p.Dir, err)
	}
	p, err = NewPaths(flagDir)
	if err != nil || p.Dir != flagDir {
		t.Fatalf("flag: got %q err %v", p.Dir, err)
	}
	info, err := os.Stat(flagDir)
	if err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("dir not created 0700: %v %v", info, err)
	}
}

func TestPathHelpers(t *testing.T) {
	p := Paths{Dir: "/d"}
	if p.GlobalDB() != "/d/global.db" || p.ProjectDB("acme-app") != "/d/acme-app.db" || p.RunDir() != "/d/run" {
		t.Fatalf("unexpected: %s %s %s", p.GlobalDB(), p.ProjectDB("acme-app"), p.RunDir())
	}
}

func TestDefaultsToHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MNEMONIC_DATA_DIR", "")
	p, err := NewPaths("")
	if err != nil || p.Dir != filepath.Join(home, ".mnemonic") {
		t.Fatalf("got %q err %v", p.Dir, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/config/ -v`
Expected: FAIL (`undefined: NewPaths`).

- [ ] **Step 3: Write minimal implementation**

`internal/platform/config/paths.go`:

```go
// Package config resolves the Mnemonic data directory layout.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type Paths struct{ Dir string }

// NewPaths resolves the data dir (flag > MNEMONIC_DATA_DIR > ~/.mnemonic)
// and creates it with 0700 permissions.
func NewPaths(flagDir string) (Paths, error) {
	dir := flagDir
	if dir == "" {
		dir = os.Getenv("MNEMONIC_DATA_DIR")
	}
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return Paths{}, fmt.Errorf("resolve home dir: %w", err)
		}
		dir = filepath.Join(home, ".mnemonic")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return Paths{}, err
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return Paths{}, fmt.Errorf("create data dir: %w", err)
	}
	return Paths{Dir: abs}, nil
}

func (p Paths) GlobalDB() string            { return filepath.Join(p.Dir, "global.db") }
func (p Paths) ProjectDB(slug string) string { return filepath.Join(p.Dir, slug+".db") }
func (p Paths) RunDir() string               { return filepath.Join(p.Dir, "run") }
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/platform/config/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/platform/config
git commit -m "feat(platform): data dir path resolution"
```

---

### Task 3: `project` slug resolver

**Files:**
- Create: `internal/platform/project/project.go`
- Test: `internal/platform/project/project_test.go`

**Interfaces:**
- Consumes: `apperr.New`, `apperr.Validation`.
- Produces: `project.Normalize(raw string) (string, error)`, `project.RemoteSlug(remote string) string`, `project.Resolve(explicit, cwd string) (string, error)` (order: explicit → `MNEMONIC_PROJECT` → `git remote get-url origin` → basename of `cwd`).

- [ ] **Step 1: Add dependency and write the failing test**

Run: `go get golang.org/x/text/unicode/norm`

`internal/platform/project/project_test.go`:

```go
package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"My Repo":          "my-repo",
		"  Acme/Widgets  ": "acme-widgets",
		"café":             "caf", // non [a-z0-9._-] runes become "-", then trimmed
		"global":           "repo-global",
		"Run":              "repo-run",
		".hidden":          "hidden",
		"a__b":             "a__b",
	}
	for in, want := range cases {
		got, err := Normalize(in)
		if err != nil || got != want {
			t.Errorf("Normalize(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := Normalize("///"); err == nil {
		t.Error("expected validation error for empty slug")
	}
}

func TestRemoteSlug(t *testing.T) {
	cases := map[string]string{
		"git@github.com:acme/widgets.git":     "acme-widgets",
		"https://github.com/acme/widgets.git": "acme-widgets",
		"https://github.com/acme/widgets/":    "acme-widgets",
		"widgets":                             "",
	}
	for in, want := range cases {
		if got := RemoteSlug(in); got != want {
			t.Errorf("RemoteSlug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolve(t *testing.T) {
	t.Setenv("MNEMONIC_PROJECT", "")
	if got, _ := Resolve("Explicit One", t.TempDir()); got != "explicit-one" {
		t.Errorf("explicit: %q", got)
	}
	t.Setenv("MNEMONIC_PROJECT", "From Env")
	if got, _ := Resolve("", t.TempDir()); got != "from-env" {
		t.Errorf("env: %q", got)
	}
	t.Setenv("MNEMONIC_PROJECT", "")
	dir := filepath.Join(t.TempDir(), "My Repo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if got, _ := Resolve("", dir); got != "my-repo" {
		t.Errorf("basename: %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/project/ -v`
Expected: FAIL (`undefined: Normalize`).

- [ ] **Step 3: Write minimal implementation**

`internal/platform/project/project.go`:

```go
// Package project resolves canonical repository slugs used as DB basenames.
package project

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/text/unicode/norm"

	"mnemonic/internal/platform/apperr"
)

var reserved = map[string]bool{"global": true, "config": true, "models": true, "run": true, "tmp": true, "wal": true}

var invalidChars = regexp.MustCompile(`[^a-z0-9._-]+`)

// Normalize turns a raw name into a safe, lowercase DB basename slug.
// Reserved names are prefixed with "repo-".
func Normalize(raw string) (string, error) {
	s := strings.ToLower(norm.NFC.String(strings.TrimSpace(raw)))
	s = invalidChars.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-.")
	if s == "" {
		return "", apperr.New(apperr.Validation, "project name is empty after normalization")
	}
	if reserved[s] {
		s = "repo-" + s
	}
	return s, nil
}

// RemoteSlug converts a git remote URL into "owner-repo" ("" if not parseable).
func RemoteSlug(remote string) string {
	r := strings.TrimSuffix(strings.TrimSpace(remote), "/")
	r = strings.TrimSuffix(r, ".git")
	parts := strings.FieldsFunc(r, func(c rune) bool { return c == '/' || c == ':' })
	if len(parts) < 2 {
		return ""
	}
	return parts[len(parts)-2] + "-" + parts[len(parts)-1]
}

// Resolve picks the project slug: explicit > MNEMONIC_PROJECT > git origin > cwd basename.
func Resolve(explicit, cwd string) (string, error) {
	if explicit != "" {
		return Normalize(explicit)
	}
	if env := os.Getenv("MNEMONIC_PROJECT"); env != "" {
		return Normalize(env)
	}
	if out, err := exec.Command("git", "-C", cwd, "remote", "get-url", "origin").Output(); err == nil {
		if slug := RemoteSlug(string(out)); slug != "" {
			return Normalize(slug)
		}
	}
	return Normalize(filepath.Base(cwd))
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/platform/project/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum internal/platform/project
git commit -m "feat(platform): project slug resolver with reserved-name guard"
```

---

### Task 4: `sqlitepool` (writer/reader split, fence, FS guard)

**Files:**
- Create: `internal/platform/sqlitepool/pool.go`, `fs.go`, `fs_darwin.go`, `fs_linux.go`, `fs_other.go`
- Test: `internal/platform/sqlitepool/pool_test.go`

**Interfaces:**
- Consumes: `apperr`.
- Produces:
  - `sqlitepool.DB{Path string; W, R *sql.DB}`; `(*DB).Write(ctx context.Context, fn func(tx *sql.Tx) error) error` (serialized, `BEGIN IMMEDIATE`, 10 s default timeout, 64-slot admission queue); `(*DB).Close() error`.
  - `sqlitepool.Pool`; `NewPool() *Pool`; `(*Pool).Get(path string, init func(*DB) error) (*DB, error)` (re-opens + re-runs `init` when the file inode changed or vanished, removing stale `-wal`/`-shm`); `(*Pool).Close() error`.
  - `sqlitepool.CheckLocalFS(dir string) error` (returns `apperr.UnsupportedFS` on NFS/SMB/CIFS/AFS).

- [ ] **Step 1: Add dependency and write the failing test**

Run: `go get modernc.org/sqlite`

`internal/platform/sqlitepool/pool_test.go`:

```go
package sqlitepool

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"mnemonic/internal/platform/apperr"
)

func createT(d *DB) error {
	_, err := d.W.Exec(`CREATE TABLE IF NOT EXISTS t(id INTEGER PRIMARY KEY AUTOINCREMENT, v TEXT)`)
	return err
}

func TestConcurrentWritersAndReaders(t *testing.T) {
	p := NewPool()
	defer p.Close()
	d, err := p.Get(filepath.Join(t.TempDir(), "a.db"), createT)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	var wg sync.WaitGroup
	errs := make(chan error, 40)
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			errs <- d.Write(ctx, func(tx *sql.Tx) error {
				_, err := tx.Exec(`INSERT INTO t(v) VALUES('x')`)
				return err
			})
		}()
		go func() {
			defer wg.Done()
			var n int
			errs <- d.R.QueryRow(`SELECT COUNT(*) FROM t`).Scan(&n)
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatalf("concurrent op failed: %v", e)
		}
	}
	var n int
	if err := d.R.QueryRow(`SELECT COUNT(*) FROM t`).Scan(&n); err != nil || n != 20 {
		t.Fatalf("rows = %d err %v, want 20", n, err)
	}
}

func TestWriteRollsBackOnError(t *testing.T) {
	p := NewPool()
	defer p.Close()
	d, _ := p.Get(filepath.Join(t.TempDir(), "a.db"), createT)
	err := d.Write(context.Background(), func(tx *sql.Tx) error {
		if _, err := tx.Exec(`INSERT INTO t(v) VALUES('x')`); err != nil {
			return err
		}
		return apperr.New(apperr.Validation, "nope")
	})
	if apperr.CodeOf(err) != apperr.Validation {
		t.Fatalf("err = %v", err)
	}
	var n int
	_ = d.R.QueryRow(`SELECT COUNT(*) FROM t`).Scan(&n)
	if n != 0 {
		t.Fatalf("rollback failed, rows = %d", n)
	}
}

func TestInodeFenceReopensReplacedDB(t *testing.T) {
	p := NewPool()
	defer p.Close()
	path := filepath.Join(t.TempDir(), "a.db")
	d1, err := p.Get(path, createT)
	if err != nil {
		t.Fatal(err)
	}
	if err := d1.Write(context.Background(), func(tx *sql.Tx) error {
		_, err := tx.Exec(`INSERT INTO t(v) VALUES('old')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	d2, err := p.Get(path, createT)
	if err != nil {
		t.Fatal(err)
	}
	if d1 == d2 {
		t.Fatal("expected a fresh handle after the DB file was replaced")
	}
	var n int
	if err := d2.R.QueryRow(`SELECT COUNT(*) FROM t`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("new generation should be empty: n=%d err=%v", n, err)
	}
	// same file again -> same handle
	d3, _ := p.Get(path, createT)
	if d3 != d2 {
		t.Fatal("expected cached handle when inode unchanged")
	}
}

func TestCheckLocalFSAcceptsTempDir(t *testing.T) {
	if err := CheckLocalFS(t.TempDir()); err != nil {
		t.Fatalf("temp dir rejected: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/sqlitepool/ -v`
Expected: FAIL (`undefined: NewPool`).

- [ ] **Step 3: Write the filesystem guard files**

`internal/platform/sqlitepool/fs.go`:

```go
package sqlitepool

import (
	"strings"

	"mnemonic/internal/platform/apperr"
)

var networkFS = map[string]bool{
	"nfs": true, "smb": true, "smb2": true, "smbfs": true,
	"cifs": true, "afs": true, "afpfs": true, "webdav": true,
}

// CheckLocalFS refuses network filesystems where SQLite WAL is unsafe.
func CheckLocalFS(dir string) error {
	name, err := fsTypeName(dir)
	if err != nil {
		return nil // best effort: unknown means allowed
	}
	if networkFS[strings.ToLower(name)] {
		return apperr.Newf(apperr.UnsupportedFS,
			"data directory %q is on network filesystem %q; set MNEMONIC_DATA_DIR to a local disk", dir, name)
	}
	return nil
}
```

`internal/platform/sqlitepool/fs_darwin.go`:

```go
//go:build darwin

package sqlitepool

import "syscall"

func fsTypeName(dir string) (string, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return "", err
	}
	b := make([]byte, 0, 16)
	for _, c := range st.Fstypename {
		if c == 0 {
			break
		}
		b = append(b, byte(c))
	}
	return string(b), nil
}
```

`internal/platform/sqlitepool/fs_linux.go`:

```go
//go:build linux

package sqlitepool

import "syscall"

func fsTypeName(dir string) (string, error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return "", err
	}
	switch uint32(st.Type) {
	case 0x6969:
		return "nfs", nil
	case 0x517B:
		return "smb", nil
	case 0xFF534D42:
		return "cifs", nil
	case 0xFE534D42:
		return "smb2", nil
	case 0x5346414F:
		return "afs", nil
	}
	return "local", nil
}
```

`internal/platform/sqlitepool/fs_other.go`:

```go
//go:build !darwin && !linux

package sqlitepool

func fsTypeName(dir string) (string, error) { return "unknown", nil }
```

- [ ] **Step 4: Write the pool**

`internal/platform/sqlitepool/pool.go`:

```go
// Package sqlitepool manages per-file SQLite handles: one serialized writer
// (BEGIN IMMEDIATE) plus a pool of read-only WAL readers, with an inode fence
// that reopens a database whose file was replaced or deleted.
package sqlitepool

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const (
	writeQueueSize  = 64
	defaultWriteTTL = 10 * time.Second
)

// DB is one SQLite file with a writer handle (W) and a reader pool (R).
type DB struct {
	Path string
	W    *sql.DB
	R    *sql.DB
	info os.FileInfo
	sem  chan struct{}
}

func dsn(path string, writer bool) string {
	q := url.Values{}
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "temp_store(2)")
	if writer {
		q.Add("_txlock", "immediate")
		q.Add("_pragma", "journal_mode(WAL)")
		q.Add("_pragma", "synchronous(NORMAL)")
		q.Add("_pragma", "journal_size_limit(67108864)")
	} else {
		q.Add("_pragma", "query_only(1)")
	}
	return "file:" + filepath.ToSlash(path) + "?" + q.Encode()
}

func open(path string) (*DB, error) {
	if err := CheckLocalFS(filepath.Dir(path)); err != nil {
		return nil, err
	}
	w, err := sql.Open("sqlite", dsn(path, true))
	if err != nil {
		return nil, err
	}
	w.SetMaxOpenConns(1)
	if err := w.Ping(); err != nil { // creates the file + applies pragmas
		_ = w.Close()
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	r, err := sql.Open("sqlite", dsn(path, false))
	if err != nil {
		_ = w.Close()
		return nil, err
	}
	r.SetMaxOpenConns(min(8, runtime.NumCPU()))
	info, err := os.Stat(path)
	if err != nil {
		_ = w.Close()
		_ = r.Close()
		return nil, err
	}
	return &DB{Path: path, W: w, R: r, info: info, sem: make(chan struct{}, writeQueueSize)}, nil
}

// Write runs fn in a serialized BEGIN IMMEDIATE transaction. It commits on
// nil and rolls back on error. A 10s deadline applies when ctx has none.
func (d *DB) Write(ctx context.Context, fn func(tx *sql.Tx) error) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultWriteTTL)
		defer cancel()
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

// Close closes both handles.
func (d *DB) Close() error {
	e1 := d.R.Close()
	e2 := d.W.Close()
	if e1 != nil {
		return e1
	}
	return e2
}

// Pool caches one *DB per path.
type Pool struct {
	mu  sync.Mutex
	dbs map[string]*DB
}

func NewPool() *Pool { return &Pool{dbs: map[string]*DB{}} }

// Get returns the cached DB for path, or opens it and runs init exactly once
// per file generation. If the file was replaced or deleted since it was
// opened, the old handles are closed, stale -wal/-shm files are removed, and
// a fresh generation is opened (init runs again).
func (p *Pool) Get(path string, init func(*DB) error) (*DB, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if d, ok := p.dbs[path]; ok {
		if cur, err := os.Stat(path); err == nil && os.SameFile(cur, d.info) {
			return d, nil
		}
		_ = d.Close()
		delete(p.dbs, path)
		_ = os.Remove(path + "-wal")
		_ = os.Remove(path + "-shm")
	}
	d, err := open(path)
	if err != nil {
		return nil, err
	}
	if init != nil {
		if err := init(d); err != nil {
			_ = d.Close()
			return nil, err
		}
	}
	p.dbs[path] = d
	return d, nil
}

// Close closes every cached DB.
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	var first error
	for k, d := range p.dbs {
		if err := d.Close(); err != nil && first == nil {
			first = err
		}
		delete(p.dbs, k)
	}
	return first
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/platform/sqlitepool/ -race -v`
Expected: PASS (all 4 tests, no data races).

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/platform/sqlitepool
git commit -m "feat(platform): sqlite pool with serialized writer, readers, inode fence, network-FS guard"
```

---

### Task 5: `migrate` runner + platform schema

**Files:**
- Create: `internal/platform/migrate/migrate.go`, `internal/platform/migrate/sql/001_platform.sql`
- Test: `internal/platform/migrate/migrate_test.go`

**Interfaces:**
- Consumes: `sqlitepool.DB` (`W`, `Write`).
- Produces: `migrate.Apply(ctx context.Context, db *sqlitepool.DB, subsystem string, fsys fs.FS) error` (files `NNN_name.sql` at the FS root, applied in numeric order, each in one transaction, recorded in `schema_migrations(subsystem, version, applied_at)`); `migrate.PlatformFS() fs.FS`.

- [ ] **Step 1: Write the failing test**

`internal/platform/migrate/migrate_test.go`:

```go
package migrate

import (
	"context"
	"path/filepath"
	"testing"
	"testing/fstest"

	"mnemonic/internal/platform/sqlitepool"
)

func newDB(t *testing.T) *sqlitepool.DB {
	t.Helper()
	p := sqlitepool.NewPool()
	t.Cleanup(func() { _ = p.Close() })
	d, err := p.Get(filepath.Join(t.TempDir(), "m.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestApplyIsOrderedAndIdempotent(t *testing.T) {
	d := newDB(t)
	fsys := fstest.MapFS{
		"002_more.sql":  {Data: []byte(`ALTER TABLE a ADD COLUMN y TEXT;`)},
		"001_first.sql": {Data: []byte(`CREATE TABLE a(x INTEGER); CREATE TABLE b(z INTEGER);`)},
	}
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if err := Apply(ctx, d, "demo", fsys); err != nil {
			t.Fatalf("apply #%d: %v", i, err)
		}
	}
	var n int
	if err := d.R.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE subsystem='demo'`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("recorded = %d err %v", n, err)
	}
	if _, err := d.R.Exec(`SELECT y FROM a`); err != nil {
		t.Fatalf("column y missing: %v", err)
	}
}

func TestFailedMigrationRollsBack(t *testing.T) {
	d := newDB(t)
	fsys := fstest.MapFS{"001_bad.sql": {Data: []byte(`CREATE TABLE ok(x INTEGER); THIS IS NOT SQL;`)}}
	if err := Apply(context.Background(), d, "demo", fsys); err == nil {
		t.Fatal("expected error")
	}
	var n int
	_ = d.R.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name='ok'`).Scan(&n)
	if n != 0 {
		t.Fatal("partial migration must roll back")
	}
}

func TestPlatformSchema(t *testing.T) {
	d := newDB(t)
	if err := Apply(context.Background(), d, "platform", PlatformFS()); err != nil {
		t.Fatal(err)
	}
	for _, tbl := range []string{"projects", "project_aliases"} {
		var n int
		_ = d.R.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, tbl).Scan(&n)
		if n != 1 {
			t.Errorf("table %s missing", tbl)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/platform/migrate/ -v`
Expected: FAIL (`undefined: Apply`).

- [ ] **Step 3: Write the platform SQL**

`internal/platform/migrate/sql/001_platform.sql`:

```sql
CREATE TABLE projects (
    slug           TEXT PRIMARY KEY,
    db_file        TEXT NOT NULL UNIQUE,
    root_path      TEXT NOT NULL DEFAULT '',
    remote_url     TEXT NOT NULL DEFAULT '',
    created_at     TEXT NOT NULL,
    last_active_at TEXT NOT NULL
);

CREATE TABLE project_aliases (
    kind       TEXT NOT NULL CHECK (kind IN ('cwd', 'remote', 'key')),
    alias_key  TEXT NOT NULL,
    slug       TEXT NOT NULL REFERENCES projects(slug) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    PRIMARY KEY (kind, alias_key)
);
CREATE INDEX idx_project_aliases_slug ON project_aliases(slug);
```

- [ ] **Step 4: Write the runner**

`internal/platform/migrate/migrate.go`:

```go
// Package migrate applies per-subsystem SQL migrations tracked in
// schema_migrations(subsystem, version).
package migrate

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"mnemonic/internal/platform/sqlitepool"
)

//go:embed sql/*.sql
var platformEmbed embed.FS

// PlatformFS returns the platform migrations (applied to global.db only).
func PlatformFS() fs.FS {
	sub, err := fs.Sub(platformEmbed, "sql")
	if err != nil {
		panic(err)
	}
	return sub
}

// Apply runs every unapplied NNN_name.sql in fsys for the given subsystem.
func Apply(ctx context.Context, db *sqlitepool.DB, subsystem string, fsys fs.FS) error {
	if _, err := db.W.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		subsystem  TEXT NOT NULL,
		version    INTEGER NOT NULL,
		applied_at TEXT NOT NULL,
		PRIMARY KEY (subsystem, version)
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	files, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, f := range files {
		base := path.Base(f)
		ver, err := strconv.Atoi(strings.SplitN(base, "_", 2)[0])
		if err != nil {
			return fmt.Errorf("migration %q: filename must start with NNN_: %w", base, err)
		}
		var applied int
		if err := db.W.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM schema_migrations WHERE subsystem=? AND version=?`, subsystem, ver,
		).Scan(&applied); err != nil {
			return err
		}
		if applied > 0 {
			continue
		}
		body, err := fs.ReadFile(fsys, f)
		if err != nil {
			return err
		}
		err = db.Write(ctx, func(tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, string(body)); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx,
				`INSERT INTO schema_migrations(subsystem, version, applied_at) VALUES (?,?,?)`,
				subsystem, ver, time.Now().UTC().Format(time.RFC3339Nano))
			return err
		})
		if err != nil {
			return fmt.Errorf("apply %s/%s: %w", subsystem, base, err)
		}
	}
	return nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/platform/migrate/ -race -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/platform/migrate
git commit -m "feat(platform): per-subsystem migration runner and platform schema"
```

---

### Task 6: Memory schema migration (tables + standalone FTS5)

**Files:**
- Create: `internal/memory/store/migrations/001_init.sql`, `internal/memory/store/store.go`
- Test: `internal/memory/store/store_test.go`

**Interfaces:**
- Consumes: `migrate.Apply`, `migrate.PlatformFS`, `sqlitepool.DB`.
- Produces: `store.Init(db *sqlitepool.DB, isGlobal bool) error` (applies `memory` migrations; also `platform` migrations when `isGlobal`); helper `store.TS(t time.Time) string`, `store.ParseTS(s string) time.Time`; `store.Querier` interface (`ExecContext`, `QueryContext`, `QueryRowContext` — satisfied by `*sql.DB` and `*sql.Tx`); test helper `newTestDB(t)` in `store_test.go`.

- [ ] **Step 1: Write the failing test**

`internal/memory/store/store_test.go`:

```go
package store

import (
	"context"
	"path/filepath"
	"testing"

	"mnemonic/internal/platform/sqlitepool"
)

func newTestDB(t *testing.T, isGlobal bool) *sqlitepool.DB {
	t.Helper()
	p := sqlitepool.NewPool()
	t.Cleanup(func() { _ = p.Close() })
	d, err := p.Get(filepath.Join(t.TempDir(), "s.db"), func(d *sqlitepool.DB) error { return Init(d, isGlobal) })
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestInitCreatesSchemaAndIsIdempotent(t *testing.T) {
	d := newTestDB(t, true)
	if err := Init(d, true); err != nil {
		t.Fatalf("second init: %v", err)
	}
	for _, tbl := range []string{
		"sessions", "observations", "observation_versions", "save_operations",
		"user_prompts", "memory_relations", "session_trails", "retrieval_trails",
		"observations_fts", "prompts_fts", "projects", "project_aliases",
	} {
		var n int
		_ = d.R.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name=?`, tbl).Scan(&n)
		if n != 1 {
			t.Errorf("table %s missing", tbl)
		}
	}
	proj := newTestDB(t, false)
	var n int
	_ = proj.R.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name='projects'`).Scan(&n)
	if n != 0 {
		t.Error("project DB must not contain the platform registry")
	}
}

func ftsCount(t *testing.T, d *sqlitepool.DB, term string) int {
	t.Helper()
	var n int
	if err := d.R.QueryRow(`SELECT COUNT(*) FROM observations_fts WHERE observations_fts MATCH ?`, `"`+term+`"`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestFTSTriggersTrackInsertUpdateSoftAndHardDelete(t *testing.T) {
	d := newTestDB(t, false)
	ctx := context.Background()
	now := TS(timeNow())
	_, err := d.W.ExecContext(ctx, `INSERT INTO observations
		(uid,project,scope,type,title,content,normalized_hash,created_at,updated_at,last_seen_at)
		VALUES ('u1','p','project','decision','Alpha title','bravo charlie','h',?,?,?)`, now, now, now)
	if err != nil {
		t.Fatal(err)
	}
	if ftsCount(t, d, "bravo") != 1 {
		t.Fatal("insert not indexed")
	}
	if _, err := d.W.Exec(`UPDATE observations SET content='delta echo' WHERE uid='u1'`); err != nil {
		t.Fatal(err)
	}
	if ftsCount(t, d, "bravo") != 0 || ftsCount(t, d, "delta") != 1 {
		t.Fatal("update not reindexed")
	}
	if _, err := d.W.Exec(`UPDATE observations SET pinned=1 WHERE uid='u1'`); err != nil {
		t.Fatal(err)
	}
	if ftsCount(t, d, "delta") != 1 {
		t.Fatal("pin must not drop the row from the index")
	}
	if _, err := d.W.Exec(`UPDATE observations SET deleted_at=? WHERE uid='u1'`, now); err != nil {
		t.Fatal(err)
	}
	if ftsCount(t, d, "delta") != 0 {
		t.Fatal("soft delete must leave the index")
	}
	if _, err := d.W.Exec(`UPDATE observations SET deleted_at=NULL WHERE uid='u1'`); err != nil {
		t.Fatal(err)
	}
	if ftsCount(t, d, "delta") != 1 {
		t.Fatal("restore must reindex")
	}
	if _, err := d.W.Exec(`DELETE FROM observations WHERE uid='u1'`); err != nil {
		t.Fatal(err)
	}
	if ftsCount(t, d, "delta") != 0 {
		t.Fatal("hard delete must leave the index")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/memory/store/ -v`
Expected: FAIL (`undefined: Init`).

- [ ] **Step 3: Write the migration SQL**

`internal/memory/store/migrations/001_init.sql`:

```sql
CREATE TABLE sessions (
    id           TEXT PRIMARY KEY,
    project      TEXT NOT NULL,
    title        TEXT NOT NULL DEFAULT '',
    directory    TEXT NOT NULL DEFAULT '',
    started_at   TEXT NOT NULL,
    ended_at     TEXT,
    summary_json TEXT,
    finalized_at TEXT,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);
CREATE INDEX idx_sessions_project_started ON sessions(project, started_at DESC);

CREATE TABLE observations (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    uid               TEXT NOT NULL UNIQUE,
    session_id        TEXT,
    project           TEXT NOT NULL,
    scope             TEXT NOT NULL DEFAULT 'project' CHECK (scope IN ('project', 'user', 'global')),
    type              TEXT NOT NULL CHECK (type IN (
                          'decision', 'architecture', 'bugfix', 'pattern',
                          'config', 'discovery', 'learning', 'preference', 'convention')),
    title             TEXT NOT NULL,
    content           TEXT NOT NULL,
    topic_key         TEXT,
    tool_name         TEXT NOT NULL DEFAULT '',
    normalized_hash   TEXT NOT NULL,
    revision_count    INTEGER NOT NULL DEFAULT 1,
    duplicate_count   INTEGER NOT NULL DEFAULT 1,
    pinned            INTEGER NOT NULL DEFAULT 0 CHECK (pinned IN (0, 1)),
    importance        REAL NOT NULL DEFAULT 0.5,
    retrieval_usage   INTEGER NOT NULL DEFAULT 0,
    superseded_by_uid TEXT,
    review_after      TEXT,
    expires_at        TEXT,
    provenance_json   TEXT NOT NULL DEFAULT '{}',
    created_at        TEXT NOT NULL,
    updated_at        TEXT NOT NULL,
    last_seen_at      TEXT NOT NULL,
    deleted_at        TEXT
);
CREATE INDEX idx_obs_topic       ON observations(topic_key, scope, updated_at DESC) WHERE deleted_at IS NULL AND topic_key IS NOT NULL;
CREATE INDEX idx_obs_dedupe      ON observations(normalized_hash, scope, type, title, last_seen_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_obs_active_time ON observations(pinned DESC, updated_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_obs_session     ON observations(session_id) WHERE session_id IS NOT NULL;
CREATE INDEX idx_obs_review      ON observations(review_after) WHERE deleted_at IS NULL AND review_after IS NOT NULL;
CREATE INDEX idx_obs_expires     ON observations(expires_at) WHERE deleted_at IS NULL AND expires_at IS NOT NULL;

CREATE TABLE observation_versions (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    observation_id INTEGER NOT NULL REFERENCES observations(id) ON DELETE CASCADE,
    version        INTEGER NOT NULL,
    title          TEXT NOT NULL,
    content        TEXT NOT NULL,
    session_id     TEXT,
    created_at     TEXT NOT NULL,
    UNIQUE (observation_id, version)
);

CREATE TABLE save_operations (
    operation_id    TEXT PRIMARY KEY,
    fingerprint     TEXT NOT NULL,
    observation_id  INTEGER REFERENCES observations(id) ON DELETE SET NULL,
    observation_uid TEXT NOT NULL,
    created_at      TEXT NOT NULL
);
CREATE INDEX idx_save_ops_created ON save_operations(created_at);

CREATE TABLE user_prompts (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    uid             TEXT NOT NULL UNIQUE,
    session_id      TEXT,
    project         TEXT NOT NULL,
    content         TEXT NOT NULL,
    normalized_hash TEXT NOT NULL,
    created_at      TEXT NOT NULL
);
CREATE INDEX idx_prompts_session_created ON user_prompts(session_id, created_at DESC);
CREATE INDEX idx_prompts_dedupe          ON user_prompts(session_id, normalized_hash, created_at DESC);

CREATE TABLE memory_relations (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    source_uid      TEXT NOT NULL REFERENCES observations(uid) ON DELETE CASCADE,
    target_uid      TEXT NOT NULL REFERENCES observations(uid) ON DELETE CASCADE,
    relation        TEXT NOT NULL DEFAULT 'pending' CHECK (relation IN (
                        'pending', 'related', 'compatible', 'scoped',
                        'supersedes', 'conflicts_with', 'not_conflict')),
    judgment_status TEXT NOT NULL DEFAULT 'pending' CHECK (judgment_status IN ('pending', 'judged', 'superseded')),
    confidence      REAL,
    reason          TEXT NOT NULL DEFAULT '',
    evidence        TEXT NOT NULL DEFAULT '',
    marked_by_kind  TEXT NOT NULL DEFAULT 'system',
    marked_by_model TEXT NOT NULL DEFAULT '',
    session_id      TEXT,
    superseded_at   TEXT,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    UNIQUE (source_uid, target_uid)
);
CREATE INDEX idx_memrel_source  ON memory_relations(source_uid) WHERE judgment_status != 'superseded';
CREATE INDEX idx_memrel_target  ON memory_relations(target_uid) WHERE judgment_status != 'superseded';
CREATE INDEX idx_memrel_pending ON memory_relations(judgment_status, created_at DESC) WHERE judgment_status = 'pending';

CREATE TABLE session_trails (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id  TEXT NOT NULL,
    action_type TEXT NOT NULL,
    target_uid  TEXT NOT NULL,
    created_at  TEXT NOT NULL
);
CREATE INDEX idx_session_trails_sid ON session_trails(session_id, id ASC);

CREATE TABLE retrieval_trails (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    route      TEXT NOT NULL,
    query      TEXT NOT NULL,
    filters    TEXT NOT NULL DEFAULT '{}',
    hit_uids   TEXT NOT NULL DEFAULT '[]',
    degraded   INTEGER NOT NULL DEFAULT 0,
    latency_ms INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL
);
CREATE INDEX idx_retrieval_trails_created ON retrieval_trails(created_at DESC);

CREATE VIRTUAL TABLE observations_fts USING fts5(
    title, content, topic_key, type, tool_name,
    tokenize = 'trigram remove_diacritics 1'
);

CREATE TRIGGER obs_fts_ai AFTER INSERT ON observations WHEN new.deleted_at IS NULL BEGIN
    INSERT INTO observations_fts(rowid, title, content, topic_key, type, tool_name)
    VALUES (new.id, new.title, new.content, COALESCE(new.topic_key, ''), new.type, new.tool_name);
END;

CREATE TRIGGER obs_fts_ad AFTER DELETE ON observations BEGIN
    DELETE FROM observations_fts WHERE rowid = old.id;
END;

CREATE TRIGGER obs_fts_au AFTER UPDATE OF title, content, topic_key, type, tool_name, deleted_at ON observations BEGIN
    DELETE FROM observations_fts WHERE rowid = old.id;
    INSERT INTO observations_fts(rowid, title, content, topic_key, type, tool_name)
    SELECT new.id, new.title, new.content, COALESCE(new.topic_key, ''), new.type, new.tool_name
    WHERE new.deleted_at IS NULL;
END;

CREATE VIRTUAL TABLE prompts_fts USING fts5(
    content,
    tokenize = 'trigram remove_diacritics 1'
);

CREATE TRIGGER prompts_fts_ai AFTER INSERT ON user_prompts BEGIN
    INSERT INTO prompts_fts(rowid, content) VALUES (new.id, new.content);
END;

CREATE TRIGGER prompts_fts_ad AFTER DELETE ON user_prompts BEGIN
    DELETE FROM prompts_fts WHERE rowid = old.id;
END;

CREATE TRIGGER prompts_fts_au AFTER UPDATE OF content ON user_prompts BEGIN
    DELETE FROM prompts_fts WHERE rowid = old.id;
    INSERT INTO prompts_fts(rowid, content) VALUES (new.id, new.content);
END;
```

> Note: the test above expects `projects`/`project_aliases` to exist in the *global* DB (platform migration) and `prompts_fts` in both. The `memory` migration is applied to both DB kinds.

- [ ] **Step 4: Write `store.go`**

`internal/memory/store/store.go`:

```go
// Package store owns every SQL statement of the memory subsystem.
package store

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"time"

	"mnemonic/internal/platform/migrate"
	"mnemonic/internal/platform/sqlitepool"
)

//go:embed migrations/*.sql
var migrationsEmbed embed.FS

func memoryFS() fs.FS {
	sub, err := fs.Sub(migrationsEmbed, "migrations")
	if err != nil {
		panic(err)
	}
	return sub
}

// Querier is satisfied by *sql.DB and *sql.Tx.
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Init applies the memory migrations (plus platform migrations on global.db).
func Init(db *sqlitepool.DB, isGlobal bool) error {
	ctx := context.Background()
	if isGlobal {
		if err := migrate.Apply(ctx, db, "platform", migrate.PlatformFS()); err != nil {
			return err
		}
	}
	return migrate.Apply(ctx, db, "memory", memoryFS())
}

// TS formats a time as UTC RFC3339Nano.
func TS(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

// ParseTS parses TS output; zero time on empty/invalid input.
func ParseTS(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
```

Also add to `store_test.go` (and import `"time"`): `func timeNow() time.Time { return time.Now() }`.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/memory/store/ -race -v`
Expected: PASS (`TestInitCreatesSchemaAndIsIdempotent`, `TestFTSTriggersTrack…`).

- [ ] **Step 6: Commit**

```bash
git add internal/memory/store
git commit -m "feat(memory): schema migration with standalone trigram FTS5 and sync triggers"
```

---

### Task 7: `api` types + store CRUD (sessions, observations, operations, projects)

**Files:**
- Create: `internal/memory/api/types.go`, `internal/memory/store/sessions.go`, `observations.go`, `operations.go`, `projects.go`
- Modify: `internal/memory/store/store_test.go` (append tests)

**Interfaces:**
- Produces (`api`):

```go
type Observation struct {
	ID             int64     `json:"-"`
	UID            string    `json:"uid"`
	SessionID      string    `json:"session_id,omitempty"`
	Project        string    `json:"project"`
	Scope          string    `json:"scope"`
	Type           string    `json:"type"`
	Title          string    `json:"title"`
	Content        string    `json:"content"`
	TopicKey       string    `json:"topic_key,omitempty"`
	ToolName       string    `json:"tool_name,omitempty"`
	NormalizedHash string    `json:"-"`
	RevisionCount  int       `json:"revision_count"`
	DuplicateCount int       `json:"duplicate_count"`
	Pinned         bool      `json:"pinned"`
	Importance     float64   `json:"importance"`
	RetrievalUsage int       `json:"retrieval_usage"`
	ReviewAfter    time.Time `json:"review_after,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	LastSeenAt     time.Time `json:"last_seen_at"`
}
```

  plus `SaveRequest`, `SaveResult`, `SearchRequest`, `SearchHit`, `SearchResult`, `GetRequest`, `Session`, `SessionStartRequest`, `SessionEndRequest`, `Service` (full text below).
- Produces (`store`): `InsertObservation(ctx, q Querier, o *api.Observation) error`; `GetObservationByUID(ctx, q, uid string) (api.Observation, error)`; `GetObservationByID(ctx, q, id int64) (api.Observation, error)`; `FindActiveByTopic(ctx, q, topicKey, scope string) (*api.Observation, error)`; `FindRecentDuplicate(ctx, q, hash, scope, typ, title string, since time.Time) (*api.Observation, error)`; `SnapshotVersion(ctx, q, obsID int64, version int, title, content, sessionID string, now time.Time) error`; `ReviseObservation(ctx, q, o *api.Observation) error`; `BumpDuplicate(ctx, q, id int64, now time.Time) error`; `AddTrail(ctx, q, sessionID, action, uid string, now time.Time) error`; `Operation{Fingerprint string; ObservationID sql.NullInt64; ObservationUID string}`; `LookupOperation(ctx, q, id string) (*Operation, error)`; `InsertOperation(ctx, q, id, fingerprint string, obsID int64, uid string, now time.Time) error`; `UpsertSession(ctx, q, s api.Session) error`; `EndSession(ctx, q, id string, summary string, now time.Time) (bool, error)`; `GetSession(ctx, q, id string) (api.Session, error)`; `TouchProject(ctx, q, slug, dbFile string, now time.Time) error`.

- [ ] **Step 1: Write `api/types.go`**

`internal/memory/api/types.go`:

```go
// Package api defines the memory service contract and DTOs shared by every
// transport (CLI, MCP, daemon HTTP) and implementation (engine.Local, client).
package api

import (
	"context"
	"time"
)

type Observation struct {
	ID             int64     `json:"-"`
	UID            string    `json:"uid"`
	SessionID      string    `json:"session_id,omitempty"`
	Project        string    `json:"project"`
	Scope          string    `json:"scope"`
	Type           string    `json:"type"`
	Title          string    `json:"title"`
	Content        string    `json:"content"`
	TopicKey       string    `json:"topic_key,omitempty"`
	ToolName       string    `json:"tool_name,omitempty"`
	NormalizedHash string    `json:"-"`
	RevisionCount  int       `json:"revision_count"`
	DuplicateCount int       `json:"duplicate_count"`
	Pinned         bool      `json:"pinned"`
	Importance     float64   `json:"importance"`
	RetrievalUsage int       `json:"retrieval_usage"`
	ReviewAfter    time.Time `json:"review_after,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	LastSeenAt     time.Time `json:"last_seen_at"`
}

type SaveRequest struct {
	Project     string `json:"project"`
	Scope       string `json:"scope,omitempty"` // project (default) | user | global
	Type        string `json:"type"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	TopicKey    string `json:"topic_key,omitempty"`
	ToolName    string `json:"tool_name,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
	OperationID string `json:"operation_id,omitempty"`
}

type SaveResult struct {
	UID           string `json:"uid"`
	Verdict       string `json:"verdict"` // create | revise | duplicate | replay
	RevisionCount int    `json:"revision_count"`
	Replayed      bool   `json:"replayed"`
	Degraded      bool   `json:"degraded"`
}

type SearchRequest struct {
	Project   string `json:"project"`
	Query     string `json:"query"`
	Scope     string `json:"scope,omitempty"`
	Type      string `json:"type,omitempty"`
	MatchMode string `json:"match_mode,omitempty"` // any (default) | all
	Limit     int    `json:"limit,omitempty"`      // default 10, max 50
}

type SearchHit struct {
	UID      string  `json:"uid"`
	Title    string  `json:"title"`
	Type     string  `json:"type"`
	Scope    string  `json:"scope"`
	Project  string  `json:"project"`
	TopicKey string  `json:"topic_key,omitempty"`
	Snippet  string  `json:"snippet"`
	Score    float64 `json:"score"`
}

type SearchResult struct {
	Hits     []SearchHit `json:"hits"`
	Degraded bool        `json:"degraded"`
}

type GetRequest struct {
	Project string `json:"project"`
	UID     string `json:"uid"`
}

type Session struct {
	ID        string    `json:"id"`
	Project   string    `json:"project"`
	Title     string    `json:"title,omitempty"`
	Directory string    `json:"directory,omitempty"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at,omitempty"`
	Summary   string    `json:"summary,omitempty"`
}

type SessionStartRequest struct {
	ID        string `json:"id"`
	Project   string `json:"project"`
	Title     string `json:"title,omitempty"`
	Directory string `json:"directory,omitempty"`
}

type SessionEndRequest struct {
	ID      string `json:"id"`
	Project string `json:"project"`
	Summary string `json:"summary,omitempty"`
}

// Service is the transport-agnostic memory contract.
type Service interface {
	Save(ctx context.Context, req SaveRequest) (SaveResult, error)
	Search(ctx context.Context, req SearchRequest) (SearchResult, error)
	Get(ctx context.Context, req GetRequest) (Observation, error)
	SessionStart(ctx context.Context, req SessionStartRequest) (Session, error)
	SessionEnd(ctx context.Context, req SessionEndRequest) error
	Close() error
}
```

- [ ] **Step 2: Append failing store tests**

Append to `internal/memory/store/store_test.go` (add imports `"time"`, `"mnemonic/internal/memory/api"`):

```go
func sampleObs(uid string) *api.Observation {
	now := time.Now().UTC()
	return &api.Observation{
		UID: uid, Project: "p", Scope: "project", Type: "decision",
		Title: "Use WAL", Content: "because concurrency", TopicKey: "arch/wal",
		NormalizedHash: "h1", RevisionCount: 1, DuplicateCount: 1, Importance: 0.5,
		CreatedAt: now, UpdatedAt: now, LastSeenAt: now,
	}
}

func TestObservationRoundTripAndFinders(t *testing.T) {
	d := newTestDB(t, false)
	ctx := context.Background()
	o := sampleObs("u1")
	if err := InsertObservation(ctx, d.W, o); err != nil || o.ID == 0 {
		t.Fatalf("insert: id=%d err=%v", o.ID, err)
	}
	got, err := GetObservationByUID(ctx, d.R, "u1")
	if err != nil || got.Title != "Use WAL" || got.TopicKey != "arch/wal" {
		t.Fatalf("get: %+v %v", got, err)
	}
	if _, err := GetObservationByUID(ctx, d.R, "missing"); apperr.CodeOf(err) != apperr.NotFound {
		t.Fatalf("missing uid should be NOT_FOUND, got %v", err)
	}
	if f, err := FindActiveByTopic(ctx, d.R, "arch/wal", "project"); err != nil || f == nil || f.UID != "u1" {
		t.Fatalf("topic finder: %v %v", f, err)
	}
	if f, _ := FindActiveByTopic(ctx, d.R, "arch/none", "project"); f != nil {
		t.Fatal("unexpected topic hit")
	}
	if f, err := FindRecentDuplicate(ctx, d.R, "h1", "project", "decision", "Use WAL", time.Now().Add(-time.Hour)); err != nil || f == nil {
		t.Fatalf("dup finder: %v %v", f, err)
	}
	if f, _ := FindRecentDuplicate(ctx, d.R, "h1", "project", "decision", "Use WAL", time.Now().Add(time.Hour)); f != nil {
		t.Fatal("since filter ignored")
	}
}

func TestReviseSnapshotsAndOperations(t *testing.T) {
	d := newTestDB(t, false)
	ctx := context.Background()
	o := sampleObs("u1")
	_ = InsertObservation(ctx, d.W, o)
	now := time.Now().UTC()
	if err := SnapshotVersion(ctx, d.W, o.ID, 1, o.Title, o.Content, "s1", now); err != nil {
		t.Fatal(err)
	}
	o.Title, o.Content, o.RevisionCount = "Use WAL v2", "new body", 2
	if err := ReviseObservation(ctx, d.W, o); err != nil {
		t.Fatal(err)
	}
	got, _ := GetObservationByUID(ctx, d.R, "u1")
	if got.RevisionCount != 2 || got.Title != "Use WAL v2" {
		t.Fatalf("revise: %+v", got)
	}
	if err := InsertOperation(ctx, d.W, "op1", "fp1", o.ID, "u1", now); err != nil {
		t.Fatal(err)
	}
	op, err := LookupOperation(ctx, d.R, "op1")
	if err != nil || op == nil || op.Fingerprint != "fp1" || !op.ObservationID.Valid {
		t.Fatalf("op: %+v %v", op, err)
	}
	if op, _ := LookupOperation(ctx, d.R, "nope"); op != nil {
		t.Fatal("expected nil for unknown op")
	}
}

func TestSessionsAndProjects(t *testing.T) {
	g := newTestDB(t, true)
	ctx := context.Background()
	now := time.Now().UTC()
	s := api.Session{ID: "s1", Project: "p", Title: "T", Directory: "/x", StartedAt: now}
	if err := UpsertSession(ctx, g.W, s); err != nil {
		t.Fatal(err)
	}
	if err := UpsertSession(ctx, g.W, s); err != nil { // idempotent resume
		t.Fatal(err)
	}
	ok, err := EndSession(ctx, g.W, "s1", "done", now)
	if err != nil || !ok {
		t.Fatalf("end: %v %v", ok, err)
	}
	got, _ := GetSession(ctx, g.R, "s1")
	if got.Summary != "done" || got.EndedAt.IsZero() {
		t.Fatalf("session: %+v", got)
	}
	if ok, _ := EndSession(ctx, g.W, "nope", "", now); ok {
		t.Fatal("ending unknown session must report false")
	}
	if err := TouchProject(ctx, g.W, "p", "p.db", now); err != nil {
		t.Fatal(err)
	}
	if err := TouchProject(ctx, g.W, "p", "p.db", now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = g.R.QueryRow(`SELECT COUNT(*) FROM projects`).Scan(&n)
	if n != 1 {
		t.Fatalf("projects = %d", n)
	}
}
```

Add `"mnemonic/internal/platform/apperr"` to the test imports. Run: `go test ./internal/memory/store/ -v`
Expected: FAIL (`undefined: InsertObservation` etc.).

- [ ] **Step 3: Write `observations.go`**

`internal/memory/store/observations.go`:

```go
package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"mnemonic/internal/memory/api"
	"mnemonic/internal/platform/apperr"
)

const obsCols = `id, uid, COALESCE(session_id,''), project, scope, type, title, content,
	COALESCE(topic_key,''), tool_name, normalized_hash, revision_count, duplicate_count,
	pinned, importance, retrieval_usage, COALESCE(review_after,''), created_at, updated_at, last_seen_at`

type scanner interface{ Scan(dest ...any) error }

func scanObs(r scanner) (api.Observation, error) {
	var o api.Observation
	var pinned int
	var review, created, updated, seen string
	err := r.Scan(&o.ID, &o.UID, &o.SessionID, &o.Project, &o.Scope, &o.Type, &o.Title, &o.Content,
		&o.TopicKey, &o.ToolName, &o.NormalizedHash, &o.RevisionCount, &o.DuplicateCount,
		&pinned, &o.Importance, &o.RetrievalUsage, &review, &created, &updated, &seen)
	if err != nil {
		return o, err
	}
	o.Pinned = pinned == 1
	o.ReviewAfter = ParseTS(review)
	o.CreatedAt, o.UpdatedAt, o.LastSeenAt = ParseTS(created), ParseTS(updated), ParseTS(seen)
	return o, nil
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// InsertObservation inserts o and sets o.ID.
func InsertObservation(ctx context.Context, q Querier, o *api.Observation) error {
	var review any
	if !o.ReviewAfter.IsZero() {
		review = TS(o.ReviewAfter)
	}
	res, err := q.ExecContext(ctx, `INSERT INTO observations
		(uid, session_id, project, scope, type, title, content, topic_key, tool_name, normalized_hash,
		 revision_count, duplicate_count, pinned, importance, retrieval_usage, review_after,
		 created_at, updated_at, last_seen_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		o.UID, nullable(o.SessionID), o.Project, o.Scope, o.Type, o.Title, o.Content,
		nullable(o.TopicKey), o.ToolName, o.NormalizedHash, o.RevisionCount, o.DuplicateCount,
		boolInt(o.Pinned), o.Importance, o.RetrievalUsage, review,
		TS(o.CreatedAt), TS(o.UpdatedAt), TS(o.LastSeenAt))
	if err != nil {
		return err
	}
	o.ID, err = res.LastInsertId()
	return err
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func GetObservationByUID(ctx context.Context, q Querier, uid string) (api.Observation, error) {
	o, err := scanObs(q.QueryRowContext(ctx, `SELECT `+obsCols+` FROM observations WHERE uid=? AND deleted_at IS NULL`, uid))
	if errors.Is(err, sql.ErrNoRows) {
		return o, apperr.Newf(apperr.NotFound, "observation %q not found", uid)
	}
	return o, err
}

func GetObservationByID(ctx context.Context, q Querier, id int64) (api.Observation, error) {
	o, err := scanObs(q.QueryRowContext(ctx, `SELECT `+obsCols+` FROM observations WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return o, apperr.Newf(apperr.NotFound, "observation id %d not found", id)
	}
	return o, err
}

// FindActiveByTopic returns the newest active observation for (topic_key, scope) or nil.
func FindActiveByTopic(ctx context.Context, q Querier, topicKey, scope string) (*api.Observation, error) {
	o, err := scanObs(q.QueryRowContext(ctx, `SELECT `+obsCols+` FROM observations
		WHERE topic_key=? AND scope=? AND deleted_at IS NULL ORDER BY updated_at DESC LIMIT 1`, topicKey, scope))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

// FindRecentDuplicate returns an active observation with identical hash/scope/type/title seen since `since`.
func FindRecentDuplicate(ctx context.Context, q Querier, hash, scope, typ, title string, since time.Time) (*api.Observation, error) {
	o, err := scanObs(q.QueryRowContext(ctx, `SELECT `+obsCols+` FROM observations
		WHERE normalized_hash=? AND scope=? AND type=? AND title=? AND deleted_at IS NULL AND last_seen_at>=?
		ORDER BY last_seen_at DESC LIMIT 1`, hash, scope, typ, title, TS(since)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func SnapshotVersion(ctx context.Context, q Querier, obsID int64, version int, title, content, sessionID string, now time.Time) error {
	_, err := q.ExecContext(ctx, `INSERT INTO observation_versions
		(observation_id, version, title, content, session_id, created_at) VALUES (?,?,?,?,?,?)`,
		obsID, version, title, content, nullable(sessionID), TS(now))
	return err
}

// ReviseObservation writes the mutable fields of o back to its row.
func ReviseObservation(ctx context.Context, q Querier, o *api.Observation) error {
	_, err := q.ExecContext(ctx, `UPDATE observations SET
		title=?, content=?, type=?, normalized_hash=?, revision_count=?, session_id=COALESCE(?, session_id),
		updated_at=?, last_seen_at=? WHERE id=?`,
		o.Title, o.Content, o.Type, o.NormalizedHash, o.RevisionCount, nullable(o.SessionID),
		TS(o.UpdatedAt), TS(o.LastSeenAt), o.ID)
	return err
}

func BumpDuplicate(ctx context.Context, q Querier, id int64, now time.Time) error {
	_, err := q.ExecContext(ctx, `UPDATE observations SET duplicate_count=duplicate_count+1,
		last_seen_at=?, updated_at=? WHERE id=?`, TS(now), TS(now), id)
	return err
}

func AddTrail(ctx context.Context, q Querier, sessionID, action, uid string, now time.Time) error {
	_, err := q.ExecContext(ctx, `INSERT INTO session_trails(session_id, action_type, target_uid, created_at)
		VALUES (?,?,?,?)`, sessionID, action, uid, TS(now))
	return err
}
```

- [ ] **Step 4: Write `operations.go`, `sessions.go`, `projects.go`**

`internal/memory/store/operations.go`:

```go
package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type Operation struct {
	Fingerprint    string
	ObservationID  sql.NullInt64
	ObservationUID string
}

// LookupOperation returns the ledger row for id, or nil when unseen.
func LookupOperation(ctx context.Context, q Querier, id string) (*Operation, error) {
	var op Operation
	err := q.QueryRowContext(ctx, `SELECT fingerprint, observation_id, observation_uid
		FROM save_operations WHERE operation_id=?`, id).Scan(&op.Fingerprint, &op.ObservationID, &op.ObservationUID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &op, nil
}

func InsertOperation(ctx context.Context, q Querier, id, fingerprint string, obsID int64, uid string, now time.Time) error {
	_, err := q.ExecContext(ctx, `INSERT INTO save_operations
		(operation_id, fingerprint, observation_id, observation_uid, created_at) VALUES (?,?,?,?,?)`,
		id, fingerprint, obsID, uid, TS(now))
	return err
}
```

`internal/memory/store/sessions.go`:

```go
package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"mnemonic/internal/memory/api"
	"mnemonic/internal/platform/apperr"
)

// UpsertSession creates the session or, if it exists, refreshes title/directory (resume).
func UpsertSession(ctx context.Context, q Querier, s api.Session) error {
	now := TS(s.StartedAt)
	_, err := q.ExecContext(ctx, `INSERT INTO sessions(id, project, title, directory, started_at, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			title = CASE WHEN excluded.title != '' THEN excluded.title ELSE sessions.title END,
			directory = CASE WHEN excluded.directory != '' THEN excluded.directory ELSE sessions.directory END,
			updated_at = excluded.updated_at`,
		s.ID, s.Project, s.Title, s.Directory, now, now, now)
	return err
}

// EndSession marks the session ended; returns false when the session does not exist.
func EndSession(ctx context.Context, q Querier, id, summary string, now time.Time) (bool, error) {
	res, err := q.ExecContext(ctx, `UPDATE sessions SET ended_at=?, summary_json=NULLIF(?, ''), updated_at=? WHERE id=?`,
		TS(now), summary, TS(now), id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func GetSession(ctx context.Context, q Querier, id string) (api.Session, error) {
	var s api.Session
	var started, ended, summary string
	err := q.QueryRowContext(ctx, `SELECT id, project, title, directory, started_at,
		COALESCE(ended_at,''), COALESCE(summary_json,'') FROM sessions WHERE id=?`, id).
		Scan(&s.ID, &s.Project, &s.Title, &s.Directory, &started, &ended, &summary)
	if errors.Is(err, sql.ErrNoRows) {
		return s, apperr.Newf(apperr.NotFound, "session %q not found", id)
	}
	s.StartedAt, s.EndedAt, s.Summary = ParseTS(started), ParseTS(ended), summary
	return s, err
}
```

`internal/memory/store/projects.go`:

```go
package store

import (
	"context"
	"time"
)

// TouchProject registers the project in global.db or refreshes last_active_at.
func TouchProject(ctx context.Context, q Querier, slug, dbFile string, now time.Time) error {
	ts := TS(now)
	_, err := q.ExecContext(ctx, `INSERT INTO projects(slug, db_file, created_at, last_active_at)
		VALUES (?,?,?,?)
		ON CONFLICT(slug) DO UPDATE SET last_active_at=excluded.last_active_at`, slug, dbFile, ts, ts)
	return err
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/memory/store/ -race -v`
Expected: PASS (all store tests including the Task 6 ones).

- [ ] **Step 6: Commit**

```bash
git add internal/memory/api internal/memory/store
git commit -m "feat(memory): api contract and store CRUD for sessions/observations/operations/projects"
```

---

### Task 8: Lexical search in the store

**Files:**
- Create: `internal/memory/store/search.go`
- Test: `internal/memory/store/search_test.go`

**Interfaces:**
- Produces: `store.SearchFilter{Query, Scope, Type, MatchMode string; Limit int}`; `store.SearchLexical(ctx, q Querier, f SearchFilter) ([]api.SearchHit, error)` — FTS5 `bm25(5,1,3,0.5,0.5)` when every term has ≥3 runes, otherwise an escaped `LIKE` fallback (≤16 terms). `Score` is higher-is-better (`-bm25`; `0` for LIKE hits ordered by recency).

- [ ] **Step 1: Write the failing test**

`internal/memory/store/search_test.go`:

```go
package store

import (
	"context"
	"testing"
	"time"

	"mnemonic/internal/memory/api"
)

func insert(t *testing.T, q Querier, uid, typ, title, content, topic string) {
	t.Helper()
	now := time.Now().UTC()
	o := &api.Observation{UID: uid, Project: "p", Scope: "project", Type: typ, Title: title, Content: content,
		TopicKey: topic, NormalizedHash: uid, RevisionCount: 1, DuplicateCount: 1, Importance: 0.5,
		CreatedAt: now, UpdatedAt: now, LastSeenAt: now}
	if err := InsertObservation(context.Background(), q, o); err != nil {
		t.Fatal(err)
	}
}

func TestSearchLexical(t *testing.T) {
	d := newTestDB(t, false)
	insert(t, d.W, "u1", "decision", "Use WAL mode", "SQLite WAL gives concurrent readers", "arch/wal")
	insert(t, d.W, "u2", "bugfix", "Fixed N+1 query", "UserList loaded profiles one by one", "")
	insert(t, d.W, "u3", "config", "Set DB path", "go build uses the db directory", "")
	ctx := context.Background()

	hits, err := SearchLexical(ctx, d.R, SearchFilter{Query: "concurrent readers", Limit: 10})
	if err != nil || len(hits) != 1 || hits[0].UID != "u1" {
		t.Fatalf("fts any: %v %v", hits, err)
	}
	if hits[0].Score <= 0 {
		t.Errorf("score should be positive (higher is better): %v", hits[0].Score)
	}
	hits, _ = SearchLexical(ctx, d.R, SearchFilter{Query: "WAL profiles", MatchMode: "all", Limit: 10})
	if len(hits) != 0 {
		t.Fatalf("all-mode must require every term: %v", hits)
	}
	hits, _ = SearchLexical(ctx, d.R, SearchFilter{Query: "WAL profiles", MatchMode: "any", Limit: 10})
	if len(hits) != 2 {
		t.Fatalf("any-mode: %v", hits)
	}
	hits, _ = SearchLexical(ctx, d.R, SearchFilter{Query: "arch/wal", Limit: 10})
	if len(hits) != 1 || hits[0].UID != "u1" {
		t.Fatalf("topic_key search: %v", hits)
	}
	hits, _ = SearchLexical(ctx, d.R, SearchFilter{Query: "WAL readers", Type: "bugfix", Limit: 10})
	if len(hits) != 0 {
		t.Fatalf("type filter: %v", hits)
	}
}

func TestSearchShortTermFallsBackToLike(t *testing.T) {
	d := newTestDB(t, false)
	insert(t, d.W, "u1", "config", "Set DB path", "go build uses the db directory", "")
	hits, err := SearchLexical(context.Background(), d.R, SearchFilter{Query: "db", Limit: 10})
	if err != nil || len(hits) != 1 {
		t.Fatalf("short term: %v %v", hits, err)
	}
	// LIKE metacharacters must be treated literally
	hits, _ = SearchLexical(context.Background(), d.R, SearchFilter{Query: "%", Limit: 10})
	if len(hits) != 0 {
		t.Fatalf("%% must be escaped: %v", hits)
	}
}

func TestSearchSkipsSoftDeletedAndHandlesQuotes(t *testing.T) {
	d := newTestDB(t, false)
	insert(t, d.W, "u1", "decision", `Quote "test" case`, "body text here", "")
	ctx := context.Background()
	if hits, err := SearchLexical(ctx, d.R, SearchFilter{Query: `"test"`, Limit: 5}); err != nil || len(hits) != 1 {
		t.Fatalf("quote handling: %v %v", hits, err)
	}
	if _, err := d.W.Exec(`UPDATE observations SET deleted_at=? WHERE uid='u1'`, TS(time.Now())); err != nil {
		t.Fatal(err)
	}
	if hits, _ := SearchLexical(ctx, d.R, SearchFilter{Query: "body text", Limit: 5}); len(hits) != 0 {
		t.Fatalf("soft-deleted must be hidden: %v", hits)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/memory/store/ -run Search -v`
Expected: FAIL (`undefined: SearchLexical`).

- [ ] **Step 3: Write minimal implementation**

`internal/memory/store/search.go`:

```go
package store

import (
	"context"
	"strings"
	"unicode/utf8"

	"mnemonic/internal/memory/api"
)

const maxLikeTerms = 16

type SearchFilter struct {
	Query     string
	Scope     string
	Type      string
	MatchMode string // "any" (default) | "all"
	Limit     int
}

// SearchLexical runs the FTS5 trigram leg, falling back to LIKE when any
// term is shorter than 3 runes (trigram cannot match those).
func SearchLexical(ctx context.Context, q Querier, f SearchFilter) ([]api.SearchHit, error) {
	terms := strings.Fields(f.Query)
	if len(terms) == 0 {
		return nil, nil
	}
	if f.Limit <= 0 {
		f.Limit = 10
	}
	short := false
	for _, t := range terms {
		if utf8.RuneCountInString(t) < 3 {
			short = true
			break
		}
	}
	if short {
		return searchLike(ctx, q, terms, f)
	}
	return searchFTS(ctx, q, terms, f)
}

func joiner(mode string) string {
	if mode == "all" {
		return " AND "
	}
	return " OR "
}

func searchFTS(ctx context.Context, q Querier, terms []string, f SearchFilter) ([]api.SearchHit, error) {
	quoted := make([]string, len(terms))
	for i, t := range terms {
		quoted[i] = `"` + strings.ReplaceAll(t, `"`, `""`) + `"`
	}
	sqlText := `SELECT o.uid, o.title, o.type, o.scope, o.project, COALESCE(o.topic_key,''),
		substr(o.content,1,300), bm25(observations_fts, 5.0, 1.0, 3.0, 0.5, 0.5) AS rank
		FROM observations_fts JOIN observations o ON o.id = observations_fts.rowid
		WHERE observations_fts MATCH ? AND o.deleted_at IS NULL`
	args := []any{strings.Join(quoted, joiner(f.MatchMode))}
	sqlText, args = appendFilters(sqlText, args, f)
	sqlText += ` ORDER BY rank LIMIT ?`
	args = append(args, f.Limit)
	return scanHits(ctx, q, sqlText, args, true)
}

func likeEscape(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

func searchLike(ctx context.Context, q Querier, terms []string, f SearchFilter) ([]api.SearchHit, error) {
	if len(terms) > maxLikeTerms {
		terms = terms[:maxLikeTerms]
	}
	clauses := make([]string, len(terms))
	var args []any
	for i, t := range terms {
		clauses[i] = `(o.title LIKE ? ESCAPE '\' OR o.content LIKE ? ESCAPE '\' OR o.topic_key LIKE ? ESCAPE '\')`
		p := "%" + likeEscape(t) + "%"
		args = append(args, p, p, p)
	}
	sqlText := `SELECT o.uid, o.title, o.type, o.scope, o.project, COALESCE(o.topic_key,''),
		substr(o.content,1,300), 0.0 FROM observations o
		WHERE o.deleted_at IS NULL AND (` + strings.Join(clauses, joiner(f.MatchMode)) + `)`
	sqlText, args = appendFilters(sqlText, args, f)
	sqlText += ` ORDER BY o.updated_at DESC LIMIT ?`
	args = append(args, f.Limit)
	return scanHits(ctx, q, sqlText, args, false)
}

func appendFilters(sqlText string, args []any, f SearchFilter) (string, []any) {
	if f.Type != "" {
		sqlText += ` AND o.type = ?`
		args = append(args, f.Type)
	}
	if f.Scope != "" {
		sqlText += ` AND o.scope = ?`
		args = append(args, f.Scope)
	}
	return sqlText, args
}

func scanHits(ctx context.Context, q Querier, sqlText string, args []any, negate bool) ([]api.SearchHit, error) {
	rows, err := q.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var hits []api.SearchHit
	for rows.Next() {
		var h api.SearchHit
		var rank float64
		if err := rows.Scan(&h.UID, &h.Title, &h.Type, &h.Scope, &h.Project, &h.TopicKey, &h.Snippet, &rank); err != nil {
			return nil, err
		}
		if negate {
			h.Score = -rank // bm25: lower is better -> flip so higher is better
		}
		hits = append(hits, h)
	}
	return hits, rows.Err()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/memory/store/ -race -v`
Expected: PASS (FTS5 `bm25()` is negative for matches, so the flipped score is positive).

- [ ] **Step 5: Commit**

```bash
git add internal/memory/store
git commit -m "feat(memory): lexical search (FTS5 bm25 + LIKE fallback)"
```

---

### Task 9: `engine.Local` — DB routing and the `Save` pipeline

**Files:**
- Create: `internal/memory/engine/local.go`, `internal/memory/engine/save.go`
- Test: `internal/memory/engine/testutil_test.go`, `internal/memory/engine/save_test.go`

**Interfaces:**
- Consumes: `store.*`, `sqlitepool.Pool`, `config.Paths`, `api.*`, `apperr.*`.
- Produces: `engine.Local` implementing `api.Service`; `engine.NewLocal(paths config.Paths) *Local`; `(*Local).WithClock(func() time.Time) *Local` (test seam); internal `(*Local).dbFor(scope, project string, create bool) (*sqlitepool.DB, error)` returning `(nil, nil)` when `create == false` and the file does not exist.

- [ ] **Step 1: Write the test helper and failing tests**

`internal/memory/engine/testutil_test.go`:

```go
package engine

import (
	"testing"
	"time"

	"mnemonic/internal/platform/config"
)

type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newEngine(t *testing.T) (*Local, *fakeClock, config.Paths) {
	t.Helper()
	paths, err := config.NewPaths(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	clk := &fakeClock{t: time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)}
	l := NewLocal(paths).WithClock(clk.now)
	t.Cleanup(func() { _ = l.Close() })
	return l, clk, paths
}
```

`internal/memory/engine/save_test.go`:

```go
package engine

import (
	"context"
	"strings"
	"testing"
	"time"

	"mnemonic/internal/memory/api"
	"mnemonic/internal/platform/apperr"
)

func base() api.SaveRequest {
	return api.SaveRequest{Project: "acme", Type: "decision", Title: "Use WAL", Content: "because concurrency"}
}

func TestSaveCreateRoutesByScope(t *testing.T) {
	l, _, paths := newEngine(t)
	ctx := context.Background()

	r, err := l.Save(ctx, base())
	if err != nil || r.Verdict != "create" || r.UID == "" || r.RevisionCount != 1 {
		t.Fatalf("project save: %+v %v", r, err)
	}
	g := base()
	g.Scope, g.Title = "user", "Prefers tabs"
	if _, err := l.Save(ctx, g); err != nil {
		t.Fatal(err)
	}
	// project DB holds the project row; global DB holds the user row + registry entry
	pdb, _ := l.dbFor("project", "acme", false)
	gdb, _ := l.dbFor("user", "acme", false)
	if pdb == nil || gdb == nil || pdb.Path != paths.ProjectDB("acme") || gdb.Path != paths.GlobalDB() {
		t.Fatalf("db routing: %v %v", pdb, gdb)
	}
	var np, ng, nreg int
	_ = pdb.R.QueryRow(`SELECT COUNT(*) FROM observations`).Scan(&np)
	_ = gdb.R.QueryRow(`SELECT COUNT(*) FROM observations`).Scan(&ng)
	_ = gdb.R.QueryRow(`SELECT COUNT(*) FROM projects WHERE slug='acme'`).Scan(&nreg)
	if np != 1 || ng != 1 || nreg != 1 {
		t.Fatalf("rows project=%d global=%d registry=%d", np, ng, nreg)
	}
}

func TestSaveValidation(t *testing.T) {
	l, _, _ := newEngine(t)
	ctx := context.Background()
	bad := []api.SaveRequest{
		{Project: "acme", Type: "decision", Content: "x"},                       // no title
		{Project: "acme", Type: "decision", Title: "t"},                         // no content
		{Project: "acme", Type: "nonsense", Title: "t", Content: "x"},           // bad type
		{Project: "acme", Type: "decision", Title: "t", Content: "x", Scope: "?"}, // bad scope
		{Type: "decision", Title: "t", Content: "x"},                            // project scope needs project
	}
	for i, req := range bad {
		if _, err := l.Save(ctx, req); apperr.CodeOf(err) != apperr.Validation {
			t.Errorf("case %d: err = %v", i, err)
		}
	}
}

func TestSaveRedactsPrivateAndCapsContent(t *testing.T) {
	l, _, _ := newEngine(t)
	ctx := context.Background()
	req := base()
	req.Content = "token is <private>sk-secret</private> ok\n<private>multi\nline</private>"
	r, err := l.Save(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	o, _ := l.Get(ctx, api.GetRequest{Project: "acme", UID: r.UID})
	if strings.Contains(o.Content, "sk-secret") || strings.Count(o.Content, "[REDACTED]") != 2 {
		t.Fatalf("content not redacted: %q", o.Content)
	}
	big := base()
	big.Title = "big"
	big.Content = strings.Repeat("é", 40000) // 80,000 bytes
	r2, _ := l.Save(ctx, big)
	o2, _ := l.Get(ctx, api.GetRequest{Project: "acme", UID: r2.UID})
	if len(o2.Content) > 32*1024 || !strings.HasSuffix(o2.Content, "é") {
		t.Fatalf("cap failed: %d bytes", len(o2.Content))
	}
}

func TestTopicKeyRevisesAndSnapshots(t *testing.T) {
	l, clk, _ := newEngine(t)
	ctx := context.Background()
	a := base()
	a.TopicKey = "architecture/storage"
	r1, _ := l.Save(ctx, a)
	clk.advance(time.Minute)
	a.Title, a.Content = "Use WAL v2", "now with readers"
	r2, err := l.Save(ctx, a)
	if err != nil || r2.Verdict != "revise" || r2.UID != r1.UID || r2.RevisionCount != 2 {
		t.Fatalf("revise: %+v %v", r2, err)
	}
	db, _ := l.dbFor("project", "acme", false)
	var oldTitle string
	var version int
	if err := db.R.QueryRow(`SELECT title, version FROM observation_versions`).Scan(&oldTitle, &version); err != nil {
		t.Fatal(err)
	}
	if oldTitle != "Use WAL" || version != 1 {
		t.Fatalf("snapshot title=%q version=%d", oldTitle, version)
	}
	// identical re-save under the same topic is a duplicate, not a revision
	r3, _ := l.Save(ctx, a)
	if r3.Verdict != "duplicate" || r3.RevisionCount != 2 {
		t.Fatalf("topic duplicate: %+v", r3)
	}
}

func TestHashDuplicateWithin24hOnly(t *testing.T) {
	l, clk, _ := newEngine(t)
	ctx := context.Background()
	r1, _ := l.Save(ctx, base())
	clk.advance(2 * time.Hour)
	r2, _ := l.Save(ctx, base())
	if r2.Verdict != "duplicate" || r2.UID != r1.UID {
		t.Fatalf("within 24h: %+v", r2)
	}
	clk.advance(25 * time.Hour)
	r3, _ := l.Save(ctx, base())
	if r3.Verdict != "create" || r3.UID == r1.UID {
		t.Fatalf("after 24h should create anew: %+v", r3)
	}
}

func TestOperationIDIdempotency(t *testing.T) {
	l, _, _ := newEngine(t)
	ctx := context.Background()
	req := base()
	req.OperationID = "op-1"
	r1, err := l.Save(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := l.Save(ctx, req)
	if err != nil || !r2.Replayed || r2.Verdict != "replay" || r2.UID != r1.UID {
		t.Fatalf("replay: %+v %v", r2, err)
	}
	changed := req
	changed.Content = "different"
	if _, err := l.Save(ctx, changed); apperr.CodeOf(err) != apperr.Conflict {
		t.Fatalf("mismatch must be FINGERPRINT_CONFLICT, got %v", err)
	}
	db, _ := l.dbFor("project", "acme", false)
	if _, err := db.W.Exec(`UPDATE save_operations SET observation_id=NULL WHERE operation_id='op-1'`); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Save(ctx, req); apperr.CodeOf(err) != apperr.Gone {
		t.Fatalf("hard-deleted op must be TOMBSTONE_GONE, got %v", err)
	}
}

func TestSaveWritesSessionTrail(t *testing.T) {
	l, _, _ := newEngine(t)
	req := base()
	req.SessionID = "s1"
	r, _ := l.Save(context.Background(), req)
	db, _ := l.dbFor("project", "acme", false)
	var action, uid string
	if err := db.R.QueryRow(`SELECT action_type, target_uid FROM session_trails WHERE session_id='s1'`).Scan(&action, &uid); err != nil {
		t.Fatal(err)
	}
	if action != "save" || uid != r.UID {
		t.Fatalf("trail %q %q", action, uid)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/memory/engine/ -v`
Expected: FAIL (`undefined: NewLocal`).

- [ ] **Step 3: Write `local.go`**

`internal/memory/engine/local.go`:

```go
// Package engine contains the in-process implementation of api.Service.
package engine

import (
	"os"
	"time"

	"github.com/google/uuid"

	"mnemonic/internal/memory/api"
	"mnemonic/internal/memory/store"
	"mnemonic/internal/platform/apperr"
	"mnemonic/internal/platform/config"
	"mnemonic/internal/platform/sqlitepool"
)

// Local implements api.Service directly on SQLite, with no daemon.
type Local struct {
	paths config.Paths
	pool  *sqlitepool.Pool
	now   func() time.Time
}

var _ api.Service = (*Local)(nil)

func NewLocal(paths config.Paths) *Local {
	return &Local{paths: paths, pool: sqlitepool.NewPool(), now: time.Now}
}

// WithClock replaces the clock (tests).
func (l *Local) WithClock(now func() time.Time) *Local {
	l.now = now
	return l
}

func (l *Local) Close() error { return l.pool.Close() }

func isGlobalScope(scope string) bool { return scope == "user" || scope == "global" }

// dbFor routes a scope to its database. With create == false it returns
// (nil, nil) when the file does not exist yet (reads never create DBs).
func (l *Local) dbFor(scope, project string, create bool) (*sqlitepool.DB, error) {
	var path string
	global := isGlobalScope(scope)
	if global {
		path = l.paths.GlobalDB()
	} else {
		if project == "" {
			return nil, apperr.New(apperr.Validation, "project is required for project scope")
		}
		path = l.paths.ProjectDB(project)
	}
	if !create {
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, err
		}
	}
	return l.pool.Get(path, func(d *sqlitepool.DB) error { return store.Init(d, global) })
}

func newUID() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}
```

- [ ] **Step 4: Write `save.go`**

`internal/memory/engine/save.go`:

```go
package engine

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"mnemonic/internal/memory/api"
	"mnemonic/internal/memory/store"
	"mnemonic/internal/platform/apperr"
)

const (
	maxContentBytes = 32 * 1024
	dedupeWindow    = 24 * time.Hour
)

var (
	privateRe  = regexp.MustCompile(`(?s)<private>.*?</private>`)
	spaceRe    = regexp.MustCompile(`\s+`)
	validTypes = map[string]bool{"decision": true, "architecture": true, "bugfix": true, "pattern": true,
		"config": true, "discovery": true, "learning": true, "preference": true, "convention": true}
	validScopes = map[string]bool{"project": true, "user": true, "global": true}
)

func redact(s string) string { return privateRe.ReplaceAllString(s, "[REDACTED]") }

func truncateUTF8(s string, max int) string {
	if len(s) <= max {
		return s
	}
	cut := max
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}

func sha(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func normalizedHash(title, content string) string {
	return sha(strings.ToLower(spaceRe.ReplaceAllString(strings.TrimSpace(title+"\n"+content), " ")))
}

func reviewAfter(typ string, now time.Time) time.Time {
	switch typ {
	case "config", "preference":
		return now.AddDate(0, 0, 90)
	case "architecture", "bugfix":
		return now.AddDate(0, 0, 365)
	default:
		return now.AddDate(0, 0, 180)
	}
}

func validateSave(r *api.SaveRequest) error {
	if r.Scope == "" {
		r.Scope = "project"
	}
	switch {
	case !validScopes[r.Scope]:
		return apperr.Newf(apperr.Validation, "invalid scope %q (project|user|global)", r.Scope)
	case !validTypes[r.Type]:
		return apperr.Newf(apperr.Validation, "invalid type %q", r.Type)
	case strings.TrimSpace(r.Title) == "":
		return apperr.New(apperr.Validation, "title is required")
	case strings.TrimSpace(r.Content) == "":
		return apperr.New(apperr.Validation, "content is required")
	case r.Scope == "project" && r.Project == "":
		return apperr.New(apperr.Validation, "project is required for project scope")
	}
	return nil
}

// Save runs the write pipeline: validate/redact -> idempotency ledger ->
// topic_key revise / hash duplicate / create -> trail, all in one transaction.
func (l *Local) Save(ctx context.Context, req api.SaveRequest) (api.SaveResult, error) {
	if err := validateSave(&req); err != nil {
		return api.SaveResult{}, err
	}
	title := truncateUTF8(redact(strings.TrimSpace(req.Title)), 512)
	content := truncateUTF8(redact(req.Content), maxContentBytes)
	hash := normalizedHash(title, content)
	fp := sha(req.Scope, req.Type, title, content, req.TopicKey, req.ToolName)
	now := l.now().UTC()

	db, err := l.dbFor(req.Scope, req.Project, true)
	if err != nil {
		return api.SaveResult{}, err
	}
	var res api.SaveResult
	err = db.Write(ctx, func(tx *sql.Tx) error {
		var err error
		if req.OperationID != "" {
			op, err := store.LookupOperation(ctx, tx, req.OperationID)
			if err != nil {
				return err
			}
			if op != nil {
				if op.Fingerprint != fp {
					return apperr.Newf(apperr.Conflict, "operation_id %q was already used with a different payload", req.OperationID)
				}
				if !op.ObservationID.Valid {
					return apperr.Newf(apperr.Gone, "operation_id %q refers to a deleted observation", req.OperationID)
				}
				o, err := store.GetObservationByID(ctx, tx, op.ObservationID.Int64)
				if err != nil {
					return err
				}
				res = api.SaveResult{UID: op.ObservationUID, Verdict: "replay", RevisionCount: o.RevisionCount, Replayed: true}
				return nil
			}
		}

		var existing *api.Observation
		if req.TopicKey != "" {
			if existing, err = store.FindActiveByTopic(ctx, tx, req.TopicKey, req.Scope); err != nil {
				return err
			}
		}
		if existing == nil {
			if existing, err = store.FindRecentDuplicate(ctx, tx, hash, req.Scope, req.Type, title, now.Add(-dedupeWindow)); err != nil {
				return err
			}
			if existing != nil {
				return finishDuplicate(ctx, tx, existing, req, fp, now, &res)
			}
			return createObservation(ctx, tx, req, title, content, hash, fp, now, &res)
		}
		if existing.NormalizedHash == hash {
			return finishDuplicate(ctx, tx, existing, req, fp, now, &res)
		}
		// revise: snapshot the prior state, then update in place
		if err := store.SnapshotVersion(ctx, tx, existing.ID, existing.RevisionCount, existing.Title, existing.Content, req.SessionID, now); err != nil {
			return err
		}
		existing.Title, existing.Content, existing.Type = title, content, req.Type
		existing.NormalizedHash, existing.RevisionCount = hash, existing.RevisionCount+1
		existing.SessionID, existing.UpdatedAt, existing.LastSeenAt = req.SessionID, now, now
		if err := store.ReviseObservation(ctx, tx, existing); err != nil {
			return err
		}
		res = api.SaveResult{UID: existing.UID, Verdict: "revise", RevisionCount: existing.RevisionCount}
		if err := recordOp(ctx, tx, req, fp, existing.ID, existing.UID, now); err != nil {
			return err
		}
		return trail(ctx, tx, req.SessionID, "revise", existing.UID, now)
	})
	if err != nil {
		return api.SaveResult{}, err
	}
	if req.Scope == "project" {
		l.registerProject(ctx, req.Project, now)
	}
	return res, nil
}

func finishDuplicate(ctx context.Context, tx *sql.Tx, existing *api.Observation, req api.SaveRequest, fp string, now time.Time, res *api.SaveResult) error {
	if err := store.BumpDuplicate(ctx, tx, existing.ID, now); err != nil {
		return err
	}
	*res = api.SaveResult{UID: existing.UID, Verdict: "duplicate", RevisionCount: existing.RevisionCount}
	if err := recordOp(ctx, tx, req, fp, existing.ID, existing.UID, now); err != nil {
		return err
	}
	return trail(ctx, tx, req.SessionID, "duplicate", existing.UID, now)
}

func createObservation(ctx context.Context, tx *sql.Tx, req api.SaveRequest, title, content, hash, fp string, now time.Time, res *api.SaveResult) error {
	uid, err := newUID()
	if err != nil {
		return err
	}
	o := &api.Observation{
		UID: uid, SessionID: req.SessionID, Project: req.Project, Scope: req.Scope, Type: req.Type,
		Title: title, Content: content, TopicKey: req.TopicKey, ToolName: req.ToolName,
		NormalizedHash: hash, RevisionCount: 1, DuplicateCount: 1, Importance: 0.5,
		ReviewAfter: reviewAfter(req.Type, now), CreatedAt: now, UpdatedAt: now, LastSeenAt: now,
	}
	if o.Project == "" {
		o.Project = "global"
	}
	if err := store.InsertObservation(ctx, tx, o); err != nil {
		return err
	}
	*res = api.SaveResult{UID: uid, Verdict: "create", RevisionCount: 1}
	if err := recordOp(ctx, tx, req, fp, o.ID, uid, now); err != nil {
		return err
	}
	return trail(ctx, tx, req.SessionID, "save", uid, now)
}

func recordOp(ctx context.Context, tx *sql.Tx, req api.SaveRequest, fp string, obsID int64, uid string, now time.Time) error {
	if req.OperationID == "" {
		return nil
	}
	return store.InsertOperation(ctx, tx, req.OperationID, fp, obsID, uid, now)
}

func trail(ctx context.Context, tx *sql.Tx, sessionID, action, uid string, now time.Time) error {
	if sessionID == "" {
		return nil
	}
	return store.AddTrail(ctx, tx, sessionID, action, uid, now)
}

// registerProject records the project in global.db; failures are non-fatal
// because the observation is already durably saved.
func (l *Local) registerProject(ctx context.Context, slug string, now time.Time) {
	g, err := l.dbFor("global", "", true)
	if err != nil {
		return
	}
	_ = g.Write(ctx, func(tx *sql.Tx) error {
		return store.TouchProject(ctx, tx, slug, slug+".db", now)
	})
}
```

- [ ] **Step 5: Add a temporary stub so the package compiles**

`api.Service` also needs `Search`, `Get`, `SessionStart`, `SessionEnd`; Task 10 implements them. Until then create `internal/memory/engine/zz_stub.go` (**do not commit it**; Task 10 step 2 deletes it):

```go
package engine

import (
	"context"

	"mnemonic/internal/memory/api"
	"mnemonic/internal/memory/store"
)

func (l *Local) Search(context.Context, api.SearchRequest) (api.SearchResult, error) {
	return api.SearchResult{}, nil
}

func (l *Local) Get(ctx context.Context, r api.GetRequest) (api.Observation, error) {
	db, err := l.dbFor("project", r.Project, false)
	if err != nil || db == nil {
		return api.Observation{}, err
	}
	return store.GetObservationByUID(ctx, db.R, r.UID)
}

func (l *Local) SessionStart(context.Context, api.SessionStartRequest) (api.Session, error) {
	return api.Session{}, nil
}

func (l *Local) SessionEnd(context.Context, api.SessionEndRequest) error { return nil }
```

Also run `go get github.com/google/uuid` (UUIDv7 for `uid`).

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/memory/engine/ -race -v`
Expected: PASS for `TestSaveCreateRoutesByScope`, `TestSaveValidation`, `TestSaveRedactsPrivateAndCapsContent`, `TestTopicKeyRevisesAndSnapshots`, `TestHashDuplicateWithin24hOnly`, `TestOperationIDIdempotency`, `TestSaveWritesSessionTrail`.

If `TestTopicKeyRevisesAndSnapshots`' last assertion fails (a re-save of the revised payload being reported as `revise`), confirm that `existing.NormalizedHash` is compared against the hash of the *redacted* title+content — that is what `hash` already is.

- [ ] **Step 7: Commit (without the stub file)**

```bash
git add internal/memory/engine/local.go internal/memory/engine/save.go internal/memory/engine/*_test.go
git commit -m "feat(memory): engine.Local with scope routing and save pipeline"
```

---

### Task 10: `engine` — Search, Get, sessions

**Files:**
- Create: `internal/memory/engine/read.go`, `internal/memory/engine/session.go`
- Delete: `internal/memory/engine/zz_stub.go`
- Test: `internal/memory/engine/read_test.go`, `internal/memory/engine/session_test.go`

**Interfaces:**
- Consumes: Task 9 `Local`, `store.SearchLexical`, `store.GetObservationByUID`, `store.UpsertSession`, `store.EndSession`.
- Produces: `(*Local).Search`, `(*Local).Get`, `(*Local).SessionStart`, `(*Local).SessionEnd` per `api.Service`. Search default scope merges the project DB and `global.db` (each only if its file exists); `Limit` defaults to 10, capped at 50; hits sorted by `Score` desc then `UID`.

- [ ] **Step 1: Write the failing tests**

`internal/memory/engine/read_test.go`:

```go
package engine

import (
	"context"
	"testing"

	"mnemonic/internal/memory/api"
	"mnemonic/internal/platform/apperr"
)

func seedTwoScopes(t *testing.T, l *Local) (projUID, userUID string) {
	t.Helper()
	ctx := context.Background()
	p, err := l.Save(ctx, api.SaveRequest{Project: "acme", Type: "decision", Title: "Use WAL mode", Content: "concurrent readers matter"})
	if err != nil {
		t.Fatal(err)
	}
	u, err := l.Save(ctx, api.SaveRequest{Project: "acme", Scope: "user", Type: "preference", Title: "Prefer tabs", Content: "tabs everywhere, readers dislike spaces"})
	if err != nil {
		t.Fatal(err)
	}
	return p.UID, u.UID
}

func TestSearchMergesProjectAndGlobalByDefault(t *testing.T) {
	l, _, _ := newEngine(t)
	pu, uu := seedTwoScopes(t, l)
	res, err := l.Search(context.Background(), api.SearchRequest{Project: "acme", Query: "readers"})
	if err != nil || len(res.Hits) != 2 {
		t.Fatalf("merged: %+v %v", res, err)
	}
	seen := map[string]bool{res.Hits[0].UID: true, res.Hits[1].UID: true}
	if !seen[pu] || !seen[uu] {
		t.Fatalf("missing hit: %+v", res.Hits)
	}
	res, _ = l.Search(context.Background(), api.SearchRequest{Project: "acme", Query: "readers", Scope: "project"})
	if len(res.Hits) != 1 || res.Hits[0].UID != pu {
		t.Fatalf("project scope: %+v", res.Hits)
	}
	res, _ = l.Search(context.Background(), api.SearchRequest{Project: "acme", Query: "readers", Scope: "user"})
	if len(res.Hits) != 1 || res.Hits[0].UID != uu {
		t.Fatalf("user scope: %+v", res.Hits)
	}
}

func TestSearchOnFreshInstallDoesNotCreateDatabases(t *testing.T) {
	l, _, paths := newEngine(t)
	res, err := l.Search(context.Background(), api.SearchRequest{Project: "ghost", Query: "anything"})
	if err != nil || len(res.Hits) != 0 {
		t.Fatalf("fresh search: %+v %v", res, err)
	}
	if db, _ := l.dbFor("project", "ghost", false); db != nil {
		t.Fatal("read must not create the project DB")
	}
	if db, _ := l.dbFor("global", "", false); db != nil {
		t.Fatalf("read must not create global.db (%s)", paths.GlobalDB())
	}
}

func TestSearchValidatesAndLimits(t *testing.T) {
	l, _, _ := newEngine(t)
	if _, err := l.Search(context.Background(), api.SearchRequest{Project: "acme", Query: "  "}); apperr.CodeOf(err) != apperr.Validation {
		t.Fatalf("empty query: %v", err)
	}
	if _, err := l.Search(context.Background(), api.SearchRequest{Project: "acme", Query: "x", Scope: "bogus"}); apperr.CodeOf(err) != apperr.Validation {
		t.Fatalf("bad scope: %v", err)
	}
	seedTwoScopes(t, l)
	res, _ := l.Search(context.Background(), api.SearchRequest{Project: "acme", Query: "readers", Limit: 1})
	if len(res.Hits) != 1 {
		t.Fatalf("limit: %d", len(res.Hits))
	}
}

func TestGetProbesProjectThenGlobal(t *testing.T) {
	l, _, _ := newEngine(t)
	pu, uu := seedTwoScopes(t, l)
	ctx := context.Background()
	for _, uid := range []string{pu, uu} {
		o, err := l.Get(ctx, api.GetRequest{Project: "acme", UID: uid})
		if err != nil || o.UID != uid {
			t.Fatalf("get %s: %+v %v", uid, o, err)
		}
	}
	if _, err := l.Get(ctx, api.GetRequest{Project: "acme", UID: "nope"}); apperr.CodeOf(err) != apperr.NotFound {
		t.Fatalf("missing: %v", err)
	}
}
```

`internal/memory/engine/session_test.go`:

```go
package engine

import (
	"context"
	"testing"

	"mnemonic/internal/memory/api"
	"mnemonic/internal/platform/apperr"
)

func TestSessionLifecycle(t *testing.T) {
	l, clk, _ := newEngine(t)
	ctx := context.Background()
	s, err := l.SessionStart(ctx, api.SessionStartRequest{ID: "s1", Project: "acme", Title: "Plan memory", Directory: "/w"})
	if err != nil || s.ID != "s1" || s.StartedAt.IsZero() {
		t.Fatalf("start: %+v %v", s, err)
	}
	// resume is idempotent
	if _, err := l.SessionStart(ctx, api.SessionStartRequest{ID: "s1", Project: "acme"}); err != nil {
		t.Fatalf("resume: %v", err)
	}
	clk.advance(10)
	if err := l.SessionEnd(ctx, api.SessionEndRequest{ID: "s1", Project: "acme", Summary: "done"}); err != nil {
		t.Fatalf("end: %v", err)
	}
	if err := l.SessionEnd(ctx, api.SessionEndRequest{ID: "nope", Project: "acme"}); apperr.CodeOf(err) != apperr.NotFound {
		t.Fatalf("end unknown: %v", err)
	}
	if _, err := l.SessionStart(ctx, api.SessionStartRequest{Project: "acme"}); apperr.CodeOf(err) != apperr.Validation {
		t.Fatalf("missing id: %v", err)
	}
}
```

- [ ] **Step 2: Remove the stub and verify tests fail**

Run: `rm internal/memory/engine/zz_stub.go && go test ./internal/memory/engine/ -run 'Search|Get|Session' -v`
Expected: FAIL to compile (`l.Search undefined` etc.).

- [ ] **Step 3: Write `read.go`**

`internal/memory/engine/read.go`:

```go
package engine

import (
	"context"
	"sort"
	"strings"

	"mnemonic/internal/memory/api"
	"mnemonic/internal/memory/store"
	"mnemonic/internal/platform/apperr"
	"mnemonic/internal/platform/sqlitepool"
)

const (
	defaultLimit = 10
	maxLimit     = 50
)

// readTargets lists the existing databases a read for (scope, project) must consult.
func (l *Local) readTargets(scope, project string) ([]*sqlitepool.DB, error) {
	var out []*sqlitepool.DB
	add := func(s string) error {
		db, err := l.dbFor(s, project, false)
		if err != nil {
			return err
		}
		if db != nil {
			out = append(out, db)
		}
		return nil
	}
	switch scope {
	case "":
		if project != "" {
			if err := add("project"); err != nil {
				return nil, err
			}
		}
		if err := add("global"); err != nil {
			return nil, err
		}
	case "project":
		if err := add("project"); err != nil {
			return nil, err
		}
	default:
		if err := add(scope); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// Search runs the lexical leg across the relevant databases and merges by score.
func (l *Local) Search(ctx context.Context, req api.SearchRequest) (api.SearchResult, error) {
	if strings.TrimSpace(req.Query) == "" {
		return api.SearchResult{}, apperr.New(apperr.Validation, "query is required")
	}
	if req.Scope != "" && !validScopes[req.Scope] {
		return api.SearchResult{}, apperr.Newf(apperr.Validation, "invalid scope %q", req.Scope)
	}
	if req.Type != "" && !validTypes[req.Type] {
		return api.SearchResult{}, apperr.Newf(apperr.Validation, "invalid type %q", req.Type)
	}
	limit := req.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	targets, err := l.readTargets(req.Scope, req.Project)
	if err != nil {
		return api.SearchResult{}, err
	}
	res := api.SearchResult{Hits: []api.SearchHit{}}
	for _, db := range targets {
		hits, err := store.SearchLexical(ctx, db.R, store.SearchFilter{
			Query: req.Query, Scope: req.Scope, Type: req.Type, MatchMode: req.MatchMode, Limit: limit,
		})
		if err != nil {
			return api.SearchResult{}, err
		}
		res.Hits = append(res.Hits, hits...)
	}
	sort.SliceStable(res.Hits, func(i, j int) bool {
		if res.Hits[i].Score != res.Hits[j].Score {
			return res.Hits[i].Score > res.Hits[j].Score
		}
		return res.Hits[i].UID < res.Hits[j].UID
	})
	if len(res.Hits) > limit {
		res.Hits = res.Hits[:limit]
	}
	return res, nil
}

// Get resolves a uid by probing the project DB first, then global.db.
func (l *Local) Get(ctx context.Context, req api.GetRequest) (api.Observation, error) {
	if req.UID == "" {
		return api.Observation{}, apperr.New(apperr.Validation, "uid is required")
	}
	targets, err := l.readTargets("", req.Project)
	if err != nil {
		return api.Observation{}, err
	}
	for _, db := range targets {
		o, err := store.GetObservationByUID(ctx, db.R, req.UID)
		if err == nil {
			return o, nil
		}
		if apperr.CodeOf(err) != apperr.NotFound {
			return api.Observation{}, err
		}
	}
	return api.Observation{}, apperr.Newf(apperr.NotFound, "observation %q not found", req.UID)
}
```

- [ ] **Step 4: Write `session.go`**

`internal/memory/engine/session.go`:

```go
package engine

import (
	"context"
	"database/sql"
	"strings"

	"mnemonic/internal/memory/api"
	"mnemonic/internal/memory/store"
	"mnemonic/internal/platform/apperr"
)

func (l *Local) SessionStart(ctx context.Context, req api.SessionStartRequest) (api.Session, error) {
	if strings.TrimSpace(req.ID) == "" {
		return api.Session{}, apperr.New(apperr.Validation, "session id is required")
	}
	if req.Project == "" {
		return api.Session{}, apperr.New(apperr.Validation, "project is required")
	}
	db, err := l.dbFor("project", req.Project, true)
	if err != nil {
		return api.Session{}, err
	}
	s := api.Session{ID: req.ID, Project: req.Project, Title: req.Title, Directory: req.Directory, StartedAt: l.now().UTC()}
	if err := db.Write(ctx, func(tx *sql.Tx) error { return store.UpsertSession(ctx, tx, s) }); err != nil {
		return api.Session{}, err
	}
	l.registerProject(ctx, req.Project, s.StartedAt)
	return store.GetSession(ctx, db.R, req.ID)
}

func (l *Local) SessionEnd(ctx context.Context, req api.SessionEndRequest) error {
	if req.ID == "" || req.Project == "" {
		return apperr.New(apperr.Validation, "session id and project are required")
	}
	db, err := l.dbFor("project", req.Project, false)
	if err != nil {
		return err
	}
	if db == nil {
		return apperr.Newf(apperr.NotFound, "session %q not found", req.ID)
	}
	return db.Write(ctx, func(tx *sql.Tx) error {
		ok, err := store.EndSession(ctx, tx, req.ID, req.Summary, l.now().UTC())
		if err != nil {
			return err
		}
		if !ok {
			return apperr.Newf(apperr.NotFound, "session %q not found", req.ID)
		}
		return nil
	})
}
```

- [ ] **Step 5: Run the full engine suite**

Run: `go test ./internal/... -race`
Expected: PASS across `platform/*`, `memory/store`, `memory/engine`.

- [ ] **Step 6: Commit**

```bash
git add -A internal/memory/engine
git commit -m "feat(memory): engine search across project+global, get, sessions"
```

---

### Task 11: CLI `mnemonic memory …` and `main`

**Files:**
- Create: `internal/memory/cli/cli.go`, `cmd/mnemonic/main.go`
- Test: `internal/memory/cli/cli_test.go`

**Interfaces:**
- Consumes: `api.Service`, `engine.NewLocal`, `config.NewPaths`, `project.Resolve`, `apperr`.
- Produces: `cli.Run(args []string, io IO, newSvc func(dataDir string) (api.Service, error)) int` where `IO{Stdin io.Reader; Stdout, Stderr io.Writer}`. Subcommands: `save`, `search`, `get`, `session start|end`. All print JSON to stdout; errors print `CODE: message` to stderr and return `apperr.ExitCode(code)`. Common flags per subcommand: `--data-dir`, `--project`. `--content -` reads stdin.

- [ ] **Step 1: Write the failing test**

`internal/memory/cli/cli_test.go`:

```go
package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"mnemonic/internal/memory/api"
	"mnemonic/internal/memory/engine"
	"mnemonic/internal/platform/config"
)

func run(t *testing.T, dataDir, stdin string, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := Run(args, IO{Stdin: strings.NewReader(stdin), Stdout: &out, Stderr: &errb},
		func(dir string) (api.Service, error) {
			p, err := config.NewPaths(dir)
			if err != nil {
				return nil, err
			}
			return engine.NewLocal(p), nil
		})
	return code, out.String(), errb.String()
}

func TestSaveSearchGetRoundTrip(t *testing.T) {
	dir := t.TempDir()
	code, out, errs := run(t, dir, "", "save", "--data-dir", dir, "--project", "acme",
		"--type", "decision", "--title", "Use WAL", "--content", "concurrent readers", "--topic-key", "arch/wal")
	if code != 0 {
		t.Fatalf("save exit %d: %s", code, errs)
	}
	var saved api.SaveResult
	if err := json.Unmarshal([]byte(out), &saved); err != nil || saved.UID == "" || saved.Verdict != "create" {
		t.Fatalf("save output: %q %v", out, err)
	}

	code, out, errs = run(t, dir, "", "search", "--data-dir", dir, "--project", "acme", "readers")
	if code != 0 {
		t.Fatalf("search exit %d: %s", code, errs)
	}
	var found api.SearchResult
	if err := json.Unmarshal([]byte(out), &found); err != nil || len(found.Hits) != 1 || found.Hits[0].UID != saved.UID {
		t.Fatalf("search output: %q %v", out, err)
	}

	code, out, _ = run(t, dir, "", "get", "--data-dir", dir, "--project", "acme", saved.UID)
	var obs api.Observation
	if code != 0 || json.Unmarshal([]byte(out), &obs) != nil || obs.Title != "Use WAL" {
		t.Fatalf("get: %d %q", code, out)
	}
}

func TestContentFromStdin(t *testing.T) {
	dir := t.TempDir()
	code, out, errs := run(t, dir, "piped body", "save", "--data-dir", dir, "--project", "acme",
		"--type", "learning", "--title", "From stdin", "--content", "-")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errs)
	}
	var saved api.SaveResult
	_ = json.Unmarshal([]byte(out), &saved)
	_, out, _ = run(t, dir, "", "get", "--data-dir", dir, "--project", "acme", saved.UID)
	if !strings.Contains(out, "piped body") {
		t.Fatalf("stdin content lost: %s", out)
	}
}

func TestErrorsMapToExitCodes(t *testing.T) {
	dir := t.TempDir()
	code, _, errs := run(t, dir, "", "get", "--data-dir", dir, "--project", "acme", "missing-uid")
	if code != 4 || !strings.HasPrefix(errs, "NOT_FOUND:") {
		t.Fatalf("not found: %d %q", code, errs)
	}
	code, _, errs = run(t, dir, "", "save", "--data-dir", dir, "--project", "acme", "--type", "bogus", "--title", "t", "--content", "c")
	if code != 2 || !strings.HasPrefix(errs, "VALIDATION_ERROR:") {
		t.Fatalf("validation: %d %q", code, errs)
	}
	code, _, _ = run(t, dir, "", "nonsense")
	if code != 2 {
		t.Fatalf("unknown command should exit 2, got %d", code)
	}
}

func TestSessionStartEnd(t *testing.T) {
	dir := t.TempDir()
	if code, _, errs := run(t, dir, "", "session", "start", "--data-dir", dir, "--project", "acme", "--id", "s1", "--title", "T"); code != 0 {
		t.Fatalf("start: %d %s", code, errs)
	}
	if code, _, errs := run(t, dir, "", "session", "end", "--data-dir", dir, "--project", "acme", "--id", "s1", "--summary", "ok"); code != 0 {
		t.Fatalf("end: %d %s", code, errs)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/memory/cli/ -v`
Expected: FAIL (`undefined: Run`).

- [ ] **Step 3: Write `cli.go`**

`internal/memory/cli/cli.go`:

```go
// Package cli implements `mnemonic memory <subcommand>`.
package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"mnemonic/internal/memory/api"
	"mnemonic/internal/platform/apperr"
	"mnemonic/internal/platform/project"
)

type IO struct {
	Stdin          io.Reader
	Stdout, Stderr io.Writer
}

type common struct {
	dataDir, project string
}

func (c *common) bind(fs *flag.FlagSet) {
	fs.StringVar(&c.dataDir, "data-dir", "", "data directory (default $MNEMONIC_DATA_DIR or ~/.mnemonic)")
	fs.StringVar(&c.project, "project", "", "project slug (default: auto-detect)")
}

func (c *common) resolveProject() (string, error) {
	cwd, _ := os.Getwd()
	return project.Resolve(c.project, cwd)
}

// Run executes the memory CLI and returns the process exit code.
func Run(args []string, streams IO, newSvc func(dataDir string) (api.Service, error)) int {
	if len(args) == 0 {
		fmt.Fprintln(streams.Stderr, "usage: mnemonic memory <save|search|get|session> [flags]")
		return 2
	}
	var run func([]string, IO, func(string) (api.Service, error)) error
	switch args[0] {
	case "save":
		run = runSave
	case "search":
		run = runSearch
	case "get":
		run = runGet
	case "session":
		run = runSession
	default:
		fmt.Fprintf(streams.Stderr, "unknown memory command %q\n", args[0])
		return 2
	}
	if err := run(args[1:], streams, newSvc); err != nil {
		code := apperr.CodeOf(err)
		msg := err.Error()
		if !strings.HasPrefix(msg, string(code)+":") {
			msg = string(code) + ": " + msg
		}
		fmt.Fprintln(streams.Stderr, msg)
		return apperr.ExitCode(code)
	}
	return 0
}

func emit(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func parse(fs *flag.FlagSet, args []string, streams IO) error {
	fs.SetOutput(streams.Stderr)
	if err := fs.Parse(args); err != nil {
		return apperr.New(apperr.Validation, err.Error())
	}
	return nil
}

func runSave(args []string, streams IO, newSvc func(string) (api.Service, error)) error {
	var c common
	var req api.SaveRequest
	fs := flag.NewFlagSet("save", flag.ContinueOnError)
	c.bind(fs)
	fs.StringVar(&req.Type, "type", "", "observation type")
	fs.StringVar(&req.Title, "title", "", "short searchable title")
	fs.StringVar(&req.Content, "content", "", "body text, or - to read stdin")
	fs.StringVar(&req.TopicKey, "topic-key", "", "stable key for upserts")
	fs.StringVar(&req.Scope, "scope", "", "project (default) | user | global")
	fs.StringVar(&req.SessionID, "session", "", "session id")
	fs.StringVar(&req.OperationID, "op-id", "", "idempotency key")
	if err := parse(fs, args, streams); err != nil {
		return err
	}
	if req.Content == "-" {
		b, err := readAll(streams.Stdin)
		if err != nil {
			return err
		}
		req.Content = b
	}
	if req.Scope == "" || req.Scope == "project" {
		p, err := c.resolveProject()
		if err != nil {
			return err
		}
		req.Project = p
	}
	svc, err := newSvc(c.dataDir)
	if err != nil {
		return err
	}
	defer svc.Close()
	res, err := svc.Save(context.Background(), req)
	if err != nil {
		return err
	}
	return emit(streams.Stdout, res)
}

func runSearch(args []string, streams IO, newSvc func(string) (api.Service, error)) error {
	var c common
	var req api.SearchRequest
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	c.bind(fs)
	fs.StringVar(&req.Scope, "scope", "", "project | user | global")
	fs.StringVar(&req.Type, "type", "", "observation type filter")
	fs.StringVar(&req.MatchMode, "match", "any", "any | all")
	fs.IntVar(&req.Limit, "limit", 10, "max results (<=50)")
	if err := parse(fs, args, streams); err != nil {
		return err
	}
	req.Query = strings.Join(fs.Args(), " ")
	p, err := c.resolveProject()
	if err != nil {
		return err
	}
	req.Project = p
	svc, err := newSvc(c.dataDir)
	if err != nil {
		return err
	}
	defer svc.Close()
	res, err := svc.Search(context.Background(), req)
	if err != nil {
		return err
	}
	return emit(streams.Stdout, res)
}

func runGet(args []string, streams IO, newSvc func(string) (api.Service, error)) error {
	var c common
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	c.bind(fs)
	if err := parse(fs, args, streams); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return apperr.New(apperr.Validation, "usage: mnemonic memory get <uid>")
	}
	p, err := c.resolveProject()
	if err != nil {
		return err
	}
	svc, err := newSvc(c.dataDir)
	if err != nil {
		return err
	}
	defer svc.Close()
	o, err := svc.Get(context.Background(), api.GetRequest{Project: p, UID: fs.Arg(0)})
	if err != nil {
		return err
	}
	return emit(streams.Stdout, o)
}

func runSession(args []string, streams IO, newSvc func(string) (api.Service, error)) error {
	if len(args) == 0 || (args[0] != "start" && args[0] != "end") {
		return apperr.New(apperr.Validation, "usage: mnemonic memory session <start|end> --id <id>")
	}
	var c common
	var id, title, dir, summary string
	fs := flag.NewFlagSet("session "+args[0], flag.ContinueOnError)
	c.bind(fs)
	fs.StringVar(&id, "id", "", "session id")
	fs.StringVar(&title, "title", "", "session title (start)")
	fs.StringVar(&dir, "directory", "", "workspace directory (start)")
	fs.StringVar(&summary, "summary", "", "summary text (end)")
	if err := parse(fs, args[1:], streams); err != nil {
		return err
	}
	p, err := c.resolveProject()
	if err != nil {
		return err
	}
	svc, err := newSvc(c.dataDir)
	if err != nil {
		return err
	}
	defer svc.Close()
	ctx := context.Background()
	if args[0] == "start" {
		s, err := svc.SessionStart(ctx, api.SessionStartRequest{ID: id, Project: p, Title: title, Directory: dir})
		if err != nil {
			return err
		}
		return emit(streams.Stdout, s)
	}
	if err := svc.SessionEnd(ctx, api.SessionEndRequest{ID: id, Project: p, Summary: summary}); err != nil {
		return err
	}
	return emit(streams.Stdout, map[string]any{"ended": id})
}

func readAll(r io.Reader) (string, error) {
	b, err := io.ReadAll(r)
	return string(b), err
}
```


- [ ] **Step 4: Write `main.go`**

`cmd/mnemonic/main.go`:

```go
package main

import (
	"fmt"
	"os"

	"mnemonic/internal/memory/api"
	"mnemonic/internal/memory/cli"
	"mnemonic/internal/memory/engine"
	"mnemonic/internal/platform/config"
)

const version = "0.1.0-dev"

func newService(dataDir string) (api.Service, error) {
	paths, err := config.NewPaths(dataDir)
	if err != nil {
		return nil, err
	}
	return engine.NewLocal(paths), nil
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: mnemonic <memory|version> ...")
		os.Exit(2)
	}
	switch args[0] {
	case "memory":
		os.Exit(cli.Run(args[1:], cli.IO{Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr}, newService))
	case "version":
		fmt.Println(version)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", args[0])
		os.Exit(2)
	}
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/memory/cli/ -race -v`
Expected: PASS (4 tests).

- [ ] **Step 6: Commit**

```bash
git add cmd internal/memory/cli
git commit -m "feat(memory): mnemonic memory save/search/get/session CLI"
```

---

### Task 12: End-to-end verification and handoff

**Files:**
- Modify: none (verification only; fix forward in the owning task's files if anything fails)

- [ ] **Step 1: Full static + unit verification**

Run:

```bash
gofmt -l . && go vet ./... && CGO_ENABLED=0 go build -o /tmp/mnemonic ./cmd/mnemonic && go test ./... -race -count=1
```

Expected: `gofmt -l` prints nothing; vet clean; build succeeds; all packages `ok`.

- [ ] **Step 2: Real-binary smoke test against a throwaway data dir**

Run:

```bash
export MNEMONIC_DATA_DIR=$(mktemp -d)
/tmp/mnemonic memory session start --project demo --id s1 --title "smoke"
/tmp/mnemonic memory save --project demo --session s1 --type decision --title "Use WAL" --content "token <private>abc</private> and concurrent readers" --topic-key arch/wal
/tmp/mnemonic memory save --scope user --type preference --title "Prefer tabs" --content "tabs readers"
/tmp/mnemonic memory search --project demo readers
/tmp/mnemonic memory search --project demo db
ls -la "$MNEMONIC_DATA_DIR"
```

Expected: first save returns `"verdict": "create"`; search "readers" returns 2 hits (project + user scope); search "db" returns `[]` hits without error (LIKE path); the data dir contains `demo.db` and `global.db`; no secret `abc` appears in any `get` output.

- [ ] **Step 3: Confirm secrets are not on disk**

Run: `strings "$MNEMONIC_DATA_DIR"/demo.db | rg -c "abc" || echo "clean"`
Expected: `clean`.

- [ ] **Step 4: Persist the milestone to Mnemonic memory and commit the plan status**

Use `mem_save` (`type: decision`, `topic_key: architecture/plan-1-complete`) summarizing: Plan 1 shipped, spec amendments (standalone FTS, no cross-DB session FK, no PersistWAL), and that Plans 2–5 remain.

```bash
git add -A
git commit --allow-empty -m "chore: Plan 1 verified (foundation + memory core)"
```

---

## Self-Review (spec coverage for Plan 1)

| Spec requirement | Task |
|---|---|
| 2.1 `platform` + `memory` package split | 1–11 |
| 3.1 data dir layout, `MNEMONIC_DATA_DIR` | 2 |
| 3.2 slug resolution (explicit/env/git/basename) + reserved names | 3 (aliases and `.mnemonic.yaml` → Plan 5) |
| 3.3 scope routing, uid probing across DBs | 9, 10 (`all_projects` fan-out → Plan 2) |
| 3.4 writer/reader split, PRAGMAs, NFS refusal, inode fence | 4 (LRU idle eviction → Plan 4 with the daemon) |
| 3.5 platform + memory DDL, FTS triggers | 5, 6 (vector tables → Plan 2) |
| 4.2 redact, 32 KiB cap, idempotency 200/409/410, topic_key revise + versions, 24h hash dup, `review_after`, trails | 9 (LLM dedup, vector conflict candidates → Plan 2) |
| 4.3 lexical leg, `<3`-rune LIKE fallback, any/all | 8, 10 (RRF/vector/composite scoring → Plan 2) |
| 5.3 CLI `save/search/get/session` | 11 (remaining CLI verbs follow their features) |
| 6 error codes + exit codes | 1, 11 |
| 7.1 SQLite concurrency/fence/FTS tests | 4, 6, 8 |

**Placeholder scan:** none; every step lists real code or an exact command.

**Type consistency:** `api.*` types defined in Task 7 are used verbatim in Tasks 9–11; `store.Querier` is satisfied by `*sql.Tx`/`*sql.DB` (used with `d.W`, `d.R`, `tx`); `dbFor(scope, project string, create bool)` is used identically in Tasks 9 and 10; `Local.registerProject` is defined in Task 9 and reused in Task 10.

**Known deliberate gaps (tracked in later plans):** conflict candidates / `judgment_required`, `retrieval_trails` writes, `retrieval_usage` updates, `mem_context`/`timeline`/`lookup`/`pin`/prompts/review/delete, vectors and ONNX, MCP, daemon, TUI, `doctor`.
