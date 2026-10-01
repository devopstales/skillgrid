package session_inject

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

const (
	injectSnippetMaxChars  = 200
	injectDefaultMaxTokens = 800
	injectSearchLimit      = 20
	// rrfK is the standard Reciprocal Rank Fusion constant, matching the
	// k=60 used by memory.ReciprocalRankFusion.
	rrfK = 60
)

// InjectItem is one ranked observation projected into the injected context:
// a redacted snippet plus its estimated token cost.
type InjectItem struct {
	ID        int64  `json:"id"`
	Source    string `json:"source"`
	Project   string `json:"project"`
	Title     string `json:"title"`
	Snippet   string `json:"snippet"`
	TokenCost int    `json:"token_cost"`
}

// RetrieveResult is the hybrid retrieval outcome: ranked injectable items,
// their total token cost, and whether the vector leg contributed (Degraded is
// true when only the BM25/FTS leg ran).
type RetrieveResult struct {
	Items       []InjectItem `json:"items"`
	Degraded    bool         `json:"degraded"`
	TotalTokens int          `json:"total_tokens"`
}

// HybridRetrieve returns injectable observations ranked by the existing fused
// search (BM25 FTS + semantic vector, RRF-fused, FTS floor), mapped into token
// budgeted InjectItems. The fused ranking is project-scoped; allProjects
// widens to every project bucket in the store's data directory. Degraded is
// true when the vector leg did not contribute (no embedder, embeddings
// disabled, or no stored vectors).
func HybridRetrieve(ctx context.Context, mem *memory.Service, projectID, query string, allProjects bool, maxTokens int) (*RetrieveResult, error) {
	if mem == nil {
		return nil, fmt.Errorf("memory service is nil")
	}
	if maxTokens <= 0 {
		maxTokens = injectDefaultMaxTokens
	}
	query = strings.TrimSpace(query)
	if query == "" {
		// An empty query produces no query vector, so the vector leg cannot
		// contribute regardless of embedder config (consistent with the
		// embedder-aware queryVector contract).
		return &RetrieveResult{Degraded: true}, nil
	}

	queryVec, degraded := queryVector(ctx, mem, query)
	var obs []memory.Observation
	var err error
	if allProjects {
		obs, err = crossProjectSearch(ctx, mem, projectID, query, queryVec)
	} else {
		obs, _, err = mem.BlendedSearch(ctx, query, "any", "", queryVec, injectSearchLimit)
	}
	if err != nil {
		return nil, err
	}
	return mapItems(obs, maxTokens, degraded), nil
}

// HybridObservations is the observation-level form of HybridRetrieve's gather
// step: it runs the fused search (BM25 FTS floor + semantic vector, RRF-fused,
// degrading to FTS-only when no embedder contributes) and returns the ranked,
// full observation rows (no injectability filter, no token mapping) so a caller
// can render its own projection. project-scoped; allProjects widens to every
// project bucket in the store's data directory. The second return reports
// whether the vector leg was skipped (degraded=true).
func HybridObservations(ctx context.Context, mem *memory.Service, projectID, query string, allProjects bool) ([]memory.Observation, bool, error) {
	if mem == nil {
		return nil, false, fmt.Errorf("memory service is nil")
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, true, nil
	}
	queryVec, degraded := queryVector(ctx, mem, query)
	var obs []memory.Observation
	var err error
	if allProjects {
		obs, err = crossProjectSearch(ctx, mem, projectID, query, queryVec)
	} else {
		obs, _, err = mem.BlendedSearch(ctx, query, "any", "", queryVec, injectSearchLimit)
	}
	if err != nil {
		return nil, degraded, err
	}
	return obs, degraded, nil
}

func queryVector(ctx context.Context, mem *memory.Service, query string) (memory.Vector, bool) {
	degraded := true
	var vec memory.Vector
	// The vector leg runs only when an embedder is attached AND embeddings are
	// enabled (BlendedSearch gates its vector leg on the same two), AND the
	// embed succeeds. Any miss means the vector leg did not contribute, so the
	// result is degraded.
	if mem.DirEmbedder() != nil && memory.EmbeddingEnabled() {
		if v, err := mem.DirEmbedder().EmbedQuery(ctx, query); err == nil && len(v.Data) > 0 {
			vec = v
			degraded = false
		}
	}
	return vec, degraded
}

