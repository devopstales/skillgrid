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
	// Registers the vec0 vtab module so migration 042 (CREATE VIRTUAL TABLE
	// vec_symbols/vec_chunks) applies on every store, not just the ones that
	// also import vectorstore. Without it a plain store open fails with
	// "no such module: vec0".
	_ "modernc.org/sqlite/vec"
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
	// Pre-squash upgrade shim. Databases created before the 001..039
	// migrations were squashed into 001_schema.sql already carry a
	// schema_version marker row (written by the old runner) but no
	// migration:001_schema.sql row. Executing 001_schema.sql against such a
	// database would re-fire CREATE TABLE / ALTER TABLE statements that have
	// already landed, so we record it without executing. A fresh database has
	// no schema_version row before the loop, so 001_schema.sql runs normally
	// there.
	var hasVersion int
	if err := db.QueryRow(`SELECT COUNT(*) FROM index_meta WHERE key = 'schema_version'`).Scan(&hasVersion); err != nil {
		return fmt.Errorf("check schema_version: %w", err)
	}
	var hasSquash bool
	if err := db.QueryRow(`SELECT COUNT(*) > 0 FROM index_meta WHERE key = ?`, "migration:001_schema.sql").Scan(&hasSquash); err != nil {
		return fmt.Errorf("check 001_schema.sql: %w", err)
	}
	squashSkipped := false
	if hasVersion > 0 && !hasSquash {
		if _, err := db.Exec(`INSERT INTO index_meta (key, schema_version) VALUES (?, 1)`, "migration:001_schema.sql"); err != nil {
			return fmt.Errorf("record 001_schema.sql: %w", err)
		}
		appliedCount++
		squashSkipped = true
	}
	// Pre-squash stores recorded 001_schema.sql without executing it (or have
	// no marker at all), so their `observations` table never gained the columns
	// 054's data-copy reads. Backfill them (and the two child tables 055
	// rebuilds) before the 054+ batch runs, but only while 054 is still due —
	// a store that already passed 054 never runs the probes. The per-column
	// probe makes it a no-op on any store that has the columns (fresh builds
	// included). migrationDone returns true once 054 is RECORDED, so the
	// backfill runs only while it is NOT yet recorded (i.e. still about to run).
	if alreadyDone, err := migrationDone(db, "062_obs_split.sql"); err != nil {
		return err
	} else if !alreadyDone {
		if err := backfillPreSplitObservations(db); err != nil {
			return fmt.Errorf("backfill pre-split observations: %w", err)
		}
	}
	for _, name := range names {
		// On a pre-squash database 001_schema.sql was already recorded above
		// without being executed; skip it explicitly (the done check below
		// would also skip it).
		if name == "001_schema.sql" && squashSkipped {
			continue
		}
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
		if err := execMigrationFile(db, name, string(sqlBytes)); err != nil {
			return err
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

// execMigrationFile executes a migration's statements ONE PER STATEMENT rather
// than as a single db.Exec batch. The difference matters: a batch prepared via
// the driver stops at the first error but leaves earlier statements already
// applied, so a file that partially applies can leave the schema in a half
// state where a later file (e.g. 054 reading columns 001 adds) fails. On a
// legacy store 001_schema.sql's ADD COLUMNs are duplicates (the columns exist
// from an earlier partial apply); executing per statement means the duplicate
// stops 001 there, and — because that duplicate means every 054-required
// column is already present — the backfill has already ensured 054's reads
// succeed, so we record 001 and move on. Fresh stores apply 001 fully (no
// duplicate) and continue normally. A statement that fails for a reason other
// than a duplicate column is a genuine schema error and aborts the upgrade.
func execMigrationFile(db *sql.DB, name, contents string) error {
	for _, stmt := range splitSQLStatements(contents) {
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			if isDuplicateColumnErr(err) {
				// Already present (partial apply on a legacy store). Safe to
				// skip: the column the next migration needs exists.
				continue
			}
			return fmt.Errorf("exec %s: %w", name, err)
		}
	}
	return nil
}

// isDuplicateColumnErr reports whether a SQL error is "duplicate column
// name", the only ADD COLUMN failure that is safe to skip on a partially
// migrated store.
func isDuplicateColumnErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate column name")
}

