package memory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// UpdateInput holds the fields mem_update can change. Empty strings are
// left untouched; only non-empty fields are applied.
type UpdateInput struct {
	Title    string
	Content  string
	Type     string
	TopicKey string
	Scope    string
}

// Update modifies an existing observation in place. Bumps updated_at; the
// observations_fts UPDATE trigger (001_initial.sql) keeps the FTS row in
// sync automatically.
func (s *Service) Update(ctx context.Context, id int64, in UpdateInput) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return errors.New("memory service not initialized")
	}
	hasContent := strings.TrimSpace(in.Title) != "" || strings.TrimSpace(in.Content) != ""

	type kv struct{ col, val string }
	kvs := []kv{}
	if v := strings.TrimSpace(in.Title); v != "" {
		kvs = append(kvs, kv{"title", v})
	}
	if v := strings.TrimSpace(in.Content); v != "" {
		kvs = append(kvs, kv{"content", v})
	}
	if v := strings.TrimSpace(in.Type); v != "" {
		if !IsValidType(v) {
			return fmt.Errorf("invalid type %q", v)
		}
		kvs = append(kvs, kv{"type", v})
	}
	if v := strings.TrimSpace(in.Scope); v != "" {
		kvs = append(kvs, kv{"scope", v})
	}
	if v := strings.TrimSpace(in.TopicKey); v != "" {
		kvs = append(kvs, kv{"topic_key", v})
	}

	if hasContent {
		// Recompute the dedup hash from the *resulting* title/content so the
		// 24h dedupe window keeps working after an in-place edit.
		var curTitle, curContent, curType string
		if err := s.store.DB.QueryRowContext(ctx, `
			SELECT title, content, type FROM observations
			WHERE id = ? AND project = ? AND deleted_at IS NULL`,
			id, s.projectID,
		).Scan(&curTitle, &curContent, &curType); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("observation %d not found", id)
			}
			return fmt.Errorf("load observation: %w", err)
		}
		if strings.TrimSpace(in.Title) != "" {
			curTitle = in.Title
		}
		if strings.TrimSpace(in.Content) != "" {
			curContent = in.Content
		}
		kvs = append(kvs, kv{"normalized_hash", normalizedHash(curTitle, curContent, curType)})
	}

	if len(kvs) == 0 {
		return errors.New("no fields to update")
	}

	sets := make([]string, 0, len(kvs)+1)
	args := make([]any, 0, len(kvs)+1)
	for _, pair := range kvs {
		sets = append(sets, pair.col+" = ?")
		args = append(args, pair.val)
	}
	sets = append(sets, "updated_at = ?")
	args = append(args, time.Now().UTC().Format(time.RFC3339), id, s.projectID)
	res, err := s.store.DB.ExecContext(ctx, `
		UPDATE observations SET `+strings.Join(sets, ", ") + `
		WHERE id = ? AND project = ? AND deleted_at IS NULL`,
		args...,
	)
	if err != nil {
		return fmt.Errorf("update observation: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("observation %d not found", id)
	}
	return nil
}

// Delete removes an observation. Soft-delete (hard=false) sets deleted_at;
// hard=true removes the row. FTS stays in sync via the trigger in
// 001_initial.sql.
func (s *Service) Delete(ctx context.Context, id int64, hard bool) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return errors.New("memory service not initialized")
	}
	verb, query, args := "soft-delete", `
		UPDATE observations SET deleted_at = ?
		WHERE id = ? AND project = ? AND deleted_at IS NULL`,
		[]any{time.Now().UTC().Format(time.RFC3339), id, s.projectID}
	if hard {
		verb, query, args = "hard-delete", `
			DELETE FROM observations WHERE id = ? AND project = ?`,
			[]any{id, s.projectID}
	}
	res, err := s.store.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s observation: %w", verb, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("observation %d not found", id)
	}
	return nil
}

