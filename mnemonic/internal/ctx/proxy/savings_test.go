package proxy

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// TestEstimateTokensStable is the deterministic token-estimate contract: the
// fixture's before/after delta must be exact. This is the invariant that makes
// X-CTX-Savings accurate on the AC fixture.
func TestEstimateTokensStable(t *testing.T) {
	// The AC fixture: count whitespace-delimited tokens on the wire bytes.
	// The JSON structure produces 7 fields (the {, "messages" etc. merge
	// around the 9 content words).
	body := []byte(`{"messages":[{"role":"user","content":"the quick brown"},{"role":"assistant","content":"fox jumps over"},{"role":"user","content":"the lazy dog"}]}`)
	if got := EstimateTokens(body); got != 7 {
		t.Fatalf("EstimateTokens(fixture) = %d, want 7", got)
	}
	// Whitespace-only tokens collapse: repeated whitespace is one boundary.
	if got := EstimateTokens([]byte(`{"a":"   "}`)); got != 2 {
		t.Errorf("EstimateTokens(whitespace) = %d, want 2 (the {\"a\": and })", got)
	}
}

// TestTransformPassthrough proves passthrough leaves the messages unchanged
// (delta 0).
func TestTransformPassthrough(t *testing.T) {
	in := []message{{Role: "user", Content: "the quick brown"}}
	out, err := transform(ModePassthrough, in)
	if err != nil {
		t.Fatalf("transform(passthrough): %v", err)
	}
	if !reflect.DeepEqual(out, in) {
		t.Errorf("passthrough should be unchanged: got %v", out)
	}
}

// TestTransformTokenMaximizesCompression proves token mode reduces every
// non-empty message, producing a smaller after-byte count (non-zero delta).
func TestTransformTokenMaximizesCompression(t *testing.T) {
	in := []message{
		{Role: "user", Content: "the quick brown"},
		{Role: "assistant", Content: "fox jumps over"},
	}
	out, err := transform(ModeToken, in)
	if err != nil {
		t.Fatalf("transform(token): %v", err)
	}
	before := EstimateTokens(mustEncode(t, in))
	after := EstimateTokens(mustEncode(t, out))
	if after >= before {
		t.Errorf("token mode must compress: before=%d after=%d", before, after)
	}
	// Each reduced message carries the deterministic summary marker.
	for i, m := range out {
		if !strings.Contains(m.Content, "[ctx-summarized") {
			t.Errorf("token mode message %d not summarized: %q", i, m.Content)
		}
	}
}

// TestTransformCacheCompressesOnlyTail proves cache mode freezes the stable
// prefix (everything up to and including the last user message) and compresses
// only the mutable tail.
func TestTransformCacheCompressesOnlyTail(t *testing.T) {
	in := []message{
		{Role: "user", Content: "frozen one"},        // head
		{Role: "assistant", Content: "frozen two"},   // head
		{Role: "user", Content: "frozen three"},      // head (last user)
		{Role: "assistant", Content: "mutable tail"}, // tail
	}
	out, err := transform(ModeCache, in)
	if err != nil {
		t.Fatalf("transform(cache): %v", err)
	}
	// Head (first 3) unchanged.
	for i := 0; i < 3; i++ {
		if out[i] != in[i] {
			t.Errorf("cache head %d changed: %v -> %v", i, in[i], out[i])
		}
	}
	// Tail (index 3) summarized.
	if !strings.Contains(out[3].Content, "[ctx-summarized") {
		t.Errorf("cache tail not summarized: %q", out[3].Content)
	}
}

func TestTransformUnknownMode(t *testing.T) {
	if _, err := transform(Mode("bogus"), []message{{Role: "user", Content: "x"}}); err == nil {
		t.Errorf("transform(bogus) should error")
	}
}

// TestSavingsWriteHeadersAccurate proves the five savings headers are written
// with the exact before/after/mode/cache-hit values (plan 16.5).
func TestSavingsWriteHeadersAccurate(t *testing.T) {
	cases := []struct {
		name      string
		sv        savings
		wantSave  string
		wantOrig  string
		wantComp  string
		wantMode  string
		wantCache string
	}{
		{"delta", savings{Before: 10, After: 4, Mode: ModeToken, CacheHit: false}, "6", "10", "4", "token", "false"},
		{"passthrough-zero", savings{Before: 7, After: 7, Mode: ModePassthrough}, "0", "7", "7", "passthrough", "false"},
		{"cache-hit", savings{Before: 9, After: 5, Mode: ModeCache, CacheHit: true}, "4", "9", "5", "cache", "true"},
		{"cache-miss", savings{Before: 9, After: 5, Mode: ModeCache, CacheHit: false}, "4", "9", "5", "cache", "false"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tc.sv.writeHeaders(w)
			if got := w.Header().Get(HeaderSavings); got != tc.wantSave {
				t.Errorf("X-CTX-Savings = %q, want %q", got, tc.wantSave)
			}
			if got := w.Header().Get(HeaderOriginal); got != tc.wantOrig {
				t.Errorf("X-CTX-Original-Tokens = %q, want %q", got, tc.wantOrig)
			}
			if got := w.Header().Get(HeaderCompressed); got != tc.wantComp {
				t.Errorf("X-CTX-Compressed-Tokens = %q, want %q", got, tc.wantComp)
			}
			if got := w.Header().Get(HeaderMode); got != tc.wantMode {
				t.Errorf("X-CTX-Mode = %q, want %q", got, tc.wantMode)
			}
			if got := w.Header().Get(HeaderCacheHit); got != tc.wantCache {
				t.Errorf("X-CTX-Cache-Hit = %q, want %q", got, tc.wantCache)
			}
		})
	}
}

func TestSavingsRatio(t *testing.T) {
	if got := (savings{Before: 10, After: 4}).savingsRatio(); got != 0.6 {
		t.Errorf("savingsRatio = %v, want 0.6", got)
	}
	if got := (savings{Before: 0, After: 0}).savingsRatio(); got != 0 {
		t.Errorf("savingsRatio(0,0) = %v, want 0 (no div by zero)", got)
	}
}

func mustEncode(t *testing.T, msgs []message) []byte {
	t.Helper()
	b, err := json.Marshal(messages{Messages: msgs})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	return b
}
