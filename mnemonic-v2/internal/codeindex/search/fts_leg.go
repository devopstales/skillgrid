package search

import (
	"context"
	"strings"
	"unicode"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// ftsLeg is the FTS floor (spec 006 §2): BM25 over chunks_fts + symbol_fts.
// It is the floor — it always answers, even when the embedder is down.
//
// Identifier-first (the T-CI-1 fix): the v1 FTS leg matched only free text and
// did NOT match the queried identifier (the live probe returned identical hits
// for a real and a fake symbol). v2's fix: an identifier-shaped query matches
// symbol_fts.name/qualified_name first (exact/prefix identifier match ranks
// above free-text), so a real symbol is found and a fake one is not.
//
// The under? prefix (T10) filters the leg to files.path under the prefix.
//
// The leg reads through the store's RawQuery seam (TICKET-01's M-3 parked
// minor — the FTS read path until a Store method exists; the controller
// pre-approves ONE additive FTS read method, but RawQuery suffices here so no
// Store widening is needed).
func ftsLeg(ctx context.Context, st store.Store, q SearchQuery) ([]legHit, error) {
	rq, ok := st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return nil, errNoRawQuery
	}
	limit := q.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	sel := limit * 3

	// Identifier-first: when the query is a single identifier (a
	// camel/snake/flat word, no spaces, no pattern characters), match
	// symbol_fts.name/qualified_name with an exact/prefix match FIRST (ranked
	// above free-text). This is the T-CI-1 fix — the v1 free-text-only
	// matching missed wireGitHooks.
	var identHits []legHit
	if ident := identifierMatchQuery(q.Query); ident != "" {
		args := []any{ident, ident + "%", ident, ident + ".%"}
		where := `(s.name = ? OR s.name LIKE ? OR s.qualified_name = ? OR s.qualified_name LIKE ?)`
		if q.Language != "" {
			where += ` AND s.language = ?`
			args = append(args, q.Language)
		}
		where, args = appendUnder(where, args, "f.path", q.Under)
		// Exact matches rank above prefix matches (ORDER BY a boolean).
		// F3: the ORDER BY comparison uses a bind parameter (?), not an
		// inlined SQL literal.
		where += ` ORDER BY (s.name = ?) DESC, s.name, s.id LIMIT ?`
		args = append(args, ident, sel)
		rows, err := rq.RawQuery(`
			SELECT s.id, s.name, s.kind, f.path, s.start_line, s.end_line
			FROM symbols s
			INNER JOIN files f ON f.id = s.file_id
			WHERE `+where, args...)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			identHits = append(identHits, legHit{
				ID:        "sym:" + itoa(rowInt(row, 0)),
				Path:      rowStr(row, 3),
				StartLine: rowInt(row, 4),
				EndLine:   rowInt(row, 5),
				Symbol:    rowStr(row, 1),
				Kind:      rowStr(row, 2),
				StoreID:   int64(rowInt(row, 0)),
			})
		}
	}

	// Free-text FTS: BM25 over chunks_fts + symbol_fts (the v1 floor).
	//
	// F1 (T-CI-2): a free-text MATCH that ERRORS propagates the error — the
	// leg degrades out + a warning is recorded (a silent zero-hit leg would
	// be the v1 bug). A MATCH that runs clean and returns 0 rows is a normal
	// zero-hit leg (no warning). The identifier-first sub-query above is a
	// separate exact/prefix read (no MATCH) and is NOT affected — it is the
	// T-CI-1 floor and must keep answering.
	var textHits []legHit
	if match := ftsMatchQuery(q.Query); match != "" {
		// chunk_fts leg.
		where := `WHERE chunks_fts MATCH ?`
		args := []any{match}
		if q.Language != "" {
			where += ` AND f.language = ?`
			args = append(args, q.Language)
		}
		where, args = appendUnder(where, args, "f.path", q.Under)
		where += ` ORDER BY MAX(bm25(chunks_fts), 1.0) LIMIT ?`
		args = append(args, sel)
		crows, err := rq.RawQuery(`
			SELECT f.path, c.start_line, c.end_line
			FROM chunks c
			INNER JOIN chunks_fts ON chunks_fts.rowid = c.id
			INNER JOIN files f ON f.id = c.file_id
			`+where, args...)
		if err != nil {
			return nil, err
		}
		for _, row := range crows {
			textHits = append(textHits, legHit{
				ID:        "chunk:" + rowStr(row, 0) + ":" + itoa(rowInt(row, 1)),
				Path:      rowStr(row, 0),
				StartLine: rowInt(row, 1),
				EndLine:   rowInt(row, 2),
				Kind:      "chunk",
			})
		}
		// symbol_fts free-text leg (the name/qualified_name/signature/kind/
		// language columns — a free-text match on a symbol).
		where = `WHERE symbol_fts MATCH ?`
		args = []any{match}
		if q.Language != "" {
			where += ` AND s.language = ?`
			args = append(args, q.Language)
		}
		where, args = appendUnder(where, args, "f.path", q.Under)
		where += ` ORDER BY MAX(bm25(symbol_fts), 1.0) LIMIT ?`
		args = append(args, sel)
		srows, err := rq.RawQuery(`
			SELECT s.id, s.name, s.kind, f.path, s.start_line, s.end_line
			FROM symbols s
			INNER JOIN symbol_fts ON symbol_fts.rowid = s.id
			INNER JOIN files f ON f.id = s.file_id
			`+where, args...)
		if err != nil {
			return nil, err
		}
		for _, row := range srows {
			textHits = append(textHits, legHit{
				ID:        "sym:" + itoa(rowInt(row, 0)),
				Path:      rowStr(row, 3),
				StartLine: rowInt(row, 4),
				EndLine:   rowInt(row, 5),
				Symbol:    rowStr(row, 1),
				Kind:      rowStr(row, 2),
				StoreID:   int64(rowInt(row, 0)),
			})
		}
	}

	// Identifier-first ordering: exact/prefix identifier matches rank ABOVE
	// free-text (the T-CI-1 fix). Dedupe by ID (a doc in both the identifier
	// and free-text sub-legs is one hit, ranked by the identifier position).
	out := []legHit{}
	seen := map[string]bool{}
	for _, h := range identHits {
		if !seen[h.ID] {
			seen[h.ID] = true
			out = append(out, h)
		}
	}
	for _, h := range textHits {
		if !seen[h.ID] {
			seen[h.ID] = true
			out = append(out, h)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// identifierMatchQuery returns the query string when it is a single
// identifier (a camelCase/snake_case/flat word, no spaces, no pattern
// characters) — the identifier-first path. Returns "" for free-text or
// pattern queries.
func identifierMatchQuery(q string) string {
	q = strings.TrimSpace(q)
	if q == "" || strings.ContainsAny(q, " \\()@") {
		return ""
	}
	hasLetter := false
	for _, r := range q {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '.' {
			return ""
		}
		if unicode.IsLetter(r) {
			hasLetter = true
		}
	}
	if !hasLetter {
		return ""
	}
	return q
}

// ftsMatchQuery builds an FTS5 MATCH expression for a free-text query. The
// trigram tokenizer requires each term to be >= 3 runes; a single < 3-rune
// term yields "" (the identifier-first path or a LIKE scan covers it).
func ftsMatchQuery(q string) string {
	terms := []string{}
	for _, t := range strings.Fields(q) {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		t = strings.ReplaceAll(t, `"`, `""`)
		terms = append(terms, `"`+t+`"`)
	}
	if len(terms) == 1 && runeLen(terms[0]) < 3 {
		return ""
	}
	return strings.Join(terms, " ")
}

// runeLen returns the rune count of a (possibly quoted) string.
func runeLen(s string) int {
	return len([]rune(strings.Trim(s, `"`)))
}

// appendUnder appends the T10 under-scope predicate + its params for the given
// path column. An empty prefix is a no-op. The prefix matches the exact path
// or a path with the prefix as a directory segment (internal/pay matches
// internal/pay and internal/pay/x.go, not internal/payment).
func appendUnder(where string, args []any, col, under string) (string, []any) {
	if under == "" {
		return where, args
	}
	under = strings.TrimSuffix(under, "/")
	where += ` AND (` + col + ` = ? OR ` + col + ` LIKE ?)`
	args = append(args, under, under+"/%")
	return where, args
}

// errNoRawQuery is the FTS leg's error when the store lacks the RawQuery seam
// (the test store always has it; a production store without it is a gap).
var errNoRawQuery = errString("fts leg: store does not expose RawQuery")

// rowStr reads a string column (nil-safe).
func rowStr(row []any, i int) string {
	if i >= len(row) {
		return ""
	}
	s, _ := row[i].(string)
	return s
}

// rowInt reads an int column (nil-safe; handles int64 and int).
func rowInt(row []any, i int) int {
	if i >= len(row) {
		return 0
	}
	switch v := row[i].(type) {
	case int64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}
