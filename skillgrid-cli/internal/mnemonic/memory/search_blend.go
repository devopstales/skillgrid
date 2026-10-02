package memory

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SearchSignals is the additive per-hit explanation on mem_search (ADR-0018).
// Every field is in [0,1].
type SearchSignals struct {
	Keyword    float64
	Vector     float64
	Recency    float64
	Entity     float64
	Decay      float64
	Importance float64
}

// SearchHit is one owner-scoped search result plus the fusion explanation.
type SearchHit struct {
	Observation Observation
	Score       float64
	MatchedVia  string
	Signals     SearchSignals
}

// SearchOwnerScopedBlend is the mem_search read path. Both legs honor the
// owner visibility filter. An empty query vector or a disabled embedder
// keeps the FTS order and reports matched_via=keyword.
func (s *Service) SearchOwnerScopedBlend(ctx context.Context, readerOwner, readerAgent, query, matchMode, scope string, limit int, queryVec Vector) ([]SearchHit, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, fmt.Errorf("memory service not initialized")
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	fts, err := s.ownerScopedFTS(ctx, readerOwner, readerAgent, query, matchMode, scope, limit)
	if err != nil {
		return nil, err
	}

	keyOf := func(id int64, project string) string {
		return fmt.Sprintf("%d:%s", id, project)
	}
	ftsRank := map[string]int{}
	byKey := map[string]Observation{}
	for i, o := range fts {
		k := keyOf(o.ID, o.Project)
		ftsRank[k] = i
		byKey[k] = o
	}

	vecRank := map[string]int{}
	cosine := map[string]float64{}
	useVec := len(queryVec.Data) > 0 && EmbeddingEnabled()
	if useVec {
		wider := limit * 3
		if wider < limit {
			wider = limit
		}
		vecHits, vErr := s.SearchByVector(ctx, queryVec, wider)
		if vErr != nil || len(vecHits) == 0 {
			useVec = false
		} else {
			placed := 0
			for _, vh := range vecHits {
				ok, rErr := s.canRead(ctx, vh.ID, "", readerOwner, readerAgent)
				if rErr != nil || !ok {
					continue
				}
				k := keyOf(vh.ID, vh.Project)
				vecRank[k] = placed
				sim := vh.Sim
				if sim < 0 {
					sim = 0
				}
				if sim > 1 {
					sim = 1
				}
				cosine[k] = sim
				placed++
				if _, seen := byKey[k]; !seen {
					o, gErr := s.Get(ctx, vh.ID)
					if gErr != nil {
						delete(vecRank, k)
						delete(cosine, k)
						continue
					}
					byKey[k] = o
				}
			}
			if len(vecRank) == 0 {
				useVec = false
			}
		}
	}

	order := make([]string, 0, len(byKey))
	if !useVec {
		for _, o := range fts {
			order = append(order, keyOf(o.ID, o.Project))
		}
	} else {
		ids := make([]string, 0, len(byKey))
		for k := range byKey {
			ids = append(ids, k)
		}
		order = ReciprocalRankFusion(ids, ftsRank, vecRank, 60)
	}

	cfg := s.decayCfg
	if cfg.HalfLifeDays <= 0 {
		cfg.HalfLifeDays = 30
	}
	now := time.Now().UTC()
	hits := make([]SearchHit, 0, len(order))
	for _, k := range order {
		o, ok := byKey[k]
		if !ok {
			continue
		}
		_, hasFTS := ftsRank[k]
		_, hasVec := vecRank[k]
		via := "keyword"
		switch {
		case hasFTS && hasVec:
			via = "hybrid"
		case hasVec:
			via = "vector"
		}
		kw := 0.0
		if hasFTS {
			kw = 1 / float64(1+ftsRank[k])
		}
		sig := SearchSignals{
			Keyword:    kw,
			Vector:     cosine[k],
			Recency:    SignalRecency(o, now, cfg),
			Entity:     0,
			Decay:      SignalDecay(o, now, cfg),
			Importance: SignalImportance(o),
		}
		score := kw
		if hasVec {
			rrf := 0.0
			if hasFTS {
				rrf += 1 / float64(60+ftsRank[k]+1)
			}
			rrf += 1 / float64(60+vecRank[k]+1)
			score = rrf / (2.0 / 61.0)
			if score > 1 {
				score = 1
			}
		}
		hits = append(hits, SearchHit{Observation: o, Score: score, MatchedVia: via, Signals: sig})
	}
	if s.decayCfg.Enabled {
		hits = sortHitsByDecay(hits, now, cfg)
	}
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits, nil
}

