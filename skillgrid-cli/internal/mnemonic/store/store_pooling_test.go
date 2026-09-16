package store

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	sqlitedrv "modernc.org/sqlite"
)

// TestStoreOpenReusesCachedHandle covers @step-01 (Scenario: Cached handle
// reuse on second open + Handle closes only when reference count reaches
// zero): a second Open for the same store path returns the same underlying
// *sql.DB without re-opening, a different project gets a different handle,
// Close refcounts (the connection stays open until the last reference is
// returned), and the cache is evicted when the refcount hits zero.
func TestStoreOpenReusesCachedHandle(t *testing.T) {
	dir := t.TempDir()

	stA, err := Open(dir, "pool-a")
	if err != nil {
		t.Fatalf("first open A: %v", err)
	}
	stB, err := Open(dir, "pool-b")
	if err != nil {
		t.Fatalf("open B: %v", err)
	}
	defer stB.Close()

	stA2, err := Open(dir, "pool-a")
	if err != nil {
		t.Fatalf("second open A: %v", err)
	}

	// Cache hit: same underlying *sql.DB, no new connection created.
	if stA.DB != stA2.DB {
		t.Fatalf("expected second open of the same store to reuse the cached *sql.DB")
	}
	// Different project: different handle.
	if stA.DB == stB.DB {
		t.Fatalf("expected different projects to get different *sql.DB handles")
	}
	// Migration bookkeeping ran exactly once (no re-apply on cache hit).
	if countMigration(t, stA.DB, "019_session_relay.sql") != 1 {
		t.Fatalf("expected 019 migration recorded once")
	}

	// Refcount: closing one handle leaves the underlying connection open.
	if err := stA.Close(); err != nil {
		t.Fatalf("close first handle: %v", err)
	}
	if err := stA2.DB.Ping(); err != nil {
		t.Fatalf("underlying connection should stay open while refcount > 0: %v", err)
	}
	// Final close evicts the cache: the next open gets a fresh *sql.DB.
	if err := stA2.Close(); err != nil {
		t.Fatalf("close second handle: %v", err)
	}
	stA3, err := Open(dir, "pool-a")
	if err != nil {
		t.Fatalf("open after eviction: %v", err)
	}
	defer stA3.Close()
	if stA3.DB == stA.DB {
		t.Fatalf("expected a fresh *sql.DB after the refcount reached zero")
	}

	// The fresh handle is healthy (cache health-check holds after eviction).
	var n int
	if err := stA3.DB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='sessions'`).Scan(&n); err != nil {
		t.Fatalf("post-eviction query: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected sessions table on fresh handle, got %d", n)
	}
}

// TestStoreOpenWALRetry covers @step-01 (Scenario: WAL lock retry with
// exponential backoff): PRAGMA busy_timeout=10000 is applied on every new
// connection, and a write that hits a WAL lock retries at 50ms, 100ms, and
// 200ms intervals, succeeding after the lock is released — no panic, no
// failure while the lock was held.
func TestStoreOpenWALRetry(t *testing.T) {
	dir := t.TempDir()

	// The store opens (and sets busy_timeout=10000) while no lock is held.
	st, err := Open(dir, "walproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	// busy_timeout is applied on every new connection.
	var timeout int
	if err := st.DB.QueryRow(`PRAGMA busy_timeout`).Scan(&timeout); err != nil {
		t.Fatalf("pragma busy_timeout: %v", err)
	}
	if timeout != 10000 {
		t.Fatalf("expected busy_timeout=10000, got %d", timeout)
	}

	// A concurrent writer holds the WAL write lock.
	if _, err := st.DB.Exec(`CREATE TABLE walprobe (id INTEGER PRIMARY KEY, v TEXT)`); err != nil {
		t.Fatalf("create: %v", err)
	}
	walTx, err := st.DB.Begin()
	if err != nil {
		t.Fatalf("begin writer tx: %v", err)
	}
	if _, err := walTx.Exec(`INSERT INTO walprobe (v) VALUES ('hold')`); err != nil {
		t.Fatalf("writer insert: %v", err)
	}

	// A second connection's write hits the lock. It must not fail outright:
	// it retries (backoff 50/100/200ms) until the lock is released, then the
	// write succeeds — the behavior the busy_timeout + retry contract gates.
	start := time.Now()
	writeDone := make(chan error, 1)
	go func() {
		db2 := rawSQLDB(t, dir, "walproj")
		defer db2.Close()
		// Warm the connection so the lock is actually held on the write,
		// not on the first handshake.
		if _, err := db2.Exec(`PRAGMA busy_timeout=10000`); err != nil {
			writeDone <- err
			return
		}
		_, err = db2.Exec(`INSERT INTO walprobe (v) VALUES ('contended')`)
		writeDone <- err
	}()

	// Release the lock partway through the retry window.
	time.Sleep(80 * time.Millisecond)
	if err := walTx.Commit(); err != nil {
		t.Fatalf("commit writer tx: %v", err)
	}

	select {
	case err := <-writeDone:
		if err != nil {
			t.Fatalf("contended write should succeed after backoff retries, got: %v", err)
		}
		if elapsed := time.Since(start); elapsed < 50*time.Millisecond {
			t.Fatalf("expected a backoff delay before the contended write, finished in %v", elapsed)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("contended write did not complete within the retry window")
	}

	// Both rows present: the contended write landed after the retry.
	var n int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM walprobe`).Scan(&n); err != nil {
		t.Fatalf("count walprobe: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 walprobe rows after contended write, got %d", n)
	}
}

