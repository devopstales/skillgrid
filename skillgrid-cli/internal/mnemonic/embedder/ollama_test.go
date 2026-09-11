package embedder

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestOllamaEmbedder (06.1): a mock Ollama server on a test port mimics
// /api/embeddings; the embedder is configured to that port and must POST
// {"model","prompt"} and return the server's embedding vector.
func TestOllamaEmbedder(t *testing.T) {
	const wantDim = 12

	var gotPath, gotContentType string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		emb := make([]float32, wantDim)
		for i := range emb {
			emb[i] = float32(i + 1) / float32(wantDim)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model":    "nomic-embed-code",
			"embedding": emb,
		})
	}))
	defer srv.Close()

	o := NewOllama(OllamaConfig{
		BaseURL:   srv.URL,
		Model:     "nomic-embed-code",
		Dimension: wantDim,
	})

	if got := o.Model(); got != "nomic-embed-code" {
		t.Fatalf("Model()=%q, want %q", got, "nomic-embed-code")
	}
	if got := o.Dimension(); got != wantDim {
		t.Fatalf("Dimension()=%d, want %d", got, wantDim)
	}

	v, err := o.Embed(context.Background(), "test text")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(v.Data) != wantDim {
		t.Fatalf("Embed dim=%d, want %d", len(v.Data), wantDim)
	}
	for i := range v.Data {
		want := float32(i+1) / float32(wantDim)
		if v.Data[i] != want {
			t.Fatalf("Data[%d]=%v, want %v", i, v.Data[i], want)
		}
	}

	q, err := o.EmbedQuery(context.Background(), "test text")
	if err != nil {
		t.Fatalf("EmbedQuery: %v", err)
	}
	if len(q.Data) != wantDim {
		t.Fatalf("EmbedQuery dim=%d, want %d", len(q.Data), wantDim)
	}

	if gotPath != "/api/embeddings" {
		t.Fatalf("request path=%q, want /api/embeddings", gotPath)
	}
	if gotContentType != "application/json" {
		t.Fatalf("Content-Type=%q, want application/json", gotContentType)
	}
	if gotBody["model"] != "nomic-embed-code" {
		t.Fatalf("body.model=%v, want nomic-embed-code", gotBody["model"])
	}
	if gotBody["prompt"] != "test text" {
		t.Fatalf("body.prompt=%v, want the input text", gotBody["prompt"])
	}
}

// TestOllamaEmbedError (06.1): an HTTP error response surfaces as an error.
func TestOllamaEmbedError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"model not found"}`))
	}))
	defer srv.Close()

	o := NewOllama(OllamaConfig{BaseURL: srv.URL, Model: "nope", Dimension: 8})
	_, err := o.Embed(context.Background(), "test text")
	if err == nil {
		t.Fatal("Embed on 404: want error, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("error=%q, want it to mention the status", err)
	}
}
