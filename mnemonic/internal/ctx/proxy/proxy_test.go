package proxy

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/ctx/config"
	ctxstore "github.com/devopstales/skillgrid/mnemonic/internal/ctx/store"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// errTestDial is a transport-level error used by tests to simulate an
// unreachable provider.
var errTestDial = errors.New("test: dial tcp connection refused")

// testConfig returns a CtxConfig tuned for proxy integration tests: savings
// headers on, minimal retries, the default cache mode, and the bind/port the
// AC expects. ProviderRetries is low so the retry-budget tests stay fast; the
// backoff is 0 so no real sleeping occurs even on the real-sleeper path.
func testConfig(mode string, retries, backoffMs int) *config.CtxConfig {
	cfg := config.DefaultCtxConfig()
	cfg.Proxy.Mode = mode
	cfg.Proxy.SavingsHeaders = true
	cfg.Proxy.ProviderRetries = retries
	cfg.Proxy.ProviderRetryBackoffMs = backoffMs
	return &cfg
}

// testLogger returns a logger that discards output (test noise suppression).
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newTestProxy builds a Proxy wired to a test upstream (an httptest server),
// an optional ctx store accessor (nil for no-telemetry tests), and the noSleep
// seam. The returned upstream is closed via t.Cleanup.
func newTestProxy(t *testing.T, cfg *config.CtxConfig, cs *ctxstore.Store, mode Mode, upstream *httptest.Server) *Proxy {
	t.Helper()
	ns := &noSleep{}
	p := NewWithSleep(cfg, cs, testLogger(), mode, ns.sleep, upstream.Client())
	// Override the detected upstream with the test server.
	p.upstream = baseUpstream{
		Name:       "test",
		BaseURL:    upstream.URL,
		AuthHeader: "x-api-key",
		AuthValue:  "test-key",
		client:     upstream.Client(),
	}
	return p
}

// newSharedStore opens a real shared project store (migrations applied) for
// telemetry tests, wrapping it in a ctx store accessor.
func newSharedStore(t *testing.T, project string) *ctxstore.Store {
	t.Helper()
	shared, err := store.Open(t.TempDir(), project)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { shared.Close() })
	return ctxstore.New(shared)
}

// newUpstreamServer builds a generic upstream that returns the given status
// and body, recording every request it sees.
func newUpstreamServer(t *testing.T, status int, body string, seen *[]string) *httptest.Server {
	t.Helper()
	return newUpstreamServerWithContentType(t, status, body, "application/json", seen)
}

