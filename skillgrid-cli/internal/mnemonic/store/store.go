// Package store wraps a project-scoped SQLite database with embedded SQL migrations.
package store

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Store is the shared SQLite handle for one project.
type Store struct {
	DB   *sql.DB
	path string
}

// Open opens or creates the SQLite database for projectID under dataDir.
func Open(dataDir, projectID string) (*Store, error) {
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
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(1)
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("apply %s: %w", pragma, err)
		}
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &Store{DB: db, path: dbPath}, nil
}

// Path returns the on-disk path of the SQLite file.
func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Close releases the database handle.
func (s *Store) Close() error {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.DB.Close()
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
