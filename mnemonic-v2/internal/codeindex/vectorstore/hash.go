package vectorstore

import (
	"context"
	"hash/fnv"
	"math"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/embedder"
)

// hash.go — the hash-64 default embedder (the v1 HashEmbedder,
// mnemonic_back/internal/embedder/embedder.go:43, lifted). It is the
// deterministic floor (no dep, cgo-free, always works): a FNV-1a token hash
// folded into a fixed-dim vector, L2-normalized so cosine is well-defined.
// The query side prepends a "query: " prefix (the asymmetric param set — the
// two sides produce distinct vectors, the v1 convention).
//
// ADR-0045: the hash-64 default is 64-dim, so the 768-dim ANN path is not
// served by it. If the 768-dim path is an OpenAI-compat provider (no new dep),
// the ADR is NOT taken; if ONNX is wired, onnxer v0.7.0 is added.

// HashEmbedder is the deterministic hash embedder (the floor). Dim is the
// output dimension (0 → the 64 default); QueryPrefix distinguishes the
// query-side embedding (asymmetric — the v1 convention).
type HashEmbedder struct {
	Dim         int
	QueryPrefix string
}

// hashDimDefault is the floor's dimension (the v1 default, 64).
const hashDimDefault = 64

// Model names the producer (the guard's key — a model-name change clears
// the vectors).
func (h HashEmbedder) Model() string { return "hash-64" }

// Dimensions is the model-wide output dimension (0 → the 64 default).
func (h HashEmbedder) Dimensions() int {
	if h.Dim > 0 {
		return h.Dim
	}
	return hashDimDefault
}

// Embed produces the indexing (corpus) vector for the texts (one per input,
// in order). The hash embedder is deterministic — the same text always yields
// the same vector (the guard's re-index is reproducible).
func (h HashEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	dim := h.Dimensions()
	out := make([][]float32, len(texts))
	for i, t := range texts {
		out[i] = hashEmbed(dim, "", t)
	}
	return out, nil
}

// hashEmbed produces the (optionally prefixed) embedding: tokenize
// (prefix+text), FNV-1a each token into a bucket, L2-normalize. The prefix
// is the asymmetric query marker ("" on the corpus side, "query: " on the
// query side).
func hashEmbed(dim int, prefix, text string) []float32 {
	if dim <= 0 {
		dim = hashDimDefault
	}
	vec := make([]float32, dim)
	for _, tok := range tokenize(prefix + text) {
		fh := fnv.New32a()
		_, _ = fh.Write([]byte(tok))
		idx := int(fh.Sum32() % uint32(dim))
		vec[idx]++
	}
	// L2-normalize so cosine is well-defined (a zero vector stays zero).
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
	return vec
}

// tokenize lowercases + field-splits + trims the punctuation the v1
// HashEmbedder trims (the deterministic token set — a drift here changes every
// vector, breaking the re-index reproducibility).
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

// EmbedQuery is a convenience for the query side (the "query: " prefix). The
// embedder.Embedder seam has no query method (the v2 seam is symmetric
// Embed), so the Engine's query path uses EmbedQuery when the embedder is a
// *HashEmbedder (asymmetric) or Embed otherwise. It is a method (not an
// interface member) to keep the embedder seam unchanged.
func (h HashEmbedder) EmbedQuery(ctx context.Context, text string) []float32 {
	prefix := h.QueryPrefix
	if prefix == "" {
		prefix = "query: "
	}
	return hashEmbed(h.Dimensions(), prefix, text)
}

// queryVector returns the query-side vector for text (the asymmetric path).
// For a *HashEmbedder it uses the "query: " prefix; for any other embedder it
// falls back to Embed (the symmetric path — the OpenAI-compat provider's query
// is the same as its corpus embed, the v1 convention for non-hash providers).
func queryVector(ctx context.Context, emb embedder.Embedder, text string) ([]float32, error) {
	if h, ok := emb.(*HashEmbedder); ok {
		return h.EmbedQuery(ctx, text), nil
	}
	if h, ok := emb.(HashEmbedder); ok {
		return h.EmbedQuery(ctx, text), nil
	}
	vecs, err := emb.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vecs) == 0 {
		return nil, nil
	}
	return vecs[0], nil
}

// _ satisfies the embedder.Embedder seam (compile-time: the hash-64 default
// is a valid seam the Engine can hold).
var _ embedder.Embedder = (*HashEmbedder)(nil)