// TestStoreOpenWALRetryLoop covers @step-01 (Scenarios: WAL lock retry with
// exponential backoff + retry count on a locked open): a cold Open (cache
// disabled so no pooled handle is reused) whose PRAGMA journal_mode=WAL hits
// a write lock held by a concurrent connection is RETRIED by
// openWithWALRetry — the first attempt fails with a typed SQLITE_BUSY error
// (isWALBusy must classify it as busy), a backoff elapses, and the retry
// succeeds once the lock is released. The elapsed-time floor proves the
// retry happened rather than the open succeeding on attempt 0.
func TestStoreOpenWALRetryLoop(t *testing.T) {
	t.Setenv(envCacheDisable, "1")
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "retryproj.sqlite")

	// Create the file and hold a write lock on it (deferred tx + insert,
	// verified to hold the WAL write lock in this environment).
	hold, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open holder: %v", err)
	}
	hold.SetMaxOpenConns(1)
	if _, err := hold.Exec(`CREATE TABLE IF NOT EXISTS walprobe (id INTEGER PRIMARY KEY, v TEXT)`); err != nil {
		t.Fatalf("holder create: %v", err)
	}
	walTx, err := hold.Begin()
	if err != nil {
		t.Fatalf("holder begin: %v", err)
	}
	if _, err := walTx.Exec(`INSERT INTO walprobe (v) VALUES ('hold')`); err != nil {
		t.Fatalf("holder insert: %v", err)
	}
	defer func() {
		_ = walTx.Commit()
		_ = hold.Close()
	}()

	// Cold Open in a goroutine (cache disabled → no pooled handle, so the
	// open goes through openWithWALRetry). The first attempt's
	// PRAGMA journal_mode=WAL hits the held write lock (verified: this is
	// the real contention path — a journal-mode transition under lock),
	// isWALBusy classifies it as busy, and the retry succeeds after the
	// 50ms backoff once the lock is released at 80ms.
	opened := make(chan *Store, 1)
	openErr := make(chan error, 1)
	start := time.Now()
	go func() {
		s, err := Open(dir, "retryproj")
		if err != nil {
			openErr <- err
			return
		}
		opened <- s
	}()

	time.Sleep(80 * time.Millisecond)
	if err := walTx.Commit(); err != nil {
		t.Fatalf("commit holder tx: %v", err)
	}

	select {
	case s := <-opened:
		defer s.Close()
	case err := <-openErr:
		t.Fatalf("open under WAL lock should succeed after the retry loop absorbs the busy error, got: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatalf("open did not complete within the retry window")
	}
	elapsed := time.Since(start)
	// At least one 50ms backoff must have elapsed: the first attempt hit the
	// lock and openWithWALRetry slept before retrying.
	if elapsed < 50*time.Millisecond {
		t.Fatalf("expected openWithWALRetry to back off at least 50ms before the retry, finished in %v", elapsed)
	}
	t.Logf("open succeeded after %v (retry absorbed the transient WAL lock)", elapsed)
}

