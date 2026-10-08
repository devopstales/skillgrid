package context_harness

import (
	"database/sql"
	"errors"
	"sort"
	"strings"
)

// RRFK is the constant in the reciprocal-rank-fusion formula score += 1/(K+rank).
// It is intentionally distinct from hybrid.RRFK so this package stays free of a
// dependency on the code-optimised hybrid package; both are 60 by design.
const RRFK = 60

// sourceSandbox and sourceIndexed are the Source labels carried by CtxResult.
const (
	sourceSandbox = "sandbox"
	sourceIndexed = "indexed"
)

// CtxResult is one fused result row returned by CtxSearch. Source records which
// leg the row is reported from (sandbox preferred over indexed when a candidate
// matches both); ToolOrKind is the sandbox tool name or the indexed file kind;
// Snippet is the first 200 chars of the matched text; Score is the fused
// reciprocal-rank value; Rank is the 1-based position in the final fused
// (descending score) order.
type CtxResult struct {
	Source     string
	ToolOrKind string
	Name       string
	Snippet    string
	Score      float64
	Rank       int
}

// CtxSearch is the thin wrapper the CLI and MCP surface call. It delegates to
// CtxQuery; kept as a named entry point so callers depend on a stable symbol
// rather than the query internals.
func CtxSearch(db *sql.DB, query string, limit int) ([]CtxResult, error) {
	return CtxQuery(db, query, limit)
}

// CtxQuery runs the two-leg RRF fusion. Leg 1 queries sandbox tool outputs
// (tool_outputs_fts); leg 2 queries indexed code files (indexed_files_fts).
//
// Candidates are fused on a shared identity key so that a candidate matched by
// BOTH legs accumulates: score += 1/(K+rank) per matching leg. A cross-leg hit
// therefore outranks a single-leg hit. Each leg is fail-open: a leg whose table
// is absent or whose MATCH errors returns an empty leg instead of failing the
// whole query, so the other leg's results are still returned.
//
// Results are sorted by fused score descending and capped to limit.
func CtxQuery(db *sql.DB, query string, limit int) ([]CtxResult, error) {
	if db == nil {
		return nil, errors.New("ctx query: nil db")
	}
	if limit <= 0 {
		limit = 20
	}
	matches := sqlQuoteFTS(query)

	sandboxHits := legSandbox(db, matches)
	indexedHits := legIndexed(db, matches)

	type fused struct {
		key        string
		score      float64
		toolOrKind string
		name       string
		snippet    string
		source     string
	}
	m := make(map[string]*fused)
	get := func(k string) *fused {
		f, ok := m[k]
		if !ok {
			// New entries start with a neutral source so the indexed loop can
			// distinguish "sandbox already claimed this key" from "indexed-only".
			f = &fused{key: k}
			m[k] = f
		}
		return f
	}

	for _, h := range sandboxHits {
		k := candidateKey(h.name)
		f := get(k)
		f.score += 1.0 / (RRFK + float64(h.rank))
		// Sandbox is processed first and always claims the reported
		// source/tool/snippet for its key (preferred over indexed).
		f.toolOrKind = h.toolOrKind
		f.name = h.name
		f.snippet = h.snippet
		f.source = sourceSandbox
	}
	for _, h := range indexedHits {
		k := candidateKey(h.name)
		f := get(k)
		f.score += 1.0 / (RRFK + float64(h.rank))
		// Only fill metadata when sandbox did NOT claim this key (indexed-only
		// candidate). A sandbox-claimed key keeps its sandbox metadata even if
		// indexed also matched (cross-leg hit reports as sandbox).
		if f.source == "" {
			f.toolOrKind = h.toolOrKind
			f.name = h.name
			f.snippet = h.snippet
			f.source = sourceIndexed
		}
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys) // deterministic order for equal scores
	sort.SliceStable(keys, func(i, j int) bool {
		return m[keys[i]].score > m[keys[j]].score
	})

	if len(keys) > limit {
		keys = keys[:limit]
	}

	results := make([]CtxResult, 0, len(keys))
	for i, k := range keys {
		f := m[k]
		results = append(results, CtxResult{
			Source:     f.source,
			ToolOrKind: f.toolOrKind,
			Name:       f.name,
			Snippet:    f.snippet,
			Score:      f.score,
			Rank:       i + 1,
		})
	}
	return results, nil
}

// legHit is one ranked hit from a single leg. name is the candidate identity
// shared across legs (sandbox tool name / indexed file name); rank is the
// 1-based FTS rank within that leg.
type legHit struct {
	name       string
	rank       int
	toolOrKind string
	snippet    string
}

// legSandbox queries tool_outputs_fts. Fail-open: any error (absent table,
// MATCH failure) returns an empty slice.
func legSandbox(db *sql.DB, matches string) []legHit {
	const q = `
SELECT f.rowid, t.tool_name, t.output
FROM tool_outputs_fts f
JOIN tool_outputs t ON t.id = f.rowid
WHERE tool_outputs_fts MATCH ?
ORDER BY rank
LIMIT 200`
	rows, err := db.Query(q, matches)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var out []legHit
	rank := 0
	for rows.Next() {
		var id int64
		var tool, output string
		if err := rows.Scan(&id, &tool, &output); err != nil {
			return out
		}
		rank++
		out = append(out, legHit{name: tool, rank: rank, toolOrKind: tool, snippet: snippet(output)})
	}
	return out
}

// legIndexed queries indexed_files_fts. Fail-open: any error returns an empty
// slice.
func legIndexed(db *sql.DB, matches string) []legHit {
	const q = `
SELECT f.rowid, i.name, i.kind, i.content
FROM indexed_files_fts f
JOIN indexed_files i ON i.id = f.rowid
WHERE indexed_files_fts MATCH ?
ORDER BY rank
LIMIT 200`
	rows, err := db.Query(q, matches)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var out []legHit
	rank := 0
	for rows.Next() {
		var id int64
		var name, kind, content string
		if err := rows.Scan(&id, &name, &kind, &content); err != nil {
			return out
		}
		rank++
		out = append(out, legHit{name: name, rank: rank, toolOrKind: kind, snippet: snippet(content)})
	}
	return out
}

// candidateKey normalises a candidate name to a stable fusion key (lowercase).
func candidateKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// snippet returns the first 200 chars of text (trimmed).
func snippet(text string) string {
	if len(text) <= 200 {
		return strings.TrimSpace(text)
	}
	return strings.TrimSpace(text[:200])
}

// sqlQuoteFTS builds an FTS5 MATCH expression from a free-text query: quote
// each whitespace-separated token and join with implicit AND (FTS5 default).
func sqlQuoteFTS(query string) string {
	fields := strings.Fields(strings.ReplaceAll(query, `"`, `""`))
	if len(fields) == 0 {
		return `""`
	}
	quoted := make([]string, len(fields))
	for i, f := range fields {
		quoted[i] = `"` + f + `"`
	}
	return strings.Join(quoted, " ")
}
