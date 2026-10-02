package codeindex

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/search"
)

func init() {
	// search already sits under hybrid, which this package imports, so the
	// symbol FTS function cannot import codeindex. It calls this lookup
	// through the registration below.
	search.SetAliasLookup(LookupAliases)
}

// LookupAliases returns qualified_name values whose alias matches text
// case-insensitively for project. An empty result is a miss, not an error.
func LookupAliases(ctx context.Context, db *sql.DB, project, text string) ([]string, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	project = strings.TrimSpace(project)
	text = strings.TrimSpace(text)
	if project == "" || text == "" {
		return nil, nil
	}
	rows, err := db.QueryContext(ctx, `
		SELECT qualified_name FROM entity_aliases
		WHERE project = ? AND alias = ? COLLATE NOCASE
		ORDER BY qualified_name`, project, text)
	if err != nil {
		return nil, fmt.Errorf("lookup entity aliases: %w", err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan entity alias: %w", err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate entity aliases: %w", err)
	}
	return names, nil
}

// insertIndexAliases records the symbol name and qualified name as index-sourced
// aliases. Identical values collapse to one row via the primary key.
func insertIndexAliases(tx *sql.Tx, project, name, qualified string) error {
	if strings.TrimSpace(project) == "" || qualified == "" {
		return nil
	}
	for _, alias := range []string{name, qualified} {
		if alias == "" {
			continue
		}
		if _, err := tx.Exec(`
			INSERT OR IGNORE INTO entity_aliases (project, alias, qualified_name, source)
			VALUES (?, ?, ?, 'index')`, project, alias, qualified); err != nil {
			return fmt.Errorf("insert entity alias %q: %w", alias, err)
		}
	}
	return nil
}

// projectIDFromConn reads the store project id from the sqlite filename
// ({project}.sqlite), matching store.Open.
func projectIDFromConn(q interface {
	QueryRow(query string, args ...any) *sql.Row
}) string {
	var file string
	if err := q.QueryRow(`SELECT file FROM pragma_database_list() WHERE name = 'main'`).Scan(&file); err != nil || file == "" {
		return ""
	}
	return strings.TrimSuffix(filepath.Base(file), ".sqlite")
}
