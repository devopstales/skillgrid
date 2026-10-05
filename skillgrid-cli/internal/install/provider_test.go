package install

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// --- test helpers for the provider step (ADR-0023, TICKET-03) ---

// writeHomeIndexing seeds the home indexing.yaml the provider step merges
// into. The operator keys (embedder.dimension, retrieval_budget.items) are
// the "unrelated" state a merge must preserve.
func writeHomeIndexing(t *testing.T, home, content string) string {
	t.Helper()
	dir := filepath.Join(home, ".skillgrid", "config.d")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "indexing.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// loadHomeIndexing parses the home indexing.yaml into a nested map.
func loadHomeIndexing(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var m map[string]any
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return m
}

func homeLLMM(t *testing.T, m map[string]any) map[string]any {
	t.Helper()
	mn, ok := m["mnemonic"].(map[string]any)
	if !ok {
		t.Fatalf("no mnemonic section: %v", m)
	}
	llm, ok := mn["llm"].(map[string]any)
	if !ok {
		t.Fatalf("no mnemonic.llm section: %v", mn)
	}
	return llm
}

func homeEmbedder(t *testing.T, m map[string]any) map[string]any {
	t.Helper()
	mn, ok := m["mnemonic"].(map[string]any)
	if !ok {
		t.Fatalf("no mnemonic section: %v", m)
	}
	emb, ok := mn["embedder"].(map[string]any)
	if !ok {
		t.Fatalf("no mnemonic.embedder section: %v", mn)
	}
	return emb
}

// tagsServer serves an Ollama /api/tags endpoint listing the given model
// names.
func tagsServer(t *testing.T, models ...string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The production probe hits ollamaBaseURL+"/api/tags"; when the base
		// itself is the httptest root the path is "/", so accept both.
		if r.URL.Path != "/api/tags" && r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		type modelJSON struct {
			Name string `json:"name"`
		}
		out := struct {
			Models []modelJSON `json:"models"`
		}{}
		for _, m := range models {
			out.Models = append(out.Models, modelJSON{Name: m})
		}
		_ = json.NewEncoder(w).Encode(out)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// --- G10: happy path install yes ensures local ollama and wires config ---
//
// SATISFIES: happy path install yes ensures local ollama and wires config
func TestSetupProviderLocal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	idxPath := writeHomeIndexing(t, home, `profile: default
mnemonic:
  embedder:
    provider: onnx
    dimension: 768
  retrieval_budget:
    items: 5
`)

	// Fake Ollama on PATH: `ollama serve` (which the step starts when the
	// probe is already served) must not fail; every other invocation is
	// recorded by the runCmd seam instead of exec.
	binDir := t.TempDir()
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := os.WriteFile(filepath.Join(binDir, "ollama"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	var invocations []string

	// Ollama base URL seam → fake /api/tags (neither model present).
	prev := ollamaBaseURL
	ollamaBaseURL = tagsServer(t).URL
	t.Cleanup(func() { ollamaBaseURL = prev })

	// runCmd seam → record instead of exec.
	prevRun := runCmd
	runCmd = func(name string, args ...string) error {
		invocations = append(invocations, name+" "+join1(args))
		return nil
	}
	t.Cleanup(func() { runCmd = prevRun })

	cfg := Config{
		Provider: "local",
		HomeDir:  home,
		RepoHome: filepath.Join(home, ".skillgrid"),
	}
	if err := setupProvider(&cfg); err != nil {
		t.Fatalf("setupProvider (local): %v", err)
	}

	// (a) both models pulled because /api/tags listed neither.
	pulled := map[string]bool{}
	for _, inv := range invocations {
		if len(inv) >= 11 && strings.HasPrefix(inv, "ollama pull ") {
			pulled[inv[len("ollama pull "):]] = true
		}
	}
	if !pulled["llama3.2:3b"] {
		t.Errorf("ollama pull llama3.2:3b not invoked; got %v", invocations)
	}
	if !pulled["nomic-embed-text"] {
		t.Errorf("ollama pull nomic-embed-text not invoked; got %v", invocations)
	}

	// (b) home config merged with the correct llm + ollama embedder…
	m := loadHomeIndexing(t, idxPath)
	llm := homeLLMM(t, m)
	if llm["enabled"] != true {
		t.Errorf("mnemonic.llm.enabled = %v, want true", llm["enabled"])
	}
	if llm["base_url"] != ollamaBaseURL+"/v1" {
		t.Errorf("mnemonic.llm.base_url = %v, want %q", llm["base_url"], ollamaBaseURL+"/v1")
	}
	if llm["model"] != "llama3.2:3b" {
		t.Errorf("mnemonic.llm.model = %v, want llama3.2:3b", llm["model"])
	}
	emb := homeEmbedder(t, m)
	if emb["provider"] != "ollama" {
		t.Errorf("mnemonic.embedder.provider = %v, want ollama", emb["provider"])
	}
	if emb["model"] != "nomic-embed-text" {
		t.Errorf("mnemonic.embedder.model = %v, want nomic-embed-text", emb["model"])
	}
	if emb["base_url"] != ollamaBaseURL+"/v1" {
		t.Errorf("mnemonic.embedder.base_url = %v, want %q", emb["base_url"], ollamaBaseURL+"/v1")
	}

	// …and the pre-existing unrelated keys are preserved.
	if emb["dimension"] != 768 {
		t.Errorf("mnemonic.embedder.dimension = %v, want 768 preserved", emb["dimension"])
	}
	mn := m["mnemonic"].(map[string]any)
	rb, ok := mn["retrieval_budget"].(map[string]any)
	if !ok || rb["items"] != 5 {
		t.Errorf("mnemonic.retrieval_budget.items = %v, want 5 preserved", rb)
	}
	if m["profile"] != "default" {
		t.Errorf("profile = %v, want default preserved", m["profile"])
	}
}

// --- G11: skip-provider leaves ensure out ---
//
// SATISFIES: skip-provider leaves ensure out
func TestSetupProviderSkipped(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	idxPath := writeHomeIndexing(t, home, `profile: default
mnemonic:
  embedder:
    provider: onnx
`)

	// Any command run or ollama touch is a failure: counters stay zero and the
	// ollama base is left at a dead URL (probe would fail if it ran).
	prevRun := runCmd
	runCmd = func(name string, args ...string) error {
		t.Errorf("runCmd(%q %v) ran despite --skip-provider", name, args)
		return nil
	}
	t.Cleanup(func() { runCmd = prevRun })
	prev := ollamaBaseURL
	ollamaBaseURL = "http://127.0.0.1:0" // dead; probe errors if reached
	t.Cleanup(func() { ollamaBaseURL = prev })

	cfg := Config{
		Provider:     "local",
		SkipProvider: true,
		HomeDir:      home,
		RepoHome:     filepath.Join(home, ".skillgrid"),
	}
	if err := setupProvider(&cfg); err != nil {
		t.Fatalf("setupProvider (skip): %v", err)
	}

	// llm is absent and embedder is unchanged (still onnx).
	m := loadHomeIndexing(t, idxPath)
	mn, _ := m["mnemonic"].(map[string]any)
	if _, present := mn["llm"]; present {
		t.Errorf("mnemonic.llm was written despite --skip-provider: %v", mn["llm"])
	}
	emb := homeEmbedder(t, m)
	if emb["provider"] != "onnx" {
		t.Errorf("mnemonic.embedder.provider = %v, want onnx (unchanged)", emb["provider"])
	}
}

// --- bonus: external wires without ollama ---
func TestSetupProviderExternalNoOllama(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	idxPath := writeHomeIndexing(t, home, `profile: default
mnemonic:
  embedder:
    provider: onnx
`)

	prevRun := runCmd
	runCmd = func(name string, args ...string) error {
		t.Errorf("ollama command %q %v ran despite external provider", name, args)
		return nil
	}
	t.Cleanup(func() { runCmd = prevRun })

	cfg := Config{
		Provider:   "external",
		HomeDir:    home,
		RepoHome:   filepath.Join(home, ".skillgrid"),
		LLMBaseURL: "https://api.example.com/v1",
		LLMApiKey:  "sk-test",
	}
	if err := setupProvider(&cfg); err != nil {
		t.Fatalf("setupProvider (external): %v", err)
	}

	m := loadHomeIndexing(t, idxPath)
	llm := homeLLMM(t, m)
	if llm["enabled"] != true {
		t.Errorf("mnemonic.llm.enabled = %v, want true", llm["enabled"])
	}
	if llm["base_url"] != "https://api.example.com/v1" {
		t.Errorf("mnemonic.llm.base_url = %v, want the external base", llm["base_url"])
	}
	if _, present := llm["api_key"]; present {
		t.Errorf("mnemonic.llm.api_key must NOT be written to home yaml: %v", llm)
	}
	emb := homeEmbedder(t, m)
	if emb["provider"] != "external" {
		t.Errorf("mnemonic.embedder.provider = %v, want external", emb["provider"])
	}
	if emb["base_url"] != "https://api.example.com/v1" {
		t.Errorf("mnemonic.embedder.base_url = %v, want the external base", emb["base_url"])
	}
}

// --- bonus: ensure skips pull when models already present ---
func TestSetupProviderEnsureSkipsPullWhenPresent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	idxPath := writeHomeIndexing(t, home, `profile: default
mnemonic:
  embedder:
    provider: onnx
`)

	prevRun := runCmd
	var invocations []string
	runCmd = func(name string, args ...string) error {
		invocations = append(invocations, name+" "+join1(args))
		return nil
	}
	t.Cleanup(func() { runCmd = prevRun })

	prev := ollamaBaseURL
	ollamaBaseURL = tagsServer(t, "llama3.2:3b", "nomic-embed-text").URL
	t.Cleanup(func() { ollamaBaseURL = prev })

	cfg := Config{
		Provider: "local",
		HomeDir:  home,
		RepoHome: filepath.Join(home, ".skillgrid"),
	}
	if err := setupProvider(&cfg); err != nil {
		t.Fatalf("setupProvider (ensure-present): %v", err)
	}

	for _, inv := range invocations {
		if strings.HasPrefix(inv, "ollama pull ") {
			t.Errorf("ollama pull %q ran though both models were present", inv)
		}
	}
	// Config still merged.
	m := loadHomeIndexing(t, idxPath)
	if llm := homeLLMM(t, m); llm["enabled"] != true || llm["model"] != "llama3.2:3b" {
		t.Errorf("config not merged when models present: llm=%v", llm)
	}
	if emb := homeEmbedder(t, m); emb["provider"] != "ollama" {
		t.Errorf("embedder.provider = %v, want ollama", emb["provider"])
	}
}

// join1 joins args with spaces for invocation logging.
func join1(args []string) string {
	out := ""
	for i, a := range args {
		if i > 0 {
			out += " "
		}
		out += a
	}
	return out
}
