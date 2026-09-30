package memory

import (
	"context"
)

// LatestCompletedSession returns the id of the most recently ended session for
// the project (status 'ended', ordered by ended_at DESC with rowid as a stable
// tiebreak). It surfaces sql.ErrNoRows when the project has no ended session.
func (s *Service) LatestCompletedSession(ctx context.Context, projectID string) (string, error) {
	var id string
	err := s.store.DB.QueryRowContext(ctx,
		`SELECT id FROM sessions
		 WHERE project = ? AND status = 'ended'
		 ORDER BY ended_at DESC, rowid DESC
		 LIMIT 1`,
		projectID,
	).Scan(&id)
	return id, err
}