// splitSQLStatements splits a migration script into individual statements.
// Semicolons terminate plain statements, but a CREATE TRIGGER / CREATE INDEX
// / CREATE VIEW body runs from its BEGIN to its matching END, whose `END;`
// (or a bare `END` on its own line, as 001_schema.sql writes) is the real
// terminator — a naive semicolon split would shred the trigger body and leave
// a dangling `END` that fails with "incomplete input". Single-quoted strings,
// -- line comments, and /* block */ comments are skipped so their contents
// never split a statement.
func splitSQLStatements(s string) []string {
	var (
		out       []string
		cur       strings.Builder
		inS       bool // inside 'string'
		inLine    bool // inside -- comment
		inBLK     bool // inside /* */
		inTrig    bool // inside a CREATE TRIGGER body (BEGIN...END)
		trigInStr bool // inside a 'string' within a trigger body
	)
	flush := func() {
		out = append(out, strings.TrimSpace(cur.String()))
		cur.Reset()
	}
	// hasTrigKeyword reports whether the accumulated statement has opened a
	// trigger (so the next BEGIN starts its body). Only CREATE TRIGGER opens
	// a body; CREATE INDEX/VIEW have no BEGIN...END to track.
	opensTrigger := func() bool {
		return strings.Contains(cur.String(), "CREATE TRIGGER")
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case inLine:
			if c == '\n' {
				inLine = false
			}
			cur.WriteByte(c)
		case inBLK:
			if c == '*' && i+1 < len(s) && s[i+1] == '/' {
				cur.WriteString("*/")
				i++
				inBLK = false
			} else {
				cur.WriteByte(c)
			}
		case inS:
			if c == '\'' {
				if i+1 < len(s) && s[i+1] == '\'' { // escaped ''
					cur.WriteString("''")
					i++
				} else {
					inS = false
					cur.WriteByte(c)
				}
			} else {
				cur.WriteByte(c)
			}
		case inTrig:
			// Inside a trigger body: look for the END terminator (word-boundary
			// match, optional trailing ';'). Everything else accumulates. String
			// literals are tracked here too: a literal containing the word END or
			// BEGIN ('...the END is near') must not terminate the body, so the
			// inS sub-state applies unchanged while inTrig is active.
			if c == '\'' && !trigInStr {
				trigInStr = true
				cur.WriteByte(c)
				break
			}
			if c == '\'' && trigInStr {
				if i+1 < len(s) && s[i+1] == '\'' { // escaped ''
					cur.WriteString("''")
					i++
					break
				}
				trigInStr = false
				cur.WriteByte(c)
				break
			}
			if trigInStr {
				cur.WriteByte(c)
				break
			}
			if (c == 'E' || c == 'e') && matchWord(s, i, "END") {
				j := i + 3
				for j < len(s) && (s[j] == ' ' || s[j] == '\t') {
					j++
				}
				if j < len(s) && s[j] == ';' {
					cur.WriteString(s[i : j+1])
					i = j
				} else {
					// bare END (001 writes a newline then END on its own line)
					cur.WriteString(s[i : i+3])
					i += 2
				}
				flush()
				inTrig = false
				trigInStr = false
			} else {
				cur.WriteByte(c)
			}
		default:
			switch c {
			case '\'':
				inS = true
				cur.WriteByte(c)
			case '-':
				if i+1 < len(s) && s[i+1] == '-' {
					inLine = true
					cur.WriteString("--")
					i++
				} else {
					cur.WriteByte(c)
				}
			case '/':
				if i+1 < len(s) && s[i+1] == '*' {
					inBLK = true
					cur.WriteString("/*")
					i++
				} else {
					cur.WriteByte(c)
				}
			case ';':
				flush()
			default:
				cur.WriteByte(c)
				// A CREATE TRIGGER statement's first BEGIN opens its body.
				if (c == 'B' || c == 'b') && matchWord(s, i, "BEGIN") {
					if opensTrigger() {
						inTrig = true
					}
				}
			}
		}
	}
	flush()
	return out
}

