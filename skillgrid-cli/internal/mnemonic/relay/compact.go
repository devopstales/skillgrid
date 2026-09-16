package relay

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CompactResult is the result of a thin knowledge compact (change 006, step
// 03).
type CompactResult struct {
	// KnowledgePath is the absolute path of the refreshed KNOWLEDGE.md.
	KnowledgePath string
	// Empty is true when the compact ran with empty/missing knowledge inputs
	// (warn + continue): a minimal KNOWLEDGE.md was written, not an error.
	Empty bool
}

// CompactKnowledge performs a THIN refresh of projectRoot/.skillgrid/.cleave/
// KNOWLEDGE.md from the handoff inputs / session notes ONLY. It has NO Fact
// Memory dependency: the relay does not read the Fact Memory store, so a compact
// succeeds on a session with handoff inputs but no Fact Memory.
//
// The knowledge body is sourced, in priority order, from:
//  1. the existing .cleave/KNOWLEDGE.md (the bundle's knowledge section), or
//  2. the session handoffs' context_summary notes (joined), or
//  3. a minimal placeholder (empty/missing inputs → warn + continue).
//
// The compact never errors on empty/missing knowledge inputs: it writes an
// empty or minimal KNOWLEDGE.md and reports CompactResult.Empty=true. It
// returns the absolute path of the refreshed file.
func CompactKnowledge(ctx context.Context, db Store, projectID, projectRoot string) (CompactResult, error) {
	if strings.TrimSpace(projectRoot) == "" {
		return CompactResult{}, fmt.Errorf("relay: project root is required")
	}

	body, empty := gatherKnowledge(ctx, db, projectID, projectRoot)

	dir := BundleDir(projectRoot)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return CompactResult{}, fmt.Errorf("relay: create cleave dir: %w", err)
	}
	p := filepath.Join(dir, FileKnowledge)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		return CompactResult{}, fmt.Errorf("relay: write %s: %w", FileKnowledge, err)
	}
	return CompactResult{KnowledgePath: p, Empty: empty}, nil
}

// gatherKnowledge assembles the KNOWLEDGE.md body from the handoff inputs /
// session notes only (no Fact Memory). It returns the body and whether the
// inputs were empty/missing (warn + continue → minimal file). The caller's ctx
// is forwarded to the store query so a deadline/cancellation reaches it.
func gatherKnowledge(ctx context.Context, db Store, projectID, projectRoot string) (string, bool) {
	// 1. The existing .cleave/KNOWLEDGE.md (the bundle's knowledge section).
	if b, err := os.ReadFile(filepath.Join(BundleDir(projectRoot), FileKnowledge)); err == nil {
		if s := strings.TrimSpace(string(b)); s != "" {
			return string(b), false
		}
	}

	// 2. The session handoffs' context_summary notes (session notes), joined.
	if db != nil && strings.TrimSpace(projectID) != "" {
		if notes := contextNotes(ctx, db, projectID); len(notes) > 0 {
			return strings.Join(notes, "\n"), false
		}
	}

	// 3. Empty/missing inputs → a minimal file (warn + continue, not an error).
	return minimalKnowledge(), true
}

// contextNotes reads the non-empty context_summary notes of the project's
// session handoffs, newest first. It forwards the caller's ctx to the query.
func contextNotes(ctx context.Context, db Store, projectID string) []string {
	rows, err := db.QueryContext(ctx, `
		SELECT context_summary FROM session_handoffs
		WHERE project = ? AND context_summary IS NOT NULL
		ORDER BY created_at DESC, id DESC`, projectID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var notes []string
	for rows.Next() {
		var s sql.NullString
		if err := rows.Scan(&s); err != nil {
			continue
		}
		if s.Valid {
			if v := strings.TrimSpace(s.String); v != "" {
				notes = append(notes, v)
			}
		}
	}
	return notes
}

// minimalKnowledge is the KNOWLEDGE.md body written when the compact has no
// knowledge inputs (empty/missing). It is deliberately minimal: a title plus a
// note that no knowledge was available to fold in.
func minimalKnowledge() string {
	return fmt.Sprintf("# KNOWLEDGE\n\n_No knowledge recorded yet (thin compact, no Fact Memory)._\n\n_compacted at %s (UTC) — handoff inputs / session notes only._\n",
		time.Now().UTC().Format(time.RFC3339))
}
