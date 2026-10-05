package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/config"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/llm"
)

// restoreLLMSeams snapshots the two process-level seams (the AskLLM package
// var via AskLLMSeam and the dedupLLMFunc via a SetDedupLLMFunc(nil) restore)
// and restores them on cleanup so the suite stays hermetic across tests. The
// dedup func has no getter, so it is reset to nil (its zero value).
func restoreLLMSeams(t *testing.T) {
	t.Helper()
	prevAsk := AskLLMSeam()
	t.Cleanup(func() {
		SetAskLLM(prevAsk)
		SetDedupLLMFunc(nil)
	})
}

// llmTestServer stands up an OpenAI-compatible chat/completions server that
// records each request (system + user + model + auth) and replies with a
// fixed content string. It returns the server URL, a pointer to the last
// recorded request, and a Close function.
func llmTestServer(t *testing.T, content string) (string, *recordedLLMRequest, func()) {
	t.Helper()
	rec := &recordedLLMRequest{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %q, want /chat/completions", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		rec.Model = req.Model
		if len(req.Messages) >= 1 {
			rec.System = req.Messages[0].Content
		}
		if len(req.Messages) >= 2 {
			rec.User = req.Messages[1].Content
		}
		rec.Auth = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"content": content}}},
		})
	}))
	return srv.URL, rec, srv.Close
}

type recordedLLMRequest struct {
	Model  string
	System string
	User   string
	Auth   string
}

// TestAttachSharedLLMDisabledNoOp is gate G5: when mnemonic.llm is disabled
// (Enabled=false), AttachSharedLLM is a no-op — it returns (nil, nil) and
// touches no seam. The AskLLM seam must remain nil (the cited-ask floor).
func TestAttachSharedLLMDisabledNoOp(t *testing.T) {
	restoreLLMSeams(t)
	// Ensure a clean slate: nothing attached from a prior test.
	SetAskLLM(nil)
	SetDedupLLMFunc(nil)

	client, err := AttachSharedLLM(config.LLMConfig{Enabled: false})
	if err != nil {
		t.Fatalf("AttachSharedLLM disabled: unexpected error %v", err)
	}
	if client != nil {
		t.Fatalf("AttachSharedLLM disabled: client = %v, want nil (no-op)", client)
	}
	if got := AskLLMSeam(); got != nil {
		t.Fatalf("AttachSharedLLM disabled: AskLLMSeam() = %v, want nil (floor intact)", got)
	}
}

// TestAttachSharedLLMMissingConfig is gate G6: Enabled=true but BaseURL or
// Model empty is a config mistake — AttachSharedLLM returns (nil, non-nil)
// and attaches nothing. The AskLLM seam stays nil so the floors hold.
func TestAttachSharedLLMMissingConfig(t *testing.T) {
	restoreLLMSeams(t)
	SetAskLLM(nil)
	SetDedupLLMFunc(nil)

	// Missing BaseURL (Model set).
	client, err := AttachSharedLLM(config.LLMConfig{Enabled: true, Model: "m", APIKey: "k"})
	if err == nil {
		t.Fatalf("AttachSharedLLM missing base_url: want error, got nil (client %v)", client)
	}
	if client != nil {
		t.Fatalf("AttachSharedLLM missing base_url: client = %v, want nil (nothing attached)", client)
	}
	if got := AskLLMSeam(); got != nil {
		t.Fatalf("AttachSharedLLM missing base_url: AskLLMSeam() = %v, want nil", got)
	}

	// Missing Model (BaseURL set) — same gate.
	_, err = AttachSharedLLM(config.LLMConfig{Enabled: true, BaseURL: "http://x", APIKey: "k"})
	if err == nil {
		t.Fatal("AttachSharedLLM missing model: want error, got nil")
	}
}

