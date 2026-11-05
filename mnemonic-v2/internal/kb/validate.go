package kb

import (
	"context"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// ValidateResult is the outcome of a single pin-check.
type ValidateResult struct {
	RelPath string
	OK      bool
	Reason  string
}

// Validate runs the pin-check for one page: the last recorded pin state
// (fresh / stale / unresolved) against the current body hash. A grounded page
// is OK only when its baseline pin is fresh; unknown/unresolved pins fail.
func Validate(ctx context.Context, st store.Store, relPath, project string) (*ValidateResult, error) {
	res := &ValidateResult{RelPath: relPath, OK: false, Reason: "not_found"}
	rows, err := st.Query(ctx,
		`SELECT id, rel_path, type, status, body_hash FROM kb_pages WHERE project = ? AND rel_path = ?`,
		project, relPath)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return res, nil
	}
	pageID := int64(0)
	bodyHash := ""
	if v, ok := rows[0][0].(int64); ok {
		pageID = v
	}
	if h, ok := rows[0][4].(string); ok {
		bodyHash = h
	}

	baseRows, err := st.Query(ctx,
		`SELECT source_fingerprint, page_version FROM kb_page_baselines WHERE project = ? AND rel_path = ?`,
		project, relPath)
	if err != nil {
		return nil, err
	}
	_, _ = st.Query(ctx, `SELECT subject, literal FROM kb_claims WHERE page_id = ?`, pageID)
	pin := "unresolved"
	if len(baseRows) > 0 {
		if sf, ok := baseRows[0][0].(string); ok && sf == bodyHash {
			pin = "fresh"
		}
	}
	if strings.EqualFold(strVal2(rows[0], 2), "decision") || strings.EqualFold(strVal2(rows[0], 2), "invariant") || strings.EqualFold(strVal2(rows[0], 2), "workflow") {
		switch pin {
		case "fresh":
			res.OK = true
			res.Reason = "fresh"
		case "stale":
			res.OK = false
			res.Reason = "stale"
		default:
			res.OK = false
			res.Reason = "unresolved"
		}
	}
	return res, nil
}

func strVal2(r []any, i int) string {
	if i >= len(r) {
		return ""
	}
	switch v := r[i].(type) {
	case string:
		return v
	case []byte:
		return string(v)
	}
	return ""
}
