package judgment

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/ctx/config"
)

// newEngine builds a Judgment from the default config (endpooint/model/threshold
// come from config, never hardcoded) and a cache, with an injectable client.
func newEngine(cfg config.Judgment, cache CacheStore) *Judgment {
	return New(cfg, cache, &http.Client{Timeout: defaultTimeout})
}

func TestPreFilterPinShortMessage(t *testing.T) {
	if got := PreFilter(Unit{Role: "user", Content: "short high-signal fact"}); got != ClassPin {
		t.Errorf("short message class = %s, want pin", got)
	}
}

func TestPreFilterDeadEmpty(t *testing.T) {
	if got := PreFilter(Unit{Role: "user", Content: "   "}); got != ClassDead {
		t.Errorf("whitespace message class = %s, want dead", got)
	}
}

func TestPreFilterDeadLabel(t *testing.T) {
	if got := PreFilter(Unit{Role: "user", Content: "##"}); got != ClassDead {
		t.Errorf("marker message class = %s, want dead", got)
	}
}

func TestPreFilterRemainderLong(t *testing.T) {
	long := strings.Repeat("word ", 100) // 100 words > pinMaxWords
	if got := PreFilter(Unit{Role: "user", Content: long}); got != ClassRemainder {
		t.Errorf("long message class = %s, want remainder", got)
	}
}

// TestPreFilterDeterministic proves the pre-filter is a pure function: the same
// unit always yields the same class (AC: pre-filter determinism).
func TestPreFilterDeterministic(t *testing.T) {
	u := Unit{Role: "user", Content: strings.Repeat("word ", 40)}
	first := PreFilter(u)
	for i := 0; i < 5; i++ {
		if got := PreFilter(u); got != first {
			t.Fatalf("pre-filter not deterministic: first=%s got=%s", first, got)
		}
	}
}

// TestJudgeFailOpenNoServer proves ADR-0016: with no model reachable, the
// judge fails open — a remainder unit degrades to KEEP (never drop, never an
// error), and a pin stays pinned.
func TestJudgeFailOpenNoServer(t *testing.T) {
	cfg := config.DefaultCtxConfig().Judgment
	cfg.Endpoint = "http://127.0.0.1:19999/v1/systemone" // nothing listens here
	j := newEngine(cfg, nil)

	// A remainder unit (long) with a down model → fail open to keep.
	d, err := j.Judge(context.Background(), Unit{Role: "user", Content: strings.Repeat("word ", 50)})
	if err != nil {
		t.Fatalf("Judge must fail open, not error: %v", err)
	}
	if d.Action != ActionKeep {
		t.Errorf("fail-open action = %s, want keep (never drop)", d.Action)
	}

	// A pin unit is kept by the deterministic pre-filter (no model needed).
	dp, _ := j.Judge(context.Background(), Unit{Role: "user", Content: "short fact"})
	if dp.Action != ActionKeep {
		t.Errorf("pin action = %s, want keep", dp.Action)
	}
}

// TestJudgeBelowThresholdTruncates proves the decision rule: a model keep-score
// BELOW the configured threshold truncates (ADR-0034).
func TestJudgeBelowThresholdTruncates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"score":0.10}`))
	}))
	defer srv.Close()

	cfg := config.DefaultCtxConfig().Judgment
	cfg.Endpoint = srv.URL
	j := newEngine(cfg, nil)

	// A remainder unit (long) so the model is consulted.
	d, err := j.Judge(context.Background(), Unit{Role: "user", Content: strings.Repeat("word ", 50)})
	if err != nil {
		t.Fatalf("Judge: %v", err)
	}
	if d.Action != ActionTruncate {
		t.Errorf("score 0.10 < threshold 0.22 → action = %s, want truncate", d.Action)
	}
	if d.KeepScore != 0.10 {
		t.Errorf("keep score = %f, want 0.10", d.KeepScore)
	}
}

// TestJudgeAboveThresholdKeeps proves the decision rule: a model keep-score AT
// OR ABOVE the configured threshold keeps the unit.
func TestJudgeAboveThresholdKeeps(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"score":0.85}`))
	}))
	defer srv.Close()

	cfg := config.DefaultCtxConfig().Judgment
	cfg.Endpoint = srv.URL
	j := newEngine(cfg, nil)

	d, err := j.Judge(context.Background(), Unit{Role: "user", Content: strings.Repeat("word ", 50)})
	if err != nil {
		t.Fatalf("Judge: %v", err)
	}
	if d.Action != ActionKeep {
		t.Errorf("score 0.85 >= threshold 0.22 → action = %s, want keep", d.Action)
	}
}

