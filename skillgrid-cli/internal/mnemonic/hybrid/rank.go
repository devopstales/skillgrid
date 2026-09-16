// Package hybrid ranks code search hits with an offline RRF fusion of
// identifier/chunk FTS, deterministic signals (proximity, TF-IDF, type/API),
// and — when the embedder is available — semantic vectors. FTS is the floor:
// a down/missing embedder degrades to FTS + signals, never a hard fail.
// Every ranked hit carries per-signal provenance.
package hybrid

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/embedder"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/search"
)

// RRFK is the reciprocal-rank-fusion constant (mirrors memory.BlendedSearch).
const RRFK = 60

// Provenance is the per-signal explanation for one ranked hit.
type Provenance struct {
	FTS      bool    `json:"fts,omitempty"`
	Signal   bool    `json:"signal,omitempty"`
	Semantic bool    `json:"semantic,omitempty"`
	Sim      float64 `json:"similarity,omitempty"`
}

// Hit is one ranked hybrid result with provenance.
type Hit struct {
	Path       string     `json:"path"`
	StartLine  int        `json:"start_line"`
	EndLine    int        `json:"end_line"`
	Symbol     string     `json:"symbol,omitempty"`
	Kind       string     `json:"kind,omitempty"`
	Snippet    string     `json:"snippet,omitempty"`
	Score      float64    `json:"score"`
	Provenance Provenance `json:"provenance"`
}

// Options tunes a hybrid search.
type Options struct {
	Limit    int
	FTSOnly  bool // select the lexical leg only (CLI --fts)
	Semantic bool // select the vector leg only (CLI --semantic)
	Embedder embedder.Embedder
	// Language, when set, scopes the semantic leg to that language
	// (exact index-level filter on the embeddings/chunk_embeddings language
	// column — the cocoindex partition key, relational form). Empty means all
	// languages.
	Language string
}

// Result is a hybrid search answer.
type Result struct {
	Query    string   `json:"query"`
	Legs     []string `json:"legs"`
	Hits     []Hit    `json:"hits"`
	Warnings []string `json:"warnings,omitempty"`
}

// Rank fuses the ranked legs into a single RRF-ordered hit list with
// per-signal provenance. The maps may be nil (leg absent). When only one leg
// is present the score reduces to 1/(k+rank) — ordering identical to that leg
// alone (FTS stays the floor).
func Rank(hits map[string]Hit, ftsRanks, sigRanks, semRanks map[string]int, k int) []Hit {
	if k <= 0 {
		k = RRFK
	}
	if len(hits) == 0 {
		return nil
	}
	type scored struct {
		id    string
		score float64
	}
	var out []scored
	for id := range hits {
		score := 0.0
		if r, ok := ftsRanks[id]; ok {
			score += 1.0 / float64(k+r+1)
		}
		if r, ok := sigRanks[id]; ok {
			score += 1.0 / float64(k+r+1)
		}
		if r, ok := semRanks[id]; ok {
			score += 1.0 / float64(k+r+1)
		}
		if score > 0 {
			out = append(out, scored{id: id, score: score})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].score != out[j].score {
			return out[i].score > out[j].score
		}
		return out[i].id < out[j].id
	})
	ranked := make([]Hit, 0, len(out))
	for _, s := range out {
		h := hits[s.id]
		h.Score = s.score
		h.Provenance.FTS = isPresent(ftsRanks, s.id)
		h.Provenance.Signal = isPresent(sigRanks, s.id)
		h.Provenance.Semantic = isPresent(semRanks, s.id)
		ranked = append(ranked, h)
	}
	return ranked
}

func isPresent(m map[string]int, key string) bool {
	_, ok := m[key]
	return ok
}

