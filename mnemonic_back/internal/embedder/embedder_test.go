package embedder

import (
	"context"
	"testing"
)

func TestHashEmbedderDeterministic(t *testing.T) {
	h := HashEmbedder{Dim: 32}
	a, err := h.Embed(context.Background(), "hello world")
	if err != nil {
		t.Fatal(err)
	}
	b, err := h.Embed(context.Background(), "hello world")
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Data) != 32 {
		t.Fatalf("dim=%d", len(a.Data))
	}
	for i := range a.Data {
		if a.Data[i] != b.Data[i] {
			t.Fatal("not deterministic")
		}
	}
}
