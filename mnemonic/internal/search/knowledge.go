// Package search adds structured knowledge-doc search: FTS over chunks_fts
// filtered to a skillgrid artifact type (report/briefing/blueprint/findings),
// with per-type quote-block metadata extraction. The agent reads the full body
// via code_read. This is the shared engine behind report_search,
// briefing_search, blueprint_search, and findings_search.
package search

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const defaultKnowledgeLimit = 10

// KnowledgeType names a skillgrid artifact type searched by knowledgeSearch.
type KnowledgeType string

const (
	KnowledgeReport    KnowledgeType = "report"
	KnowledgeBriefing  KnowledgeType = "briefing"
	KnowledgeBlueprint KnowledgeType = "blueprint"
	KnowledgeFindings  KnowledgeType = "findings"
)

// KnowledgeHit is one knowledge-doc search result: the FTS-ranked file plus the
// type-specific metadata extracted from its head.
type KnowledgeHit struct {
	Path    string            `json:"path"`
	Title   string            `json:"title"`
	Snippet string            `json:"snippet"`
	Score   float64           `json:"score"`
	Meta    map[string]string `json:"meta,omitempty"`
}

// KnowledgeSearch runs FTS over chunks_fts, filters hits to files whose base
// name is one of the given bases, de-dupes to one hit per file, extracts the
// per-type metadata from each file's head, and returns them ranked.
func KnowledgeSearch(db *sql.DB, query string, limit int, kind KnowledgeType, bases []string, extract func(string) map[string]string) ([]KnowledgeHit, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	ftsQuery := buildCodeFTSQuery(query)
	if ftsQuery == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = defaultKnowledgeLimit
	}
	// Over-fetch raw chunk hits so that, after filtering to the target files
	// and de-duplicating to one hit per file, we can still fill `limit`.
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
		return nil, fmt.Errorf("knowledge search: %w", err)
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
			return nil, fmt.Errorf("scan knowledge hit: %w", err)
		}
		if !baseIn(r.path, bases) {
			continue
		}
		r.score = -rank
		if _, seen := byFile[r.path]; !seen {
			byFile[r.path] = &r
			order = append(order, r.path)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate knowledge hits: %w", err)
	}
	if len(order) == 0 {
		return nil, nil
	}

	hits := make([]KnowledgeHit, 0, len(order))
	for _, p := range order {
		if len(hits) >= limit {
			break
		}
		r := byFile[p]
		body := strings.TrimSpace(r.snippet)
		title := extractTitle(body)
		h := KnowledgeHit{
			Path:    p,
			Title:   title,
			Snippet: snippet(body),
			Score:   r.score,
			Meta:    extract(body),
		}
		hits = append(hits, h)
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	return hits, nil
}

// ReportSearch returns structured QA report hits (gate/generated/change).
func ReportSearch(db *sql.DB, query string, limit int) ([]KnowledgeHit, error) {
	return KnowledgeSearch(db, query, limit, KnowledgeReport, []string{"report.md"}, extractReportMeta)
}

// BriefingSearch returns structured briefing hits (status/date).
func BriefingSearch(db *sql.DB, query string, limit int) ([]KnowledgeHit, error) {
	return KnowledgeSearch(db, query, limit, KnowledgeBriefing, []string{"briefing.md"}, extractBriefingMeta)
}

// BlueprintSearch returns structured blueprint hits (status/tier).
func BlueprintSearch(db *sql.DB, query string, limit int) ([]KnowledgeHit, error) {
	return KnowledgeSearch(db, query, limit, KnowledgeBlueprint, []string{"blueprint.md"}, extractBlueprintMeta)
}

// FindingsSearch returns structured research findings hits (status/mode/date/type).
func FindingsSearch(db *sql.DB, query string, limit int) ([]KnowledgeHit, error) {
	return KnowledgeSearch(db, query, limit, KnowledgeFindings, []string{"findings.md"}, extractFindingsMeta)
}

