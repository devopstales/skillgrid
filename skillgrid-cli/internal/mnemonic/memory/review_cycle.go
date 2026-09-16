package memory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// DefaultReviewInterval is how long a review cycle runs before the
// observation becomes due again. Long enough that re-review is meaningful
// (weeks), short enough that stale notes surface in practice.
const DefaultReviewInterval = 30 * 24 * time.Hour

// ReviewDue is an observation whose review_after has passed.
type ReviewDue struct {
	ID          int64  `json:"id"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	TopicKey    string `json:"topic_key,omitempty"`
	Scope       string `json:"scope,omitempty"`
	ReviewAfter string `json:"review_after"`
}

// ListReviews returns observations due for review, oldest review_after first.
func (s *Service) ListReviews(ctx context.Context, limit int) ([]ReviewDue, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, errors.New("memory service not initialized")
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT id, type, title, content, topic_key, scope, review_after
		FROM observations
		WHERE project = ? AND deleted_at IS NULL
		  AND review_after IS NOT NULL AND review_after != ''
		  AND review_after <= datetime('now')
		ORDER BY review_after ASC, id ASC
		LIMIT ?`,
		s.projectID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list reviews: %w", err)
	}
	defer rows.Close()
	var out []ReviewDue
	for rows.Next() {
		var r ReviewDue
		var topicKey, scope sql.NullString
		if err := rows.Scan(&r.ID, &r.Type, &r.Title, &r.Content, &topicKey, &scope, &r.ReviewAfter); err != nil {
			return nil, fmt.Errorf("scan review: %w", err)
		}
		if topicKey.Valid {
			r.TopicKey = topicKey.String
		}
		if scope.Valid {
			r.Scope = scope.String
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// MarkReviewed advances the review cycle so an observation is not due again
// before now + interval. It also starts the cycle the first time it is
// called (review_after was NULL).
func (s *Service) MarkReviewed(ctx context.Context, id int64) (reviewAfter string, err error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return "", errors.New("memory service not initialized")
	}
	now := time.Now().UTC()
	// Max(NULL, now) so re-marking a future review_after pushes it forward.
	next := now.Add(DefaultReviewInterval).Format(time.RFC3339)
	res, err := s.store.DB.ExecContext(ctx, `
		UPDATE observations
		SET review_after = ?
		WHERE id = ? AND project = ? AND deleted_at IS NULL`,
		next, id, s.projectID,
	)
	if err != nil {
		return "", fmt.Errorf("mark reviewed: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", fmt.Errorf("observation %d not found", id)
	}
	return next, nil
}

// SetReviewAfter sets the review cycle for an observation (used by mem_save
// when a capture_prompt=true flow wants a review scheduled).
func (s *Service) SetReviewAfter(ctx context.Context, id int64, reviewAfter string) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return errors.New("memory service not initialized")
	}
	res, err := s.store.DB.ExecContext(ctx, `
		UPDATE observations SET review_after = ?
		WHERE id = ? AND project = ? AND deleted_at IS NULL`,
		reviewAfter, id, s.projectID,
	)
	if err != nil {
		return fmt.Errorf("set review_after: %w", err)
	}
	if _, err := res.RowsAffected(); err != nil {
		return err
	}
	return nil
}