func crossProjectSearch(ctx context.Context, mem *memory.Service, projectID, query string, queryVec memory.Vector) ([]memory.Observation, error) {
	// Cross-bucket fusion: each bucket (the primary project plus every sibling
	// project) contributes one ranked list from its own BlendedSearch, already
	// RRF-fused internally (FTS+vector). Concatenating the lists would hide
	// sibling hits whenever the primary has >= injectSearchLimit matches, so
	// we fuse across buckets with RRF:
	//   score(key) = sum over buckets of 1/(rrfK + rank_in_bucket + 1)
	// The same formula/constant as memory.ReciprocalRankFusion (that helper
	// only blends two lists; here we have N bucket lists), keyed by
	// "project:id" because ids collide across buckets. Ties break on key
	// string (deterministic).
	ranked := [][]memory.Observation{}
	primary, _, err := mem.BlendedSearch(ctx, query, "any", "", queryVec, injectSearchLimit)
	if err != nil {
		return nil, err
	}
	ranked = append(ranked, primary)
	if dir, e := dataDirFromService(mem); e == nil {
		for _, name := range siblingProjects(dir, projectID) {
			st, oErr := store.Open(dir, name)
			if oErr != nil {
				continue
			}
			svc := memory.New(st, name)
			if emb := mem.DirEmbedder(); emb != nil {
				svc.SetDirEmbedder(emb)
			}
			hits, _, sErr := svc.BlendedSearch(ctx, query, "any", "", queryVec, injectSearchLimit)
			st.Close()
			if sErr != nil {
				continue
			}
			ranked = append(ranked, hits)
		}
	}

	byKey := map[string]memory.Observation{}
	keys := []string{}
	scores := map[string]float64{}
	for _, list := range ranked {
		for rank, o := range list {
			key := o.Project + ":" + itoa(o.ID)
			if _, ok := byKey[key]; !ok {
				byKey[key] = o
				keys = append(keys, key)
			}
			scores[key] += 1.0 / float64(rrfK+rank+1)
		}
	}
	// keys[i] corresponds to out[i]; sort both together by fused score (desc),
	// key string (asc) on ties.
	idx := make([]int, len(keys))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		sa, sb := scores[keys[idx[a]]], scores[keys[idx[b]]]
		if sa != sb {
			return sa > sb
		}
		return keys[idx[a]] < keys[idx[b]]
	})
	out := make([]memory.Observation, len(keys))
	for i, k := range idx {
		out[i] = byKey[keys[k]]
	}
	return out, nil
}

func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}

func dataDirFromService(mem *memory.Service) (string, error) {
	p, err := mem.StorePath()
	if err != nil || p == "" {
		return "", err
	}
	return filepath.Dir(p), nil
}

func siblingProjects(dir, exclude string) []string {
	entries, err := filepath.Glob(filepath.Join(dir, "*.sqlite"))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		name := strings.TrimSuffix(filepath.Base(e), ".sqlite")
		if name != exclude {
			out = append(out, name)
		}
	}
	return out
}

func mapItems(obs []memory.Observation, maxTokens int, degraded bool) *RetrieveResult {
	res := &RetrieveResult{Degraded: degraded}
	for _, o := range obs {
		if !ObservationInjectable(o) {
			continue
		}
		snippet := redactFullPaths(o.Content)
		if len(snippet) > injectSnippetMaxChars {
			snippet = snippet[:injectSnippetMaxChars] + "…"
		}
		cost := EstimateTokens(snippet)
		if cost <= 0 {
			continue
		}
		if res.TotalTokens+cost > maxTokens {
			continue
		}
		res.TotalTokens += cost
		res.Items = append(res.Items, InjectItem{
			ID:        o.ID,
			Source:    "observation",
			Project:   o.Project,
			Title:     o.Title,
			Snippet:   snippet,
			TokenCost: cost,
		})
	}
	return res
}