func sortHitsByDecay(hits []SearchHit, now time.Time, cfg DecayConfig) []SearchHit {
	if len(hits) < 2 {
		return hits
	}
	pinned := make([]SearchHit, 0, len(hits))
	rest := make([]SearchHit, 0, len(hits))
	for _, h := range hits {
		if h.Observation.Pinned {
			pinned = append(pinned, h)
		} else {
			rest = append(rest, h)
		}
	}
	type row struct {
		h SearchHit
		s float64
		i int
	}
	rows := make([]row, len(rest))
	for i, h := range rest {
		rows[i] = row{h: h, s: Reinforcement(h.Observation, now, cfg) * h.Score, i: i}
	}
	for i := 1; i < len(rows); i++ {
		j := i
		for j > 0 && (rows[j].s > rows[j-1].s || (rows[j].s == rows[j-1].s && rows[j].i < rows[j-1].i)) {
			rows[j], rows[j-1] = rows[j-1], rows[j]
			j--
		}
	}
	out := make([]SearchHit, 0, len(hits))
	out = append(out, pinned...)
	for _, r := range rows {
		out = append(out, r.h)
	}
	return out
}

// ownerScopedFTS is SearchOwnerScoped without the in-memory re-rank. It bumps
// retrieval usage after the scan, on the pre-bump structs the caller ranks.
func (s *Service) ownerScopedFTS(ctx context.Context, readerOwner, readerAgent, query, matchMode, scope string, limit int) ([]Observation, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, fmt.Errorf("memory service not initialized")
	}
	ftsQuery := buildFTSQuery(query, matchMode)
	if ftsQuery == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	scope = strings.TrimSpace(scope)
	scopeClause := ""
	args := []any{ftsQuery, s.projectID}
	if scope != "" {
		scopeClause = " AND o.scope = ?"
		args = append(args, scope)
	}
	clause, clauseArgs := s.visibilityFilter(readerOwner, readerOwner, readerAgent)
	args = append(args, clauseArgs...)
	args = append(args, limit)
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT o.id, o.session_id, o.type, o.title, o.content, o.project, o.scope,
		       o.topic_key, o.source, o.normalized_hash, o.revision_count, o.prompt_id, o.created_at, o.updated_at,
		       COALESCE(o.pinned, 0), COALESCE(o.duplicate_count, 0), o.last_seen_at, o.expires_at, o.tool_name,
		       o.owner, COALESCE(o.visibility, 'private'), COALESCE(o.status, 'active'), COALESCE(o.retrieval_usage, 0),
		       o.importance_score, o.recency_decay, o.maturity_tier, o.provenance, o.memory_type,
		       o.valid_at, o.invalid_at, o.superseded_by
		FROM observations o
		INNER JOIN observations_fts ON observations_fts.rowid = o.id
		WHERE observations_fts MATCH ? AND o.deleted_at IS NULL AND o.project = ?`+scopeClause+`
		  AND (o.invalid_at IS NULL OR o.invalid_at = '' OR strftime('%s', o.invalid_at) > strftime('%s', 'now'))
		  AND `+clause+`
		ORDER BY COALESCE(o.pinned, 0) DESC, bm25(observations_fts)
		LIMIT ?`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("owner scoped search: %w", err)
	}
	defer rows.Close()
	out, err := scanObservations(rows)
	if err != nil {
		return nil, err
	}
	for _, o := range out {
		s.BumpRetrievalUsage(ctx, o.ID)
	}
	return out, nil
}
