package embedder

import (
	"context"
	"strings"
	"testing"
)

func TestOnnxDefault(t *testing.T) {
	o := NewOnnx(OnnxConfig{})

	if got := o.Model(); got != "nomic-embed-code" {
		t.Fatalf("Model()=%q, want %q", got, "nomic-embed-code")
	}
	if got := o.Dimension(); got != 768 {
		t.Fatalf("Dimension()=%d, want 768", got)
	}

	// modelPath must be under ~/.skillgrid/models/
	mp := o.modelPath()
	if !strings.Contains(mp, ".skillgrid/models") {
		t.Fatalf("modelPath()=%q, want it to contain .skillgrid/models", mp)
	}

	// Model file is absent → deterministic hash fallback, 768-dim, no error.
	v, err := o.Embed(context.Background(), "func main() { fmt.Println(\"hi\") }")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(v.Data) != 768 {
		t.Fatalf("Embed dim=%d, want 768", len(v.Data))
	}

	q, err := o.EmbedQuery(context.Background(), "fmt.Println")
	if err != nil {
		t.Fatalf("EmbedQuery: %v", err)
	}
	if len(q.Data) != 768 {
		t.Fatalf("EmbedQuery dim=%d, want 768", len(q.Data))
	}

	// The hash fallback is deterministic.
	v2, _ := o.Embed(context.Background(), "func main() { fmt.Println(\"hi\") }")
	for i := range v.Data {
		if v.Data[i] != v2.Data[i] {
			t.Fatalf("Embed not deterministic at index %d", i)
		}
	}
}

func TestNullAdapter(t *testing.T) {
	n := NewNull()

	if got := n.Model(); got != "off" {
		t.Fatalf("Model()=%q, want %q", got, "off")
	}
	if got := n.Dimension(); got != 0 {
		t.Fatalf("Dimension()=%d, want 0", got)
	}

	v, err := n.Embed(context.Background(), "anything")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(v.Data) != 0 {
		t.Fatalf("Embed dim=%d, want 0", len(v.Data))
	}

	q, err := n.EmbedQuery(context.Background(), "anything")
	if err != nil {
		t.Fatalf("EmbedQuery: %v", err)
	}
	if len(q.Data) != 0 {
		t.Fatalf("EmbedQuery dim=%d, want 0", len(q.Data))
	}
}
