// Package search provides BM25-ranked FTS over indexed code chunks. This file
// adds structured ADR search: FTS over the chunks_fts table filtered to the
// 04-adr-*.md decision records, with frontmatter (status/supersedes/date) and a
// one-hop supersession chain. The agent reads the full body via code_read.
package search

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const defaultAdrSearchLimit = 10

// AdrHit is one ADR search result: the FTS-ranked file plus its parsed
// frontmatter and a one-hop supersession pointer.
type AdrHit struct {
	Path        string  `json:"path"`
	Title       string  `json:"title"`
	Status      string  `json:"status"`
	Supersedes  string  `json:"supersedes"`
	Date        string  `json:"date"`
	Snippet     string  `json:"snippet"`
	Score       float64 `json:"score"`
	SupersededBy string `json:"supersededBy,omitempty"`
}

// AdrSearch runs FTS over chunks_fts, filters hits to **/04-adr-*.md, parses
// each ADR's frontmatter (status/supersedes/date + H1 title), de-dupes to one
// hit per file, and follows the supersession chain one hop (a superseded ADR is
// flagged with the superseding ADR's path).
func AdrSearch(db *sql.DB, query string, limit int) ([]AdrHit, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	ftsQuery := buildCodeFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = defaultAdrSearchLimit
	}
	// Over-fetch raw chunk hits so that, after filtering to ADR files and
	// de-duplicating to one hit per file, we can still fill `limit`.
	rows, err := db.Query(`
		SELECT f.path, c.text, bm25(chunks_fts) AS rank
		FROM chunks c
		INNER JOIN chunks_fts ON chunks_fts.rowid = c.id
		INNER JOIN files f ON f.id = c.file_id
		WHERE chunks_fts MATCH ?
		ORDER BY rank
		LIMIT ?`,
		ftsQuery, limit*3,
	)
	if err != nil {
		return nil, fmt.Errorf("adr search: %w", err)
	}
	defer rows.Close()

	type raw struct {
		path    string
		snippet string
		score   float64
	}
	byFile := map[string]*raw{}
	order := []string{}
	for rows.Next() {
		var r raw
		var rank float64
		if err := rows.Scan(&r.path, &r.snippet, &rank); err != nil {
			return nil, fmt.Errorf("scan adr hit: %w", err)
		}
		if !isAdrPath(r.path) {
			continue
		}
		r.score = -rank
		if _, seen := byFile[r.path]; !seen {
			byFile[r.path] = &r
			order = append(order, r.path)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate adr hits: %w", err)
	}
	if len(order) == 0 {
		return nil, nil
	}

	hits := make([]AdrHit, 0, len(order))
	for _, p := range order {
		if len(hits) >= limit {
			break
		}
		r := byFile[p]
		body := strings.TrimSpace(r.snippet)
		title, fm := parseAdrFrontmatter(body)
		h := AdrHit{
			Path:    p,
			Title:   title,
			Snippet: snippet(body),
			Score:   r.score,
		}
		if v, ok := fm["status"]; ok {
			h.Status = v
		}
		if v, ok := fm["date"]; ok {
			h.Date = v
		}
		if v, ok := fm["supersedes"]; ok && v != "none" && v != "" {
			h.Supersedes = v
		}
		// One-hop supersession: if another matched ADR names this file's number
		// as its supersedes target, flag this ADR as superseded by that file.
		if m := adrNum.FindStringSubmatch(filepath.Base(p)); m != nil {
			num := m[1]
			for other, or := range byFile {
				if other == p {
					continue
				}
				if fmSupersedesNumber(or.snippet, num) {
					h.SupersededBy = other
					break
				}
			}
		}
		hits = append(hits, h)
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	return hits, nil
}

// isAdrPath reports whether a repo-relative path is an ADR record.
func isAdrPath(path string) bool {
	base := filepath.Base(filepath.ToSlash(path))
	return adrFileRe.MatchString(base)
}

var (
	adrFileRe = regexp.MustCompile(`^04-adr-\d{4}-.+\.md$`)
	adrNum    = regexp.MustCompile(`^04-adr-(\d{4})-`)
)

// parseAdrFrontmatter extracts the H1 title and the frontmatter map (status,
// supersedes, date) from an ADR body. The H1 is the first `# ` line; the
// frontmatter is the block between the first `---` and the second `---` after
// the H1. Returns ("", nil) when the body is empty.
func parseAdrFrontmatter(body string) (string, map[string]string) {
	lines := strings.Split(body, "\n")
	var title string
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "# ") {
			title = strings.TrimSpace(strings.TrimPrefix(t, "# "))
			break
		}
	}
	fm := map[string]string{}
	inFM := false
	fmStart := -1
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if t == "---" {
			if !inFM {
				inFM = true
				fmStart = i
				continue
			}
			// Closing delimiter: parse the keys between fmStart+1 and i.
			for j := fmStart + 1; j < i; j++ {
				kv := strings.SplitN(strings.TrimSpace(lines[j]), ":", 2)
				if len(kv) != 2 {
					continue
				}
				key := strings.ToLower(strings.TrimSpace(kv[0]))
				val := strings.Trim(strings.TrimSpace(kv[1]), `"`)
				fm[key] = val
			}
			break
		}
	}
	if fmStart == -1 && inFM {
		// Unterminated frontmatter (defensive): parse to EOF.
		for j := fmStart + 1; j < len(lines); j++ {
			kv := strings.SplitN(strings.TrimSpace(lines[j]), ":", 2)
			if len(kv) != 2 {
				continue
			}
			key := strings.ToLower(strings.TrimSpace(kv[0]))
			val := strings.Trim(strings.TrimSpace(kv[1]), `"`)
			fm[key] = val
		}
	}
	return title, fm
}

// fmSupersedesNumber reports whether the frontmatter of body names `num` as
// its supersedes target (a "supersedes: 04-adr-NNNN" or "supersedes: NNNN").
func fmSupersedesNumber(body, num string) bool {
	_, fm := parseAdrFrontmatter(body)
	v, ok := fm["supersedes"]
	if !ok {
		return false
	}
	v = strings.TrimSpace(v)
	return v == num || strings.Contains(v, num)
}

// snippet trims a chunk to a short preview for the hit.
func snippet(text string) string {
	t := strings.Join(strings.Fields(text), " ")
	const max = 300
	if len(t) <= max {
		return t
	}
	return t[:max] + "…"
}