// Search runs the full offline hybrid search against db: identifier FTS over
// symbols + chunk FTS, deterministic signals, and the optional embedding leg.
// A down/nil embedder degrades to FTS + signals (the floor) — it never errors
// the whole run.
func Search(ctx context.Context, db *sql.DB, query string, opts Options) (*Result, error) {
	if db == nil {
		return nil, fmt.Errorf("hybrid search: database not initialized")
	}
	if query == "" {
		return nil, fmt.Errorf("hybrid search: query is required")
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}

	res := &Result{Query: query}
	hits := map[string]Hit{}
	ftsRanks := map[string]int{}
	sigRanks := map[string]int{}
	semRanks := map[string]int{}

	if !opts.Semantic {
		// Identifier FTS over symbols (step 02).
		if symHits, err := search.SymbolFTS(db, query, limit*3); err == nil {
			for _, sh := range symHits {
				id := fmt.Sprintf("sym:%d", sh.ID)
				hits[id] = Hit{
					Path:      sh.Path,
					StartLine: sh.StartLine,
					EndLine:   sh.EndLine,
					Symbol:    sh.Name,
					Kind:      sh.Kind,
					Snippet:   sh.Signature,
				}
			}
		}
		// Chunk FTS (the existing code_search leg).
		if chunkHits, err := search.CodeSearch(db, query, limit*3); err == nil {
			for _, ch := range chunkHits {
				id := fmt.Sprintf("chunk:%s:%d", ch.Path, ch.StartLine)
				hits[id] = Hit{
					Path:      ch.Path,
					StartLine: ch.StartLine,
					EndLine:   ch.EndLine,
					Snippet:   ch.Snippet,
				}
			}
		}
		// Deterministic signals (identifier token overlap + kind).
		sigHits, warnings := signalSearch(db, query)
		if warnings != nil {
			res.Warnings = append(res.Warnings, warnings...)
		}
		for _, sh := range sigHits {
			id := fmt.Sprintf("sym:%d", sh.ID)
			if _, ok := hits[id]; !ok {
				hits[id] = Hit{
					Path:      sh.Path,
					StartLine: sh.StartLine,
					EndLine:   sh.EndLine,
					Symbol:    sh.Name,
					Kind:      sh.Kind,
					Snippet:   sh.Signature,
				}
			}
		}

		if len(hits) > 0 {
			for i, id := range orderByFTS(db, query, hits) {
				ftsRanks[id] = i
			}
			for i, id := range orderBySignal(db, query, hits) {
				sigRanks[id] = i
			}
		}
		res.Legs = append(res.Legs, "fts", "signal")
	}

	// Embedding leg: only when an embedder is provided and not FTS-only. Two
	// sub-legs (both language-scoped when opts.Language is set): symbol-level
	// (name+signature vectors) and chunk-level (AST-boundary text vectors, 037).
	if !opts.FTSOnly && opts.Embedder != nil && opts.Embedder.Model() != "" {
		symLeg, symWarn, symErr := vectorLeg(ctx, db, query, opts.Language, opts.Embedder, limit*3)
		chunkLeg, chunkWarn, chunkErr := chunkVectorLeg(ctx, db, query, opts.Language, opts.Embedder, limit*3)
		if symErr != nil || chunkErr != nil {
			// Down embedder: degrade to FTS + signals (the floor).
			if symErr != nil {
				res.Warnings = append(res.Warnings, "embedder down: "+symErr.Error())
			} else {
				res.Warnings = append(res.Warnings, "embedder down: "+chunkErr.Error())
			}
		}
		for i, vh := range symLeg {
			id := vh.ID
			if _, ok := hits[id]; !ok {
				hits[id] = Hit{
					Path:      vh.Path,
					StartLine: vh.StartLine,
					EndLine:   vh.EndLine,
					Symbol:    vh.Symbol,
					Kind:      vh.Kind,
				}
			}
			h := hits[id]
			h.Provenance.Sim = vh.Sim
			hits[id] = h
			semRanks[id] = i
		}
		// Chunk-level hits are keyed by path+line (distinct from symbol ids).
		for i, vh := range chunkLeg {
			id := vh.ID
			if _, ok := hits[id]; !ok {
				hits[id] = Hit{
					Path:      vh.Path,
					StartLine: vh.StartLine,
					EndLine:   vh.EndLine,
					Kind:      "chunk",
				}
			}
			h := hits[id]
			h.Provenance.Sim = vh.Sim
			hits[id] = h
			if _, ok := semRanks[id]; !ok {
				semRanks[id] = i
			}
		}
		if len(symLeg) > 0 || len(chunkLeg) > 0 {
			res.Legs = append(res.Legs, "semantic")
		}
		if symWarn != "" {
			res.Warnings = append(res.Warnings, symWarn)
		}
		if chunkWarn != "" {
			res.Warnings = append(res.Warnings, chunkWarn)
		}
	}

	if opts.Semantic {
		res.Legs = append(res.Legs, "semantic")
	}

	ranked := Rank(hits, ftsRanks, sigRanks, semRanks, RRFK)
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	res.Hits = ranked
	if res.Hits == nil {
		res.Hits = []Hit{}
	}
	return res, nil
}

