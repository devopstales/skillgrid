package session_inject

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

const (
	injectSnippetMaxChars  = 200
	injectDefaultMaxTokens = 800
	injectSearchLimit      = 20
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
		return &RetrieveResult{Degraded: !memory.EmbeddingEnabled()}, nil
	}

	queryVec, degraded := queryVector(ctx, mem, query)
	var obs []memory.Observation
	var err error
	if allProjects {
		obs, err = crossProjectSearch(ctx, mem, projectID, query, queryVec)
	} else {
		obs, err = mem.BlendedSearch(ctx, query, "any", "", queryVec, injectSearchLimit)
	}
	if err != nil {
		return nil, err
	}
	return mapItems(obs, maxTokens, degraded), nil
}

func queryVector(ctx context.Context, mem *memory.Service, query string) (memory.Vector, bool) {
	degraded := true
	var vec memory.Vector
	if mem.DirEmbedder() != nil && memory.EmbeddingEnabled() {
		if v, err := mem.DirEmbedder().EmbedQuery(ctx, query); err == nil && len(v.Data) > 0 {
			vec = v
			degraded = false
		}
	}
	return vec, degraded
}

func crossProjectSearch(ctx context.Context, mem *memory.Service, projectID, query string, queryVec memory.Vector) ([]memory.Observation, error) {
	primary, err := mem.BlendedSearch(ctx, query, "any", "", queryVec, injectSearchLimit)
	if err != nil {
		return nil, err
	}
	var out []memory.Observation
	seen := map[string]bool{}
	for _, o := range primary {
		seen[o.Project+":"+itoa(o.ID)] = true
		out = append(out, o)
	}
	dir, e := dataDirFromService(mem)
	if e != nil {
		return out, nil
	}
	for _, name := range siblingProjects(dir, projectID) {
		st, oErr := store.Open(dir, name)
		if oErr != nil {
			continue
		}
		svc := memory.New(st, name)
		if emb := mem.DirEmbedder(); emb != nil {
			svc.SetDirEmbedder(emb)
		}
		hits, sErr := svc.BlendedSearch(ctx, query, "any", "", queryVec, injectSearchLimit)
		st.Close()
		if sErr != nil {
			continue
		}
		for _, o := range hits {
			key := o.Project + ":" + itoa(o.ID)
			if !seen[key] {
				seen[key] = true
				out = append(out, o)
			}
		}
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
