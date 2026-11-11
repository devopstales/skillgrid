package projectwiki

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/devopstales/skillgrid/mnemonic/internal/core/store"
)

// indexLag counts the live vault pages at root whose indexed body_hash
// differs (or that are not indexed yet) — the stale-index measure
// shared by Status (count) and AssessQuery (lagged flag). A vault
// root that does not exist is no lag (nothing to be stale against),
// never an error.
func indexLag(ctx context.Context, st store.Store, root, project string) (int64, bool, error) {
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		return 0, false, nil
	}
	pages, err := collectPages(root)
	if err != nil {
		return 0, false, err
	}
	rows, err := st.Query(ctx,
		`SELECT rel_path, body_hash FROM projectwiki_pages WHERE project = ? AND deleted_at IS NULL`, project)
	if err != nil {
		return 0, false, err
	}
	indexed := make(map[string]string, len(rows))
	for _, r := range rows {
		indexed[strVal(r, 0)] = strVal(r, 1)
	}
	var stale int64
	for _, p := range pages {
		if h, ok := indexed[p.RelPath]; !ok || h != p.BodyHash {
			stale++
		}
	}
	return stale, stale > 0, nil
}

// StatusResult is the first verb's read model: index freshness + run state.
type StatusResult struct {
	Project       string
	Indexed       int64
	Phase         string
	RunID         string
	LastFullRunAt string
	Baselines     int64
	Events        int64
	// Blueprint T7 breakdown: page counts by type, stale count
	// (live vault file whose indexed body_hash lags), pending
	// generation jobs in the active run's plan, orphan count
	// (pages with in_degree <= 1, the lint orphan rule).
	ByType  map[string]int64 `json:"by_type,omitempty"`
	Stale   int64            `json:"stale"`
	Pending int64            `json:"pending"`
	Orphans int64            `json:"orphans"`
}

// Status reads the project wiki's derived state for one project: how many pages are
// indexed, the last full-run phase/id, baseline + event counts, and the
// T7 breakdown (by-type counts, stale vs the live vault at root,
// pending plan jobs, orphans). An empty root skips the stale count.
func Status(ctx context.Context, st store.Store, root, project string) (*StatusResult, error) {
	res := &StatusResult{Project: project, Phase: "idle", ByType: map[string]int64{}}
	rows, err := st.Query(ctx, `SELECT COUNT(*) FROM projectwiki_pages WHERE project = ? AND deleted_at IS NULL`, project)
	if err != nil {
		return nil, err
	}
	if len(rows) > 0 {
		if v, ok := rows[0][0].(int64); ok {
			res.Indexed = v
		}
	}
	typeRows, err := st.Query(ctx,
		// The orphan rule is defined once — lint.go's orphanInDegreeMax
		// (spec 007 §2); this SQL embeds that constant, not a second
		// literal.
		fmt.Sprintf(`SELECT type, COUNT(*), COALESCE(SUM(CASE WHEN in_degree <= %d THEN 1 ELSE 0 END), 0) FROM projectwiki_pages WHERE project = ? AND deleted_at IS NULL GROUP BY type`, orphanInDegreeMax), project)
	if err != nil {
		return nil, err
	}
	for _, r := range typeRows {
		if v, ok := r[1].(int64); ok {
			res.ByType[strVal(r, 0)] = v
		}
		if v, ok := r[2].(int64); ok {
			res.Orphans += v
		}
	}
	if root != "" {
		stale, _, err := indexLag(ctx, st, root, project)
		if err != nil {
			return nil, err
		}
		res.Stale = stale
	}
	if rs, err := LoadRun(ctx, st, project); err == nil && rs != nil {
		res.Phase = rs.Phase
		res.RunID = rs.RunID
		res.LastFullRunAt = rs.LastFullRunAt
		for _, job := range rs.Plan {
			if job.Status == JobPending {
				res.Pending++
			}
		}
	} else if err != nil {
		return nil, err
	}
	bRows, err := st.Query(ctx, `SELECT COUNT(*) FROM projectwiki_page_baselines WHERE project = ?`, project)
	if err != nil {
		return nil, err
	}
	if len(bRows) > 0 {
		if v, ok := bRows[0][0].(int64); ok {
			res.Baselines = v
		}
	}
	eRows, err := st.Query(ctx, `SELECT COUNT(*) FROM projectwiki_events WHERE project = ?`, project)
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
