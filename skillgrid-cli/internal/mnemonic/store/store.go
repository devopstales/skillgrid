package store

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Store wraps a project-scoped SQLite database.
type Store struct {
	DB   *sql.DB
	path string
}

// Open opens or creates the SQLite database for projectID under dataDir.
func Open(dataDir, projectID string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dataDir, projectID+".sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

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

// Close closes the underlying database connection.
func (s *Store) Close() error {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.DB.Close()
}

func migrate(db *sql.DB) error {
	sqlBytes, err := migrationsFS.ReadFile("migrations/001_initial.sql")
	if err != nil {
		return fmt.Errorf("read migration: %w", err)
	}

	if _, err := db.Exec(string(sqlBytes)); err != nil {
		return fmt.Errorf("exec migration: %w", err)
	}

	return nil
}
