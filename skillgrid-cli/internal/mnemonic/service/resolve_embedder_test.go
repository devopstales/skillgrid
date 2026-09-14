package service

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/embedder"
)

// osMkdirConfigD creates dir/config.d (idempotent).
func osMkdirConfigD(dir string) error {
	return os.MkdirAll(filepath.Join(dir, "config.d"), 0o755)
}

// osWriteConfigYAML writes the given YAML to dir/config.d/indexing.yaml.
func osWriteConfigYAML(dir, body string) error {
	return os.WriteFile(filepath.Join(dir, "config.d", "indexing.yaml"), []byte(body), 0o644)
}

// osWriteFile writes data to path.
func osWriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

// TestResolveEmbedderRoutesOllama covers 014 step 06 M1: resolveEmbedder (the
// runtime selection path) must route to the ollama embedder, agreeing with
// embedder.Default (which selects via embedder.BuildFromConfig). Pre-fix it
// silently mapped provider:ollama to *Onnx (the default branch), so the two
// selection paths disagreed. With a live mock server, BuildFromConfig's eager
// Verify probe succeeds and the *Ollama embedder is returned.
func TestResolveEmbedderRoutesOllama(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"nomic-embed-code","embedding":[0.1,0.2,0.3]}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	if err := osMkdirConfigD(dir); err != nil {
		t.Fatalf("mkdir config.d: %v", err)
	}
	body := "mnemonic:\n  embedder:\n    provider: ollama\n    base_url: " + srv.URL + "\n    model: nomic-embed-code\n    dimension: 3\n"
	if err := osWriteConfigYAML(dir, body); err != nil {
		t.Fatalf("write indexing.yaml: %v", err)
	}

	e := resolveEmbedder(dir)
	if e == nil {
		t.Fatalf("provider=ollama: resolveEmbedder returned nil, want *Ollama")
	}
	if _, ok := e.(*embedder.Ollama); !ok {
		t.Fatalf("provider=ollama: resolveEmbedder → %T, want *Ollama", e)
	}
}

// TestResolveEmbedderRoutesLocal covers the local-provider half of 014 step 06
// M1: with provider:local and a model file present, resolveEmbedder must
// return the *LocalONNX embedder (when the onnxer runtime is available on the
// host) — NOT the *Onnx default that the pre-fix code returned for any
// non-onnx/external provider. When the runtime library is absent, selection
// degrades to the Null Adapter (06.4), which is the correct contract; both
// outcomes are asserted.
func TestResolveEmbedderRoutesLocal(t *testing.T) {
	modelDir := t.TempDir()
	// A minimal non-empty .onnx file so the model-path resolution succeeds.
	// The onnxer runtime is absent on most hosts, in which case Load fails and
	// selection degrades to Null (the tested contract is that it is NOT *Onnx).
	if err := osWriteFile(filepath.Join(modelDir, "test-model.onnx"), []byte("\x0a\x03foo")); err != nil {
		t.Fatalf("write model: %v", err)
	}

	dir := t.TempDir()
	if err := osMkdirConfigD(dir); err != nil {
		t.Fatalf("mkdir config.d: %v", err)
	}
	body := "mnemonic:\n  embedder:\n    provider: local\n    model: test-model\n    dimension: 4\n    model_dir: " + modelDir + "\n"
	if err := osWriteConfigYAML(dir, body); err != nil {
		t.Fatalf("write indexing.yaml: %v", err)
	}

	e := resolveEmbedder(dir)
	if e == nil {
		t.Fatalf("provider=local: resolveEmbedder returned nil, want *LocalONNX or Null")
	}
	if _, isOnnx := e.(*embedder.Onnx); isOnnx {
		t.Fatalf("provider=local: resolveEmbedder → *Onnx (the pre-fix silent default), want *LocalONNX or Null")
	}
	if _, ok := e.(*embedder.LocalONNX); !ok {
		if _, isNull := e.(embedder.NullEmbedder); !isNull {
			t.Fatalf("provider=local: resolveEmbedder → %T, want *LocalONNX or Null (runtime-absent degrade)", e)
		}
	}
}

// TestResolveEmbedderOffIsNull covers the off-provider half of 014 step 06 M1:
// provider:off must resolve to the Null Adapter (BuildFromConfig's off branch),
// never *Onnx. The FTS+signals floor is reached via the caller's
// nil-or-zero-length gate (Null.Embed returns a zero-length vector).
func TestResolveEmbedderOffIsNull(t *testing.T) {
	dir := t.TempDir()
	if err := osMkdirConfigD(dir); err != nil {
		t.Fatalf("mkdir config.d: %v", err)
	}
	body := "mnemonic:\n  embedder:\n    provider: off\n"
	if err := osWriteConfigYAML(dir, body); err != nil {
		t.Fatalf("write indexing.yaml: %v", err)
	}

	e := resolveEmbedder(dir)
	if e == nil {
		t.Fatalf("provider=off: resolveEmbedder returned nil, want Null (BuildFromConfig)")
	}
	if _, ok := e.(embedder.NullEmbedder); !ok {
		t.Fatalf("provider=off: resolveEmbedder → %T, want Null", e)
	}
}
