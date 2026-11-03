package clm

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Edit is the model's proposed mirror edit.
type Edit struct {
	Nonce    string
	Messages []string
}

// Revision is a persisted accepted edit.
type Revision struct {
	SessionID         string
	ProjectID         string
	Revision          int
	AnchorCount       int
	AnchorDigest      string
	Messages          string
	SizeEstimate      int
	CalibrationFactor float64
	WithholdIDs       string
	EditTrace         string
}

// ValidateRevision checks the edit nonce matches and the message sequence is legal.
func ValidateRevision(edit *Edit, expectedNonce string) error {
	if edit.Nonce != expectedNonce {
		return fmt.Errorf("nonce mismatch: got %q want %q", edit.Nonce, expectedNonce)
	}
	if len(edit.Messages) == 0 {
		return fmt.Errorf("empty message list")
	}
	return nil
}

// PersistRevision inserts a revision row.
func PersistRevision(ctx context.Context, db *sql.DB, r *Revision) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.ExecContext(ctx,
		`INSERT INTO context_revisions (id, session_id, project_id, revision, anchor_count, anchor_digest, messages, size_estimate, calibration_factor, withhold_ids, edit_trace, created_at)
		 VALUES (COALESCE((SELECT max(id) FROM context_revisions),0)+1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.SessionID, r.ProjectID, r.Revision, r.AnchorCount, r.AnchorDigest,
		r.Messages, r.SizeEstimate, r.CalibrationFactor, r.WithholdIDs, r.EditTrace, now)
	return err
}