// TestAttachSharedLLMWiresSeams is gate G7: Enabled=true with a valid
// BaseURL + Model wires the two process-level seams (AskLLM + dedup func) to
// the same shared *llm.Client, and returns that client for the memory-level
// attach. The AskLLM seam must be the returned client (identity check).
func TestAttachSharedLLMWiresSeams(t *testing.T) {
	restoreLLMSeams(t)
	SetAskLLM(nil)
	SetDedupLLMFunc(nil)

	url, _, close := llmTestServer(t, "ok")
	defer close()

	client, err := AttachSharedLLM(config.LLMConfig{Enabled: true, BaseURL: url, Model: "m", APIKey: "k"})
	if err != nil {
		t.Fatalf("AttachSharedLLM enabled+valid: unexpected error %v", err)
	}
	if client == nil {
		t.Fatal("AttachSharedLLM enabled+valid: client = nil, want the shared *llm.Client")
	}
	got := AskLLMSeam()
	if got == nil {
		t.Fatal("AttachSharedLLM enabled+valid: AskLLMSeam() = nil, want the client")
	}
	// Identity: the seam must be the very client we built, not an adapter.
	if got != client {
		t.Fatalf("AttachSharedLLM enabled+valid: AskLLMSeam() is not the returned client (got %T)", got)
	}
}

// TestExtractionAdapterCompletes is gate G8: the extraction adapter routes
// Extract(ctx, text) to the shared Completer with the extraction system
// prompt, and returns the server's message content. The request body must
// carry the extraction system prompt (proves the adapter sends it, not an
// arbitrary string).
func TestExtractionAdapterCompletes(t *testing.T) {
	restoreLLMSeams(t)
	url, rec, close := llmTestServer(t, `{"learnings":[{"text":"x","type":"decision"}]}`)
	defer close()

	client := llm.New(llm.Config{BaseURL: url, Model: "m", Timeout: 2 * time.Second})
	adapter := newExtractionLLMAdapter(client)
	got, err := adapter.Extract(context.Background(), "some session text")
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}
	want := `{"learnings":[{"text":"x","type":"decision"}]}`
	if got != want {
		t.Fatalf("Extract = %q, want %q", got, want)
	}
	if !strings.Contains(rec.System, "extract") || !strings.Contains(rec.System, "JSON") {
		t.Fatalf("request system prompt = %q, want it to name extraction + strict JSON", rec.System)
	}
	if !strings.Contains(rec.User, "some session text") {
		t.Fatalf("request user = %q, want it to carry the session text", rec.User)
	}
}

// TestDreamAdapterCompletes: the dream adapter implements both DreamLLM
// methods (consolidate + synthesize) against the shared Completer with fixed
// system prompts. Each returns the server content, and the user prompt must
// carry a numbered list of the observations / summaries (the brief's contract
// for the future lazy distill wiring).
func TestDreamAdapterCompletes(t *testing.T) {
	restoreLLMSeams(t)
	url, rec, close := llmTestServer(t, "consolidated record")
	defer close()

	client := llm.New(llm.Config{BaseURL: url, Model: "m", Timeout: 2 * time.Second})
	adapter := NewDreamLLMAdapter(client)
	ctx := context.Background()

	got, err := adapter.ConsolidatePrompt(ctx, "auth", []string{"fact one", "fact two"})
	if err != nil {
		t.Fatalf("ConsolidatePrompt returned error: %v", err)
	}
	if got != "consolidated record" {
		t.Fatalf("ConsolidatePrompt = %q, want the server content", got)
	}
	if !strings.Contains(rec.User, "auth") || !strings.Contains(rec.User, "fact one") || !strings.Contains(rec.User, "fact two") {
		t.Fatalf("consolidate user prompt = %q, want the topic + both observations", rec.User)
	}
	// Numbered observations list: the two facts must appear as a numbered
	// (1-indexed or 0-indexed) block, not a bare join.
	if !strings.Contains(rec.User, "1.") && !strings.Contains(rec.User, "[1]") {
		t.Fatalf("consolidate user prompt = %q, want a numbered observations list", rec.User)
	}

	got, err = adapter.SynthesizePrompt(ctx, []string{"sum one", "sum two"})
	if err != nil {
		t.Fatalf("SynthesizePrompt returned error: %v", err)
	}
	if got != "consolidated record" {
		t.Fatalf("SynthesizePrompt = %q, want the server content", got)
	}
	if !strings.Contains(rec.User, "sum one") || !strings.Contains(rec.User, "sum two") {
		t.Fatalf("synthesize user prompt = %q, want both summaries", rec.User)
	}
	if !strings.Contains(rec.User, "1.") && !strings.Contains(rec.User, "[1]") {
		t.Fatalf("synthesize user prompt = %q, want a numbered summaries list", rec.User)
	}
}
