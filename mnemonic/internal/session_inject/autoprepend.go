package session_inject

import (
	"context"
	"database/sql"
	"errors"

	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
)

// AutoPrepend finds the most recently ended session for the project and
// returns its L1 summary; a project with no ended session returns "".
func AutoPrepend(ctx context.Context, mem *memory.Service, projectID string, maxTokens int) (string, error) {
	sessionID, err := mem.LatestCompletedSession(ctx, projectID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return DistillSummary(ctx, mem, sessionID, maxTokens)
}
