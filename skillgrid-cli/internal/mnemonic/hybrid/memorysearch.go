package hybrid

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/embedder"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/facts"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/skills"
)

// MemoryFact is one fact surfaced by SearchMemory, carrying its content and
// per-leg provenance so the caller can show where it came from. Scope is fixed
// to "fact" so the MCP result tags each row by source.
type MemoryFact struct {
	Scope      string     `json:"scope"`
	ID         int64      `json:"id"`
	Content    string     `json:"content"`
	Score      float64    `json:"score"`
	Provenance Provenance `json:"provenance"`
}

// MemorySkill is one skill surfaced by SearchMemory. Scope is fixed to "skill".
type MemorySkill struct {
	Scope       string     `json:"scope"`
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Language    string     `json:"language"`
	Description string     `json:"description,omitempty"`
	Score       float64    `json:"score"`
	Provenance  Provenance `json:"provenance"`
}

// MemoryOptions controls SearchMemory. Embedder is the adapter for the vector
// leg; nil (or a down embedder) degrades the result to BM25-only — never a
// hard fail. IncludeDeleted widens the fact leg to soft-deleted rows.
// ReadingSession is the session id the fact_search trail event is attributed
// to (the fact store records it on retrieval).
type MemoryOptions struct {
	Embedder       embedder.Embedder
	IncludeDeleted bool
	ReadingSession string
}

// MemoryResult is the fused fact+skill result set, the active legs, and
// warnings describing any degraded state.
type MemoryResult struct {
	Query    string        `json:"query"`
	Legs     []string      `json:"legs"`
	Facts    []MemoryFact  `json:"facts"`
	Skills   []MemorySkill `json:"skills"`
	Warnings []string      `json:"warnings,omitempty"`
}

// SearchMemory performs the hybrid (BM25 FTS5 + optional vector) search over
// the facts and skills stores, fusing each scope's lexical and vector legs with
// RRF. The vector leg is active only when the embedder is present and working;
// otherwise the result is BM25-only (degraded, never a hard fail).
func SearchMemory(ctx context.Context, factStore *facts.Store, skillStore *skills.Store, query string, limit int, opts MemoryOptions) (*MemoryResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("hybrid: search: query is required")
	}
	if limit <= 0 {
		limit = 20
	}

	emb := opts.Embedder
	vecActive := emb != nil && emb.Model() != ""

	res := &MemoryResult{Query: query, Legs: []string{"fts"}}

	// --- Facts scope -------------------------------------------------------
	ftsFacts, err := factStore.SearchWith(ctx, opts.ReadingSession, query, limit, opts.IncludeDeleted)
	if err != nil {
		return nil, fmt.Errorf("hybrid: facts BM25: %w", err)
	}
	ftsFactRanks := make(map[string]int, len(ftsFacts))
	for i, f := range ftsFacts {
		ftsFactRanks[factKey(f.ID)] = i
	}
	var semFactRanks map[string]int
	var factSim map[string]float64
	if vecActive {
		var verr error
		semFactRanks, factSim, verr = vectorRanks(ctx, emb, query, ftsFacts, func(f facts.Fact) string { return f.Content })
		if verr != nil {
			res.Warnings = append(res.Warnings, "facts vector leg: embedder down, BM25-only for facts")
			semFactRanks, factSim = nil, nil
		}
	}
	res.Facts = fuseFacts(ftsFacts, ftsFactRanks, semFactRanks, factSim)

	// --- Skills scope ------------------------------------------------------
	ftsSkills, err := skillStore.SearchWith(ctx, query, limit, opts.IncludeDeleted)
	if err != nil {
		return nil, fmt.Errorf("hybrid: skills BM25: %w", err)
	}
	ftsSkillRanks := make(map[string]int, len(ftsSkills))
	for i, s := range ftsSkills {
		ftsSkillRanks[skillKey(s.ID)] = i
	}
	var semSkillRanks map[string]int
	var skillSim map[string]float64
	if vecActive {
		var verr error
		semSkillRanks, skillSim, verr = vectorRanks(ctx, emb, query, ftsSkills, func(s skills.Skill) string { return s.Name + "\n" + s.Description })
		if verr != nil {
			res.Warnings = append(res.Warnings, "skills vector leg: embedder down, BM25-only for skills")
			semSkillRanks, skillSim = nil, nil
		}
	}
	res.Skills = fuseSkills(ftsSkills, ftsSkillRanks, semSkillRanks, skillSim)

	// --- Legs bookkeeping --------------------------------------------------
	if semFactRanks != nil || semSkillRanks != nil {
		res.Legs = append(res.Legs, "semantic")
	}
	return res, nil
}

