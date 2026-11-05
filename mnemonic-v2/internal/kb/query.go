package kb

import (
	"context"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// QueryFilters narrows a kb_query stage-1 retrieval.
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

// Query runs the stage-1 retrieval: FTS over kb_pages_fts with BM25 rank,
// joined to kb_pages for projection. Ranked links + 1-line snippets only —
// full hydration is stage 2 (kb_read).
func Query(ctx context.Context, st store.Store, query string, f QueryFilters) ([]QueryResult, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 10
	}
	sql := `SELECT p.rel_path, p.title, p.type, p.status, p.in_degree,
	       MAX(bm25(kb_pages_fts, 1.0, 1.0, 1.0, 1.0), 1.0) AS score
	FROM kb_pages_fts
	JOIN kb_pages p ON p.id = kb_pages_fts.rowid
	WHERE kb_pages_fts MATCH ? AND p.project = ? AND p.deleted_at IS NULL
	ORDER BY MAX(bm25(kb_pages_fts, 1.0, 1.0, 1.0, 1.0), 1.0) DESC
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
