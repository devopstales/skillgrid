package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// compressBody is the shared test fixture: a 3-message conversation whose
// token count drops measurably under token/cache transforms.
const compressBody = `{"model":"claude-3-5-sonnet","max_tokens":1024,"messages":[{"role":"user","content":"the quick brown fox"},{"role":"assistant","content":"jumps over the lazy dog"},{"role":"user","content":"tell me a joke please"}]}`

// [compress-endpoint-loopback] Loopback /v1/compress: a loopback POST
// /v1/compress with a message-list body returns a compressed message list
// (valid JSON, messages transformed per the current mode) and NO upstream
// call is made (the upstream httptest server saw 0 requests).
func TestCompressLoopbackNoUpstream(t *testing.T) {
	var seen []string
	up := newUpstreamServer(t, http.StatusOK, `{"type":"message"}`, &seen)
	defer up.Close()
	cfg := testConfig("token", 0, 0)
	p := newTestProxy(t, cfg, nil, ModeToken, up)

	req := httptest.NewRequest(http.MethodPost, "/v1/compress", strings.NewReader(compressBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:9999"
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	// No upstream call: the compress endpoint is a local transform, not a
	// forwarded request.
	if len(seen) != 0 {
		t.Fatalf("upstream requests = %d, want 0 (compress must not dial upstream)", len(seen))
	}
	// The response is a valid JSON object with a transformed messages array.
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	msgBytes, ok := doc["messages"]
	if !ok {
		t.Fatalf("response has no messages field: %s", w.Body.String())
	}
	// Token mode summarizes every non-empty message.
	if !strings.Contains(string(msgBytes), "[ctx-summarized") {
		t.Errorf("messages not summarized in token mode: %s", msgBytes)
	}
	// Non-message fields are preserved (reuses applyTransform's in-place shape).
	if doc["model"] == nil {
		t.Errorf("model field not preserved in compress response")
	}
	// No savings headers: compress has no before/after vs. an upstream response.
	if got := w.Header().Get(HeaderSavings); got != "" {
		t.Errorf("X-CTX-Savings = %q, want empty (no upstream on compress)", got)
	}
}

// [compress-endpoint-loopback] Cache mode: /v1/compress transforms only the
// mutable tail (messages after the frozen head) and leaves the stable prefix
// intact — the same headSplit boundary the forward path uses.
func TestCompressLoopbackCacheMode(t *testing.T) {
	var seen []string
	up := newUpstreamServer(t, http.StatusOK, `{"type":"message"}`, &seen)
	defer up.Close()
	cfg := testConfig("cache", 0, 0)
	p := newTestProxy(t, cfg, nil, ModeCache, up)

	req := httptest.NewRequest(http.MethodPost, "/v1/compress", strings.NewReader(compressBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:9999"
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if len(seen) != 0 {
		t.Fatalf("upstream requests = %d, want 0", len(seen))
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	// The frozen head (up to and including the last user message — here all 3
	// messages, since the last message is a user message) stays intact.
	msgBytes, _ := doc["messages"]
	if !strings.Contains(string(msgBytes), "the quick brown fox") {
		t.Errorf("cache mode must leave the frozen head intact: %s", msgBytes)
	}
}

// [compress-endpoint-loopback] Non-loopback /v1/compress: a POST from a
// non-loopback origin returns 404 — the route is invisible, deliberately 404
// (not 403) so a scanner cannot distinguish it from a missing path.
func TestCompressNonLoopback404Not403(t *testing.T) {
	var seen []string
	up := newUpstreamServer(t, http.StatusOK, `{"type":"message"}`, &seen)
	defer up.Close()
	cfg := testConfig("token", 0, 0)
	p := newTestProxy(t, cfg, nil, ModeToken, up)

	req := httptest.NewRequest(http.MethodPost, "/v1/compress", strings.NewReader(compressBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "10.0.0.5:44321" // non-loopback origin
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 for non-loopback origin", w.Code)
	}
	if w.Code == http.StatusForbidden {
		t.Fatalf("status = 403; must be 404 so the route is invisible to scanners")
	}
	// The non-loopback request must not reach the transform or the upstream.
	if len(seen) != 0 {
		t.Fatalf("upstream requests = %d, want 0", len(seen))
	}
}

// [compress-endpoint-loopback] Non-loopback 404 is method-agnostic: a GET
// (not just POST) from a non-loopback origin also returns 404.
func TestCompressNonLoopbackAnyMethod404(t *testing.T) {
	var seen []string
	up := newUpstreamServer(t, http.StatusOK, `{"type":"message"}`, &seen)
	defer up.Close()
	cfg := testConfig("token", 0, 0)
	p := newTestProxy(t, cfg, nil, ModeToken, up)

	req := httptest.NewRequest(http.MethodGet, "/v1/compress", nil)
	req.RemoteAddr = "192.168.1.10:5555"
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("GET from non-loopback: status = %d, want 404", w.Code)
	}
}

// [compress-endpoint-loopback] ADR-0016 fail-open: a /v1/compress body that
// fails the transform (JSON object where an array is expected) degrades to
// the untransformed input with 200 — no 500, no upstream call.
func TestCompressFailOpenReturnsInput(t *testing.T) {
	var seen []string
	up := newUpstreamServer(t, http.StatusOK, `{"type":"message"}`, &seen)
	defer up.Close()
	cfg := testConfig("token", 0, 0)
	p := newTestProxy(t, cfg, nil, ModeToken, up)

	// messages is a JSON object, not an array: the transform faults.
	badBody := `{"messages":{"role":"user"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/compress", strings.NewReader(badBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:9999"
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (fail-open must not 500)", w.Code)
	}
	if len(seen) != 0 {
		t.Fatalf("upstream requests = %d, want 0", len(seen))
	}
	// The untransformed input list is returned so the caller still completes.
	if !strings.Contains(w.Body.String(), `"role":"user"`) {
		t.Errorf("fail-open response should echo the input: %s", w.Body.String())
	}
}

// [compress-endpoint-loopback] Nil store: /v1/compress works with a nil ctx
// store (no telemetry, no 500).
func TestCompressNilStoreNoop(t *testing.T) {
	up := newUpstreamServer(t, http.StatusOK, `{"type":"message"}`, nil)
	defer up.Close()
	cfg := testConfig("token", 0, 0)
	p := newTestProxy(t, cfg, nil, ModeToken, up)

	req := httptest.NewRequest(http.MethodPost, "/v1/compress", strings.NewReader(compressBody))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:9999"
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (nil store must not 500)", w.Code)
	}
	// Drain the body so the test does not leak a goroutine holding it.
	if _, err := io.ReadAll(w.Body); err != nil {
		t.Fatalf("reading response: %v", err)
	}
}

// [mode-switch-emitted] GET /stats: after a runtime mode switch, /stats
// reports the current mode as JSON. The key T06 requirement is that the
// current mode is exposed.
func TestStatsReportsCurrentMode(t *testing.T) {
	var seen []string
	up := newUpstreamServer(t, http.StatusOK, `{"type":"message"}`, &seen)
	defer up.Close()
	cs := newSharedStore(t, "ctxstats")
	cfg := testConfig("cache", 0, 0)
	p := newTestProxy(t, cfg, cs, ModeCache, up)

	// Switch cache -> token at runtime.
	old, err := p.SetMode(ModeToken, "test switch")
	if err != nil {
		t.Fatalf("SetMode: %v", err)
	}
	if old != ModeCache {
		t.Fatalf("SetMode returned old = %q, want cache", old)
	}

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	req.RemoteAddr = "127.0.0.1:9999"
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var stats map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &stats); err != nil {
		t.Fatalf("stats response is not valid JSON: %v", err)
	}
	if stats["mode"] != "token" {
		t.Errorf("stats mode = %v, want token (current mode after switch)", stats["mode"])
	}
}