// vectorLeg computes the query vector and ranks stored symbol embeddings by
// cosine similarity. When language is non-empty it is an exact index-level
// filter (the cocoindex partition key, relational form). Returns a warning
// string when degenerate rows are found.
func vectorLeg(ctx context.Context, db *sql.DB, query, language string, emb embedder.Embedder, limit int) ([]vectorHit, string, error) {
	qVec, err := emb.Embed(ctx, query)
	if err != nil {
		return nil, "", fmt.Errorf("embed query: %w", err)
	}
	if len(qVec.Data) == 0 {
		return nil, "embedder returned empty vector", nil
	}
	q := `
		SELECT e.symbol_id, s.name, s.kind, s.start_line, s.end_line, f.path, e.vector
		FROM embeddings e
		JOIN symbols s ON s.id = e.symbol_id
		JOIN files f ON f.id = s.file_id
	`
	var args []any
	if language != "" {
		q += ` WHERE s.language = ?`
		args = append(args, language)
	}
	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, "", fmt.Errorf("select embeddings: %w", err)
	}
	defer rows.Close()
	var hits []vectorHit
	var degenerate int
	for rows.Next() {
		var symbolID int64
		var name, kind, path string
		var startLine, endLine int
		var blob []byte
		if err := rows.Scan(&symbolID, &name, &kind, &startLine, &endLine, &path, &blob); err != nil {
			continue
		}
		vec, derr := memory.DecodeVector(blob)
		if derr != nil || len(vec.Data) == 0 {
			degenerate++
			continue
		}
		// An all-zeros vector has cosine similarity 0 with everything — it
		// carries no direction, so the vector leg skips it (04.11).
		if memory.CosineSimilarity(vec, vec) == 0 {
			degenerate++
			continue
		}
		sim := memory.CosineSimilarity(qVec, vec)
		hits = append(hits, vectorHit{
			ID:        fmt.Sprintf("sym:%d", symbolID),
			Path:      path,
			StartLine: startLine,
			EndLine:   endLine,
			Symbol:    name,
			Kind:      kind,
			Sim:       sim,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Sim != hits[j].Sim {
			return hits[i].Sim > hits[j].Sim
		}
		return hits[i].ID < hits[j].ID
	})
	if len(hits) > limit {
		hits = hits[:limit]
	}
	warn := ""
	if degenerate > 0 {
		warn = fmt.Sprintf("%d degenerate embeddings skipped", degenerate)
	}
	return hits, warn, nil
}

// vectorHit is one ranked embedding match.
type vectorHit struct {
	ID        string
	Path      string
	StartLine int
	EndLine   int
	Symbol    string
	Kind      string
	Sim       float64
}

