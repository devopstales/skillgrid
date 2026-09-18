package memfs

import (
	"context"
	"database/sql"
	"fmt"
	"path"
	"strings"
)

// Entry is one node in a code-index listing (ls).
type Entry struct {
	Name      string
	Kind      string // "dir" | "file" | "symbol"
	Signature string
	StartLine int
	EndLine   int
	SymbolID  int64
}

// List returns the nodes under the given code path.
//   - dir path  → immediate subdirectories (Name, Kind "dir") + files (Name, Kind "file")
//   - file path → the file's symbols (Name, Kind "symbol", Signature, StartLine, EndLine)
//   - symbol path → the single symbol
//
// An empty (unindexed) store returns 0 entries and a nil error; the CLI prints
// the "no code index" note.
func (fs *MemFS) List(ctx context.Context, codePath string) ([]Entry, error) {
	if fs == nil || fs.db == nil {
		return nil, fmt.Errorf("memfs: not initialized")
	}
	cp, err := ResolveCodePath(fs.projectID, codePath)
	if err != nil {
		return nil, err
	}
	switch cp.Kind {
	case "dir":
		return fs.listDir(ctx, cp.Dir)
	case "file":
		return fs.listSymbols(ctx, cp.Dir, cp.File, "")
	case "symbol":
		return fs.listSymbols(ctx, cp.Dir, cp.File, cp.Symbol)
	}
	return nil, fmt.Errorf("memfs list: unknown kind %q", cp.Kind)
}

// fullPath joins a dir prefix and a file basename into a repo-relative path.
func fullPath(dir, file string) string {
	if dir == "" {
		return file
	}
	return dir + "/" + file
}

// listDir lists immediate subdirectories and files under dirPrefix.
func (fs *MemFS) listDir(ctx context.Context, dirPrefix string) ([]Entry, error) {
	var rows *sql.Rows
	var err error
	if dirPrefix == "" {
		rows, err = fs.db.QueryContext(ctx, `SELECT path FROM files ORDER BY path LIMIT 200`)
	} else {
		rows, err = fs.db.QueryContext(ctx,
			`SELECT path FROM files WHERE path LIKE ? ORDER BY path LIMIT 200`, dirPrefix+"/%")
	}
	if err != nil {
		return nil, fmt.Errorf("memfs list: %w", err)
	}
	defer rows.Close()

	type node struct {
		name string
		kind string
	}
	seen := map[string]bool{}
	var out []Entry
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("memfs list scan: %w", err)
		}
		rel := strings.TrimPrefix(p, dirPrefix)
		if dirPrefix != "" {
			rel = strings.TrimPrefix(rel, "/")
		}
		segs := splitPath(rel)
		if len(segs) == 0 {
			continue
		}
		first := segs[0]
		if len(segs) > 1 {
			if !seen[first] {
				seen[first] = true
				out = append(out, Entry{Name: first, Kind: "dir"})
			}
		} else {
			if !seen[first] {
				seen[first] = true
				out = append(out, Entry{Name: first, Kind: "file"})
			}
		}
	}
	return out, rows.Err()
}

// listSymbols lists the symbols in a file (optionally a single named one).
func (fs *MemFS) listSymbols(ctx context.Context, dir, file, symbol string) ([]Entry, error) {
	fp := fullPath(dir, file)
	q := `
		SELECT s.name, s.signature, s.start_line, s.end_line, s.id
		FROM symbols s JOIN files f ON f.id = s.file_id
		WHERE f.path = ?`
	args := []any{fp}
	if symbol != "" {
		q += ` AND s.name = ?`
		args = append(args, symbol)
	}
	q += ` ORDER BY s.start_line, s.name LIMIT 200`
	rows, err := fs.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("memfs list: %w", err)
	}
	defer rows.Close()
	var out []Entry
	for rows.Next() {
		var e Entry
		var sig sql.NullString
		if err := rows.Scan(&e.Name, &sig, &e.StartLine, &e.EndLine, &e.SymbolID); err != nil {
			return nil, fmt.Errorf("memfs list scan: %w", err)
		}
		e.Kind = "symbol"
		e.Signature = sig.String
		out = append(out, e)
	}
	return out, rows.Err()
}

// matchesCodeGlob reports whether the pattern matches a file path or basename.
func matchesCodeGlob(pattern, p string) bool {
	if ok, _ := path.Match(pattern, p); ok {
		return true
	}
	if ok, _ := path.Match(pattern, path.Base(p)); ok {
		return true
	}
	return false
}