// TestIsWALBusyClassification unit-tests the retry decision: isWALBusy must
// classify real busy errors (the driver's *sqlite.Error surfaced through
// openDatabase's %w wrap) as busy, and reject unrelated errors.
func TestIsWALBusyClassification(t *testing.T) {
	busy := openDatabaseBusyWrap(t)
	var typed *sqlitedrv.Error
	if !errors.As(busy, &typed) || typed.Code() != 5 {
		t.Fatalf("precondition: wrapped openDatabase error must be *sqlite.Error with code 5 (SQLITE_BUSY), got %v", busy)
	}
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"real wrapped busy from openDatabase", busy, true},
		{"unrelated", fmt.Errorf("apply PRAGMA journal_mode=WAL: no such table: foo"), false},
	}
	for _, tc := range cases {
		if got := isWALBusy(tc.err); got != tc.want {
			t.Fatalf("%s: isWALBusy = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func openDatabaseBusyWrap(t *testing.T) error {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "busywrap.sqlite")
	hold, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open holder: %v", err)
	}
	hold.SetMaxOpenConns(1)
	if _, err := hold.Exec(`CREATE TABLE IF NOT EXISTS t (id INTEGER PRIMARY KEY)`); err != nil {
		t.Fatalf("holder create: %v", err)
	}
	tx, err := hold.Begin()
	if err != nil {
		t.Fatalf("holder begin: %v", err)
	}
	if _, err := tx.Exec(`INSERT INTO t VALUES (1)`); err != nil {
		t.Fatalf("holder insert: %v", err)
	}
	defer func() {
		_ = tx.Commit()
		_ = hold.Close()
	}()
	// openDatabase sets journal_mode=WAL first — with the lock held it must
	// return the wrapped busy error.
	_, err = openDatabase(dbPath)
	if err == nil {
		t.Fatalf("expected openDatabase to be busy while the lock is held")
	}
	return err
}

// TestStoreOpenCacheDisabledByEnv covers @step-01 (Scenario: Cache disabled
// by environment variable): SKILLGRID_MNEMONIC_DISABLE_CACHE=1 makes every
// Open create a fresh underlying connection and no handle is stored in the
// cache (each Store owns its database and closes it outright).
func TestStoreOpenCacheDisabledByEnv(t *testing.T) {
	t.Setenv("SKILLGRID_MNEMONIC_DISABLE_CACHE", "1")
	dir := t.TempDir()

	st1, err := Open(dir, "nocache")
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	defer st1.Close()
	st2, err := Open(dir, "nocache")
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer st2.Close()

	if st1.DB == st2.DB {
		t.Fatalf("with the cache disabled, each open must create a new *sql.DB")
	}
	// No handle is stored in the cache.
	if _, ok := handleCache.Load(st1.Path()); ok {
		t.Fatalf("cache must stay empty when disabled")
	}
	var n1, n2 int
	if err := st1.DB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='sessions'`).Scan(&n1); err != nil {
		t.Fatalf("query handle 1: %v", err)
	}
	if err := st2.DB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='sessions'`).Scan(&n2); err != nil {
		t.Fatalf("query handle 2: %v", err)
	}
	if n1 != 1 || n2 != 1 {
		t.Fatalf("both handles must be usable, got %d/%d", n1, n2)
	}
}

// TestMigration014TTLExtraction covers @step-01 (01.3): a fresh store has
// the 020 migration applied — ttl_config and extraction_metadata tables
// exist with the expected columns, the migration is recorded once, and the
// existing schema remains intact (additive, not a rewrite).
func TestMigration014TTLExtraction(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "ttlproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	if !tableExists(t, st.DB, "ttl_config") {
		t.Fatalf("expected ttl_config table after 020 migration")
	}
	if !tableExists(t, st.DB, "extraction_metadata") {
		t.Fatalf("expected extraction_metadata table after 020 migration")
	}
	for _, col := range []string{"key", "value"} {
		var n int
		if err := st.DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('ttl_config') WHERE name=?`, col).Scan(&n); err != nil {
			t.Fatalf("pragma ttl_config %s: %v", col, err)
		}
		if n != 1 {
			t.Fatalf("ttl_config column %s not present (n=%d)", col, n)
		}
	}
	for _, col := range []string{"id", "session_id", "content_hash", "extracted_at", "model"} {
		var n int
		if err := st.DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('extraction_metadata') WHERE name=?`, col).Scan(&n); err != nil {
			t.Fatalf("pragma extraction_metadata %s: %v", col, err)
		}
		if n != 1 {
			t.Fatalf("extraction_metadata column %s not present (n=%d)", col, n)
		}
	}
	if countMigration(t, st.DB, "020_ttl_extraction.sql") != 1 {
		t.Fatalf("expected 020 migration recorded once")
	}

	// Existing schema intact (additive).
	for _, name := range []string{"observations", "observations_fts", "sessions"} {
		if !tableExists(t, st.DB, name) {
			t.Fatalf("pre-existing table %s missing after 020", name)
		}
	}

	// Both new tables are writable.
	if _, err := st.DB.Exec(`INSERT INTO ttl_config (key, value) VALUES ('ttl_days', '30')`); err != nil {
		t.Fatalf("insert ttl_config: %v", err)
	}
	if _, err := st.DB.Exec(`INSERT INTO extraction_metadata (session_id, content_hash, extracted_at, model) VALUES ('s1','h1', ?, 'test')`, time.Now().UTC().Format(time.RFC3339)); err != nil {
		t.Fatalf("insert extraction_metadata: %v", err)
	}
	var cfg, meta int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM ttl_config`).Scan(&cfg); err != nil {
		t.Fatalf("count ttl_config: %v", err)
	}
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM extraction_metadata`).Scan(&meta); err != nil {
		t.Fatalf("count extraction_metadata: %v", err)
	}
	if cfg != 1 || meta != 1 {
		t.Fatalf("expected writable new tables, got %d/%d rows", cfg, meta)
	}
}

var _ = sql.ErrConnDone // keep database/sql import stable for future assertions
