package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/config"
	"github.com/devopstales/skillgrid/mnemonic/internal/llm"
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

// TestOpenProjectAttachesSharedLLM is the end-to-end wiring gap the Spec
// reviewer flagged: the LLM-on branch of openProject (service.go) has no test.
// It writes an indexing.yaml that enables mnemonic.llm against a live
// OpenAI-compatible server, opens the project (which runs AttachSharedLLM +
// the memory-level attach), and proves the shared client actually reaches the
// wire on a real Complete — not just that the seams are non-nil. config.Load
// walks up from configRoot, so the file in a fresh t.TempDir() is the only
// source and the test is isolated from any home/repo config.
func TestOpenProjectAttachesSharedLLM(t *testing.T) {
	restoreLLMSeams(t)
	SetAskLLM(nil)
	SetDedupLLMFunc(nil)

	url, _, close := llmTestServer(t, `{"verdict":"add"}`)
	defer close()

	root := t.TempDir()
	cfgDir := filepath.Join(root, ".skillgrid", "config.d")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir config.d: %v", err)
	}
	yamlDoc := "mnemonic:\n  llm:\n    enabled: true\n    base_url: " + url + "\n    model: qwen3:8b\n    api_key: sk-e2e\n"
	if err := os.WriteFile(filepath.Join(cfgDir, "indexing.yaml"), []byte(yamlDoc), 0o600); err != nil {
		t.Fatalf("write indexing.yaml: %v", err)
	}

	svc := New(t.TempDir())
	h, cleanup, err := svc.openProject("llm-e2e", root)
	if err != nil {
		t.Fatalf("openProject: %v", err)
	}
	defer cleanup()

	// The process-level AskLLM seam must now be a shared client (identity is
	// proven by a live call reaching the test server).
	seam := AskLLMSeam()
	if seam == nil {
		t.Fatal("openProject: AskLLMSeam() = nil, want the shared client")
	}
	got, err := seam.Complete(context.Background(), "sys", "hello")
	if err != nil {
		t.Fatalf("shared client Complete: %v", err)
	}
	if got != `{"verdict":"add"}` {
		t.Fatalf("shared client Complete = %q, want the server's verdict JSON", got)
	}
	// The dedup func must also be attached (the memory-level seam is wired in
	// the same openProject branch; a live Classify reaching the server proves
	// the process-level func is non-nil and functional).
	if _, err := newDedupLLMBackend().Classify(context.Background(), "obs", []string{"cand"}); err != nil {
		t.Fatalf("dedup backend Classify (should reach the wired LLM): %v", err)
	}
	_ = h
}

// TestClassifyRaceFreeDedupFunc is the regression guard for the dedup seam
// data race: Classify reads the package-level dedupLLMFunc (via
// getDedupLLMFunc, under RLock) while openProject re-runs SetDedupLLMFunc
// (a write under Lock) on every retrieval/compaction/ask. Without the RLock
// guard, `go test -race` flags the unsynchronized read. This test hammers the
// writer and reader concurrently so the race detector (or a torn nil read)
// fires if the guard regresses.
func TestClassifyRaceFreeDedupFunc(t *testing.T) {
	restoreLLMSeams(t)
	fn := func(ctx context.Context, system, user string) (string, error) {
		return `{"verdict":"add"}`, nil
	}
	SetDedupLLMFunc(fn)
	b := newDedupLLMBackend()

	const iters = 50
	var wg sync.WaitGroup
	wg.Add(2)
	// Writer: repeatedly (re)attach the func, exactly as openProject does.
	go func() {
		defer wg.Done()
		for i := 0; i < iters; i++ {
			SetDedupLLMFunc(fn)
		}
	}()
	// Reader: repeatedly Classify, which reads the func under RLock.
	go func() {
		defer wg.Done()
		for i := 0; i < iters; i++ {
			if _, err := b.Classify(context.Background(), "obs", []string{"cand"}); err != nil {
				t.Errorf("Classify returned error: %v", err)
				return
			}
		}
	}()
	wg.Wait()
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