// TestJudgeEnvelopeScore proves the Ollama envelope shape (a score embedded in
// message content) is parsed, and the decision follows the threshold.
func TestJudgeEnvelopeScore(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":{"content":"{\"score\": 0.42}"}}`))
	}))
	defer srv.Close()

	cfg := config.DefaultCtxConfig().Judgment
	cfg.Endpoint = srv.URL
	j := newEngine(cfg, nil)

	d, err := j.Judge(context.Background(), Unit{Role: "user", Content: strings.Repeat("word ", 50)})
	if err != nil {
		t.Fatalf("Judge: %v", err)
	}
	if d.KeepScore != 0.42 {
		t.Errorf("envelope keep score = %f, want 0.42", d.KeepScore)
	}
	if d.Action != ActionKeep {
		t.Errorf("score 0.42 >= 0.22 → action = %s, want keep", d.Action)
	}
}

// TestFingerprintDeterministic proves the fingerprint is a stable function of
// the unit (same input → same fingerprint; different → different).
func TestFingerprintDeterministic(t *testing.T) {
	u := Unit{Role: "user", Content: "hello world"}
	if Fingerprint(u) != Fingerprint(u) {
		t.Error("fingerprint not deterministic for same unit")
	}
	if Fingerprint(u) == Fingerprint(Unit{Role: "user", Content: "different"}) {
		t.Error("different units must have different fingerprints")
	}
	// Role is part of the fingerprint: same content, different role differs.
	if Fingerprint(u) == Fingerprint(Unit{Role: "assistant", Content: "hello world"}) {
		t.Error("same content with different role must differ")
	}
}

// TestCompressTruncateOnly proves the compressor is truncate-only: a keep
// decision leaves the unit byte-identical; a truncate decision caps to a strict
// prefix (never expands, never deletes).
func TestCompressTruncateOnly(t *testing.T) {
	// Keep: unchanged.
	u := Unit{Role: "user", Content: "a b c d e f g"}
	if got := Compress(u, Decision{Action: ActionKeep}); got.Content != u.Content {
		t.Errorf("keep compressed = %q, want unchanged %q", got.Content, u.Content)
	}

	// Truncate: capped to a strict prefix, never longer.
	long := strings.Repeat("word ", 200)
	lu := Unit{Role: "user", Content: long}
	c := Compress(lu, Decision{Action: ActionTruncate})
	if len(c.Content) >= len(long) {
		t.Errorf("truncate did not shorten: len %d >= %d", len(c.Content), len(long))
	}
	words := strings.Fields(c.Content)
	if len(words) > truncateBudgetWords {
		t.Errorf("truncated to %d words, want <= %d", len(words), truncateBudgetWords)
	}
	// The kept prefix must be the original's first words (no rewrite).
	for i, w := range words {
		if i >= len(strings.Fields(long)) || w != strings.Fields(long)[i] {
			t.Fatalf("truncated content is not a strict prefix of the original")
		}
	}
}

// TestCompressNeverExpands proves a unit already within the truncate budget is
// left as-is (a no-op, never expanded).
func TestCompressNeverExpands(t *testing.T) {
	u := Unit{Role: "user", Content: "short unit"}
	if got := Compress(u, Decision{Action: ActionTruncate}); got.Content != u.Content {
		t.Errorf("within-budget truncate expanded/changed: %q", got.Content)
	}
}

// memoryCache is a tiny in-memory CacheStore for cache tests.
type memoryCache struct {
	m map[string][2]any // fingerprint → {action string, score float64}
}

func (m *memoryCache) GetJudgmentCache(fp string) (string, float64, bool) {
	v, ok := m.m[fp]
	if !ok {
		return "", 0, false
	}
	return v[0].(string), v[1].(float64), true
}

func (m *memoryCache) PutJudgmentCache(fp, action string, score float64, judgedAt int64) {
	m.m[fp] = [2]any{action, score}
}

// TestJudgeCacheHitNoModel proves a fingerprint cache hit serves the decision
// with NO model call (the model endpoint is pointed at a dead address; the hit
// still resolves).
func TestJudgeCacheHitNoModel(t *testing.T) {
	mc := &memoryCache{m: map[string][2]any{}}
	cfg := config.DefaultCtxConfig().Judgment
	cfg.Endpoint = "http://127.0.0.1:19999/v1/systemone" // dead — a miss would fail open
	j := New(cfg, mc, &http.Client{Timeout: defaultTimeout})

	u := Unit{Role: "user", Content: strings.Repeat("word ", 50)}

	// Prime the cache with a truncate decision.
	mc.PutJudgmentCache(Fingerprint(u), "trunc", 0.10, 0)

	d, err := j.Judge(context.Background(), u)
	if err != nil {
		t.Fatalf("Judge: %v", err)
	}
	if !d.FromCache {
		t.Error("decision not served from cache")
	}
	if d.Action != ActionTruncate {
		t.Errorf("cached action = %s, want truncate", d.Action)
	}
}

// TestJudgeCachesModelDecision proves a model decision for a remainder unit is
// persisted so a second identical unit is served from cache (FromCache=true).
func TestJudgeCachesModelDecision(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"score":0.05}`))
	}))
	defer srv.Close()

	mc := &memoryCache{m: map[string][2]any{}}
	cfg := config.DefaultCtxConfig().Judgment
	cfg.Endpoint = srv.URL
	j := New(cfg, mc, &http.Client{Timeout: defaultTimeout})

	u := Unit{Role: "user", Content: strings.Repeat("word ", 50)}

	if _, err := j.Judge(context.Background(), u); err != nil {
		t.Fatalf("first Judge: %v", err)
	}
	if calls != 1 {
		t.Fatalf("model calls after first judge = %d, want 1", calls)
	}

	// Second identical unit: served from cache, no new model call.
	d, err := j.Judge(context.Background(), u)
	if err != nil {
		t.Fatalf("second Judge: %v", err)
	}
	if !d.FromCache {
		t.Error("second decision not served from cache")
	}
	if calls != 1 {
		t.Errorf("model calls after cached judge = %d, want 1 (no new call)", calls)
	}
	if d.Action != ActionTruncate {
		t.Errorf("cached action = %s, want truncate", d.Action)
	}
}
