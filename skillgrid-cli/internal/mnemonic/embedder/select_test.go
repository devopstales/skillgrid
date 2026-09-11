package embedder

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/config"
)

// writeIndexing writes an indexing.yaml (mnemonic.embedder section) into a
// fresh config.d directory and returns it.
func writeIndexing(t *testing.T, embedderYAML string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.d", "indexing.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "mnemonic:\n" + embedderYAML
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestEmbedderConfigDrivenSelection (06.3): indexing.yaml selects the
// provider; BuildFromConfig returns the matching embedder type.
func TestEmbedderConfigDrivenSelection(t *testing.T) {
	// ollama → *Ollama
	dir := writeIndexing(t, "  embedder:\n    provider: ollama\n    model: test-model\n    dimension: 32\n")
	cfg := config.Load(dir)
	if cfg.Embedder.Provider != "ollama" {
		t.Fatalf("config provider=%q, want ollama", cfg.Embedder.Provider)
	}
	e := BuildFromConfig(toEmbedderConfig(cfg.Embedder))
	if _, ok := e.(*Ollama); !ok {
		t.Fatalf("provider=ollama → %T, want *Ollama", e)
	}

	// local → *LocalONNX (model file present). When the onnxer runtime
	// library is absent on this host, selection degrades to Null (06.4) —
	// the file is found, the library is not; both are verified below.
	modelDir := t.TempDir()
	writeTestModel(t, modelDir, "test-model.onnx")
	dir = writeIndexing(t, "  embedder:\n    provider: local\n    model: test-model\n    dimension: 4\n    model_dir: "+modelDir+"\n")
	e = BuildFromConfig(toEmbedderConfig(config.Load(dir).Embedder))
	if !onnxRuntimeAvailable() {
		if _, ok := e.(NullEmbedder); !ok {
			t.Fatalf("provider=local (runtime absent) → %T, want NullEmbedder", e)
		}
	} else if _, ok := e.(*LocalONNX); !ok {
		t.Fatalf("provider=local → %T, want *LocalONNX", e)
	}

	// onnx → *Onnx (existing behavior)
	dir = writeIndexing(t, "  embedder:\n    provider: onnx\n")
	e = BuildFromConfig(toEmbedderConfig(config.Load(dir).Embedder))
	if _, ok := e.(*Onnx); !ok {
		t.Fatalf("provider=onnx → %T, want *Onnx", e)
	}

	// external → *External (existing behavior)
	dir = writeIndexing(t, "  embedder:\n    provider: external\n    base_url: http://localhost:9999\n    model: ext\n    dimension: 8\n")
	e = BuildFromConfig(toEmbedderConfig(config.Load(dir).Embedder))
	if _, ok := e.(*External); !ok {
		t.Fatalf("provider=external → %T, want *External", e)
	}

	// off → Null
	dir = writeIndexing(t, "  embedder:\n    provider: off\n")
	e = BuildFromConfig(toEmbedderConfig(config.Load(dir).Embedder))
	if _, ok := e.(NullEmbedder); !ok {
		t.Fatalf("provider=off → %T, want NullEmbedder", e)
	}

	// no provider → the default (onnx) embedder: config.Load falls back to
	// DefaultEmbedder() (provider "onnx"), so an absent provider key keeps
	// the current default. An explicitly empty provider in a section maps to
	// Null via the selector's "" branch.
	dir = writeIndexing(t, "")
	if got := config.Load(dir).Embedder.Provider; got != "onnx" {
		t.Fatalf("absent provider key → config provider %q, want the onnx default", got)
	}
	e = BuildFromConfig(toEmbedderConfig(config.Load(dir).Embedder))
	if _, ok := e.(*Onnx); !ok {
		t.Fatalf("absent provider key → %T, want *Onnx (current default)", e)
	}
	if _, ok := BuildFromConfig(EmbedderConfig{}).(NullEmbedder); !ok {
		t.Fatal("empty provider → want NullEmbedder")
	}
}

// toEmbedderConfig maps the config package's EmbedderConfig onto the
// embedder package's selection struct (no import cycle).
func toEmbedderConfig(c config.EmbedderConfig) EmbedderConfig {
	return EmbedderConfig{
		Provider:  c.Provider,
		Dimension: c.Dimension,
		Indexing:  AsymParams{Instructions: c.Indexing.Instructions, InputType: c.Indexing.InputType, MaxTokens: c.Indexing.MaxTokens},
		Query:     AsymParams{Instructions: c.Query.Instructions, InputType: c.Query.InputType, MaxTokens: c.Query.MaxTokens},
		BaseURL:   c.BaseURL,
		Model:     c.Model,
		APIKey:    c.APIKey,
		ModelDir:  c.ModelDir,
	}
}

// TestConfigLoadsNewProviders (06.3): the YAML provider + model_dir keys are
// parsed by config.Load.
func TestConfigLoadsNewProviders(t *testing.T) {
	dir := writeIndexing(t, "  embedder:\n    provider: local\n    model: foo\n    model_dir: /tmp/models\n")
	cfg := config.Load(dir)
	if cfg.Embedder.Provider != "local" {
		t.Fatalf("provider=%q, want local", cfg.Embedder.Provider)
	}
	if cfg.Embedder.ModelDir != "/tmp/models" {
		t.Fatalf("model_dir=%q, want /tmp/models", cfg.Embedder.ModelDir)
	}

	// a malformed provider value must not break loading (it is preserved as
	// text and degraded at selection time)
	dir = writeIndexing(t, "  embedder:\n    provider: bogus\n")
	if got := config.Load(dir).Embedder.Provider; got != "bogus" {
		t.Fatalf("provider=%q, want bogus (preserved)", got)
	}
}

// TestConfigLoadValidYAML is a guard that the fixture YAML parses at all.
func TestConfigLoadValidYAML(t *testing.T) {
	var doc map[string]any
	if err := yaml.Unmarshal([]byte("mnemonic:\n  embedder:\n    provider: ollama\n"), &doc); err != nil {
		t.Fatal(err)
	}
}
