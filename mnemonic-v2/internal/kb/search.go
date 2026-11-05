package kb

// search.go — the raw FTS power path (008 §3 kb_search): more rows than
// kb_query's stage-1 default, longer snippets. Same index (kb_pages_fts),
// same BM25 ranking — the difference is the shape, not the retrieval.

import (
	"context"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// SearchResult is one raw-FTS row.
type SearchResult struct {
	RelPath string  `json:"rel_path"`
	Title   string  `json:"title"`
	Type    string  `json:"type"`
	Snippet string  `json:"snippet"`
	Score   float64 `json:"score"`
}

// Search runs the raw FTS over kb_pages_fts (008 §3 kb_search). The snippet
// is a window around the first matched term (the whole leading body when no
// term is found). limit <= 0 → 25 (the power path's default; kb_query's
// stage-1 default is 10).
func Search(ctx context.Context, st store.Store, query, project string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 25
	}
	sql := `SELECT p.rel_path, p.title, p.type, p.body,
	       MAX(bm25(kb_pages_fts, 1.0, 1.0, 1.0, 1.0), 1.0) AS score
	FROM kb_pages_fts
	JOIN kb_pages p ON p.id = kb_pages_fts.rowid
	WHERE kb_pages_fts MATCH ? AND p.project = ? AND p.deleted_at IS NULL
	ORDER BY MAX(bm25(kb_pages_fts, 1.0, 1.0, 1.0, 1.0), 1.0) DESC
	LIMIT ?`
	rows, err := st.Query(ctx, sql, query, project, limit)
	if err != nil {
		return nil, err
	}
	terms := strings.Fields(query)
	out := make([]SearchResult, 0, len(rows))
	for _, r := range rows {
		body := strVal(r, 3)
		out = append(out, SearchResult{
			RelPath: strVal(r, 0),
			Title:   strVal(r, 1),
			Type:    strVal(r, 2),
			Snippet: snippetOf(body, terms, 300),
			Score:   fltVal(r, 4),
		})
	}
	return out, nil
}

// snippetOf builds a ~maxLen-character window around the first occurrence of
// any query term (case-insensitive); the body's leading characters when none
// is present. " … " marks a trimmed window.
func snippetOf(body string, terms []string, maxLen int) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	lower := strings.ToLower(body)
	pos := -1
	for _, t := range terms {
		if i := strings.Index(lower, strings.ToLower(t)); i >= 0 && (pos < 0 || i < pos) {
			pos = i
		}
	}
	if pos < 0 {
		if len(body) <= maxLen {
			return body
		}
		return body[:maxLen] + " …"
	}
	start := pos - maxLen/4
	if start < 0 {
		start = 0
	}
	end := pos + maxLen*3/4
	if end > len(body) {
		end = len(body)
	}
	s := body[start:end]
	prefix, suffix := "", ""
	if start > 0 {
		prefix = "… "
	}
	if end < len(body) {
		suffix = " …"
	}
	return prefix + s + suffix
}
