package embedder

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTestModel drops a minimal ONNX protobuf (an Identity Constant node:
// input -> output, no weights) into dir as name. Real embedding inference
// needs a real encoder model + tokenizer (wired in a follow-up); this file
// is enough for onnxer to build a session and run inference.
func writeTestModel(t *testing.T, dir, name string) string {
	t.Helper()
	proto := []byte{
		0x08, 0x08, // ir_version = 8
		0x12, 0x08, 0x74, 0x65, 0x73, 0x74, 0x6d, 0x6f, 0x64, 0x65, 0x6c, // graph.name = "testmodel"
		0x22, 0x19,
		0x0a, 0x05, 0x69, 0x6e, 0x70, 0x75, 0x74, // node[0].input[0] = "input"
		0x12, 0x05, 0x6f, 0x75, 0x74, 0x70, 0x75, 0x74, // node[0].output[0] = "output"
		0x1a, 0x08, 0x43, 0x6f, 0x6e, 0x73, 0x74, 0x61, 0x6e, 0x74, // node[0].op_type = "Constant"
		0x4a, 0x05, 0x08, 0x01, 0x10, 0x01, 0x28, 0x01, // node[0].attribute[0]: type=FLOAT(1), i=1
		0x52, 0x05, 0x69, 0x6e, 0x70, 0x75, 0x74, // graph.input[0].name = "input"
		0x5a, 0x05, 0x6f, 0x75, 0x74, 0x70, 0x75, 0x74, // graph.output[0].name = "output"
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, proto, 0o644); err != nil {
		t.Fatalf("write model: %v", err)
	}
	return path
}

// TestLocalONNXEmbedder (06.2): the loader resolves the model file from a
// configured directory, loads it (descriptive error when missing), and
// inference runs via onnxer.
func TestLocalONNXEmbedder(t *testing.T) {
	dir := t.TempDir()
	writeTestModel(t, dir, "test-model.onnx")

	l := NewLocal(LocalConfig{ModelDir: dir, Model: "test-model", Dimension: 4})

	if got := l.Model(); got != "test-model" {
		t.Fatalf("Model()=%q, want %q", got, "test-model")
	}
	if got := l.Dimension(); got != 4 {
		t.Fatalf("Dimension()=%d, want 4", got)
	}
	if got := l.ModelPath(); got != filepath.Join(dir, "test-model.onnx") {
		t.Fatalf("ModelPath()=%q, want %q", got, filepath.Join(dir, "test-model.onnx"))
	}

	// Model found → resolves to the file and loads. If the onnxer runtime
	// library is absent on this host, loading reports
	// ErrRuntimeUnavailable (the caller degrades to Null at selection time)
	// instead of a model error — the file itself was found either way.
	if _, err := l.loadSession(); err != nil {
		if !errors.Is(err, ErrRuntimeUnavailable) {
			t.Fatalf("loadSession with model present: %v (want nil or ErrRuntimeUnavailable)", err)
		}
		t.Logf("onnx runtime library absent on this host: %v", err)
	}
}

// TestLocalONNXEmbedderMissing (06.2): a missing model file (or directory)
// yields a descriptive error, not a panic.
func TestLocalONNXEmbedderMissing(t *testing.T) {
	l := NewLocal(LocalConfig{
		ModelDir:  filepath.Join(t.TempDir(), "does-not-exist"),
		Model:     "ghost-model",
		Dimension: 4,
	})

	if err := l.Load(ctx0); err == nil {
		t.Fatal("Load with missing model: want error, got nil")
	} else if !strings.Contains(err.Error(), "ghost-model.onnx") {
		t.Fatalf("error=%q, want it to name the missing model file", err)
	}

	_, err := l.Embed(context.Background(), "test text")
	if err == nil {
		t.Fatal("Embed with missing model: want error, got nil")
	} else if !strings.Contains(err.Error(), "ghost-model.onnx") {
		t.Fatalf("Embed error=%q, want it to name the missing model file", err)
	}
}

// ctx0 is a package-level background context for load-path tests.
var ctx0 = context.Background()
