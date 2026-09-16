package memfs

import (
	"context"
	"database/sql"
	"fmt"
	"path"
	"strings"
)

// Find performs a filesystem-style glob pattern search across observations
// in the given scope. The pattern is matched against the basename of the
// topic_key (the last path segment) and against the observation title.
//
// Supported glob characters: * (any chars), ? (single char).
//
//   - Find(ctx, "auth*", "project/A/") → observations whose topic_key basename
//     or title matches "auth*"
//   - Find(ctx, "*.go", "") → all observations matching "*.go" across all scopes
//
// scope may be "" to search all observations in the project.
func (fs *MemFS) Find(ctx context.Context, pattern string, scope string) ([]Observation, error) {
	if fs == nil || fs.db == nil {
		return nil, fmt.Errorf("memfs: not initialized")
	}
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil, fmt.Errorf("memfs find: empty pattern")
	}

	var (
		rows *sql.Rows
		err  error
	)
	if scope != "" {
		f, perr := ResolveURI(scope)
		if perr != nil {
			return nil, perr
		}
		prefix := f.Prefix()
		rows, err = fs.db.QueryContext(ctx, `
			SELECT id, title, content, topic_key, memory_type, created_at
			FROM observations
			WHERE project = ? AND deleted_at IS NULL
			  AND (topic_key = ? OR topic_key LIKE ?)
			ORDER BY created_at DESC, id DESC
			LIMIT 200`,
			fs.projectID, prefix, prefix+"/%",
		)
	} else {
		rows, err = fs.db.QueryContext(ctx, `
			SELECT id, title, content, topic_key, memory_type, created_at
			FROM observations
			WHERE project = ? AND deleted_at IS NULL
			ORDER BY created_at DESC, id DESC
			LIMIT 200`,
			fs.projectID,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("memfs find: %w", err)
	}
	defer rows.Close()

	var out []Observation
	for rows.Next() {
		var o Observation
		var tk sql.NullString
		var mt sql.NullString
		if err := rows.Scan(&o.ID, &o.Title, &o.Content, &tk, &mt, &o.CreatedAt); err != nil {
			return nil, fmt.Errorf("memfs find scan: %w", err)
		}
		o.TopicKey = tk.String
		o.MemoryType = mt.String
		if matchesGlob(pattern, o) {
			out = append(out, o)
		}
	}
	return out, rows.Err()
}

// matchesGlob reports whether the observation matches the glob pattern.
// It checks the basename of topic_key and the title.
func matchesGlob(pattern string, o Observation) bool {
	// Check topic_key basename.
	if o.TopicKey != "" {
		base := path.Base(o.TopicKey)
		if ok, _ := path.Match(pattern, base); ok {
			return true
		}
	}
	// Check title.
	if o.Title != "" {
		if ok, _ := path.Match(pattern, o.Title); ok {
			return true
		}
	}
	return false
}
