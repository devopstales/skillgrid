package memory

import (
	"context"
	"errors"
	"fmt"
	"sort"
)

// SetEmbedding stores a precomputed embedding for an observation. blob is the
// little-endian float32 layout produced by EncodeVector; model names the
// producer for provenance (and to gate re-embedding on model swap). Passing a
// nil/empty blob clears the embedding (a no-op reindex can drop vectors).
func (s *Service) SetEmbedding(ctx context.Context, id int64, blob []byte, model string) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return errors.New("memory service not initialized")
	}
	if len(blob) > 0 && len(blob)%4 != 0 {
		return fmt.Errorf("embedding length %d is not a multiple of 4", len(blob))
	}
	// Only stamp created_at when there is no prior timestamp, preserving the
	// original creation time on re-embeds.
	res, err := s.store.DB.ExecContext(ctx, `
		UPDATE observations
		SET embedding = ?,
		    embedding_model = ?,
		    embedding_created_at = COALESCE(embedding_created_at, datetime('now'))
		WHERE id = ? AND project = ? AND deleted_at IS NULL`,
		blobOrNULL(blob), model, id, s.projectID,
	)
	if err != nil {
		return fmt.Errorf("set embedding: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("observation %d not found", id)
	}
	return nil
}

// blobOrNULL converts an empty slice to a NULL parameter.
func blobOrNULL(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}

// VectorHit pairs an observation with its similarity to the query vector.
type VectorHit struct {
	ID      int64
	Sim     float64
	Project string
}

// SearchByVector ranks the embedded observations of the project by cosine
// similarity to queryVec and returns the top limit. Observations without an
// embedding are skipped (they are still reachable via FTS). This is the
// semantic leg of the blended search; run it only when EmbeddingEnabled() and
// the caller has computed a query vector.
func (s *Service) SearchByVector(ctx context.Context, queryVec Vector, limit int) ([]VectorHit, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, errors.New("memory service not initialized")
	}
	if len(queryVec.Data) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT id, embedding
		FROM observations
		WHERE project = ? AND deleted_at IS NULL AND embedding IS NOT NULL`,
		s.projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("select embeddings: %w", err)
	}
	defer rows.Close()
	var hits []VectorHit
	for rows.Next() {
		var id int64
		var blob []byte
		if err := rows.Scan(&id, &blob); err != nil {
			return nil, fmt.Errorf("scan embedding row: %w", err)
		}
		vec, err := DecodeVector(blob)
		if err != nil {
			continue
		}
		sim := CosineSimilarity(queryVec, vec)
		hits = append(hits, VectorHit{ID: id, Sim: sim, Project: s.projectID})
	}
	if err := rows.Err(); err != nil {
		return nil, err
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
	return hits, nil
}

// BlendedSearch is the P4 orchestration hook: it runs the FTS leg
// (SearchWithScope) and, when a non-empty query vector is supplied, the
// vector leg (SearchByVector), then merges the two ranked lists with
// ReciprocalRankFusion before truncating to limit.
//
// Degradation guarantees (the "FTS5 is the floor" rule):
//   - queryVec empty  → returns the FTS leg unchanged (RRF is a no-op).
//   - no embeddings    → vector leg empty → RRF reduces to FTS ordering.
func (s *Service) BlendedSearch(ctx context.Context, query, matchMode, scope string, queryVec Vector, limit int) ([]Observation, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, errors.New("memory service not initialized")
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	fts, err := s.SearchWithScope(ctx, query, matchMode, scope, limit)
	if err != nil {
		return nil, err
	}
	if len(queryVec.Data) == 0 || !EmbeddingEnabled() {
		return fts, nil // FTS floor: no vector leg to blend with
	}

	vecHits, err := s.SearchByVector(ctx, queryVec, limit*3) // grab a wider net before blending
	if err != nil || len(vecHits) == 0 {
		return fts, nil // floor: keep FTS when vector leg has no candidates
	}

	key := func(o Observation) string {
		return fmt.Sprintf("%d:%s", o.ID, o.Project)
	}
	ftsRanks := make(map[string]int, len(fts))
	idByKey := make(map[string]Observation, len(fts))
	for i, o := range fts {
		k := key(o)
		ftsRanks[k] = i
		idByKey[k] = o
	}
	// Vector hits may reference rows that did NOT match FTS — fetch them so
	// the semantic-only recalls appear.
	vecIDs := make(map[string]int, len(vecHits))
	for i, vh := range vecHits {
		k := fmt.Sprintf("%d:%s", vh.ID, vh.Project)
		vecIDs[k] = i
		if _, ok := idByKey[k]; !ok {
			o, gErr := s.Get(ctx, vh.ID)
			if gErr == nil {
				idByKey[k] = o
			}
		}
	}
	allKeys := make([]string, 0, len(idByKey))
	for k := range idByKey {
		allKeys = append(allKeys, k)
	}
	merged := ReciprocalRankFusion(allKeys, ftsRanks, vecIDs, 60)
	out := make([]Observation, 0, len(merged))
	for _, k := range merged {
		if o, ok := idByKey[k]; ok {
			out = append(out, o)
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
