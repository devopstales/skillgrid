package embedder

import (
	"context"
	"hash/fnv"
	"math"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// Embedder produces Pure Go embedding vectors (no CGO).
type Embedder interface {
	Embed(ctx context.Context, text string) (memory.Vector, error)
	Model() string
}

// HashEmbedder is a deterministic Pure Go stub embedder suitable for tests
// and offline ranking when a real local model is not wired.
type HashEmbedder struct {
	Dim int
}

const defaultDim = 64

func (h HashEmbedder) Model() string { return "hash-embedder-v1" }

func (h HashEmbedder) Embed(ctx context.Context, text string) (memory.Vector, error) {
	_ = ctx
	dim := h.Dim
	if dim <= 0 {
		dim = defaultDim
	}
	vec := make([]float32, dim)
	for _, tok := range tokenize(text) {
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
func Default() Embedder {
	if !memory.EmbeddingEnabled() {
		return nil
	}
	return HashEmbedder{}
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
