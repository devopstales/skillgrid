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
	resp   string
	err    error
	lastIn string
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

const keyLearningsText = `Some narrative output.

## Key Learnings:

1. bcrypt cost=12 is the right balance for our server
2. JWT refresh tokens need atomic rotation to avoid races
3. FTS5 queries must be sanitized before MATCH`

// TestExtractWithLLM is 05.1: ExtractWithLLM calls the LLM seam with the input
// text and parses the structured JSON learnings into PassiveItem values that
// shape through the same shapePassiveItem/shapePassiveContent pipeline as the
// regex path.
func TestExtractWithLLM(t *testing.T) {
	llm := &mockLLM{resp: llmJSONOK}
	items, err := ExtractWithLLM(context.Background(), keyLearningsText, llm)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if !strings.Contains(llm.lastIn, keyLearningsText) {
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

// captureTitles returns the (title|type) keys of live observations for the
// fixture project, for comparing a capture against the regex-only result.
func captureTitles(t *testing.T, fx *fixture) map[string]bool {
	t.Helper()
	rows, err := fx.st.DB.Query(
		`SELECT title, type FROM observations WHERE project = ? AND deleted_at IS NULL`,
		fx.svc.projectID,
	)
	if err != nil {
		t.Fatalf("query observations: %v", err)
	}
	defer rows.Close()
	seen := map[string]bool{}
	for rows.Next() {
		var title, typ string
		if err := rows.Scan(&title, &typ); err != nil {
			t.Fatalf("scan observation: %v", err)
		}
		seen[title+"|"+typ] = true
	}
	return seen
}

// TestCapturePassiveLLMFailureFallsBackToRegex is 05.2: when the LLM seam
// errors, CapturePassive falls back to the regex floor and produces exactly
// the same observations as the regex-only path, with no error propagated.
func TestCapturePassiveLLMFailureFallsBackToRegex(t *testing.T) {
	ctx := context.Background()

	// Regex-only baseline (no LLM enabled): the current default behavior.
	base := newFixture(t, "capture-llm-fail-base")
	baseRes, err := base.svc.CapturePassive(ctx, PassiveInput{
		Content:   keyLearningsText,
		SessionID: session1,
	})
	if err != nil {
		t.Fatalf("baseline capture: %v", err)
	}
	if baseRes.Saved == 0 {
		t.Fatalf("baseline: expected saved learnings, got %d (skipped %d)", baseRes.Saved, baseRes.Skipped)
	}
	baseSet := captureTitles(t, base)

	// LLM enabled but failing: the regex floor must take over.
	fx := newFixture(t, "capture-llm-fail")
	llm := &mockLLM{err: errors.New("llm down")}
	fx.svc.SetExtractionLLM(llm)
	fx.svc.EnableExtractionLLM(true)
	res, err := fx.svc.CapturePassive(ctx, PassiveInput{
		Content:   keyLearningsText,
		SessionID: session1,
	})
	if err != nil {
		t.Fatalf("capture with failing LLM must not propagate an error: %v", err)
	}
	if llm.lastIn == "" {
		t.Fatal("expected the LLM to have been tried")
	}
	if res.Saved == 0 {
		t.Fatalf("fallback: expected saved learnings, got %d (skipped %d)", res.Saved, res.Skipped)
	}
	// The fallback result must be byte-for-byte the regex-only result.
	got := captureTitles(t, fx)
	if len(got) != len(baseSet) {
		t.Fatalf("fallback produced %d observations, regex-only produced %d", len(got), len(baseSet))
	}
	for k := range baseSet {
		if !got[k] {
			t.Fatalf("fallback missing regex-only observation %q", k)
		}
	}
}
