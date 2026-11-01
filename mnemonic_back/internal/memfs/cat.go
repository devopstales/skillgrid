package memfs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// Cat returns the source for a code node:
//   - file::symbol → the chunks whose [start_line..end_line] intersect the
//     symbol's [start_line..end_line], concatenated in line order.
//   - file         → the file's full text (all chunks in line order).
//
// An unknown file or symbol (or an unindexed store) returns an error.
func (fs *MemFS) Cat(ctx context.Context, codePath string) (string, error) {
	if fs == nil || fs.db == nil {
		return "", fmt.Errorf("memfs: not initialized")
	}
	cp, err := ResolveCodePath(fs.projectID, codePath)
	if err != nil {
		return "", err
	}
	if cp.Kind == "dir" {
		return "", errors.New("memfs cat: path is a directory; cat a file or file::symbol")
	}
	fp := fullPath(cp.Dir, cp.File)

	// Confirm the file exists.
	var fileID int64
	err = fs.db.QueryRowContext(ctx, `SELECT id FROM files WHERE path = ?`, fp).Scan(&fileID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("memfs cat: no indexed file %q", fp)
	}
	if err != nil {
		return "", fmt.Errorf("memfs cat: %w", err)
	}

	if cp.Kind == "symbol" {
		var startLine, endLine int
		err = fs.db.QueryRowContext(ctx,
			`SELECT start_line, end_line FROM symbols WHERE file_id = ? AND name = ?`,
			fileID, cp.Symbol).Scan(&startLine, &endLine)
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("memfs cat: no symbol %q in %q", cp.Symbol, fp)
		}
		if err != nil {
			return "", fmt.Errorf("memfs cat: %w", err)
		}
		return fs.catSpan(ctx, fileID, startLine, endLine)
	}
	// Whole file: all its chunks in line order.
	return fs.catSpan(ctx, fileID, 0, wholeFileEnd)
}

// catSpan concatenates the chunks of a file intersecting [startLine..endLine].
func (fs *MemFS) catSpan(ctx context.Context, fileID int64, startLine, endLine int) (string, error) {
	rows, err := fs.db.QueryContext(ctx,
		`SELECT text FROM chunks WHERE file_id = ? AND start_line <= ? AND end_line >= ? ORDER BY start_line`,
		fileID, endLine, startLine)
	if err != nil {
		return "", fmt.Errorf("memfs cat: %w", err)
	}
	defer rows.Close()
	var parts []string
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return "", fmt.Errorf("memfs cat scan: %w", err)
		}
		parts = append(parts, text)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(parts) == 0 {
		return "", errors.New("memfs cat: no source chunks for this node")
	}
	return strings.Join(parts, "\n"), nil
}