// newUpstreamServerWithContentType is newUpstreamServer with an explicit
// response Content-Type (for streaming-passthrough tests).
func newUpstreamServerWithContentType(t *testing.T, status int, body, contentType string, seen *[]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if seen != nil {
			*seen = append(*seen, string(b))
		}
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

// TestServeHTTPPassthroughDeltaZero proves the passthrough mode: the client's
// exact body is forwarded verbatim, the response is passed through, and the
// savings header reports delta 0 (before == after).
func TestServeHTTPPassthroughDeltaZero(t *testing.T) {
	clientBody := []byte(`{"messages":[{"role":"user","content":"the quick brown"}]}`)
	var seen []string
	up := newUpstreamServer(t, http.StatusOK, `{"type":"message"}`, &seen)
	defer up.Close()
	cfg := testConfig("passthrough", 0, 0)
	p := newTestProxy(t, cfg, nil, ModePassthrough, up)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(string(clientBody)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	// Passthrough forwards the ORIGINAL bytes (not the re-encoded shape).
	if len(seen) != 1 || seen[0] != string(clientBody) {
		t.Fatalf("forwarded body = %q, want verbatim client body %q", seen, clientBody)
	}
	if got := w.Header().Get(HeaderSavings); got != "0" {
		t.Errorf("X-CTX-Savings = %q, want 0 (passthrough delta)", got)
	}
	if got := w.Header().Get(HeaderMode); got != "passthrough" {
		t.Errorf("X-CTX-Mode = %q, want passthrough", got)
	}
	if got := w.Header().Get(HeaderOriginal); got != w.Header().Get(HeaderCompressed) {
		t.Errorf("original %q != compressed %q (passthrough must be equal)", w.Header().Get(HeaderOriginal), w.Header().Get(HeaderCompressed))
	}
}

// TestServeHTTPTokenExactSavings is the AC fixture: the 3-message body the AC
// names. Token mode must compress every message, the savings header must equal
// tokens_before - tokens_after exactly, and the forwarded body must be the
// compressed shape (smaller than the original).
func TestServeHTTPTokenExactSavings(t *testing.T) {
	clientBody := []byte(`{"messages":[{"role":"user","content":"the quick brown"},{"role":"assistant","content":"fox jumps over"},{"role":"user","content":"the lazy dog"}]}`)
	var seen []string
	up := newUpstreamServer(t, http.StatusOK, `{"type":"message"}`, &seen)
	defer up.Close()
	cfg := testConfig("token", 0, 0)
	p := newTestProxy(t, cfg, nil, ModeToken, up)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(string(clientBody)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	before := EstimateTokens(clientBody)
	after := EstimateTokens([]byte(seen[0]))
	if after >= before {
		t.Fatalf("token mode must compress the wire: before=%d after=%d", before, after)
	}
	wantDelta := before - after
	if got := w.Header().Get(HeaderSavings); got != itoa(wantDelta) {
		t.Errorf("X-CTX-Savings = %q, want %d (before-after)", got, wantDelta)
	}
	if got := w.Header().Get(HeaderOriginal); got != itoa(before) {
		t.Errorf("X-CTX-Original-Tokens = %q, want %d", got, before)
	}
	if got := w.Header().Get(HeaderCompressed); got != itoa(after) {
		t.Errorf("X-CTX-Compressed-Tokens = %q, want %d", got, after)
	}
	if got := w.Header().Get(HeaderMode); got != "token" {
		t.Errorf("X-CTX-Mode = %q, want token", got)
	}
	// The compressed body must carry the deterministic summary marker.
	if !strings.Contains(seen[0], "[ctx-summarized") {
		t.Errorf("forwarded body not summarized: %q", seen[0])
	}
}

// TestServeHTTPFailOpenOnTransformFault proves ADR-0016 fail-open at the
// transport: a request body that parses as JSON but whose messages array
// cannot be encoded (a transform fault) degrades to passthrough — the
// ORIGINAL body is forwarded, the mode header reports passthrough, the delta
// is 0, and the client still gets the upstream's 200.
func TestServeHTTPFailOpenOnTransformFault(t *testing.T) {
	// A body with a top-level "messages" that is a JSON object (not an array):
	// parseMessages unmarshals into messages.Messages []message and fails, so
	// the proxy fails open to passthrough with the original bytes.
	clientBody := []byte(`{"messages":{"role":"user"}}`)
	var seen []string
	up := newUpstreamServer(t, http.StatusOK, `{"type":"message"}`, &seen)
	defer up.Close()
	cfg := testConfig("token", 0, 0)
	p := newTestProxy(t, cfg, nil, ModeToken, up)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(string(clientBody)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (fail-open must not 500)", w.Code)
	}
	if len(seen) != 1 || seen[0] != string(clientBody) {
		t.Fatalf("fail-open must forward the original body, got %q", seen)
	}
	if got := w.Header().Get(HeaderMode); got != "passthrough" {
		t.Errorf("X-CTX-Mode = %q, want passthrough (degraded)", got)
	}
	if got := w.Header().Get(HeaderSavings); got != "0" {
		t.Errorf("X-CTX-Savings = %q, want 0 (degraded to passthrough)", got)
	}
}

// TestServeHTTPProviderRetryVerbatim proves the provider-retry path: a
// persistent non-2xx is retried up to the budget (1 + maxRetries attempts) and
// then returned to the client verbatim (the provider's last status + body),
// not a 500.
func TestServeHTTPProviderRetryVerbatim(t *testing.T) {
	var seen []string
	up := newUpstreamServer(t, http.StatusTooManyRequests, `{"error":"rate limited"}`, &seen)
	defer up.Close()
	cfg := testConfig("passthrough", 2, 0) // 2 retries, no backoff
	p := newTestProxy(t, cfg, nil, ModePassthrough, up)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"messages":[]}`))
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	// 1 initial + 2 retries = 3 attempts hit the upstream.
	if len(seen) != 3 {
		t.Fatalf("upstream attempts = %d, want 3 (1 + 2 retries)", len(seen))
	}
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("client status = %d, want 429 (verbatim provider status)", w.Code)
	}
	if got := w.Body.String(); got != `{"error":"rate limited"}` {
		t.Errorf("client body = %q, want verbatim provider body", got)
	}
}

// TestServeHTTPOpenAICompatEndToEnd is the OpenAI-compatibility end-to-end
// check (AC 2b): the proxy must forward an OpenAI-shape request to the
// configured upstream and pass the response (and savings headers) through to
// the client unchanged.
func TestServeHTTPOpenAICompatEndToEnd(t *testing.T) {
	clientBody := []byte(`{"model":"gpt-4","messages":[{"role":"user","content":"hi there"}],"temperature":0.2}`)
	var seen []string
	up := newUpstreamServer(t, http.StatusOK, `{"id":"chatcmpl-1","object":"chat.completion","choices":[{"message":{"role":"assistant","content":"hello"}}]}`, &seen)
	defer up.Close()
	cfg := testConfig("passthrough", 0, 0)
	p := newTestProxy(t, cfg, nil, ModePassthrough, up)

	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(string(clientBody)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	// Passthrough forwards the exact OpenAI request body verbatim.
	if len(seen) != 1 || seen[0] != string(clientBody) {
		t.Fatalf("forwarded body = %q, want verbatim OpenAI body %q", seen, clientBody)
	}
	// The upstream response passes through to the client unchanged.
	if got := w.Body.String(); got != `{"id":"chatcmpl-1","object":"chat.completion","choices":[{"message":{"role":"assistant","content":"hello"}}]}` {
		t.Errorf("client body = %q, want verbatim upstream response", got)
	}
	// Savings headers are still written on the OpenAI-compat path.
	if got := w.Header().Get(HeaderMode); got != "passthrough" {
		t.Errorf("X-CTX-Mode = %q, want passthrough", got)
	}
	if got := w.Header().Get(HeaderSavings); got != "0" {
		t.Errorf("X-CTX-Savings = %q, want 0 (passthrough)", got)
	}
}

// TestServeHTTPTransportError502 proves that when the provider is unreachable
// (a transport-level error with no upstream status), the proxy renders a 502
// after exhausting retries — the client still completes, no panic, no 500.
func TestServeHTTPTransportError502(t *testing.T) {
	// A client whose transport always errors (connection refused) — no upstream
	// status is ever produced.
	client := &http.Client{Transport: errorTransport{errTestDial}}
	cfg := testConfig("passthrough", 1, 0)
	p := NewWithSleep(cfg, nil, testLogger(), ModePassthrough, func(time.Duration) {}, client)
	p.upstream = baseUpstream{Name: "anthropic", BaseURL: "http://127.0.0.1:1", AuthHeader: "x-api-key", AuthValue: "k", client: client}

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"messages":[]}`))
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("client status = %d, want 502 (provider unreachable)", w.Code)
	}
	if !strings.Contains(w.Body.String(), "provider unreachable") {
		t.Errorf("502 body = %q, want a 'provider unreachable' message", w.Body.String())
	}
}