// matchWord reports whether s[i:] starts the identifier w (case-insensitive)
// as a whole word (not followed by another identifier character).
func matchWord(s string, i int, w string) bool {
	if i+len(w) > len(s) {
		return false
	}
	for k := 0; k < len(w); k++ {
		a, b := s[i+k], w[k]
		if a >= 'A' && a <= 'Z' {
			a += 'a' - 'A'
		}
		if a >= 'a' && a <= 'z' {
			b += 'a' - 'A'
		}
		if a != b {
			return false
		}
	}
	if i+len(w) < len(s) {
		n := s[i+len(w)]
		if (n >= 'a' && n <= 'z') || (n >= 'A' && n <= 'Z') || (n >= '0' && n <= '9') || n == '_' {
			return false
		}
	}
	return true
}

// migrationDone reports whether a named migration has already been recorded
// in index_meta. Used to gate the pre-split backfill on 054 still being due.
func migrationDone(db *sql.DB, name string) (bool, error) {
	var done bool
	if err := db.QueryRow(`SELECT COUNT(*) > 0 FROM index_meta WHERE key = ?`, "migration:"+name).Scan(&done); err != nil {
		return false, fmt.Errorf("check %s: %w", name, err)
	}
	return done, nil
}

// preSplitObservationColumns lists every `observations` column that 054's
// data-copy reads (all 41 non-id columns) but that a pre-squash store may be
// missing. 001_schema.sql adds most via ALTER TABLE; 043 adds archived_at /
// archive_reason; 044 adds valid_at / invalid_at / superseded_by. A
// pre-squash store that skipped 001 (recorded without executing) never gained
// any of them, so 054's bare-column SELECT fails at prepare time. The
// per-column probe makes this a no-op on stores that already have every
// column (fresh builds, or legacy stores that ran the individual 002–053
// migrations before the squash).
var preSplitObservationColumns = []struct {
	name string
	ddl  string
}{
	// 054 observation_state
	{"status", "status TEXT NOT NULL DEFAULT 'active'"},
	{"visibility", "visibility TEXT NOT NULL DEFAULT 'private'"},
	{"archived_at", "archived_at TEXT"},
	{"archive_reason", "archive_reason TEXT"},
	{"review_after", "review_after TEXT"},
	{"expires_at", "expires_at TEXT"},
	{"pinned", "pinned INTEGER NOT NULL DEFAULT 0"},
	// 054 observation_provenance
	{"source", "source TEXT NOT NULL DEFAULT 'agent'"},
	{"prompt_id", "prompt_id INTEGER"},
	{"tool_name", "tool_name TEXT"},
	{"provenance", "provenance TEXT"},
	{"owner", "owner TEXT"},
	{"memory_type", "memory_type TEXT"},
	{"graph_ref", "graph_ref INTEGER"},
	// 054 observation_stats
	{"duplicate_count", "duplicate_count INTEGER NOT NULL DEFAULT 0"},
	{"last_seen_at", "last_seen_at TEXT"},
	{"retrieval_usage", "retrieval_usage INTEGER NOT NULL DEFAULT 0"},
	{"importance_score", "importance_score REAL"},
	{"recency_decay", "recency_decay REAL"},
	{"maturity_tier", "maturity_tier TEXT"},
	{"risk_score", "risk_score REAL"},
	// 054 observation_temporal
	{"valid_at", "valid_at TEXT"},
	{"invalid_at", "invalid_at TEXT"},
	{"superseded_by", "superseded_by INTEGER"},
	// 054 observation_vectors
	{"embedding", "embedding BLOB"},
	{"embedding_model", "embedding_model TEXT"},
	{"embedding_created_at", "embedding_created_at TEXT"},
}

