// Package search provides BM25-ranked FTS over indexed code chunks.
package search

import (
	"database/sql"
	"fmt"
	"strings"
)

const defaultCodeSearchLimit = 20

// CodeHit is one FTS match over an indexed code chunk.
type CodeHit struct {
	Path      string  `json:"path"`
	StartLine int     `json:"start_line"`
	EndLine   int     `json:"end_line"`
	Snippet   string  `json:"snippet"`
	Score     float64 `json:"score"`
}

// CodeSearch runs BM25-ranked FTS over indexed code chunks.
func CodeSearch(db *sql.DB, query string, limit int) ([]CodeHit, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	ftsQuery := buildCodeFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = defaultCodeSearchLimit
	}
	rows, err := db.Query(`
		SELECT f.path, c.start_line, c.end_line, c.text, bm25(chunks_fts) AS rank
		FROM chunks c
		INNER JOIN chunks_fts ON chunks_fts.rowid = c.id
		INNER JOIN files f ON f.id = c.file_id
		WHERE chunks_fts MATCH ?
		ORDER BY rank
		LIMIT ?`,
		ftsQuery, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("code search: %w", err)
	}
	defer rows.Close()
	var hits []CodeHit
	for rows.Next() {
		var hit CodeHit
		var rank float64
		if err := rows.Scan(&hit.Path, &hit.StartLine, &hit.EndLine, &hit.Snippet, &rank); err != nil {
			return nil, fmt.Errorf("scan code hit: %w", err)
		}
		hit.Score = -rank
		hits = append(hits, hit)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate code hits: %w", err)
	}
	return hits, nil
}

func buildCodeFTSQuery(query string) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return ""
	}
	query = strings.ReplaceAll(query, `"`, `""`)
	return `"` + query + `"`
}
