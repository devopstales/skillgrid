package embedder

import (
	"context"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// NullEmbedder is the "off" provider: a Null Adapter with no vector leg.
// Embed always returns an empty vector (and no error) so callers degrade to
// the FTS + signals floor without a hard fail.
type NullEmbedder struct{}

// NewNull returns the Null Adapter.
func NewNull() NullEmbedder { return NullEmbedder{} }

func (NullEmbedder) Model() string  { return "off" }
func (NullEmbedder) Dimension() int { return 0 }

func (NullEmbedder) Embed(_ context.Context, _ string) (memory.Vector, error) {
	return memory.Vector{}, nil
}

func (NullEmbedder) EmbedQuery(_ context.Context, _ string) (memory.Vector, error) {
	return memory.Vector{}, nil
}
