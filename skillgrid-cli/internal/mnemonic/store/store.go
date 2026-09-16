// Package store wraps a project-scoped SQLite database with embedded SQL migrations.
package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Store is the shared SQLite handle for one project.
type Store struct {
	DB   *sql.DB
	path string
	// cacheKey is the pool key (store path) when this Store wraps a pooled
	// handle, or "" when it owns its database outright (cache disabled).
	cacheKey string
}

// cachedEntry is a pooled *sql.DB shared by every live Store for the same
// store path. The pool eliminates N+1 store opens: repeated Open calls for
// the same dataDir+projectID return the same underlying database handle.
type cachedEntry struct {
	db   *sql.DB
	refs atomic.Int64
}

// handleCache is the process-global store handle pool. Keyed by store path
// (dataDir + projectID). SKILLGRID_MNEMONIC_DISABLE_CACHE=1 bypasses it
// (rollback boundary — every Open creates a fresh connection).
var (
	handleCache sync.Map
	cacheMu     sync.Mutex // serializes evict-and-close against live refs
)

const (
	envCacheDisable = "SKILLGRID_MNEMONIC_DISABLE_CACHE"
	maxOpenAttempts = 3
)

// walBackoffs are the retry delays after a WAL-locked open attempt.
var walBackoffs = []time.Duration{50 * time.Millisecond, 100 * time.Millisecond, 200 * time.Millisecond}

func cacheDisabled() bool {
	return strings.EqualFold(os.Getenv(envCacheDisable), "1")
}

// openCount records every call to Open (successful or not). It is a process-
// global diagnostic that is never read on the production path — the mcp tests
// use it to prove a single tool call opens its project store exactly once
// (no double-open). It is safe under concurrent use.
var (
	openCountMu sync.Mutex
	openCount   int
)

// OpenCount returns the total number of times Open has been called in this
// process. Test-only helper.
func OpenCount() int {
	openCountMu.Lock()
	defer openCountMu.Unlock()
	return openCount
}

// RecordOpen increments the process-global open counter.
func RecordOpen() {
	openCountMu.Lock()
	openCount++
	openCountMu.Unlock()
}

// ResetOpenCount clears the counter. Test-only helper.
func ResetOpenCount() {
	openCountMu.Lock()
	openCount = 0
	openCountMu.Unlock()
}

// Open opens or creates the SQLite database for projectID under dataDir.
// Open is refcounted: a second Open for the same store path returns the
// cached handle (same underlying *sql.DB) instead of re-opening the
// database. Close releases one reference; the database is only closed once
// every reference has been returned. Set SKILLGRID_MNEMONIC_DISABLE_CACHE=1
// to bypass the cache (every Open creates a fresh connection).
func Open(dataDir, projectID string) (*Store, error) {
	RecordOpen()
	if strings.TrimSpace(projectID) == "" {
		return nil, fmt.Errorf("project id is required")
	}
	if strings.Contains(projectID, "..") {
		return nil, fmt.Errorf("invalid project id %q", projectID)
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	dbPath := filepath.Join(dataDir, projectID+".sqlite")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	if !cacheDisabled() {
		if s, ok := acquireCached(dbPath); ok {
			return s, nil
		}
	}
	db, err := openWithWALRetry(dbPath)
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	if cacheDisabled() {
		return &Store{DB: db, path: dbPath}, nil
	}
	return cacheNewHandle(db, dbPath)
}

// openWithWALRetry opens the SQLite database, retrying with exponential
// backoff (50ms, 100ms, 200ms) when a concurrent writer holds the WAL lock.
// It fails after maxOpenAttempts (3) total attempts.
func openWithWALRetry(dbPath string) (*sql.DB, error) {
	var lastErr error
	for attempt := 0; attempt < maxOpenAttempts; attempt++ {
		db, err := openDatabase(dbPath)
		if err == nil {
			return db, nil
		}
		lastErr = err
		if !isWALBusy(err) || attempt+1 >= maxOpenAttempts {
			break
		}
		time.Sleep(walBackoffs[attempt])
	}
	return nil, lastErr
}

// isWALBusy reports whether err is a transient SQLite busy error
// (SQLITE_BUSY, code 5) caused by a concurrent writer holding the WAL lock.
// It prefers the driver's typed error (*sqlite.Error with
// Code() == sqlite3.SQLITE_BUSY); the text fallback matches the driver's
// rendered busy messages, e.g. "database is locked (5) (SQLITE_BUSY)"
// (verified against modernc.org/sqlite v1.45.0, error.go / conn.go).
func isWALBusy(err error) bool {
	if err == nil {
		return false
	}
	var typed *sqlite.Error
	if errors.As(err, &typed) {
		return typed.Code() == sqlite3.SQLITE_BUSY
	}
	msg := err.Error()
	return strings.Contains(msg, "SQLITE_BUSY") ||
		strings.Contains(msg, "The database file is locked") ||
		strings.Contains(msg, "database is locked") ||
		strings.Contains(msg, "database table is locked")
}

func openDatabase(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(1)
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		// 10s busy timeout: the session-close distill hook (change 013 step 02)
		// reopens the store in a detached goroutine a beat after the handler's
		// store closes; a longer timeout absorbs that transient WAL lock.
		"PRAGMA busy_timeout=10000",
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("apply %s: %w", pragma, err)
		}
	}
	return db, nil
}

