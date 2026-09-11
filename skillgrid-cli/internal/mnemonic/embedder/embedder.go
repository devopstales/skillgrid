// Package embedder produces code embedding vectors for the hybrid search
// core. The Embedder interface is pluggable: ONNX nomic-embed-code is the
// default provider, an external OpenAI-compatible endpoint is a configurable
// provider, and "off" is the Null Adapter (no vector leg). The embedder is
// asymmetric-capable: separate indexing_params (corpus) and query_params
// (query); the output dimension is model-wide.
package embedder

import (
	"context"
	"hash/fnv"
	"math"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/config"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// Embedder produces embedding vectors (no CGo required for the default ONNX
// provider). The asymmetric param sets are honored by providers that need
// different treatment of the corpus (indexing) side vs. the query side.
type Embedder interface {
	// Embed produces a vector for text under the indexing (corpus) params.
	Embed(ctx context.Context, text string) (memory.Vector, error)
	// EmbedQuery produces a vector for a query under the query params.
	EmbedQuery(ctx context.Context, text string) (memory.Vector, error)
	// Model names the producer (used to gate re-embedding on model swap).
	Model() string
	// Dimension is the model-wide output dimension.
	Dimension() int
}

// HashEmbedder is a deterministic Pure Go stub embedder suitable for tests
// and offline ranking when a real local model is not wired. It is
// asymmetric-capable (the query side prepends a "query:" prefix so the two
// param sets produce distinct vectors).
type HashEmbedder struct {
	Dim int
	// QueryPrefix distinguishes the query-side embedding (asymmetric).
	QueryPrefix string
}

const defaultDim = 64

func (h HashEmbedder) Model() string { return "hash-embedder-v1" }

func (h HashEmbedder) Dimension() int {
	if h.Dim > 0 {
		return h.Dim
	}
	return defaultDim
}

func (h HashEmbedder) Embed(ctx context.Context, text string) (memory.Vector, error) {
	return hashEmbed(h.Dim, "", text)
}

func (h HashEmbedder) EmbedQuery(ctx context.Context, text string) (memory.Vector, error) {
	prefix := h.QueryPrefix
	if prefix == "" {
		prefix = "query: "
	}
	return hashEmbed(h.Dim, prefix, text)
}

// NewHash returns a HashEmbedder with the given dimension (0 = default 64).
func NewHash(dim int) HashEmbedder {
	return HashEmbedder{Dim: dim, QueryPrefix: "query: "}
}

func hashEmbed(dim int, prefix, text string) (memory.Vector, error) {
	if dim <= 0 {
		dim = defaultDim
	}
	vec := make([]float32, dim)
	for _, tok := range tokenize(prefix + text) {
		fh := fnv.New32a()
		_, _ = fh.Write([]byte(tok))
		idx := int(fh.Sum32() % uint32(dim))
		vec[idx] += 1
	}
	// L2 normalize so cosine is well-defined.
	var norm float64
	for _, v := range vec {
		norm += float64(v) * float64(v)
	}
	if norm > 0 {
		inv := float32(1 / math.Sqrt(norm))
		for i := range vec {
			vec[i] *= inv
		}
	}
	return memory.Vector{Data: vec}, nil
}

// Default returns the process embedder when MNEMONIC_EMBED is on, else nil.
// With embedding enabled it selects the provider from the project's
// indexing.yaml (config.Load("."), 06.3); a load failure degrades to the
// Null Adapter instead of failing (06.4). Callers keep their nil/zero-length
// checks: nil (embedding off) and Null (degraded) both skip the vector leg.
func Default() Embedder {
	if !memory.EmbeddingEnabled() {
		return nil
	}
	return buildDefaultFromConfig(config.Load("."))
}

// buildDefaultFromConfig is the config-driven selection used by Default; it
// takes an already-loaded config so tests and callers with a config root can
// reuse it.
func buildDefaultFromConfig(cfg config.Indexing) Embedder {
	ec := cfg.Embedder
	return BuildFromConfig(EmbedderConfig{
		Provider:  ec.Provider,
		Dimension: ec.Dimension,
		Indexing:  AsymParams{Instructions: ec.Indexing.Instructions, InputType: ec.Indexing.InputType, MaxTokens: ec.Indexing.MaxTokens},
		Query:     AsymParams{Instructions: ec.Query.Instructions, InputType: ec.Query.InputType, MaxTokens: ec.Query.MaxTokens},
		BaseURL:   ec.BaseURL,
		Model:     ec.Model,
		APIKey:    ec.APIKey,
		ModelDir:  ec.ModelDir,
	})
}

func tokenize(text string) []string {
	fields := strings.Fields(strings.ToLower(text))
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.Trim(f, ".,;:!?\"'`()[]{}")
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}
