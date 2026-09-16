package memfs

import (
	"context"
	"database/sql"
	"fmt"
)

// List returns the observations under the given scope path.
//
//   - scope "project/A/preferences" → observations whose topic_key equals
//     "project/A/preferences" or starts with "project/A/preferences/"
//   - scope "project/A/" → all observations whose topic_key starts with "project/A/"
//   - scope "user/B/" → all observations whose topic_key starts with "user/B/"
//
// The scope path is resolved via ResolveURI to get the prefix.
func (fs *MemFS) List(ctx context.Context, scope string) ([]Observation, error) {
	if fs == nil || fs.db == nil {
		return nil, fmt.Errorf("memfs: not initialized")
	}
	f, err := ResolveURI(scope)
	if err != nil {
		return nil, err
	}
	prefix := f.Prefix()
	// Match exact topic_key OR children under the prefix.
	like := prefix + "/%"
	rows, err := fs.db.QueryContext(ctx, `
		SELECT id, title, content, topic_key, memory_type, created_at
		FROM observations
		WHERE project = ? AND deleted_at IS NULL
		  AND topic_key IS NOT NULL AND topic_key != ''
		  AND (topic_key = ? OR topic_key LIKE ?)
		ORDER BY created_at DESC, id DESC
		LIMIT 200`,
		fs.projectID, prefix, like,
	)
	if err != nil {
		return nil, fmt.Errorf("memfs list: %w", err)
	}
	defer rows.Close()

	var out []Observation
	for rows.Next() {
		var o Observation
		var tk sql.NullString
		var mt sql.NullString
		if err := rows.Scan(&o.ID, &o.Title, &o.Content, &tk, &mt, &o.CreatedAt); err != nil {
			return nil, fmt.Errorf("memfs list scan: %w", err)
		}
		o.TopicKey = tk.String
		o.MemoryType = mt.String
		out = append(out, o)
	}
	return out, rows.Err()
}
