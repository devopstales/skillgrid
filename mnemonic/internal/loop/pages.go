package loop

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PagesDir is the gitignored cache of compiled views. SQLite stays the
// source of truth; these files are regenerated and never edited by hand.
func PagesDir(root string) string {
	return filepath.Join(root, ".skillgrid", "cache", "mnemonic")
}

// WritePages regenerates the architecture and decisions pages under root.
func WritePages(ctx context.Context, db *sql.DB, root, project string) error {
	dir := PagesDir(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	arch, err := architecturePage(ctx, db, project)
	if err != nil {
		return err
	}
	dec, err := decisionsPage(ctx, db, project)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "architecture.md"), []byte(arch), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "decisions.md"), []byte(dec), 0o644)
}

func architecturePage(ctx context.Context, db *sql.DB, project string) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "# Architecture\n\nProject: %s\n\nGenerated from the code index. Do not edit.\n\n", project)
	b.WriteString("## Communities\n\n")
	rows, err := db.QueryContext(ctx, `
		SELECT id, label, symbol_count FROM community_meta ORDER BY symbol_count DESC, id LIMIT 20`)
	if err != nil {
		return "", err
	}
	n := 0
	for rows.Next() {
		var id, count int
		var label string
		if err := rows.Scan(&id, &label, &count); err != nil {
			rows.Close()
			return "", err
		}
		fmt.Fprintf(&b, "- %d %s (%d symbols)\n", id, label, count)
		n++
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return "", err
	}
	if n == 0 {
		b.WriteString("- (none indexed)\n")
	}
	b.WriteString("\n## God nodes\n\n")
	gods, err := db.QueryContext(ctx, `
		SELECT s.name, COUNT(e.id) AS degree
		FROM symbols s
		LEFT JOIN edges e ON e.from_id = s.id OR e.to_id = s.id
		GROUP BY s.id
		ORDER BY degree DESC, s.name
		LIMIT 15`)
	if err != nil {
		return "", err
	}
	defer gods.Close()
	gn := 0
	for gods.Next() {
		var name string
		var degree int
		if err := gods.Scan(&name, &degree); err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "- %s degree %d\n", name, degree)
		gn++
	}
	if gn == 0 {
		b.WriteString("- (none)\n")
	}
	b.WriteString("\n## Cross-module edges\n\n")
	b.WriteString("Edges whose endpoints sit in different communities.\n\n")
	cross, err := db.QueryContext(ctx, `
		SELECT s1.name, s2.name, e.kind
		FROM edges e
		JOIN symbols s1 ON s1.id = e.from_id
		JOIN symbols s2 ON s2.id = e.to_id
		JOIN communities c1 ON c1.symbol_id = s1.id
		JOIN communities c2 ON c2.symbol_id = s2.id
		WHERE c1.id != c2.id
		ORDER BY s1.name, s2.name
		LIMIT 20`)
	if err != nil {
		return "", err
	}
	defer cross.Close()
	cn := 0
	for cross.Next() {
		var a, b2, kind string
		if err := cross.Scan(&a, &b2, &kind); err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "- %s -%s-> %s\n", a, kind, b2)
		cn++
	}
	if cn == 0 {
		b.WriteString("- (none)\n")
	}
	return b.String(), cross.Err()
}

func decisionsPage(ctx context.Context, db *sql.DB, project string) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "# Decisions\n\nProject: %s\n\nLive topic keys only. Superseded rows are omitted.\n\n", project)
	rows, err := db.QueryContext(ctx, `
		SELECT topic_key, title, type
		FROM observations
		WHERE project = ? AND deleted_at IS NULL
		  AND topic_key IS NOT NULL AND TRIM(topic_key) != ''
		  AND (invalid_at IS NULL OR invalid_at = '')
		ORDER BY topic_key, id`, project)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		var key, title, typ string
		if err := rows.Scan(&key, &title, &typ); err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "## %s\n\n%s (%s)\n\n", key, title, typ)
		n++
	}
	if n == 0 {
		b.WriteString("(no live topic keys)\n")
	}
	return b.String(), rows.Err()
}
