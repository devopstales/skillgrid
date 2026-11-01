package embedder

import (
	"context"
	"net"
	"testing"
)

// deadPort returns a TCP port with nothing listening on it.
func deadPort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	p := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return p
}

// TestEmbedderLoadFailureReturnsNull (06.4): a requested provider that cannot
// load degrades to the Null Adapter (no error, no panic); NullEmbedder.Embed
// returns a zero-length vector without error.
func TestEmbedderLoadFailureReturnsNull(t *testing.T) {
	// ollama on an unreachable port → Null (construction succeeds, the
	// server is simply absent; selection must not surface that as a hard
	// failure).
	e := BuildFromConfig(EmbedderConfig{
		Provider:  ProviderOllama,
		BaseURL:   "http://127.0.0.1:" + itoa(deadPort(t)),
		Model:     "nomic-embed-code",
		Dimension: 64,
	})
	if _, ok := e.(NullEmbedder); !ok {
		t.Fatalf("unreachable ollama → %T, want NullEmbedder (degraded)", e)
	}
	v, err := e.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("NullEmbedder.Embed: %v", err)
	}
	if len(v.Data) != 0 {
		t.Fatalf("NullEmbedder.Embed dim=%d, want 0", len(v.Data))
	}
	q, err := e.EmbedQuery(context.Background(), "hello")
	if err != nil {
		t.Fatalf("NullEmbedder.EmbedQuery: %v", err)
	}
	if len(q.Data) != 0 {
		t.Fatalf("NullEmbedder.EmbedQuery dim=%d, want 0", len(q.Data))
	}

	// local with a missing model file → Null (no error).
	e = BuildFromConfig(EmbedderConfig{
		Provider:  ProviderLocal,
		ModelDir:  "/nonexistent/skillgrid-models-dir",
		Model:     "ghost",
		Dimension: 8,
	})
	if _, ok := e.(NullEmbedder); !ok {
		t.Fatalf("missing local model → %T, want NullEmbedder (degraded)", e)
	}

	// unknown provider → Null (no error).
	if _, ok := BuildFromConfig(EmbedderConfig{Provider: "bogus"}).(NullEmbedder); !ok {
		t.Fatal("unknown provider → want NullEmbedder")
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	return string(b[pos:])
}
