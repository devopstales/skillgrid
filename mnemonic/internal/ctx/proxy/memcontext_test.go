package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	ctxstore "github.com/devopstales/skillgrid/mnemonic/internal/ctx/store"
	"github.com/devopstales/skillgrid/mnemonic/internal/ctx/traffic"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// TestProxyMemoryContextInjection is the KEY compression-context proof: a
// candidate traffic memory is NOT injected into the forwarded context; after
// Gate.Apply it IS (as a leading system message); after Gate.Revoke it is NOT
// again. The upstream records the exact body it receives so the assertions are
// on the wire bytes, not the in-process transform.
func TestProxyMemoryContextInjection(t *testing.T) {
	const (
		memID   = "mem-ctx"
		session = "sess-ctx"
		memText = "prefer compact diffs in output"
	)
	var seen []string
	up := newUpstreamServer(t, http.StatusOK, `{"type":"message"}`, &seen)
	defer up.Close()

	cs := newSharedStore(t, "p-ctx-mem")
	g := traffic.NewGate(cs)
	cs.PutTrafficMemory(&ctxstore.TrafficMemory{
		MemoryID:   memID,
		SessionID:  session,
		Category:   "preference",
		MemoryText: memText,
		Confidence: 0.9,
		CreatedAt:  1,
		Status:     "candidate",
	})

	cfg := testConfig("passthrough", 0, 0)
	p := newTestProxy(t, cfg, cs, ModePassthrough, up)
	p.SetGate(g)

	clientBody := `{"messages":[{"role":"user","content":"the quick brown"}]}`
	doForward := func() []byte {
		req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(clientBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(SessionHeader, session)
		w := httptest.NewRecorder()
		p.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("forward status = %d, want 200", w.Code)
		}
		if len(seen) == 0 {
			t.Fatal("upstream saw no request")
		}
		return []byte(seen[len(seen)-1])
	}
	leadSystem := func(body []byte) string {
		var doc struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.Unmarshal(body, &doc); err != nil {
			t.Fatalf("forwarded body is not the expected JSON: %v", err)
		}
		if len(doc.Messages) == 0 || doc.Messages[0].Role != "system" {
			return ""
		}
		return doc.Messages[0].Content
	}

	// 1. Candidate (status=candidate): the memory is NOT in the forwarded context.
	if got := leadSystem(doForward()); strings.Contains(got, memText) {
		t.Fatalf("candidate memory leaked into compression context: %q", got)
	}

	// 2. After Apply: the memory IS injected as the leading system message.
	if err := g.Apply(memID); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got := leadSystem(doForward()); !strings.Contains(got, memText) {
		t.Fatalf("applied memory not in compression context: %q", got)
	}

	// 3. After Revoke: the memory is NOT in the forwarded context again.
	if err := g.Revoke(memID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if got := leadSystem(doForward()); strings.Contains(got, memText) {
		t.Fatalf("revoked memory still in compression context: %q", got)
	}
}

// TestProxyMemoryContextFailOpen proves ADR-0016 on the request path: a nil
// gate (no gate installed) leaves the forwarded body byte-for-byte identical
// to the client's original — no context injection, no 500.
func TestProxyMemoryContextFailOpen(t *testing.T) {
	clientBody := []byte(`{"messages":[{"role":"user","content":"the quick brown"}]}`)
	var seen []string
	up := newUpstreamServer(t, http.StatusOK, `{"type":"message"}`, &seen)
	defer up.Close()

	cfg := testConfig("passthrough", 0, 0)
	p := newTestProxy(t, cfg, nil, ModePassthrough, up)
	// No SetGate: the proxy has no gate.

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(string(clientBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(SessionHeader, "sess-none")
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if len(seen) != 1 || seen[0] != string(clientBody) {
		t.Fatalf("forwarded body = %q, want the verbatim client body (no gate = unchanged)", seen)
	}
}

// TestProxyMemoryContextStoreFaultFailOpen proves ADR-0016 for a store that
// cannot read the memories (closed handle): the gate degrades to no context,
// the body is unchanged, and the request still 200s (never 500s).
func TestProxyMemoryContextStoreFaultFailOpen(t *testing.T) {
	clientBody := []byte(`{"messages":[{"role":"user","content":"the quick brown"}]}`)
	var seen []string
	up := newUpstreamServer(t, http.StatusOK, `{"type":"message"}`, &seen)
	defer up.Close()

	shared := newRawSharedStore(t, "p-ctx-fault")
	cs := ctxstore.New(shared)
	g := traffic.NewGate(cs)
	_ = cs.PutTrafficMemory(&ctxstore.TrafficMemory{
		MemoryID: "mem-fault", SessionID: "sess-fault", Category: "preference",
		MemoryText: "prefer compact diffs", Confidence: 0.9, CreatedAt: 1, Status: "applied",
	})
	// Close the underlying store so the gate's read now faults (SQLITE_BUSY on a
	// closed handle): a genuine store fault, not merely an empty result.
	shared.Close()

	cfg := testConfig("passthrough", 0, 0)
	p := newTestProxy(t, cfg, cs, ModePassthrough, up)
	p.SetGate(g)

	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(string(clientBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(SessionHeader, "sess-fault")
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (store fault must not 500)", w.Code)
	}
	if len(seen) != 1 || seen[0] != string(clientBody) {
		t.Fatalf("forwarded body = %q, want the verbatim client body (store fault = no context)", seen)
	}
}

// TestActiveMemoryContextNilGate proves the helper's fail-open contract
// directly: a nil gate yields an empty context block and no error, and
// prependMemoryContext leaves the body unchanged for an empty block.
func TestActiveMemoryContextNilGate(t *testing.T) {
	ctx, err := activeMemoryContext(nil, "sess")
	if err != nil {
		t.Fatalf("activeMemoryContext(nil) = %v, want nil (fail-open)", err)
	}
	if ctx != "" {
		t.Fatalf("activeMemoryContext(nil) = %q, want empty (fail-open)", ctx)
	}

	body := []byte(`{"messages":[{"role":"user","content":"hi"}]}`)
	if got := prependMemoryContext(body, ""); !strings.Contains(string(got), `"content":"hi"`) {
		t.Fatalf("prependMemoryContext(empty) changed the body: %s", got)
	}
}

// TestPrependMemoryContextInsertsLeadingSystem checks the helper in isolation:
// a non-empty context becomes the leading system message ahead of the existing
// conversation, and non-message fields survive the re-encode.
func TestPrependMemoryContextInsertsLeadingSystem(t *testing.T) {
	body := []byte(`{"model":"test-model","messages":[{"role":"user","content":"hi"}]}`)
	got := prependMemoryContext(body, "[traffic context]\n- remember X")

	var doc struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(got, &doc); err != nil {
		t.Fatalf("re-encoded body is not JSON: %v", err)
	}
	if doc.Model != "test-model" {
		t.Errorf("non-message field lost: model = %q, want test-model", doc.Model)
	}
	if len(doc.Messages) != 2 {
		t.Fatalf("messages = %d, want 2 (injected system + original)", len(doc.Messages))
	}
	if doc.Messages[0].Role != "system" || !strings.Contains(doc.Messages[0].Content, "remember X") {
		t.Errorf("leading message = %+v, want a system message carrying the context", doc.Messages[0])
	}
	if doc.Messages[1].Role != "user" || doc.Messages[1].Content != "hi" {
		t.Errorf("trailing message = %+v, want the original user message preserved", doc.Messages[1])
	}
}

// newRawSharedStore opens a real shared project store (migrations applied) and
// returns it WITHOUT a t.Cleanup close: the fault test closes the handle itself
// to simulate a store fault, so a cleanup close would double-close.
func newRawSharedStore(t *testing.T, project string) *store.Store {
	t.Helper()
	s, err := store.Open(t.TempDir(), project)
	if err != nil {
		t.Fatalf("open shared store: %v", err)
	}
	return s
}