// acquireCached returns the pooled handle for dbPath, or a fresh one. A
// ping before return is the health-check on reuse (change 014): a dead or
// closed pooled handle is evicted and replaced instead of handed out.
func acquireCached(dbPath string) (*Store, bool) {
	v, ok := handleCache.Load(dbPath)
	if !ok {
		return nil, false
	}
	entry := v.(*cachedEntry)
	entry.refs.Add(1)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	err := entry.db.PingContext(ctx)
	cancel()
	if err == nil {
		return &Store{DB: entry.db, path: dbPath, cacheKey: dbPath}, true
	}
	// Unhealthy pooled handle: drop the pool reference; evict only if this
	// was the last one holding the database.
	if remaining := entry.refs.Add(-1); remaining <= 0 {
		evictCached(dbPath, entry)
	}
	return nil, false
}

// cacheNewHandle registers a freshly opened database in the pool and returns
// the Store wrapping it.
func cacheNewHandle(db *sql.DB, dbPath string) (*Store, error) {
	entry := &cachedEntry{db: db}
	entry.refs.Store(1)
	actual, loaded := handleCache.LoadOrStore(dbPath, entry)
	e := actual.(*cachedEntry)
	if loaded {
		// Lost the race: release the db we just opened and reuse the winner.
		db.Close()
		e.refs.Add(1)
	}
	return &Store{DB: e.db, path: dbPath, cacheKey: dbPath}, nil
}

// evictCached closes the pooled database and removes dbPath from the pool.
// The mutex serializes the load-compare-swap so a database that still has
// live references is never closed underneath them.
func evictCached(dbPath string, entry *cachedEntry) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if v, ok := handleCache.Load(dbPath); ok && v == entry {
		handleCache.Delete(dbPath)
	} else {
		return
	}
	entry.db.Close()
}

// Path returns the on-disk path of the SQLite file.
func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Close releases one reference to the database handle. Pooled handles are
// only closed once every reference has been returned; a handle that owns
// its database outright (cache disabled) is closed immediately.
func (s *Store) Close() error {
	if s == nil || s.DB == nil {
		return nil
	}
	if s.cacheKey != "" {
		return s.releasePooled()
	}
	return s.DB.Close()
}

// releasePooled decrements the pool reference count for s.cacheKey. When the
// count reaches zero the pooled database is closed and evicted from the
// cache.
func (s *Store) releasePooled() error {
	v, ok := handleCache.Load(s.cacheKey)
	if !ok {
		// Already evicted (concurrent final Close); nothing to release.
		return nil
	}
	entry := v.(*cachedEntry)
	if entry.db != s.DB {
		// The pool entry was rebound to a new *sql.DB (RebindPassDB) after this
		// Store captured the old one. This Store is the owner whose pass run
		// caused the rebind, and its close already closed the old DB inside
		// RebindPassDB — decrementing here would leak the passDB entry's refs.
		return nil
	}
	if remaining := entry.refs.Add(-1); remaining <= 0 {
		evictCached(s.cacheKey, entry)
		return nil
	}
	return nil
}

