package memory

import (
	"context"
	"fmt"
	"time"
)

// PruneOldEvents deletes session_events rows older than days and returns the
// count deleted. Best-effort: a failure is returned but the caller (serve
// startup) logs it and continues — retention is not a session-critical path.
func (s *Service) PruneOldEvents(ctx context.Context, days int) (int64, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return 0, fmt.Errorf("memory service not initialized")
	}
	if days <= 0 {
		days = 90
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Format(time.RFC3339)
	res, err := s.store.DB.ExecContext(ctx,
		`DELETE FROM session_events WHERE timestamp < ?`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("prune old events: %w", err)
	}
	return res.RowsAffected()
}