// TestServeHTTPWritesTelemetry proves the proxy records a per-turn savings row
// and a session row on a real shared store: after one proxied request, the
// ctx_proxy_savings and ctx_proxy_sessions tables each carry a row reflecting
// the request.
func TestServeHTTPWritesTelemetry(t *testing.T) {
	cs := newSharedStore(t, "ctxtelemetry")
	clientBody := []byte(`{"messages":[{"role":"user","content":"hello world test"}]}`)
	up := newUpstreamServer(t, http.StatusOK, `{"type":"message"}`, nil)
	defer up.Close()
	cfg := testConfig("token", 0, 0)
	p := newTestProxy(t, cfg, cs, ModeToken, up)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(string(clientBody)))
	req.Header.Set(SessionHeader, "sess-telemetry")
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	// A per-turn savings row was written for the session.
	savings, err := cs.ListProxySavings("sess-telemetry")
	if err != nil {
		t.Fatalf("ListProxySavings: %v", err)
	}
	if len(savings) != 1 {
		t.Fatalf("savings rows = %d, want 1", len(savings))
	}
	if savings[0].Mode != "token" {
		t.Errorf("savings mode = %q, want token", savings[0].Mode)
	}
	if savings[0].TokensAfter >= savings[0].TokensBefore {
		t.Errorf("token mode must compress: before=%d after=%d", savings[0].TokensBefore, savings[0].TokensAfter)
	}
	// A session row was written with the aggregated counters.
	sess, err := cs.GetProxySession("sess-telemetry")
	if err != nil {
		t.Fatalf("GetProxySession: %v", err)
	}
	if sess == nil {
		t.Fatalf("session row not written for %q", "sess-telemetry")
	}
	if sess.Requests != 1 {
		t.Errorf("session requests = %d, want 1", sess.Requests)
	}
	if sess.TokensBefore != savings[0].TokensBefore || sess.TokensAfter != savings[0].TokensAfter {
		t.Errorf("session counters (%d/%d) != savings row (%d/%d)", sess.TokensBefore, sess.TokensAfter, savings[0].TokensBefore, savings[0].TokensAfter)
	}
}

