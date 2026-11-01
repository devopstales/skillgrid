package memfs

import (
	"context"
	"fmt"
)

// listCap is the maximum number of file/symbol rows a listing or find returns
// before truncation. A single policy value shared by ls/tree/find and asserted
// by the scale test so the cap and its assertion can't drift.
const listCap = 200

// wholeFileEnd is the sentinel end-line for "the whole file" cat spans (a
// value larger than any plausible chunk end_line).
const wholeFileEnd = 1 << 30

// fileQuery returns a parameterized "list file paths under dirPrefix" query.
// dirPrefix "" lists all indexed files; otherwise it matches the dir subtree.
// This is the single home for the cap + prefix-scope logic (review #3/#4).
func fileQuery(dirPrefix string) (string, []any) {
	q := "SELECT path FROM files"
	var args []any
	if dirPrefix != "" {
		q += " WHERE path LIKE ?"
		args = append(args, dirPrefix+"/%")
	}
	q += fmt.Sprintf(" ORDER BY path LIMIT %d", listCap)
	return q, args
}

// symbolNameQuery returns a parameterized "symbol name + signature" query
// under dirPrefix (capped) — the single home for Find's symbols leg (review #4).
func symbolNameQuery(dirPrefix string) (string, []any) {
	q := "SELECT s.name, s.signature FROM files f JOIN symbols s ON s.file_id = f.id"
	var args []any
	if dirPrefix != "" {
		q += " WHERE f.path LIKE ?"
		args = append(args, dirPrefix+"/%")
	}
	q += fmt.Sprintf(" ORDER BY s.name LIMIT %d", listCap)
	return q, args
}

// fileCountQuery returns a parameterized "file path + symbol count" query
// under dirPrefix (capped) — the single home for the tree's per-file count
// (review #4).
func fileCountQuery(dirPrefix string) (string, []any) {
	q := "SELECT f.path, (SELECT COUNT(*) FROM symbols s WHERE s.file_id = f.id) FROM files f"
	var args []any
	if dirPrefix != "" {
		q += " WHERE f.path LIKE ?"
		args = append(args, dirPrefix+"/%")
	}
	q += fmt.Sprintf(" ORDER BY f.path LIMIT %d", listCap)
	return q, args
}

// listFilePaths returns the indexed file paths under dirPrefix (capped).
func (fs *MemFS) listFilePaths(ctx context.Context, dirPrefix string) ([]string, error) {
	q, args := fileQuery(dirPrefix)
	rows, err := fs.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("memfs: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("memfs list scan: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// NoCodeIndexNote is the single home for the "unindexed store" message
// (review #7). The CLI and the tree renderer both use it.
func (fs *MemFS) NoCodeIndexNote() string {
	return "mem fs: no code index for project " + fs.projectID + " (run skillgrid index)"
}

func (fs *MemFS) noCodeIndexNote() string { return fs.NoCodeIndexNote() }
