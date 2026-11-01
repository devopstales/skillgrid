package memfs

import (
	"context"
	"database/sql"
	"fmt"
	"path"
	"strings"
)

// Find globs the code index within an optional scope. The pattern is matched
// (via path.Match) against indexed file paths/basenames and against symbol
// names. scope (a code path prefix) narrows the search to files under it.
//
//   - Find(ctx, "*.go", "")         → all files matching *.go + symbols named *.go
//   - Find(ctx, "Handler", "")       → symbols named Handler
//   - Find(ctx, "*.go", "src/auth/") → files under src/auth/ matching *.go
//
// An unindexed store returns 0 entries, nil error.
func (fs *MemFS) Find(ctx context.Context, pattern, scope string) ([]Entry, error) {
	if fs == nil || fs.db == nil {
		return nil, fmt.Errorf("memfs: not initialized")
	}
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil, fmt.Errorf("memfs find: empty pattern")
	}
	scopeDir := ""
	if scope != "" {
		cp, err := ResolveCodePath(fs.projectID, scope)
		if err != nil {
			return nil, err
		}
		scopeDir = cp.Dir
	}

	// Materialize files first (single query, closed), then symbols, so the
	// single-connection store pool is never held by two live row sets.
	files, err := fs.listFilePaths(ctx, scopeDir)
	if err != nil {
		return nil, err
	}

	var out []Entry
	seenFile := map[string]bool{}
	for _, p := range files {
		if seenFile[p] {
			continue
		}
		if matchesCodeGlob(pattern, p) {
			seenFile[p] = true
			out = append(out, Entry{Name: p, Kind: "file"})
		}
	}

	type symRef struct {
		name string
		sig  string
	}
	var syms []symRef
	{
		q, args := symbolNameQuery(scopeDir)
		rows, err := fs.db.QueryContext(ctx, q, args...)
		if err != nil {
			return nil, fmt.Errorf("memfs find: %w", err)
		}
		for rows.Next() {
			var sr symRef
			var sig sql.NullString
			if err := rows.Scan(&sr.name, &sig); err != nil {
				rows.Close()
				return nil, fmt.Errorf("memfs find scan: %w", err)
			}
			sr.sig = sig.String
			syms = append(syms, sr)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	}
	for _, sr := range syms {
		if ok, _ := path.Match(pattern, sr.name); ok {
			out = append(out, Entry{Name: sr.name, Kind: "symbol", Signature: sr.sig})
		}
	}
	return out, nil
}
