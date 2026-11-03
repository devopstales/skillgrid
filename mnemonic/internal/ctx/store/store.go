// Package store provides thin, typed accessors over the six ctx_* tables that
// live on the shared project SQLite store (ADR-0033/0012). The package opens
// NO database of its own: New wraps a caller-supplied *store.Store handle.
//
// Fail-open (ADR-0016): a query for a ctx_* table that does not exist (e.g. a
// store opened before the 056-061 migrations ran) degrades to an empty result
// or zero value — never a hard error that would 500 an endpoint.
package store

import (
	"database/sql"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// Store is a ctx accessor over the shared project store. It owns no
// connection: every query runs on the *sql.DB of the *store.Store passed to
// New (ADR-0033 — the ctx store opens no second DB handle / connection and
// creates only ctx_-prefixed tables).
type Store struct {
	db     *sql.DB
	shared *store.Store
}

// New wraps the shared store handle. It opens no database of its own; the
// accessor queries hit the same database file as shared.Path().
func New(shared *store.Store) *Store {
	if shared == nil || shared.DB == nil {
		return nil
	}
	return &Store{db: shared.DB, shared: shared}
}

// DB returns the underlying shared *sql.DB (test/integration use).
func (s *Store) DB() *sql.DB {
	if s == nil {
		return nil
	}
	return s.db
}

// Shared returns the wrapped *store.Store.
func (s *Store) Shared() *store.Store {
	if s == nil {
		return nil
	}
	return s.shared
}

// failOpenOpen reports whether err means the ctx table does not exist on this
// database. The driver renders the missing table as "no such table: <name>"
// (sometimes prefixed with "SQL logic error:"); we parse the name after the
// marker and compare it to the accessor's own table, so a different table's
// missing error is not swallowed.
func failOpenOpen(err error, table string) bool {
	if err == nil {
		return false
	}
	idx := indexErr(err.Error(), "no such table:")
	if idx < 0 {
		return false
	}
	name := strings.TrimLeft(err.Error()[idx+len("no such table:"):], " \t")
	return trimAfter(name) == table
}

// trimAfter trims everything after the first non-identifier character (the
// driver appends " (1)" — the sqlite error code — after the table name).
func trimAfter(s string) string {
	i := 0
	for i < len(s) && isIdentChar(s[i]) {
		i++
	}
	return s[:i]
}

func isIdentChar(c byte) bool {
	switch {
	case c == '_':
		return true
	case c >= 'a' && c <= 'z':
		return true
	case c >= 'A' && c <= 'Z':
		return true
	case c >= '0' && c <= '9':
		return true
	}
	return false
}

// indexErr reports the first index of marker in msg, or -1.
func indexErr(msg, marker string) int {
	for i := 0; i+len(marker) <= len(msg); i++ {
		if msg[i:i+len(marker)] == marker {
			return i
		}
	}
	return -1
}

// emptyRows is the fail-open stand-in for a query on a missing table: a row
// set whose Next() reports false immediately and whose Close/Err are safe
// no-ops, so callers iterate and close it exactly as they would a live
// *sql.Rows (a zero-value &sql.Rows{} panics in Next and Close).
type emptyRows struct{}

func (emptyRows) Next() bool             { return false }
func (emptyRows) Scan(dest ...any) error { return nil }
func (emptyRows) Close() error           { return nil }
func (emptyRows) Err() error             { return nil }

// queryFailOpen runs a table-scoped SELECT and, on a missing-table error,
// returns a closed, empty row set instead of the error (ADR-0016). Any other
// error propagates. The caller still defers Close and checks rows.Err().
func (s *Store) queryFailOpen(table, query string, args ...any) (*sql.Rows, error) {
	rows, err := s.db.Query(query, args...)
	if err == nil {
		return rows, nil
	}
	if !failOpenOpen(err, table) {
		if rows != nil {
			rows.Close()
		}
		return nil, err
	}
	// The driver returns a nil *sql.Rows alongside the error on a
	// missing-table fault, so there is nothing to close here.
	return nil, nil
}

// execFailOpen runs an INSERT/UPDATE against a ctx table and degrades a
// missing-table error to a no-op (the write is lost, not fatal — ADR-0016).
// Any other error propagates.
func (s *Store) execFailOpen(table, query string, args ...any) error {
	_, err := s.db.Exec(query, args...)
	if err != nil && failOpenOpen(err, table) {
		return nil
	}
	return err
}
