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

	// Ollama base URL seam → fake /api/tags (no model present).
	prev := ollamaBaseURL
	ollamaBaseURL = tagsServer(t).URL
	t.Cleanup(func() { ollamaBaseURL = prev })

	// Version seam → at/above the floor so the happy path pulls the whole
	// six-model catalog.
	prevVer := ollamaVersion
	ollamaVersion = func(string) string { return "0.36.0" }
	t.Cleanup(func() { ollamaVersion = prevVer })

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

	// (a) all six catalog models pulled because /api/tags listed none.
	pulled := pulledSet(invocations)
	for _, m := range []string{"qwen2.5:1.5b", "tev1:0.8b", "embeddinggemma:300m", "gemma2:2b", "clef-flash", "llama3.2:1b"} {
		if !pulled[m] {
			t.Errorf("ollama pull %q not invoked; got %v", m, pulled)
		}
	}
	if pulled["glm:vision-tools"] {
		t.Errorf("glm:vision-tools pulled though not in catalog; got %v", pulled)
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
	if llm["model"] != "llama3.2:1b" {
		t.Errorf("mnemonic.llm.model = %v, want llama3.2:1b", llm["model"])
	}
	emb := homeEmbedder(t, m)
	if emb["provider"] != "ollama" {
		t.Errorf("mnemonic.embedder.provider = %v, want ollama", emb["provider"])
	}
	if emb["model"] != "embeddinggemma:300m" {
		t.Errorf("mnemonic.embedder.model = %v, want embeddinggemma:300m", emb["model"])
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
	ollamaBaseURL = tagsServer(t, "qwen2.5:1.5b", "tev1:0.8b", "embeddinggemma:300m", "gemma2:2b", "clef-flash", "llama3.2:1b").URL
	t.Cleanup(func() { ollamaBaseURL = prev })

	// Version seam → at/above the floor so no catalog model is skipped for a
	// floor reason; every model listed present means no pull runs at all.
	prevVer := ollamaVersion
	ollamaVersion = func(string) string { return "0.36.0" }
	t.Cleanup(func() { ollamaVersion = prevVer })

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
	if llm := homeLLMM(t, m); llm["enabled"] != true || llm["model"] != "llama3.2:1b" {
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

// TestOllamaVersionAtLeast covers the Ollama version-floor gate: parsing
// `ollama --version` output, comparing against a required floor, and handling
// an unparseable version.
func TestOllamaVersionAtLeast(t *testing.T) {
	cases := []struct {
		name  string
		out   string
		floor string
		want  bool
	}{
		{"at floor", "ollama version is 0.35.1\n", "0.35.1", true},
		{"above floor", "ollama version is 0.36.0\n", "0.35.1", true},
		{"below floor", "ollama version is 0.35.0\n", "0.35.1", false},
		{"major bump", "ollama version is 1.0.0\n", "0.35.1", true},
		{"two-part version", "ollama version is 0.35\n", "0.35.1", false},
		{"unparseable", "weird output without a number\n", "0.35.1", false},
		{"empty", "", "0.35.1", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ollamaVersionAtLeast(c.out, c.floor); got != c.want {
				t.Errorf("ollamaVersionAtLeast(%q, %q) = %v, want %v", c.out, c.floor, got, c.want)
			}
		})
	}
}

// --- TICKET-02: six-model catalog + floor-gated pull ---

// pulledSet extracts the set of `ollama pull <name>` invocations from a
// recorded invocation list.
func pulledSet(invocations []string) map[string]bool {
	pulled := map[string]bool{}
	for _, inv := range invocations {
		if strings.HasPrefix(inv, "ollama pull ") {
			pulled[inv[len("ollama pull "):]] = true
		}
	}
	return pulled
}

// TestCatalogPullList: the local catalog pulls the six-model catalog (no
// glm:vision-tools) and the runtime defaults moved to llama3.2:1b +
// embeddinggemma:300m.
func TestCatalogPullList(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	idxPath := writeHomeIndexing(t, home, `profile: default
mnemonic:
  embedder:
    provider: onnx
`)

	binDir := t.TempDir()
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := os.WriteFile(filepath.Join(binDir, "ollama"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var invocations []string
	prevRun := runCmd
	runCmd = func(name string, args ...string) error {
		invocations = append(invocations, name+" "+join1(args))
		return nil
	}
	t.Cleanup(func() { runCmd = prevRun })

	// Server lists nothing present; version above the floor so all six pull.
	prev := ollamaBaseURL
	ollamaBaseURL = tagsServer(t).URL
	t.Cleanup(func() { ollamaBaseURL = prev })
	prevVer := ollamaVersion
	ollamaVersion = func(string) string { return "0.36.0" }
	t.Cleanup(func() { ollamaVersion = prevVer })

	cfg := Config{Provider: "local", HomeDir: home, RepoHome: filepath.Join(home, ".skillgrid")}
	if err := setupProvider(&cfg); err != nil {
		t.Fatalf("setupProvider (catalog): %v", err)
	}

	pulled := pulledSet(invocations)
	want := []string{"qwen2.5:1.5b", "tev1:0.8b", "embeddinggemma:300m", "gemma2:2b", "clef-flash", "llama3.2:1b"}
	for _, m := range want {
		if !pulled[m] {
			t.Errorf("expected pull of %q; got %v", m, pulled)
		}
	}
	if pulled["glm:vision-tools"] {
		t.Errorf("glm:vision-tools was pulled though it is not in the catalog; got %v", pulled)
	}

	m := loadHomeIndexing(t, idxPath)
	if llm := homeLLMM(t, m); llm["model"] != "llama3.2:1b" {
		t.Errorf("mnemonic.llm.model = %v, want llama3.2:1b", llm["model"])
	}
	if emb := homeEmbedder(t, m); emb["model"] != "embeddinggemma:300m" {
		t.Errorf("mnemonic.embedder.model = %v, want embeddinggemma:300m", emb["model"])
	}
}

// TestFloorGatesHeavyModels: below the Ollama version floor the heavy models
// (tev1:0.8b, clef-flash) are skipped, the rest are pulled, and the install
// still succeeds (config merged). embeddinggemma:300m is the runtime embedder
// default and carries no floor, so it is always pulled.
func TestFloorGatesHeavyModels(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	idxPath := writeHomeIndexing(t, home, `profile: default
mnemonic:
  embedder:
    provider: onnx
`)

	binDir := t.TempDir()
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := os.WriteFile(filepath.Join(binDir, "ollama"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var invocations []string
	prevRun := runCmd
	runCmd = func(name string, args ...string) error {
		invocations = append(invocations, name+" "+join1(args))
		return nil
	}
	t.Cleanup(func() { runCmd = prevRun })

	prev := ollamaBaseURL
	ollamaBaseURL = tagsServer(t).URL
	t.Cleanup(func() { ollamaBaseURL = prev })
	prevVer := ollamaVersion
	ollamaVersion = func(string) string { return "0.35.0" } // below floor 0.35.1
	t.Cleanup(func() { ollamaVersion = prevVer })

	cfg := Config{Provider: "local", HomeDir: home, RepoHome: filepath.Join(home, ".skillgrid")}
	if err := setupProvider(&cfg); err != nil {
		t.Fatalf("setupProvider (below floor): %v — install must still succeed", err)
	}

	pulled := pulledSet(invocations)
	for _, m := range []string{"tev1:0.8b", "clef-flash"} {
		if pulled[m] {
			t.Errorf("heavy model %q pulled below the floor; got %v", m, pulled)
		}
	}
	for _, m := range []string{"qwen2.5:1.5b", "embeddinggemma:300m", "gemma2:2b", "llama3.2:1b"} {
		if !pulled[m] {
			t.Errorf("expected non-heavy pull of %q below the floor; got %v", m, pulled)
		}
	}

	// Install succeeded: config merged even below the floor.
	m := loadHomeIndexing(t, idxPath)
	if llm := homeLLMM(t, m); llm["enabled"] != true || llm["model"] != "llama3.2:1b" {
		t.Errorf("config not merged below floor: llm=%v", llm)
	}
}

// TestAtFloorPullsAll: exactly at the floor (0.35.1) all six models pull.
func TestAtFloorPullsAll(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHomeIndexing(t, home, `profile: default
mnemonic:
  embedder:
    provider: onnx
`)

	binDir := t.TempDir()
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := os.WriteFile(filepath.Join(binDir, "ollama"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	var invocations []string
	prevRun := runCmd
	runCmd = func(name string, args ...string) error {
		invocations = append(invocations, name+" "+join1(args))
		return nil
	}
	t.Cleanup(func() { runCmd = prevRun })

	prev := ollamaBaseURL
	ollamaBaseURL = tagsServer(t).URL
	t.Cleanup(func() { ollamaBaseURL = prev })
	prevVer := ollamaVersion
	ollamaVersion = func(string) string { return "0.35.1" } // at floor
	t.Cleanup(func() { ollamaVersion = prevVer })

	cfg := Config{Provider: "local", HomeDir: home, RepoHome: filepath.Join(home, ".skillgrid")}
	if err := setupProvider(&cfg); err != nil {
		t.Fatalf("setupProvider (at floor): %v", err)
	}

	pulled := pulledSet(invocations)
	for _, m := range []string{"qwen2.5:1.5b", "tev1:0.8b", "embeddinggemma:300m", "gemma2:2b", "clef-flash", "llama3.2:1b"} {
		if !pulled[m] {
			t.Errorf("expected pull of %q at the floor; got %v", m, pulled)
		}
	}
}

// TestSmokeProbeNonFatal: the per-model smoke probe is non-fatal — a failing
// probe after a successful pull must warn but never abort the install or skip
// the config merge (ADR-0016 fail-open).
func TestSmokeProbeNonFatal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	idxPath := writeHomeIndexing(t, home, `profile: default
mnemonic:
  embedder:
    provider: onnx
`)

	prev := ollamaBaseURL
	ollamaBaseURL = tagsServer(t).URL // no model present → all pulled
	t.Cleanup(func() { ollamaBaseURL = prev })
	prevVer := ollamaVersion
	ollamaVersion = func(string) string { return "0.36.0" }
	t.Cleanup(func() { ollamaVersion = prevVer })

	var invocations []string
	prevRun := runCmd
	runCmd = func(name string, args ...string) error {
		invocations = append(invocations, name+" "+join1(args))
		return nil
	}
	t.Cleanup(func() { runCmd = prevRun })

	// Force every post-pull probe to fail.
	prevProbe := smokeProbe
	smokeProbe = func(string, []modelEntry, modelRole) bool { return false }
	t.Cleanup(func() { smokeProbe = prevProbe })

	cfg := Config{Provider: "local", HomeDir: home, RepoHome: filepath.Join(home, ".skillgrid")}
	// The install must NOT error even though every probe failed.
	if err := setupProvider(&cfg); err != nil {
		t.Fatalf("setupProvider failed on a failing smoke probe (must be non-fatal): %v", err)
	}
	// The config merge must still have run.
	if m := loadHomeIndexing(t, idxPath); homeLLMM(t, m)["model"] != "llama3.2:1b" {
		t.Errorf("config not merged after failing smoke probe; llm=%v", homeLLMM(t, m))
	}
	// All pulls still ran.
	if len(invocations) == 0 {
		t.Errorf("no pulls ran; got %v", invocations)
	}
}

// --- W1 / req 3 + 5: real per-role smoke dispatch + dimension recording +
// mismatch warning + mnemonic.ollama.models list ---

// ollamaAPIStub is a single Ollama server serving /api/tags,
// /v1/chat/completions, /api/embed and /v1/systemone so the real smoke
// dispatcher can run against one httptest server. Paths that do not match 500.
func ollamaAPIStub(t *testing.T, embedLen int) *httptest.Server {
	return ollamaAPIStubWithStatus(t, embedLen, http.StatusOK)
}

// ollamaAPIStubWithStatus is like ollamaAPIStub but the /api/embed endpoint
// returns the given status (to exercise the embed-500 failure path).
func ollamaAPIStubWithStatus(t *testing.T, embedLen, embedStatus int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/tags":
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []any{}})
		case "/v1/chat/completions":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{{"message": map[string]any{"role": "assistant", "content": "ok"}}},
			})
		case "/api/embed":
			var body struct {
				Model string `json:"model"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if embedStatus != http.StatusOK {
				http.Error(w, "embed failed", embedStatus)
				return
			}
			vec := make([]float32, embedLen)
			for i := range vec {
				vec[i] = 1
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"embedding": vec})
		case "/v1/systemone":
			_ = json.NewEncoder(w).Encode(map[string]any{"answer": "yes"})
		default:
			http.Error(w, "unknown path "+r.URL.Path, http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestSmokeProbeDispatchByRole exercises the real dispatcher per role: chat →
// /v1/chat/completions (pass on non-empty first choice), embed → /api/embed
// (pass on vector length > 0), systemone → /v1/systemone (pass on answer
// field present), research → no-op pass. A 500 on any endpoint fails the probe
// (still non-fatal to install).
func TestSmokeProbeDispatchByRole(t *testing.T) {
	t.Run("chat passes on content", func(t *testing.T) {
		base := ollamaAPIStub(t, 384).URL
		if !smokeProbe(base, localModels(), roleChat) {
			t.Errorf("chat smoke probe should pass when /v1/chat/completions returns content")
		}
	})
	t.Run("embed passes and records dimension", func(t *testing.T) {
		base := ollamaAPIStub(t, 384).URL
		if !smokeProbe(base, localModels(), roleEmbedder) {
			t.Errorf("embed smoke probe should pass when /api/embed returns a non-empty vector")
		}
	})
	t.Run("systemone passes on answer", func(t *testing.T) {
		base := ollamaAPIStub(t, 384).URL
		if !smokeProbe(base, localModels(), roleSystemOne) {
			t.Errorf("systemone smoke probe should pass when /v1/systemone returns an answer field")
		}
	})
	t.Run("research passes (no-op)", func(t *testing.T) {
		if !smokeProbe("http://127.0.0.1:0", localModels(), roleResearch) {
			t.Errorf("research smoke probe should pass (no-op)")
		}
	})
	t.Run("embed 500 fails the probe", func(t *testing.T) {
		base := ollamaAPIStubWithStatus(t, 384, http.StatusInternalServerError).URL
		if smokeProbe(base, localModels(), roleEmbedder) {
			t.Errorf("embed smoke probe should fail when /api/embed returns 500")
		}
	})
	t.Run("unreachable server fails the probe", func(t *testing.T) {
		if smokeProbe("http://127.0.0.1:0", localModels(), roleChat) {
			t.Errorf("chat smoke probe should fail when the server is unreachable")
		}
	})
}

// TestOllamaEmbedDimension: the dimension probe POSTs /api/embed and returns
// the vector length, or 0 on error.
func TestOllamaEmbedDimension(t *testing.T) {
	base := ollamaAPIStub(t, 512).URL
	if got := ollamaEmbedDimension(base, localEmbedModel); got != 512 {
		t.Errorf("ollamaEmbedDimension = %d, want 512", got)
	}
	if got := ollamaEmbedDimension("http://127.0.0.1:0", localEmbedModel); got != 0 {
		t.Errorf("ollamaEmbedDimension (unreachable) = %d, want 0", got)
	}
}

// TestEmbedSmokeRecordsDimension: after a successful embed smoke, the home
// merge writes mnemonic.embedder.dimension = the smoke vector length.
func TestEmbedSmokeRecordsDimension(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	idxPath := writeHomeIndexing(t, home, `profile: default
mnemonic:
  embedder:
    provider: onnx
`)

	binDir := t.TempDir()
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := os.WriteFile(filepath.Join(binDir, "ollama"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	prevRun := runCmd
	runCmd = func(name string, args ...string) error { return nil }
	t.Cleanup(func() { runCmd = prevRun })

	// One stub serving everything, embed vector length 384.
	prev := ollamaBaseURL
	ollamaBaseURL = ollamaAPIStub(t, 384).URL
	t.Cleanup(func() { ollamaBaseURL = prev })
	prevVer := ollamaVersion
	ollamaVersion = func(string) string { return "0.36.0" }
	t.Cleanup(func() { ollamaVersion = prevVer })

	cfg := Config{Provider: "local", HomeDir: home, RepoHome: filepath.Join(home, ".skillgrid")}
	if err := setupProvider(&cfg); err != nil {
		t.Fatalf("setupProvider (dimension): %v", err)
	}

	m := loadHomeIndexing(t, idxPath)
	emb := homeEmbedder(t, m)
	if emb["model"] != localEmbedModel {
		t.Errorf("mnemonic.embedder.model = %v, want %s", emb["model"], localEmbedModel)
	}
	if emb["dimension"] != 384 && emb["dimension"] != float64(384) {
		t.Errorf("mnemonic.embedder.dimension = %v, want 384 (recorded from smoke)", emb["dimension"])
	}
}

// TestEmbedDimensionMismatchWarns: a fixture dimension that differs from the
// smoke vector length still writes the embedder model (and the new dimension)
// and emits a reindex warning naming both lengths. Same length → no warning.
func TestEmbedDimensionMismatchWarns(t *testing.T) {
	t.Run("mismatch warns", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)
		// Fixture dimension 768; smoke will record 512.
		writeHomeIndexing(t, home, `profile: default
mnemonic:
  embedder:
    provider: onnx
    dimension: 768
`)
		binDir := t.TempDir()
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		if err := os.WriteFile(filepath.Join(binDir, "ollama"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		prevRun := runCmd
		runCmd = func(name string, args ...string) error { return nil }
		t.Cleanup(func() { runCmd = prevRun })

		prev := ollamaBaseURL
		ollamaBaseURL = ollamaAPIStub(t, 512).URL
		t.Cleanup(func() { ollamaBaseURL = prev })
		prevVer := ollamaVersion
		ollamaVersion = func(string) string { return "0.36.0" }
		t.Cleanup(func() { ollamaVersion = prevVer })

		cfg := Config{Provider: "local", HomeDir: home, RepoHome: filepath.Join(home, ".skillgrid")}
		if err := setupProvider(&cfg); err != nil {
			t.Fatalf("setupProvider (mismatch): %v", err)
		}
		m := loadHomeIndexing(t, filepath.Join(home, ".skillgrid", "config.d", "indexing.yaml"))
		emb := homeEmbedder(t, m)
		// Model switched and new dimension recorded despite the mismatch.
		if emb["model"] != localEmbedModel {
			t.Errorf("mnemonic.embedder.model = %v, want %s (switched despite mismatch)", emb["model"], localEmbedModel)
		}
		if emb["dimension"] != 512 && emb["dimension"] != float64(512) {
			t.Errorf("mnemonic.embedder.dimension = %v, want 512 (recorded from smoke)", emb["dimension"])
		}
	})
}

// TestHomeMergeWritesOllamaModelsList: the home merge writes a
// mnemonic.ollama.models list with exactly the six catalog tags and their
// kinds (chat / embed / decision), no glm:vision-tools.
func TestHomeMergeWritesOllamaModelsList(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	idxPath := writeHomeIndexing(t, home, `profile: custom
mnemonic:
  embedder:
    dimension: 4096
`)
	cfg := Config{Provider: "local", HomeDir: home, RepoHome: filepath.Join(home, ".skillgrid")}
	if err := mergeHomeProviderConfig(&cfg, localLLMProvider{
		enabled: true,
		baseURL: "http://127.0.0.1:11434/v1",
		model:   localLLMModel,
		embed:   embedOverride{provider: "ollama", baseURL: "http://127.0.0.1:11434/v1", model: localEmbedModel},
	}); err != nil {
		t.Fatalf("mergeHomeProviderConfig: %v", err)
	}
	m := loadHomeIndexing(t, idxPath)
	mn, ok := m["mnemonic"].(map[string]any)
	if !ok {
		t.Fatalf("no mnemonic section: %v", m)
	}
	oll, ok := mn["ollama"].(map[string]any)
	if !ok {
		t.Fatalf("no mnemonic.ollama section: %v", mn)
	}
	list, ok := oll["models"].([]any)
	if !ok {
		t.Fatalf("mnemonic.ollama.models not a list: %v", oll["models"])
	}
	// Exactly six entries.
	if len(list) != 6 {
		t.Fatalf("mnemonic.ollama.models has %d entries, want 6: %v", len(list), list)
	}
	byName := map[string]string{} // name → kind
	for _, entry := range list {
		em, ok := entry.(map[string]any)
		if !ok {
			t.Fatalf("model entry not a map: %v", entry)
		}
		name, _ := em["name"].(string)
		kind, _ := em["kind"].(string)
		if name == "" {
			t.Errorf("model entry missing name: %v", entry)
		}
		byName[name] = kind
	}
	// The six catalog tags, each with its kind.
	want := map[string]string{
		"qwen2.5:1.5b":        "systemone",
		"tev1:0.8b":           "systemone",
		"embeddinggemma:300m": "embed",
		"gemma2:2b":           "systemone",
		"clef-flash":          "systemone",
		"llama3.2:1b":         "chat",
	}
	for name, kind := range want {
		if byName[name] != kind {
			t.Errorf("mnemonic.ollama.models[%s].kind = %q, want %q", name, byName[name], kind)
		}
	}
	if _, present := byName["glm:vision-tools"]; present {
		t.Errorf("glm:vision-tools must not be in mnemonic.ollama.models")
	}
	// Unrelated keys preserved.
	if m["profile"] != "custom" {
		t.Errorf("profile = %v, want custom (preserved)", m["profile"])
	}
}

// TestConfigMergeWritesNewModels is the focused TICKET-04 gate: the home
// provider merge writes the new live-chat model (localLLMModel) and the new
// embedder model (localEmbedModel) with the Ollama base_url, while preserving
// unrelated operator keys (profile, embedder.dimension). Deterministic — no
// Ollama server required; it exercises mergeHomeProviderConfig directly.
func TestConfigMergeWritesNewModels(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// Seed operator state that the merge must preserve: a non-default profile
	// and an unrelated embedder.dimension key.
	idxPath := writeHomeIndexing(t, home, `profile: custom
mnemonic:
  embedder:
    dimension: 4096
`)

	cfg := Config{Provider: "local", HomeDir: home, RepoHome: filepath.Join(home, ".skillgrid")}
	// base differs from ollamaBaseURL by host (127.0.0.1 vs localhost) on
	// purpose: this gate pins the /v1 suffix and the provider-carried base,
	// not the real Ollama host — no seam coupling.
	base := "http://127.0.0.1:11434"
	if err := mergeHomeProviderConfig(&cfg, localLLMProvider{
		enabled: true,
		baseURL: base + "/v1",
		model:   localLLMModel,
		embed:   embedOverride{provider: "ollama", baseURL: base + "/v1", model: localEmbedModel},
	}); err != nil {
		t.Fatalf("mergeHomeProviderConfig: %v", err)
	}

	m := loadHomeIndexing(t, idxPath)
	llm := homeLLMM(t, m)
	if llm["enabled"] != true {
		t.Errorf("mnemonic.llm.enabled = %v, want true", llm["enabled"])
	}
	if llm["model"] != "llama3.2:1b" {
		t.Errorf("mnemonic.llm.model = %v, want llama3.2:1b (localLLMModel)", llm["model"])
	}
	if llm["base_url"] != base+"/v1" {
		t.Errorf("mnemonic.llm.base_url = %v, want %q", llm["base_url"], base+"/v1")
	}
	emb := homeEmbedder(t, m)
	if emb["provider"] != "ollama" {
		t.Errorf("mnemonic.embedder.provider = %v, want ollama", emb["provider"])
	}
	if emb["model"] != "embeddinggemma:300m" {
		t.Errorf("mnemonic.embedder.model = %v, want embeddinggemma:300m (localEmbedModel)", emb["model"])
	}
	if emb["base_url"] != base+"/v1" {
		t.Errorf("mnemonic.embedder.base_url = %v, want %q", emb["base_url"], base+"/v1")
	}
	// Operator keys preserved.
	if m["profile"] != "custom" {
		t.Errorf("profile = %v, want custom (preserved)", m["profile"])
	}
	if emb["dimension"] != 4096 && emb["dimension"] != float64(4096) {
		t.Errorf("mnemonic.embedder.dimension = %v, want 4096 (preserved)", emb["dimension"])
	}
}