// backfillPreSplitObservations adds the pre-split columns and child tables a
// 054-migrating store is missing. SQLite has no `ADD COLUMN IF NOT EXISTS`
// and this driver's trigger/USING forms for guarding it do not parse, so each
// add is probed with pragma_table_info first and run only when absent.
// Idempotent: a store that already has every column (fresh build) is a no-op,
// and the whole step is gated on 054 still being due.
func backfillPreSplitObservations(db *sql.DB) error {
	// Fresh stores have no observations table yet (001_schema.sql creates it
	// later in this same migrate() run), so there is nothing to backfill. The
	// per-column probe below would also no-op, but the table-existence guard
	// short-circuits the whole step for the common fresh case.
	var tableExists int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type IN ('table','view') AND name = 'observations'`).Scan(&tableExists); err != nil {
		return fmt.Errorf("probe observations table: %w", err)
	}
	if tableExists == 0 {
		return nil
	}
	for _, c := range preSplitObservationColumns {
		var exists int
		if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('observations') WHERE name = ?`, c.name).Scan(&exists); err != nil {
			return fmt.Errorf("probe %s: %w", c.name, err)
		}
		if exists > 0 {
			continue
		}
		if _, err := db.Exec(`ALTER TABLE observations ADD COLUMN ` + c.ddl); err != nil {
			return fmt.Errorf("add %s: %w", c.name, err)
		}
	}
	// observation_versions + acl_grants: 055 rebuilds them to repoint their
	// FKs at observations_core, reading existing rows. A pre-squash store that
	// never ran 001_schema.sql's CREATE TABLE for them would make 055 fail with
	// "no such table", so create them when absent. But when 001_schema.sql is
	// still due (a legacy store about to execute it), 001 creates them itself —
	// creating them here would pre-empt 001's CREATE TABLE IF NOT EXISTS with a
	// stale `REFERENCES observations(id)` FK (001's version is then corrected
	// by 055, ours would not be). So only create when 001 has already run.
	if alreadyDone, err := migrationDone(db, "001_schema.sql"); err != nil {
		return err
	} else if alreadyDone {
		// observations_fts: 054's INSTEAD OF triggers (and the legacy
		// observations_fts_* triggers that ride along on RENAME) all write to
		// observations_fts. A pre-squash store that skipped 001 never created
		// it, so the triggers fail with "no such table". 054's triggers
		// maintain the index themselves, so we do NOT create the legacy
		// AFTER INSERT/UPDATE/DELETE triggers here (they would double-write).
		var ftsExists int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name = 'observations_fts'`).Scan(&ftsExists); err != nil {
			return fmt.Errorf("probe observations_fts: %w", err)
		}
		if ftsExists == 0 {
			if _, err := db.Exec(`
				CREATE VIRTUAL TABLE observations_fts USING fts5(
					title, content, type, project,
					content='observations', content_rowid='id',
					tokenize='porter'
				)
			`); err != nil {
				return fmt.Errorf("create observations_fts: %w", err)
			}
		}
		if _, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS observation_versions (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				observation_id INTEGER NOT NULL REFERENCES observations(id) ON DELETE CASCADE,
				revision INTEGER NOT NULL,
				content TEXT NOT NULL,
				created_at TEXT NOT NULL
			)
		`); err != nil {
			return fmt.Errorf("create observation_versions: %w", err)
		}
		if _, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS acl_grants (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				observation_id INTEGER NOT NULL REFERENCES observations(id) ON DELETE CASCADE,
				grantee TEXT NOT NULL,
				grant_type TEXT NOT NULL DEFAULT 'agent',
				created_at TEXT NOT NULL
			)
		`); err != nil {
			return fmt.Errorf("create acl_grants: %w", err)
		}
	}
	return nil
}
