// Package embedder is the Embedder seam for the mnemonic-v2 memory function
// (spec 003-intelligence §2.2, INV-3). The Embedder interface is the seam: a
// nil seam is "absent" (the vector leg simply does not run, the result is
// flagged degraded), and a present-but-erroring seam is "failing" (one bounded
// retry via degrade.Run, then the floor — no vector written).
package embedder

import "context"

// Embedder produces embedding vectors for observation text (spec
// 003-intelligence §2.2). The OpenAI-compatible provider
// (internal/intelligence/llm.Client) implements both LLM and Embedder; the
// Null adapter below models the "absent" state.
type Embedder interface {
	// Embed returns one vector per input text, in the same order. A
	// dimension mismatch against the vector column is an error at first
	// write, never a silent truncation.
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	// Model is the name of the model that produced the vectors, recorded on
	// observations.embedding_model (INV-1 provenance).
	Model() string
	// Dimensions is the embedding dimension; it must match the vector column.
	Dimensions() int
}

// Null is the "off" adapter (provider "none"): an absent seam, not a failing
// one. Embed IS called (it is a present adapter) but does no I/O — it returns
// one zero-value vector per input text and never errors — so no vector is ever
// written and callers degrade to the FTS + signals floor, flagging the result.
type Null struct{}

// NewNull returns the Null adapter (the "absent" embedder).
func NewNull() Null { return Null{} }

func (Null) Model() string   { return "" }
func (Null) Dimensions() int { return 0 }
func (Null) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	// Absent, not failing: one zero-value vector per input text, no I/O, no
	// error. The caller's floor (no vector written) owns the actual result.
	return make([][]float32, len(texts)), nil
}