// TestServeHTTPTelemetryWritesStatsHistory proves the per-turn bucket writer is
// wired into recordTelemetry: after ONE proxied turn, a ctx_stats_history row
// exists for the turn's hour bucket with the compressed token delta, one
// request, and one session (the per-saving lower bound). This is the
// integration gap the reviewer flagged — before the wiring, the writer was
// built + unit-tested but never called in production, so GET /stats-history
// stayed empty in a deployed process.
func TestServeHTTPTelemetryWritesStatsHistory(t *testing.T) {
	cs := newSharedStore(t, "ctxstatshist")
	clientBody := []byte(`{"messages":[{"role":"user","content":"hello world test"}]}`)
	up := newUpstreamServer(t, http.StatusOK, `{"type":"message"}`, nil)
	defer up.Close()
	cfg := testConfig("token", 0, 0)
	p := newTestProxy(t, cfg, cs, ModeToken, up)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(string(clientBody)))
	req.Header.Set(SessionHeader, "sess-statshist")
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	// The turn's savings row (source of the fold) must carry the delta.
	savings, err := cs.ListProxySavings("sess-statshist")
	if err != nil {
		t.Fatalf("ListProxySavings: %v", err)
	}
	if len(savings) != 1 {
		t.Fatalf("savings rows = %d, want 1", len(savings))
	}
	wantSaved := savings[0].TokensBefore - savings[0].TokensAfter

	// The per-turn writer must have folded the turn into the hour bucket.
	buckets, err := cs.ListStatsBuckets("hourly")
	if err != nil {
		t.Fatalf("ListStatsBuckets(hourly): %v", err)
	}
	if len(buckets) != 1 {
		t.Fatalf("hourly buckets = %d, want 1", len(buckets))
	}
	b := buckets[0]
	if b.TokensSaved != wantSaved {
		t.Errorf("hourly tokens_saved = %d, want %d (the turn's delta)", b.TokensSaved, wantSaved)
	}
	if b.Requests != 1 {
		t.Errorf("hourly requests = %d, want 1", b.Requests)
	}
	if b.Sessions != 1 {
		t.Errorf("hourly sessions = %d, want 1 (per-saving lower bound)", b.Sessions)
	}
}

// TestServeHTTPTelemetryNilStoreNoop proves ADR-0016 fail-open at telemetry:
// with a nil ctx store accessor the proxy still serves and returns the correct
// savings headers — the telemetry write is a no-op, never a 500.
func TestServeHTTPTelemetryNilStoreNoop(t *testing.T) {
	clientBody := []byte(`{"messages":[{"role":"user","content":"the quick brown"}]}`)
	up := newUpstreamServer(t, http.StatusOK, `{"type":"message"}`, nil)
	defer up.Close()
	cfg := testConfig("token", 0, 0)
	// nil ctx store accessor → telemetry no-ops.
	p := newTestProxy(t, cfg, nil, ModeToken, up)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(string(clientBody)))
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (nil store must not 500)", w.Code)
	}
	if got := w.Header().Get(HeaderSavings); got == "" {
		t.Errorf("X-CTX-Savings missing with nil store; headers must still be written")
	}
}

