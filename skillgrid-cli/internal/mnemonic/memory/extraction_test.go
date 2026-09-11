package memory

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// mockLLM is a test double for the ExtractionLLM seam: it records the text it
// was called with and returns a canned response (or error).
type mockLLM struct {
	resp    string
	err     error
	lastIn  string
	callErr string
}

func (m *mockLLM) Extract(ctx context.Context, text string) (string, error) {
	m.lastIn = text
	if m.err != nil {
		return "", m.err
	}
	return m.resp, nil
}

const llmJSONOK = `{
  "learnings": [
    {"text": "bcrypt cost=12 is the right balance for our server"},
    {"text": "FTS5 queries must be sanitized before MATCH"}
  ]
}`

// TestExtractWithLLM is 05.1: ExtractWithLLM calls the LLM seam with the input
// text and parses the structured JSON learnings into PassiveItem values that
// shape through the same shapePassiveItem/shapePassiveContent pipeline as the
// regex path.
func TestExtractWithLLM(t *testing.T) {
	text := "Some narrative.\n\n## Key Learnings:\n\n1. bcrypt cost=12 is the right balance for our server\n2. FTS5 queries must be sanitized before MATCH"

	llm := &mockLLM{resp: llmJSONOK}
	items, err := ExtractWithLLM(context.Background(), text, llm)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if !strings.Contains(llm.lastIn, text) {
		t.Fatalf("LLM not called with the input text: got %q", llm.lastIn)
	}
	// Items must shape into the standard passive format (valid type + non-blank
	// title), i.e. the same shapePassiveItem contract the regex path uses.
	for i, item := range items {
		title, typ := shapePassiveItem(item)
		if title == "" || !IsValidType(typ) {
			t.Fatalf("item %d: bad shape title=%q typ=%q", i, title, typ)
		}
		if shapePassiveContent(item, title, typ) == "" {
			t.Fatalf("item %d: empty content", i)
		}
	}
}

// TestExtractWithLLMFailure is 05.1: when the LLM seam errors, ExtractWithLLM
// returns the error (the caller decides to fall back).
func TestExtractWithLLMFailure(t *testing.T) {
	llm := &mockLLM{err: errors.New("llm down")}
	items, err := ExtractWithLLM(context.Background(), "text", llm)
	if err == nil {
		t.Fatalf("expected error, got items=%d", len(items))
	}
	if len(items) != 0 {
		t.Fatalf("expected no items on error, got %d", len(items))
	}
}
