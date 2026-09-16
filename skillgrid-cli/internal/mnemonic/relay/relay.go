package relay

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"
)

// Store is the database handle the relay reads/writes. *sql.DB satisfies it.
// The relay only ever touches the two additive 019 tables (session_handoffs /
// session_archives) — it never rewrites sessions or observations.
type Store interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	// QueryContext is used by the thin compact (step 03) to fold the session
	// handoffs' context_summary notes into KNOWLEDGE.md. *sql.DB satisfies it.
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// Handoff writes the cleave bundle under projectRoot/.skillgrid/.cleave/ and
// then records the session_handoffs row. Fail closed: the row is only written
// AFTER the three cleave files are successfully on disk, so a file-write
// failure leaves no orphan row (and WriteBundle removes any half-bundle). It
// returns the handoff_id and the three cleave file paths.
//
// handoffID is the operator-facing identifier (also the resume target). Blank
// generates one from the source session + timestamp.
func Handoff(ctx context.Context, db Store, projectID, handoffID, projectRoot string, b Bundle) (string, []string, error) {
	if db == nil {
		return "", nil, fmt.Errorf("relay: store is required")
	}
	if strings.TrimSpace(projectID) == "" {
		return "", nil, fmt.Errorf("relay: project id is required")
	}
	if strings.TrimSpace(projectRoot) == "" {
		return "", nil, fmt.Errorf("relay: project root is required")
	}
	if handoffID == "" {
		handoffID = generateHandoffID(b.SourceSession)
	}
	if strings.TrimSpace(b.NextPrompt) == "" {
		return "", nil, fmt.Errorf("relay: next_prompt is required (resume must not invent a prompt)")
	}

	paths, err := WriteBundle(projectRoot, b.SourceSession, b)
	if err != nil {
		// File write failed before any row was written → no orphan row.
		return "", nil, err
	}

	cleavePath := ".skillgrid/.cleave"
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = db.ExecContext(ctx, `
		INSERT INTO session_handoffs
			(project, handoff_id, source_session, status, cleave_path, context_summary, created_at)
		VALUES (?, ?, ?, 'pending', ?, ?, ?)`,
		projectID, handoffID, nullString(b.SourceSession), cleavePath, nullString(b.ContextSummary), now)
	if err != nil {
		// Row insert failed: roll back the just-written files so we don't
		// leave files with no row (the other orphan direction).
		for _, p := range paths {
			_ = os.Remove(p)
		}
		return "", nil, fmt.Errorf("relay: record handoff row: %w", err)
	}
	return handoffID, paths, nil
}

// Resume reads the cleave bundle for handoffID and returns the NEXT_PROMPT as
// the resume prompt. Fail closed: an unknown handoff id, or a missing /
// incomplete .cleave/ bundle, returns an error (no invented prompt). When
// archive is true and a handoff row exists, a session_archives row is recorded
// and the handoff is flipped to 'archived'; the archive_id
// (session_archives.id) is returned when present.
func Resume(ctx context.Context, db Store, projectID, handoffID, projectRoot string, archive bool) (string, string, int64, error) {
	if db == nil {
		return "", handoffID, 0, fmt.Errorf("relay: store is required")
	}
	if strings.TrimSpace(projectID) == "" {
		return "", handoffID, 0, fmt.Errorf("relay: project id is required")
	}
	if strings.TrimSpace(handoffID) == "" {
		return "", handoffID, 0, fmt.Errorf("relay: handoff_id is required")
	}

	// Confirm the id is known (fail closed on unknown id) and learn the
	// cleave path. source_session is nullable, so scan into NullString.
	var cleavePath string
	var sourceSession sql.NullString
	err := db.QueryRowContext(ctx, `
		SELECT cleave_path, source_session
		FROM session_handoffs
		WHERE project = ? AND handoff_id = ?`,
		projectID, handoffID).Scan(&cleavePath, &sourceSession)
	if err == sql.ErrNoRows {
		return "", handoffID, 0, fmt.Errorf("relay: unknown handoff id %q for project %q", handoffID, projectID)
	}
	if err != nil {
		return "", handoffID, 0, fmt.Errorf("relay: lookup handoff: %w", err)
	}

	bundle, rerr := ReadBundle(projectRoot)
	if rerr != nil {
		return "", handoffID, 0, rerr
	}
	if strings.TrimSpace(bundle.NextPrompt) == "" {
		return "", handoffID, 0, fmt.Errorf("relay: NEXT_PROMPT is empty for handoff %q (no prompt to resume)", handoffID)
	}
	prompt := bundle.NextPrompt

	if archive {
		now := time.Now().UTC().Format(time.RFC3339)
		var sessionArg any
		if sourceSession.Valid {
			sessionArg = sourceSession.String
		}
		res, aerr := db.ExecContext(ctx, `
			INSERT INTO session_archives (project, session_id, handoff_id, path, created_at)
			VALUES (?, ?, ?, ?, ?)`,
			projectID, sessionArg, handoffID, cleavePath, now)
		if aerr != nil {
			// The prompt is already valid; surface the archive error but keep
			// the prompt (archiving is the optional part).
			return prompt, handoffID, 0, fmt.Errorf("relay: record archive: %w", aerr)
		}
		id64, _ := res.LastInsertId()
		// Fail closed on the status flip: a session_archives row is already
		// written, so a failed UPDATE would leave the handoff 'pending' with a
		// live archive row — the reverse of the no-orphan invariant. Surface
		// the error so the caller sees the archive is incomplete (the archive
		// row was written but the handoff status flip failed).
		if _, uerr := db.ExecContext(ctx, `
			UPDATE session_handoffs SET status = 'archived', archived_at = ?
			WHERE project = ? AND handoff_id = ?`, now, projectID, handoffID); uerr != nil {
			return prompt, handoffID, id64,
				fmt.Errorf("relay: archived but failed to mark handoff archived: %w", uerr)
		}
		return prompt, handoffID, id64, nil
	}
	return prompt, handoffID, 0, nil
}

// HandoffCount returns the number of session_handoffs rows for projectID (0
// when none). Used by session_status (step 03).
func HandoffCount(ctx context.Context, db Store, projectID string) (int, error) {
	var n int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM session_handoffs WHERE project = ?`, projectID).Scan(&n)
	return n, err
}

func nullString(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}

func generateHandoffID(sourceSession string) string {
	base := sourceSession
	if base == "" {
		base = "handoff"
	}
	return fmt.Sprintf("%s-%d", base, time.Now().UnixMilli())
}