// chunkVectorLeg ranks stored CHUNK embeddings (AST-boundary text vectors,
// 037) by cosine similarity. Hits are keyed by path+start_line (distinct from
// the symbol leg's sym:<id> ids). language, when set, is an exact index-level
// filter. A store predating 037 (no chunk_embeddings rows) yields an empty
// leg, never an error.
func chunkVectorLeg(ctx context.Context, db *sql.DB, query, language string, emb embedder.Embedder, limit int) ([]vectorHit, string, error) {
	qVec, err := emb.Embed(ctx, query)
	if err != nil {
		return nil, "", fmt.Errorf("embed query: %w", err)
	}
	if len(qVec.Data) == 0 {
		return nil, "embedder returned empty vector", nil
	}
	// A chunk's language is the language of the first symbol in its file
	// (files have no language column; symbols do), mirroring the indexer's
	// embedChunks. The filter is a correlated subquery so it works whether or
	// not the store has a language column on files.
	q := `
		SELECT e.chunk_id, c.start_line, c.end_line, f.path, e.vector
		FROM chunk_embeddings e
		JOIN chunks c ON c.id = e.chunk_id
		JOIN files f ON f.id = c.file_id
	`
	var args []any
	if language != "" {
		q += ` WHERE (SELECT s.language FROM symbols s
			WHERE s.file_id = c.file_id
			ORDER BY s.start_line, s.id LIMIT 1) = ?`
		args = append(args, language)
	}
	rows, err := db.Query(q, args...)
	if err != nil {
		// chunk_embeddings absent (pre-037 store): degrade to no chunk leg.
		return nil, "", nil
	}
	defer rows.Close()
	var hits []vectorHit
	for rows.Next() {
		var chunkID int64
		var startLine, endLine int
		var path string
		var blob []byte
		if err := rows.Scan(&chunkID, &startLine, &endLine, &path, &blob); err != nil {
			continue
		}
		vec, derr := memory.DecodeVector(blob)
		if derr != nil || len(vec.Data) == 0 {
			continue
		}
		if memory.CosineSimilarity(vec, vec) == 0 {
			continue
		}
		sim := memory.CosineSimilarity(qVec, vec)
		hits = append(hits, vectorHit{
			ID:        fmt.Sprintf("chunk:%s:%d", path, startLine),
			Path:      path,
			StartLine: startLine,
			EndLine:   endLine,
			Kind:      "chunk",
			Sim:       sim,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, "", nil
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Sim != hits[j].Sim {
			return hits[i].Sim > hits[j].Sim
		}
		return hits[i].ID < hits[j].ID
	})
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits, "", nil
}

// signalSearch returns deterministic-signal hits: symbols whose identifier
// tokens overlap the query, ordered by overlap strength (a TF-IDF-style
// proximity signal).
func signalSearch(db *sql.DB, query string) ([]search.SymbolHit, []string) {
	var warnings []string
	qTokens := search.SplitIdentifier(query)
	if len(qTokens) == 0 {
		return nil, warnings
	}
	qset := map[string]bool{}
	for _, t := range qTokens {
		qset[t] = true
	}
	rows, err := db.Query(`
		SELECT s.id, s.name, s.qualified_name, s.kind, s.language, s.signature,
		       f.path, s.start_line, s.end_line
		FROM symbols s INNER JOIN files f ON f.id = s.file_id
		LIMIT 500
	`)
	if err != nil {
		return nil, []string{"signal search: " + err.Error()}
	}
	defer rows.Close()
	type tok struct {
		id   int64
		over int
	}
	var toks []tok
	for rows.Next() {
		var id int64
		var name, qualified, kind, lang, sig, path string
		var startLine, endLine int
		if err := rows.Scan(&id, &name, &qualified, &kind, &lang, &sig, &path, &startLine, &endLine); err != nil {
			continue
		}
		over := 0
		for _, st := range search.SplitIdentifier(name) {
			if qset[st] {
				over++
			}
		}
		if over > 0 {
			toks = append(toks, tok{id: id, over: over})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, []string{"signal search: " + err.Error()}
	}
	sort.SliceStable(toks, func(i, j int) bool {
		if toks[i].over != toks[j].over {
			return toks[i].over > toks[j].over
		}
		return toks[i].id < toks[j].id
	})
	var out []search.SymbolHit
	for _, t := range toks {
		if h, err := symbolByID(db, t.id); err == nil {
			out = append(out, h)
		}
	}
	return out, warnings
}

// symbolByID fetches one symbol row as a SymbolHit.
func symbolByID(db *sql.DB, id int64) (search.SymbolHit, error) {
	var h search.SymbolHit
	err := db.QueryRow(`
		SELECT s.id, s.name, s.qualified_name, s.kind, s.language, s.signature,
		       f.path, s.start_line, s.end_line
		FROM symbols s INNER JOIN files f ON f.id = s.file_id
		WHERE s.id = ?`, id).
		Scan(&h.ID, &h.Name, &h.QualifiedName, &h.Kind, &h.Language, &h.Signature, &h.Path, &h.StartLine, &h.EndLine)
	return h, err
}

// orderByFTS ranks hit IDs by their bm25 position for the query (symbol hits
// first by bm25, then chunk hits by bm25).
func orderByFTS(db *sql.DB, query string, hits map[string]Hit) []string {
	symHits, _ := search.SymbolFTS(db, query, len(hits))
	var ids []string
	seen := map[string]bool{}
	for _, sh := range symHits {
		id := fmt.Sprintf("sym:%d", sh.ID)
		if _, ok := hits[id]; ok && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	chunkHits, _ := search.CodeSearch(db, query, len(hits))
	for _, ch := range chunkHits {
		id := fmt.Sprintf("chunk:%s:%d", ch.Path, ch.StartLine)
		if _, ok := hits[id]; ok && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids
}

// orderBySignal ranks hit IDs by deterministic-signal strength.
func orderBySignal(db *sql.DB, query string, hits map[string]Hit) []string {
	sig, _ := signalSearch(db, query)
	var ids []string
	seen := map[string]bool{}
	for _, sh := range sig {
		id := fmt.Sprintf("sym:%d", sh.ID)
		if _, ok := hits[id]; ok && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids
}
