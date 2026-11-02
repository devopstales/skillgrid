package search

import (
	"context"
	"encoding/binary"
	"math"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// semanticLeg is the semantic leg (spec 006 §4): embed the query, then cosine
// over the stored BLOB vectors (brute-force scan — the pre-08a floor, since
// TICKET-08a's ANN indexes don't exist yet). It is language-scoped (the
// language filter is exact — the embeddings.language/chunk_embeddings.language
// partition key).
//
// Pre-08a vector read: the EmbedSymbol/EmbedChunk write side exists on the
// Store, but there is no vector READ method (the ANN indexes don't exist yet).
// The leg reads the BLOBs through the store's RawQuery seam (the same seam the
// FTS leg uses) and decodes the little-endian float32 blobs. No additive Store
// method is needed (the controller pre-approves ONE, but RawQuery suffices).
//
// A nil/failing embedder degrades through the Hybrid's T-CI-2 path (the leg is
// dropped, not empty). This function only runs when the embedder is up.
func semanticLeg(ctx context.Context, st store.Store, q SearchQuery) ([]legHit, error) {
	rq, ok := st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return nil, errNoRawQuery
	}

	// Embed the query.
	vecs, err := q.Embedder.Embed(ctx, []string{q.Query})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 || len(vecs[0]) == 0 {
		return nil, nil // embedder returned no vector (degenerate)
	}
	qvec := vecs[0]

	// Brute-force cosine over the stored symbol BLOB vectors (the pre-08a
	// floor). language-scoped (exact); under-scope via the files.path join.
	// ponytail: the scan is unbounded (every vector for the model is fetched
	// + cosine-computed in Go). For the pre-08a floor (the ANN indexes don't
	// exist yet) this is acceptable; upgrade path: TICKET-08a's vec0/HNSW
	// index + a LIMIT on the SQL if the repo grows past a few thousand
	// vectors.
	where := `WHERE e.model = ?`
	args := []any{q.Embedder.Model()}
	if q.Language != "" {
		where += ` AND e.language = ?`
		args = append(args, q.Language)
	}
	where, args = appendUnder(where, args, "f.path", q.Under)
	limit := q.Limit
	if limit <= 0 {
		limit = defaultLimit
	}
	rows, err := rq.RawQuery(`
		SELECT s.id, s.name, s.kind, f.path, s.start_line, s.end_line, e.vector
		FROM embeddings e
		INNER JOIN symbols s ON s.id = e.symbol_id
		INNER JOIN files f ON f.id = s.file_id
		`+where, args...)
	if err != nil {
		return nil, err
	}
	type cand struct {
		h   legHit
		vec []float32
	}
	var cands []cand
	for _, row := range rows {
		blob, _ := row[6].([]byte)
		vec, ok := decodeFloat32BLOB(blob)
		if !ok || len(vec) == 0 || len(vec) != len(qvec) {
			continue
		}
		cands = append(cands, cand{
			h: legHit{
				ID:        "sym:" + itoa(rowInt(row, 0)),
				Path:      rowStr(row, 3),
				StartLine: rowInt(row, 4),
				EndLine:   rowInt(row, 5),
				Symbol:    rowStr(row, 1),
				Kind:      rowStr(row, 2),
				StoreID:   int64(rowInt(row, 0)),
			},
			vec: vec,
		})
	}
	type scored struct {
		h   legHit
		sim float64
	}
	var scored_ []scored
	for _, c := range cands {
		sim := cosineSimilarity(qvec, c.vec)
		if sim > 0 {
			scored_ = append(scored_, scored{h: c.h, sim: sim})
		}
	}
	// Sort by similarity (desc), then id for determinism.
	for i := 1; i < len(scored_); i++ {
		for j := i; j > 0 && scored_[j].sim > scored_[j-1].sim; j-- {
			scored_[j], scored_[j-1] = scored_[j-1], scored_[j]
		}
	}
	if limit > 0 && len(scored_) > limit*3 {
		scored_ = scored_[:limit*3]
	}
	out := make([]legHit, 0, len(scored_))
	for _, s := range scored_ {
		out = append(out, s.h)
	}
	return out, nil
}

// decodeFloat32BLOB decodes a little-endian float32 blob (the store's
// encoding, spec 006 §4.1). ok=false when the byte length is not a multiple of
// 4 (a truncated or absent vector).
func decodeFloat32BLOB(b []byte) ([]float32, bool) {
	if len(b) == 0 || len(b)%4 != 0 {
		return nil, false
	}
	out := make([]float32, len(b)/4)
	for i := range out {
		out[i] = float32FromLE(b[i*4:])
	}
	return out, true
}

// float32FromLE decodes one little-endian float32.
func float32FromLE(b []byte) float32 {
	if len(b) < 4 {
		return 0
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(b))
}

// cosineSimilarity returns the cosine similarity of two equal-length vectors
// in [-1, 1]. A zero-magnitude vector yields 0.
func cosineSimilarity(a, b []float32) float64 {
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