// vectorRanks embeds the query and every doc, then returns (rankMap, simMap)
// keyed by doc key, ranked by descending cosine similarity to the query. It
// returns (nil, nil, nil) when there are no docs, and an error when the
// embedder is down (query or per-doc embed fails) — the caller degrades to
// BM25-only on error.
func vectorRanks[T any](ctx context.Context, emb embedder.Embedder, query string, docs []T, docText func(T) string) (map[string]int, map[string]float64, error) {
	if len(docs) == 0 {
		return nil, nil, nil
	}
	qv, err := emb.EmbedQuery(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	type scored struct {
		doc T
		cos float64
	}
	var scoredDocs []scored
	for _, d := range docs {
		v, err := emb.Embed(ctx, docText(d))
		if err != nil {
			return nil, nil, err
		}
		scoredDocs = append(scoredDocs, scored{d, memory.CosineSimilarity(qv, v)})
	}
	// Sort by descending cosine.
	for i := 1; i < len(scoredDocs); i++ {
		for j := i; j > 0 && scoredDocs[j].cos > scoredDocs[j-1].cos; j-- {
			scoredDocs[j], scoredDocs[j-1] = scoredDocs[j-1], scoredDocs[j]
		}
	}
	// Only docs with a positive similarity count toward the semantic rank.
	rankMap := make(map[string]int)
	simMap := make(map[string]float64)
	rank := 0
	for _, sd := range scoredDocs {
		if sd.cos <= 0 {
			continue
		}
		key := docKey(sd.doc)
		rankMap[key] = rank
		simMap[key] = sd.cos
		rank++
	}
	if len(rankMap) == 0 {
		return nil, nil, nil
	}
	return rankMap, simMap, nil
}

// docKey maps a fact or skill to its canonical key via a type switch.
func docKey(d any) string {
	switch v := d.(type) {
	case facts.Fact:
		return factKey(v.ID)
	case skills.Skill:
		return skillKey(v.ID)
	}
	return ""
}

// fuseFacts RRF-fuses the FTS and semantic fact ranks into ranked MemoryFacts.
func fuseFacts(fts []facts.Fact, ftsRanks, semRanks map[string]int, sim map[string]float64) []MemoryFact {
	byKey := make(map[string]facts.Fact, len(fts))
	for _, f := range fts {
		byKey[factKey(f.ID)] = f
	}
	ranked := Rank(hitsFromFacts(byKey), ftsRanks, nil, semRanks, RRFK)
	out := make([]MemoryFact, 0, len(ranked))
	for _, h := range ranked {
		f := byKey[h.Path]
		pv := h.Provenance
		if sim != nil {
			pv.Semantic = true
			pv.Sim = sim[h.Path]
		}
		out = append(out, MemoryFact{
			Scope:      "fact",
			ID:         f.ID,
			Content:    f.Content,
			Score:      h.Score,
			Provenance: pv,
		})
	}
	return out
}

// fuseSkills is the skills analogue of fuseFacts.
func fuseSkills(fts []skills.Skill, ftsRanks, semRanks map[string]int, sim map[string]float64) []MemorySkill {
	byKey := make(map[string]skills.Skill, len(fts))
	for _, s := range fts {
		byKey[skillKey(s.ID)] = s
	}
	ranked := Rank(hitsFromSkills(byKey), ftsRanks, nil, semRanks, RRFK)
	out := make([]MemorySkill, 0, len(ranked))
	for _, h := range ranked {
		s := byKey[h.Path]
		pv := h.Provenance
		if sim != nil {
			pv.Semantic = true
			pv.Sim = sim[h.Path]
		}
		out = append(out, MemorySkill{
			Scope:       "skill",
			ID:          s.ID,
			Name:        s.Name,
			Language:    s.Language,
			Description: s.Description,
			Score:       h.Score,
			Provenance:  pv,
		})
	}
	return out
}

func hitsFromFacts(byKey map[string]facts.Fact) map[string]Hit {
	out := make(map[string]Hit, len(byKey))
	for k, f := range byKey {
		out[k] = Hit{
			Path:       k,
			Snippet:    truncate(f.Content, 200),
			Provenance: Provenance{FTS: true},
		}
	}
	return out
}

func hitsFromSkills(byKey map[string]skills.Skill) map[string]Hit {
	out := make(map[string]Hit, len(byKey))
	for k, s := range byKey {
		out[k] = Hit{
			Path:       k,
			Snippet:    truncate(s.Description, 200),
			Provenance: Provenance{FTS: true},
		}
	}
	return out
}

func factKey(id int64) string  { return "fact:" + strconv.FormatInt(id, 10) }
func skillKey(id int64) string { return "skill:" + strconv.FormatInt(id, 10) }

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 0 {
		return ""
	}
	return s[:n] + "…"
}
