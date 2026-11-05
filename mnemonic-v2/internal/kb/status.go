package kb

import (
	"context"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// StatusResult is the first verb's read model: index freshness + run state.
type StatusResult struct {
	Project       string
	Indexed       int64
	Phase         string
	RunID         string
	LastFullRunAt string
	Baselines     int64
	Events        int64
}

// Status reads the KB's derived state for one project: how many pages are
// indexed, the last full-run phase/id, baseline + event counts.
func Status(ctx context.Context, st store.Store, project string) (*StatusResult, error) {
	res := &StatusResult{Project: project, Phase: "idle"}
	rows, err := st.Query(ctx, `SELECT COUNT(*) FROM kb_pages WHERE project = ? AND deleted_at IS NULL`, project)
	if err != nil {
		return nil, err
	}
	if len(rows) > 0 {
		if v, ok := rows[0][0].(int64); ok {
			res.Indexed = v
		}
	}
	rs, err := st.Query(ctx, `SELECT phase, run_id, source_fingerprint, last_full_run_at FROM kb_run_state WHERE project = ?`, project)
	if err != nil {
		return nil, err
	}
	if len(rs) > 0 {
		if p, ok := rs[0][0].(string); ok {
			res.Phase = p
		}
		if rid, ok := rs[0][1].(string); ok {
			res.RunID = rid
		}
		if l, ok := rs[0][3].(string); ok {
			res.LastFullRunAt = l
		}
	}
	bRows, err := st.Query(ctx, `SELECT COUNT(*) FROM kb_page_baselines WHERE project = ?`, project)
	if err != nil {
		return nil, err
	}
	if len(bRows) > 0 {
		if v, ok := bRows[0][0].(int64); ok {
			res.Baselines = v
		}
	}
	eRows, err := st.Query(ctx, `SELECT COUNT(*) FROM kb_events WHERE project = ?`, project)
	if err != nil {
		return nil, err
	}
	if len(eRows) > 0 {
		if v, ok := eRows[0][0].(int64); ok {
			res.Events = v
		}
	}
	return res, nil
}
