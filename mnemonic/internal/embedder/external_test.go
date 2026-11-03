package embedder

import (
	"context"
	"strings"
	"testing"
)

func TestExternalConfigurable(t *testing.T) {
	e := NewExternal(ExternalConfig{
		BaseURL:   "http://localhost:11434",
		Model:     "nomic-embed-text",
		APIKey:    "test-key",
		Dimension: 512,
	})

	if got := e.Model(); got != "nomic-embed-text" {
		t.Fatalf("Model()=%q, want %q", got, "nomic-embed-text")
	}
	if got := e.Dimension(); got != 512 {
		t.Fatalf("Dimension()=%d, want 512", got)
	}
}

func TestExternalNoBaseURL(t *testing.T) {
	e := NewExternal(ExternalConfig{Model: "nomic-embed-text"})

	_, err := e.Embed(context.Background(), "some text")
	if err == nil {
		t.Fatal("Embed with no BaseURL: want error, got nil")
	}
	if !strings.Contains(err.Error(), "base_url is required") {
		t.Fatalf("error=%q, want it to mention base_url is required", err.Error())
	}

	_, err = e.EmbedQuery(context.Background(), "some text")
	if err == nil {
		t.Fatal("EmbedQuery with no BaseURL: want error, got nil")
	}
	if !strings.Contains(err.Error(), "base_url is required") {
		t.Fatalf("error=%q, want it to mention base_url is required", err.Error())
	}
}
