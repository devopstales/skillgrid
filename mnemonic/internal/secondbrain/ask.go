// Package secondbrain is the "ask the brain" layer over mnemonic's existing
// retrieval floor. AskCited is the deterministic, no-LLM, token-bounded,
// citation-bearing answer: it reuses the hybrid gather (FTS5 floor + vector RRF,
// degrading to BM25-only when no embedder) and renders cited, redacted snippets.
// It is the floor every higher ask mode (LLM synthesis) builds on.
package secondbrain

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
	"github.com/devopstales/skillgrid/mnemonic/internal/context_harness"
)

const (
	// defaultMaxTokens is the budget when the caller does not supply one.
	defaultMaxTokens = 2000
	// citationSnippet caps each citation's redacted content in characters.
	citationSnippet = 200
)

// Citation is one token-bounded, redacted reference to a matched observation.
// ID/Title/Type are required; Snippet is the redacted, token-capped content.
type Citation struct {
	ID      int64  `json:"id"`
	Title   string `json:"title"`
	Type    string `json:"type"`
	Snippet string `json:"snippet"`
	Project string `json:"project"`
}

// AskResult is the cited floor of mem_ask. Answer is empty in the no-LLM mode
// (populated by a higher mode only); Citations carry the evidence. MatchedVia
// is "keyword" (degraded) or "hybrid" (vector leg active). TotalTokens is the
// underscore-prefixed budget accounting (OCBI meta convention).
type AskResult struct {
	Answer      string     `json:"answer,omitempty"`
	Citations   []Citation `json:"citations"`
	MatchedVia  string     `json:"matched_via"`
	Degraded    bool       `json:"degraded"`
	Sources     []string   `json:"sources,omitempty"`
	TotalTokens int        `json:"_total_tokens"`
}

// AskCited is the deterministic no-LLM floor of mem_ask. It gathers relevant
// observations via the existing hybrid search (FTS5 + vector RRF; degrades to
// BM25-only when no embedder is active — degraded=true, matched_via=keyword),
// dedups by observation ID, and renders token-bounded, citation-bearing,
// path-redacted results. projectID scopes the answer; allProjects=true spans
// every store in the data dir. An empty query is a no-op, not an error; errors
// are returned as values, never thrown.
func AskCited(ctx context.Context, svc *service.Service, query string, projectID string, allProjects bool, maxTokens int) (*AskResult, error) {
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return &AskResult{}, nil
	}
	if svc == nil {
		return nil, fmt.Errorf("secondbrain: service is nil")
	}

	h, cleanup, err := svc.Open(projectID)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	hits, degraded, err := context_harness.HybridObservations(ctx, h.Memory(), projectID, query, allProjects)
	if err != nil {
		return nil, err
	}

	seen := map[int64]bool{}
	deduped := make([]memory.Observation, 0, len(hits))
	for _, o := range hits {
		if seen[o.ID] {
			continue
		}
		seen[o.ID] = true
		deduped = append(deduped, o)
	}

	res := &AskResult{Degraded: degraded, MatchedVia: "keyword"}
	if !degraded {
		res.MatchedVia = "hybrid"
	}

	for _, o := range deduped {
		snippet := redact(o.Content)
		if len(snippet) > citationSnippet {
			snippet = snippet[:citationSnippet] + "…"
		}
		cost := context_harness.EstimateTokens(snippet)
		if cost <= 0 {
			continue
		}
		if res.TotalTokens+cost > maxTokens {
			continue
		}
		res.TotalTokens += cost
		res.Citations = append(res.Citations, Citation{
			ID: o.ID, Title: o.Title, Type: o.Type, Snippet: snippet, Project: o.Project,
		})
		res.Sources = append(res.Sources, fmt.Sprintf("obs:%d", o.ID))
	}
	sort.SliceStable(res.Citations, func(i, j int) bool { return res.Citations[i].ID < res.Citations[j].ID })
	return res, nil
}

// redact strips full local paths from a snippet before it is returned to the
// caller (privacy floor: a citation never leaks a path). Project-relative
// paths are left unchanged.
func redact(s string) string {
	for _, prefix := range []string{"/Users/", "/home/", `C:\`} {
		s = strings.ReplaceAll(s, prefix, "<path>/")
	}
	return s
}