// baseIn reports whether the base name of path equals one of bases.
func baseIn(path string, bases []string) bool {
	base := filepath.Base(filepath.ToSlash(path))
	for _, b := range bases {
		if base == b {
			return true
		}
	}
	return false
}

// extractTitle returns the first `# ` heading of a body, or "".
func extractTitle(body string) string {
	for _, l := range strings.Split(body, "\n") {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(t, "# "))
		}
	}
	return ""
}

// kbQuoteRe matches a quote-block metadata line whose KEY is bolded:
//   > **Gate:** PASS          (report.md)
//   > **STATUS:** draft       (briefing.md)
//   > **Date:** 2026-10-09    (findings.md)
// The bold `**` is required so plain prose quote lines ("> moves to ...") are
// not mistaken for metadata. Keys are alphanumeric only so a trailing
// parenthetical stays in the value, and values may contain colons.
var kbQuoteRe = regexp.MustCompile(`(?m)^>\s*\*\*([A-Za-z]+)\s*:\s*\*\*\s*(.+?)\s*$`)

// quoteBlockMeta parses quote-block metadata lines from the head of a body.
// Keys are lower-cased and trimmed; values have surrounding bold/backticks
// stripped. Returns an empty map when the body has no metadata lines.
func quoteBlockMeta(body string) map[string]string {
	m := map[string]string{}
	for _, mm := range kbQuoteRe.FindAllStringSubmatch(body, -1) {
		key := strings.ToLower(strings.TrimSpace(strings.Trim(mm[1], ":* ")))
		val := cleanMetaValue(mm[2])
		if key == "" || val == "" {
			continue
		}
		m[key] = val
	}
	return m
}

// trailingSourceTagRe matches a lowercase source tag in parens at the very
// end of a value, e.g. " (qa)" or " (qa re-verify)". It is anchored to the
// end so interior parentheticals ("moves to ... at ship", "research (single
// pass)") are preserved.
var trailingSourceTagRe = regexp.MustCompile(` \(([a-z][a-z -]*)\)$`)

// cleanMetaValue strips surrounding emphasis/backticks and a trailing
// source tag such as " (qa)". Interior parentheticals are preserved.
func cleanMetaValue(v string) string {
	v = strings.TrimSpace(v)
	v = strings.Trim(v, "`")
	v = strings.TrimPrefix(v, "**")
	v = strings.TrimSuffix(v, "**")
	v = trailingSourceTagRe.ReplaceAllString(v, "")
	return strings.TrimSpace(v)
}

// extractReportMeta pulls gate/generated/change from a report.md head.
func extractReportMeta(body string) map[string]string {
	m := quoteBlockMeta(body)
	out := map[string]string{}
	if v, ok := m["gate"]; ok {
		out["gate"] = v
	}
	if v, ok := m["generated"]; ok {
		out["generated"] = v
	}
	if v, ok := m["change"]; ok {
		out["change"] = v
	}
	return out
}

// extractBriefingMeta pulls status/date from a briefing.md head.
func extractBriefingMeta(body string) map[string]string {
	m := quoteBlockMeta(body)
	out := map[string]string{}
	if v, ok := m["status"]; ok {
		out["status"] = v
	}
	if v, ok := m["date"]; ok {
		out["date"] = v
	}
	return out
}

// extractBlueprintMeta pulls status/tier from a blueprint.md head.
func extractBlueprintMeta(body string) map[string]string {
	m := quoteBlockMeta(body)
	out := map[string]string{}
	if v, ok := m["status"]; ok {
		out["status"] = v
	}
	if v, ok := m["tier"]; ok {
		out["tier"] = v
	}
	return out
}

// extractFindingsMeta pulls status/mode/date/type from a findings.md head.
func extractFindingsMeta(body string) map[string]string {
	m := quoteBlockMeta(body)
	out := map[string]string{}
	for _, k := range []string{"status", "mode", "date", "type"} {
		if v, ok := m[k]; ok {
			out[k] = v
		}
	}
	return out
}
