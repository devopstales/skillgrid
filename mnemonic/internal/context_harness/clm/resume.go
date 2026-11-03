package clm

import (
	"context"
	"database/sql"
)

// Resume reconstructs the highest context_revisions row for a session and
// returns it (for anchor re-validation). A mismatch at the call site falls
// back to raw context (the caller decides). Text columns are scanned through
// sql.NullString so an explicit NULL (the migration's DEFAULT ” is not
// applied to an explicit NULL) reads as an empty string.
func Resume(ctx context.Context, db *sql.DB, sessionID string) (*Revision, error) {
	row := db.QueryRowContext(ctx,
		`SELECT revision, anchor_count, anchor_digest, messages, size_estimate, calibration_factor, withhold_ids, edit_trace
		 FROM context_revisions WHERE session_id = ? ORDER BY revision DESC LIMIT 1`, sessionID)
	var (
		r           Revision
		anchor      sql.NullString
		messages    sql.NullString
		withhold    sql.NullString
		trace       sql.NullString
		factor      float64
		anchorCount int
		sizeEst     int
	)
	err := row.Scan(&r.Revision, &anchorCount, &anchor, &messages, &sizeEst, &factor, &withhold, &trace)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.AnchorCount = anchorCount
	r.AnchorDigest = anchor.String
	r.Messages = messages.String
	r.SizeEstimate = sizeEst
	r.CalibrationFactor = factor
	r.WithholdIDs = withhold.String
	r.EditTrace = trace.String
	r.SessionID = sessionID
	return &r, nil
}
