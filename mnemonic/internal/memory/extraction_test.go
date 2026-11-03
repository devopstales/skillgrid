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

// nuanceText has a "## Key Learnings" section (regex-extractable) PLUS a
// free-form paragraph with a nuanced learning that the regex floor cannot
// see. The nuance avoids the "learned"/"lesson"/"discovery" keywords, and the
// whole text is kept free of line-initial "lesson:"/"discovery:" style
// labels, so extractLearnings' inline-label fallback stays silent and the
// regex floor sees exactly the two sectioned items.
const nuanceText = `We finished the auth work and it went fine.

## Key Learnings:

1. bcrypt cost=12 is the right balance for our server
2. JWT refresh tokens need atomic rotation to avoid races

The refresh-token endpoint silently drops the rotation lock under load, so
the token pair can desync when two requests race. We will gate it with the
advisory lock in the next pass.`

// llmJSONNuance is the mock LLM's superset: both regex-extractable items plus
// the nuanced one only the LLM sees (worded like the free-form sentence).
const llmJSONNuance = `{
  "learnings": [
    {"text": "bcrypt cost=12 is the right balance for our server"},
    {"text": "JWT refresh tokens need atomic rotation to avoid races"},
    {"text": "the refresh-token endpoint silently drops the rotation lock under load, so the token pair can desync when two requests race"}
  ]
}`

// TestExtractionQualityLLMVsRegex is 05.3: the LLM pass (mocked to return the
// regex items PLUS a nuanced one the regex misses) yields a combined result
// that (a) is a superset of the regex-only result and (b) has no duplicates —
// dedup by content hash. A deterministic stand-in for "identical quality or
// better than regex-only".
func TestExtractionQualityLLMVsRegex(t *testing.T) {
	ctx := context.Background()

	// Test premise: the regex floor extracts the section items AND the
	// free-form sentences as raw lines (5 items), but it cannot consolidate
	// the multi-clause nuance into one learning — the LLM's consolidated
	// "rotation lock" learning is a distinct item the regex floor never
	// produces.
	regexItems := extractLearnings(nuanceText)
	if len(regexItems) != 5 {
		t.Fatalf("premise: regex should extract 5 items (2 sectioned + 3 free-form lines), got %d", len(regexItems))
	}
	for _, it := range regexItems {
		if strings.Contains(it.Text, "so the token pair can desync when two requests race") {
			t.Fatalf("premise: regex must not produce the consolidated nuance, got %q", it.Text)
		}
	}

	// LLM pass: mocked to return the superset (regex items + the nuance).
	llm := &mockLLM{resp: llmJSONNuance}
	llmItems, err := ExtractWithLLM(ctx, nuanceText, llm)
	if err != nil {
		t.Fatalf("llm extract: %v", err)
	}

	// Combined, deduplicated set.
	combined := dedupePassiveItems(llmItems, regexItems)

	// Every regex item must be represented in the combined set (superset).
	regexKeys := map[string]bool{}
	for _, it := range regexItems {
		title, typ := shapePassiveItem(it)
		regexKeys[title+"|"+typ] = true
	}
	// Superset: every regex key must be present in the combined set.
	combinedKeys := map[string]bool{}
	// Dedup: no two combined items share a content hash.
	seenHash := map[string]int{}
	for _, it := range combined {
		title, typ := shapePassiveItem(it)
		combinedKeys[title+"|"+typ] = true
		seenHash[passiveItemHash(it, title, typ)]++
	}
	covered := 0
	for k := range regexKeys {
		if combinedKeys[k] {
			covered++
		}
	}
	if covered != len(regexKeys) {
		t.Fatalf("combined set covers %d/%d regex items", covered, len(regexKeys))
	}
	for h, n := range seenHash {
		if n > 1 {
			t.Fatalf("duplicate item in combined set (hash %s appears %d times)", h, n)
		}
	}
	// The nuanced item (absent from the regex floor) must be present.
	nuanceFound := false
	for _, it := range combined {
		if strings.Contains(it.Text, "rotation lock") {
			nuanceFound = true
		}
	}
	if !nuanceFound {
		t.Fatal("combined set missing the nuanced learning the regex floor cannot see")
	}
	// 5 regex items (2 shared verbatim with the LLM, 3 regex-only) + 1
	// LLM-only consolidated nuance, de-duplicated.
	if len(combined) != 6 {
		t.Fatalf("expected 6 de-duplicated items (5 regex + 1 nuance), got %d", len(combined))
	}
}

// TestCapturePassiveLLMOptInCombinedDedup is 05.3: end-to-end, with the LLM
// enabled, CapturePassive stores the LLM's de-duplicated set (its superset
// already contains every regex item) — no duplicate rows — while the default
// (LLM disabled) path on the same text stays regex-only.
func TestCapturePassiveLLMOptInCombinedDedup(t *testing.T) {
	ctx := context.Background()

	// Default (LLM disabled) on the same text: the unchanged regex-only path.
	base := newFixture(t, "capture-llm-off")
	if _, err := base.svc.CapturePassive(ctx, PassiveInput{
		Content:   nuanceText,
		SessionID: session1,
	}); err != nil {
		t.Fatalf("baseline capture: %v", err)
	}
	// The unchanged regex-only path stores all 5 raw items (it does not
	// consolidate the nuance).
	if baseKeys := captureTitles(t, base); len(baseKeys) != 5 {
		t.Fatalf("regex-only baseline should store the 5 raw items, got %d: %v", len(baseKeys), baseKeys)
	}

	llm := &mockLLM{resp: llmJSONNuance}
	fx := newFixture(t, "capture-llm-optin")
	fx.svc.SetExtractionLLM(llm)
	fx.svc.EnableExtractionLLM(true)
	if _, err := fx.svc.CapturePassive(ctx, PassiveInput{
		Content:   nuanceText,
		SessionID: session1,
	}); err != nil {
		t.Fatalf("capture: %v", err)
	}
	got := captureTitles(t, fx)

	// The opt-in store must hold the LLM's de-duplicated set: both sectioned
	// items (which the mock also returns) PLUS the nuanced one the regex
	// floor misses — with no duplicate rows.
	nuanceFound := false
	for k := range got {
		if strings.Contains(k, "rotation lock") {
			nuanceFound = true
		}
	}
	if !nuanceFound {
		t.Fatal("opt-in LLM capture missing the nuanced learning")
	}
	for _, want := range []string{
		"bcrypt cost=12 is the right balance for our server",
		"JWT refresh tokens need atomic rotation to avoid races",
	} {
		found := false
		for k := range got {
			if strings.HasPrefix(k, want) {
				found = true
			}
		}
		if !found {
			t.Fatalf("opt-in store missing sectioned item %q", want)
		}
	}
	if len(got) != 3 {
		t.Fatalf("expected exactly 3 de-duplicated observations (the LLM set), got %d: %v", len(got), got)
	}
	// Row count must equal the distinct key count (no duplicate rows).
	n, ok := fx.obsCount(t)
	if !ok {
		t.Fatal("obs count")
	}
	if n != len(got) {
		t.Fatalf("duplicate rows: %d rows but %d distinct keys", n, len(got))
	}
}
