package store

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"sync"
	"time"
)

// envCacheDisable bypasses the handle pool (rollback boundary — every Open
// creates a fresh connection).
const envCacheDisable = "MNEMONIC_DISABLE_CACHE"

// pool is the process-global refcounted store-handle pool (spec
// 001-storage §5: refcounted handle pool prevents N+1 opens of the same
// project). Keyed by the project's on-disk sqlite path.
var (
	poolMu sync.Mutex
	pool   = map[string]*refHandle{}
)

// refHandle is a pooled *sql.DB shared by every live Store for the same
// project path. The DB closes only when the last reference is released.
type refHandle struct {
	db   *sql.DB
	refs int
}

// poolCount returns the number of live pool entries (test helper).
func poolCount() int {
	poolMu.Lock()
	defer poolMu.Unlock()
	return len(pool)
}

// poolDisabled reports whether MNEMONIC_DISABLE_CACHE=1 bypasses the pool.
func poolDisabled() bool {
	return strings.EqualFold(os.Getenv(envCacheDisable), "1")
}

// poolAcquire returns the shared *sql.DB for path, opening a fresh one if
// needed, and takes a reference.
func poolAcquire(path string, openFn func() (*sql.DB, error)) (*sql.DB, error) {
	if poolDisabled() {
		return openFn()
	}
	poolMu.Lock()
	defer poolMu.Unlock()
	if h, ok := pool[path]; ok {
		if err := pingContext(h.db); err == nil {
			h.refs++
			return h.db, nil
		}
		// Unhealthy pooled handle: evict and reopen.
		delete(pool, path)
	}
	db, err := openFn()
	if err != nil {
		return nil, err
	}
	pool[path] = &refHandle{db: db, refs: 1}
	return db, nil
}

// poolRelease drops one reference to path; the DB closes at zero.
func poolRelease(path string) error {
	if poolDisabled() {
		return nil // bypass mode: the Store closes its own DB
	}
	poolMu.Lock()
	defer poolMu.Unlock()
	h, ok := pool[path]
	if !ok {
		return nil // already evicted (concurrent final release)
	}
	h.refs--
	if h.refs <= 0 {
		delete(pool, path)
		return h.db.Close()
	}
	return nil
}

// poolReleaseDB closes the given db directly (cache-disabled mode, where the
// Store owns its handle outright).
func poolReleaseDB(db *sql.DB) error {
	if db == nil {
		return nil
	}
	return db.Close()
}

// pingContext pings with a short timeout (reuses the handle if healthy).
func pingContext(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return db.PingContext(ctx)
}
