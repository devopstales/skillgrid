package projectwiki

import (
	"context"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/core/store"
)

// QueryFilters narrows a projectwiki_query stage-1 retrieval.
type QueryFilters struct {
	Type         string
	Project      string
	Limit        int
	MatchMode    string
	GroundedOnly bool
}

// QueryResult is one stage-1 hit: ranked link + score (+ pin when grounded).
type QueryResult struct {
	RelPath  string
	Title    string
	Type     string
	Status   string
	Grounded bool
	Snippet  string
	Pin      string
	Score    float64
}

// Query runs the stage-1 retrieval: FTS over projectwiki_pages_fts with BM25 rank,
// joined to projectwiki_pages for projection. Ranked links + 1-line snippets only —
// full hydration is stage 2 (projectwiki_read).
func Query(ctx context.Context, st store.Store, query string, f QueryFilters) ([]QueryResult, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 10
	}
	sql := `SELECT p.rel_path, p.title, p.type, p.status, p.in_degree,
	       MAX(bm25(projectwiki_pages_fts, 1.0, 1.0, 1.0, 1.0), 1.0) AS score
	FROM projectwiki_pages_fts
	JOIN projectwiki_pages p ON p.id = projectwiki_pages_fts.rowid
	WHERE projectwiki_pages_fts MATCH ? AND p.project = ? AND p.deleted_at IS NULL
	ORDER BY MAX(bm25(projectwiki_pages_fts, 1.0, 1.0, 1.0, 1.0), 1.0) DESC
	LIMIT ?`
	rows, err := st.Query(ctx, sql, query, f.Project, limit)
	if err != nil {
		return nil, err
	}
	out := []QueryResult{}
	for _, r := range rows {
		res := QueryResult{
			RelPath: strVal(r, 0),
			Title:   strVal(r, 1),
			Type:    strVal(r, 2),
			Status:  strVal(r, 3),
			Score:   fltVal(r, 5),
		}
		res.Grounded = isGrounded(res.Type)
		res.Pin = pinFor(res)
		out = append(out, res)
	}
	return out, nil
}

// ThinThreshold is the spec 005 §6 thin cutoff: a stage-1 result with
// fewer hits than this is "thin" and carries the mem_lookup fallback
// hint (KQ-O4: the value is a measured tunable; 3 is the shipped
// default until projectwiki_eval sweeps it).
const ThinThreshold = 3

// AssessQuery computes the spec 005 §6/§7 result metadata for one
// stage-1 query: thin (resultCount < ThinThreshold) with its
// fallback hint, and degraded = "stale_index" when the derived index
// lags the vault files at root (a live page whose body_hash differs
// from the indexed row, or a vault page not yet indexed). An empty
// root skips the staleness check. The ProjectWiki never calls
// mem_lookup itself — the agent is the router (005 §6).
func AssessQuery(ctx context.Context, st store.Store, root, project string, resultCount int) (thin bool, fallback, degraded string, err error) {
	if resultCount < ThinThreshold {
		thin, fallback = true, "mem_lookup"
	}
	if root == "" {
		return thin, fallback, "", nil
	}
	_, lagged, err := indexLag(ctx, st, root, project)
	if err != nil {
		return thin, fallback, "", err
	}
	if lagged {
		return thin, fallback, "stale_index", nil
	}
	return thin, fallback, "", nil
}

func strVal(r []any, i int) string {
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

func fltVal(r []any, i int) float64 {
	if i >= len(r) {
		return 0
	}
	switch v := r[i].(type) {
	case float64:
		return v
	case int64:
		return float64(v)
	}
	return 0
}

func isGrounded(t string) bool {
	switch strings.ToLower(t) {
	case "decision", "invariant", "workflow":
		return true
	}
	return false
}

func pinFor(r QueryResult) string {
	if !r.Grounded {
		return ""
	}
	return "fresh"
}