// RebindPassDB re-registers the store's pooled handle with a freshly opened
// *sql.DB, swapping it in atomically and closing the previous pooled DB.
//
// The codeindex passes (community/process/knowledge) must run on a fresh
// single-connection *sql.DB: the store's pool is MaxOpenConns=1, so a second
// connection deadlocks once the committed 005 tx holds the write lock.
// Indexer.Run opens that fresh passDB after the commit, but if it leaves the
// pool entry pointing at the (now closed) old *sql.DB, every other live
// handle for the same store — which cached the OLD *sql.DB at Open time —
// keeps using the closed pool handle and fails with "sql: database is closed".
// RebindPassDB fixes that: it atomically swaps the pool's DB pointer to passDB
// (inheriting the old entry's live reference count, so no references are lost
// or leaked) and closes the old DB only if it is distinct from passDB. It is a
// no-op for a Store that owns its database outright (cacheKey == "").
func (s *Store) RebindPassDB(passDB *sql.DB) error {
	if s == nil || s.DB == nil || s.cacheKey == "" || passDB == nil {
		return nil
	}
	old := s.DB
	if old == passDB {
		return nil
	}
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if v, ok := handleCache.Load(s.cacheKey); ok {
		entry := v.(*cachedEntry)
		if entry.db == old {
			// Still the registered entry: swap in passDB, keep the live ref
			// count, and close the old DB after the swap (deferred so the
			// lock is released first — entry.db.Close is pool-independent).
			entry.db = passDB
			if entry.refs.Load() <= 0 {
				handleCache.Delete(s.cacheKey)
			}
		}
	}
	return old.Close()
}

// Migrate applies every embedded migration to db (idempotent, tracked in
// index_meta). Exported for test fixtures that need the real schema on a
// raw *sql.DB with a multi-connection pool (see graph package tests).
func Migrate(db *sql.DB) error {
	return migrate(db)
}

func migrate(db *sql.DB) error {
	// Ensure the migration bookkeeping table exists. Older databases that
	// predate this table get it here; the schema_version row is the source
	// of truth for how many migrations have been applied, and the
	// migration:<name> rows track individual application so non-idempotent
	// statements (e.g. ADD COLUMN) run exactly once.
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS index_meta (
			key TEXT PRIMARY KEY,
			schema_version INTEGER NOT NULL
		)
	`); err != nil {
		return fmt.Errorf("create index_meta: %w", err)
	}
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	var appliedCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM index_meta WHERE key LIKE 'migration:%'`).Scan(&appliedCount); err != nil {
		return fmt.Errorf("count applied: %w", err)
	}
	for _, name := range names {
		var done bool
		if err := db.QueryRow(`SELECT COUNT(*) > 0 FROM index_meta WHERE key = ?`, "migration:"+name).Scan(&done); err != nil {
			return fmt.Errorf("check %s: %w", name, err)
		}
		if done {
			continue
		}
		sqlBytes, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		if _, err := db.Exec(string(sqlBytes)); err != nil {
			return fmt.Errorf("exec %s: %w", name, err)
		}
		if _, err := db.Exec(`INSERT INTO index_meta (key, schema_version) VALUES (?, 1)`, "migration:"+name); err != nil {
			return fmt.Errorf("record %s: %w", name, err)
		}
		appliedCount++
	}
	// Keep the legacy schema_version marker in sync with the total applied.
	if _, err := db.Exec(`
		INSERT INTO index_meta (key, schema_version) VALUES ('schema_version', ?)
		ON CONFLICT(key) DO UPDATE SET schema_version = excluded.schema_version`,
		appliedCount,
	); err != nil {
		return fmt.Errorf("record schema_version: %w", err)
	}
	return nil
}