// TimelineEntry is a neighbour in the chronological window around an anchor.
type TimelineEntry struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	TopicKey  string `json:"topic_key,omitempty"`
	Scope     string `json:"scope,omitempty"`
	CreatedAt string `json:"created_at"`
	Direction string `json:"direction"` // "before" or "after"
}

// Timeline is the progressive-disclosure middle layer: compact context
// around an anchor observation (mem_search → mem_timeline →
// mem_get_observation).
type Timeline struct {
	Before []TimelineEntry `json:"before"`
	After  []TimelineEntry `json:"after"`
}

// Timeline returns observations created `before` and `after` the anchor,
// bounded to `window` on each side and `limit` entries per direction.
// "before" is newest-first (closest to the anchor), "after" is
// oldest-first (closest to the anchor).
func (s *Service) Timeline(ctx context.Context, anchorID int64, window time.Duration, limit int) (Timeline, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return Timeline{}, errors.New("memory service not initialized")
	}
	if window <= 0 {
		window = 1 * time.Hour
	}
	if limit <= 0 {
		limit = 5
	}
	var anchorAt string
	if err := s.store.DB.QueryRowContext(ctx, `
		SELECT created_at FROM observations
		WHERE id = ? AND project = ? AND deleted_at IS NULL`,
		anchorID, s.projectID,
	).Scan(&anchorAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Timeline{}, fmt.Errorf("observation %d not found (anchor)", anchorID)
		}
		return Timeline{}, fmt.Errorf("anchor lookup: %w", err)
	}
	minutes := int(window.Minutes())

	before, err := s.timelineSide(ctx, anchorID, anchorAt, minutes, limit, true)
	if err != nil {
		return Timeline{}, err
	}
	after, err := s.timelineSide(ctx, anchorID, anchorAt, minutes, limit, false)
	if err != nil {
		return Timeline{}, err
	}
	return Timeline{Before: before, After: after}, nil
}

func (s *Service) timelineSide(ctx context.Context, anchorID int64, anchorAt string, minutes, limit int, isBefore bool) ([]TimelineEntry, error) {
	var (
		order  string
		where  string
		label  string
	)
	if isBefore {
		label = "before"
		order = "DESC"
		where = `
			WHERE id != ? AND project = ? AND deleted_at IS NULL
			  AND datetime(created_at) < datetime(?)
			  AND datetime(created_at) >= datetime(?, '-'||?||' minutes')`
	} else {
		label = "after"
		order = "ASC"
		where = `
			WHERE id != ? AND project = ? AND deleted_at IS NULL
			  AND datetime(created_at) > datetime(?)
			  AND datetime(created_at) <= datetime(?, '+'||?||' minutes')`
	}
	sql := `
		SELECT id, type, title, content, topic_key, scope, created_at, '` + label + `'
		FROM observations ` + where + `
		ORDER BY created_at ` + order + `, id ` + order + `
		LIMIT ?`
	return s.scanTimeline(ctx, sql, anchorID, anchorAt, minutes, limit)
}

func (s *Service) scanTimeline(ctx context.Context, query string, anchorID int64, anchorAt string, minutes, limit int) ([]TimelineEntry, error) {
	rows, err := s.store.DB.QueryContext(ctx, query,
		anchorID, s.projectID, anchorAt, anchorAt, fmt.Sprintf("%d", minutes), limit,
	)
	if err != nil {
		return nil, fmt.Errorf("timeline query: %w", err)
	}
	defer rows.Close()
	var out []TimelineEntry
	for rows.Next() {
		var e TimelineEntry
		var topicKey, scope sql.NullString
		if err := rows.Scan(&e.ID, &e.Type, &e.Title, &e.Content, &topicKey, &scope, &e.CreatedAt, &e.Direction); err != nil {
			return nil, fmt.Errorf("scan timeline row: %w", err)
		}
		if topicKey.Valid {
			e.TopicKey = topicKey.String
		}
		if scope.Valid {
			e.Scope = scope.String
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
