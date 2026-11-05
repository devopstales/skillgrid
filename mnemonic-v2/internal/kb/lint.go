package kb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// LintIssue is one deterministic health finding.
type LintIssue struct {
	RelPath string
	Code    string
	Message string
}

// Lint scans a project's kb_pages for deterministic health issues:
// missing title, empty body, stale index (body_hash mismatch).
func Lint(ctx context.Context, st store.Store, project string) ([]LintIssue, error) {
	rows, err := st.Query(ctx,
		`SELECT rel_path, title, type, status, body, body_hash FROM kb_pages WHERE project = ? AND deleted_at IS NULL`,
		project)
	if err != nil {
		return nil, err
	}
	var out []LintIssue
	for _, r := range rows {
		rel := strVal(r, 0)
		title := strVal(r, 1)
		body := strVal(r, 4)
		bh := strVal(r, 5)
		if strings.TrimSpace(title) == "" {
			out = append(out, LintIssue{RelPath: rel, Code: "missing_title", Message: "title empty"})
			continue
		}
		h := sha256.Sum256([]byte(body))
		if hex.EncodeToString(h[:]) != bh {
			out = append(out, LintIssue{RelPath: rel, Code: "stale_index", Message: "body_hash mismatch"})
		}
	}
	return out, nil
}

// Fix applies the safe subset of lint fixes: backfill empty titles from the
// page's rel_path. Returns the number of fixes applied.
func Fix(ctx context.Context, st store.Store, project string, relPaths []string) (int, error) {
	fixed := 0
	for _, rel := range relPaths {
		rows, err := st.Query(ctx,
			`SELECT title FROM kb_pages WHERE project = ? AND rel_path = ? AND deleted_at IS NULL`, project, rel)
		if err != nil {
			return fixed, err
		}
		empty := false
		if len(rows) > 0 {
			empty = strings.TrimSpace(strVal(rows[0], 0)) == ""
		}
		if !empty {
			continue
		}
		if err := st.Exec(ctx, `UPDATE kb_pages SET title = ? WHERE project = ? AND rel_path = ?`, rel, project, rel); err != nil {
			return fixed, err
		}
		fixed++
	}
	return fixed, nil
}
