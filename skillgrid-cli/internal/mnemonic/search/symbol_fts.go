// Package search: Identifier-Aware FTS over the symbols table.
//
// symbol_fts feeds the existing FTS5 table (created by the 011 migration).
// The unicode61 tokenizer splits camelCase/snake_case identifiers into
// lowercase word tokens, so a query for "parseConfig" matches a symbol
// stored as "parse_config" (both tokenize to parse config) and vice versa.
package search

import (
	"database/sql"
	"fmt"
	"strings"
	"unicode"
)

// SymbolHit is one identifier-FTS match over an indexed symbol.
type SymbolHit struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	QualifiedName string  `json:"qualified_name"`
	Kind          string  `json:"kind"`
	Language      string  `json:"language"`
	Signature     string  `json:"signature"`
	Path          string  `json:"path"`
	StartLine     int     `json:"start_line"`
	EndLine       int     `json:"end_line"`
	Score         float64 `json:"score"`
}

// SplitIdentifier breaks an identifier into FTS5 unicode61-compatible word
// tokens by splitting on non-alphanumeric boundaries (underscore, dash, dot)
// and at lower->upper / digit->letter case transitions. "parseConfig" ->
// ["parse","config"]; "parse_config" -> ["parse","config"]; "HTTPServer" ->
// ["http","server"]; "parse2" -> ["parse2"].
func SplitIdentifier(s string) []string {
	var out []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			out = append(out, strings.ToLower(string(cur)))
			cur = cur[:0]
		}
	}
	for i, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()
			continue
		}
		if i > 0 {
			prev := rune(s[i-1])
			// Split at any letter<->digit boundary so both "parse2" and
			// "parse" tokenization stay discoverable (parse2 and parse).
			if (unicode.IsLetter(prev) && !unicode.IsLetter(r)) ||
				(!unicode.IsLetter(prev) && unicode.IsLetter(r)) {
				flush()
			}
		}
		cur = append(cur, r)
	}
	flush()
	return out
}

// IdentifierQuery builds a MATCH expression that finds symbols whose
// identifier tokens overlap the query. Each query word is split into
// identifier tokens; tokens are OR-joined so any shared token matches.
// Returns "" for an empty query.
func IdentifierQuery(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return ""
	}
	seen := map[string]bool{}
	var tokens []string
	for _, word := range strings.Fields(q) {
		for _, tok := range SplitIdentifier(word) {
			if len(tok) == 0 || seen[tok] {
				continue
			}
			seen[tok] = true
			tokens = append(tokens, `"`+tok+`"`)
		}
	}
	return strings.Join(tokens, " OR ")
}

// SymbolFTS runs identifier-aware FTS over indexed symbols. A query of a
// single identifier (or any of its camel/snake spellings) finds the symbol.
// limit <= 0 uses the default.
func SymbolFTS(db *sql.DB, query string, limit int) ([]SymbolHit, error) {
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	m := IdentifierQuery(query)
	if m == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = defaultCodeSearchLimit
	}
	rows, err := db.Query(`
		SELECT s.id, s.name, s.qualified_name, s.kind, s.language, s.signature,
		       f.path, s.start_line, s.end_line, bm25(symbol_fts) AS rank
		FROM symbols s
		INNER JOIN symbol_fts ON symbol_fts.rowid = s.id
		INNER JOIN files f ON f.id = s.file_id
		WHERE symbol_fts MATCH ?
		ORDER BY rank
		LIMIT ?`, m, limit)
	if err != nil {
		return nil, fmt.Errorf("symbol fts: %w", err)
	}
	defer rows.Close()
	var hits []SymbolHit
	for rows.Next() {
		var h SymbolHit
		var rank float64
		if err := rows.Scan(&h.ID, &h.Name, &h.QualifiedName, &h.Kind, &h.Language, &h.Signature, &h.Path, &h.StartLine, &h.EndLine, &rank); err != nil {
			return nil, fmt.Errorf("scan symbol hit: %w", err)
		}
		h.Score = -rank
		hits = append(hits, h)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate symbol hits: %w", err)
	}
	return hits, nil
}