// TestServeHTTPSessionIDDerivedProves the proxy derives a stable session id
// from the client address when the X-CTX-Session header is absent, so the
// telemetry row is still keyed (deterministic per RemoteAddr).
func TestServeHTTPSessionIDDerived(t *testing.T) {
	cs := newSharedStore(t, "ctxderived")
	up := newUpstreamServer(t, http.StatusOK, `{"type":"message"}`, nil)
	defer up.Close()
	cfg := testConfig("passthrough", 0, 0)
	p := newTestProxy(t, cfg, cs, ModePassthrough, up)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"messages":[]}`))
	req.RemoteAddr = "127.0.0.1:9999"
	// No X-CTX-Session header: the proxy derives one from RemoteAddr.
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	sessions, err := cs.ListProxySessions()
	if err != nil {
		t.Fatalf("ListProxySessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("session rows = %d, want 1", len(sessions))
	}
	if !strings.HasPrefix(sessions[0].SessionID, "sess-") {
		t.Errorf("derived session id = %q, want a sess- prefixed id", sessions[0].SessionID)
	}
}

// TestServeHTTPStreamResponsePassthrough proves the streaming passthrough
// behavior: the proxy streams the upstream response body into the client
// (no full buffering) and preserves the upstream's Content-Type, so large
// streamed responses do not require the whole body in memory.
func TestServeHTTPStreamResponsePassthrough(t *testing.T) {
	big := strings.Repeat("x", 256*1024) // 256 KiB
	var seen []string
	up := newUpstreamServerWithContentType(t, http.StatusOK, big, "text/event-stream", &seen)
	defer up.Close()
	cfg := testConfig("passthrough", 0, 0)
	p := newTestProxy(t, cfg, nil, ModePassthrough, up)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"messages":[]}`))
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Errorf("Content-Type = %q, want the upstream's text/event-stream", got)
	}
	if w.Body.String() != big {
		t.Fatalf("streamed body length = %d, want %d", len(w.Body.String()), len(big))
	}
}

// TestTransformPreservesNonMessageFields proves the in-place transform keeps
// the request's non-message fields (model, max_tokens, temperature, ...) on
// the wire — only the messages array is compressed, so the upstream still
// receives the full request shape (review fix for the lossy re-encode).
func TestTransformPreservesNonMessageFields(t *testing.T) {
	body := []byte(`{"model":"claude-3-5-sonnet","max_tokens":1024,"temperature":0.7,"messages":[{"role":"user","content":"the quick brown"},{"role":"assistant","content":"fox jumps over"},{"role":"user","content":"the lazy dog"}]}`)
	out, cacheHit, err := applyTransform(ModeToken, body)
	if err != nil {
		t.Fatalf("applyTransform: %v", err)
	}
	if out == nil {
		t.Fatalf("applyTransform returned nil body")
	}
	if cacheHit {
		t.Errorf("cacheHit = true, want false (token mode)")
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("forwarded body is not valid JSON: %v", err)
	}
	if doc["model"] != "claude-3-5-sonnet" {
		t.Errorf("model = %v, want preserved claude-3-5-sonnet", doc["model"])
	}
	if doc["max_tokens"] != float64(1024) {
		t.Errorf("max_tokens = %v, want preserved 1024", doc["max_tokens"])
	}
	if doc["temperature"] != 0.7 {
		t.Errorf("temperature = %v, want preserved 0.7", doc["temperature"])
	}
	// The messages array is the one field that was compressed.
	msgBytes, err := json.Marshal(doc["messages"])
	if err != nil {
		t.Fatalf("marshal messages: %v", err)
	}
	if !strings.Contains(string(msgBytes), "[ctx-summarized") {
		t.Errorf("messages not summarized: %s", msgBytes)
	}
}

// errorTransport is a round-tripper that always returns a fixed error,
// simulating an unreachable provider.
type errorTransport struct{ err error }

func (e errorTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, e.err
}

// itoa is a tiny int→string helper so the tests read without importing
// strconv at the call site.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}
