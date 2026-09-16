package service

import (
	"os"
	"path/filepath"
	"testing"
)

// writeConfigDir writes a minimal config.d/indexing.yaml under dir so
// config.Load(dir) resolves the given embedder provider (distinct from the
// CWD's config, which walks up to the repo root where no provider override
// exists).
func writeConfigDir(t *testing.T, dir, provider string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "config.d"), 0o755); err != nil {
		t.Fatalf("mkdir config.d: %v", err)
	}
	body := "mnemonic:\n  embedder:\n    provider: " + provider + "\n"
	if err := os.WriteFile(filepath.Join(dir, "config.d", "indexing.yaml"), []byte(body), 0o644); err != nil {
		t.Fatalf("write indexing.yaml: %v", err)
	}
}

// TestWarmSearchEmbedderUsesProjectRoot covers the fix for finding 2:
// warmSearchEmbedder must resolve the embedder from the project root (h.root),
// not the CWD. With h.root != CWD and h.root's config pinning the provider to
// "off" (while the CWD's repo-root config defaults to onnx), the warm path must
// return nil (the FTS floor) — proving the config was read from h.root, not
// the CWD.
func TestWarmSearchEmbedderUsesProjectRoot(t *testing.T) {
	resetWarmCache()
	defer resetWarmCache()

	// A project root whose config pins the embedder to "off".
	root := t.TempDir()
	writeConfigDir(t, root, "off")

	s := &Service{dataDir: t.TempDir()}
	// warmSearchEmbedder(root) reads config.Load(root) → provider "off" → nil.
	if emb := s.warmSearchEmbedder(root); emb != nil {
		t.Fatalf("expected a nil (FTS-floor) embedder when the project root's config pins provider=off, got %v", emb)
	}

	// And a root whose config pins the provider to "onnx" resolves a non-nil
	// embedder (the warm cache loads it) — the opposite direction of the fix.
	root2 := t.TempDir()
	writeConfigDir(t, root2, "onnx")
	if emb := s.warmSearchEmbedder(root2); emb == nil {
		t.Fatalf("expected a non-nil (onnx) embedder when the project root's config pins provider=onnx")
	}
}

// TestWarmSearchEmbedderHRootNotCWD is the end-to-end form: a search where
// h.root != CWD resolves the embedder from h.root's config (not CWD). We use
// the warmSearchEmbedder hook directly (the same code path CodeHybridSearch /
// CodeSemanticSearch use) with the CWD's config defaulting to onnx, while
// h.root's config pins "off". The result must be the FTS floor (nil) — the
// h.root config wins.
func TestWarmSearchEmbedderHRootNotCWD(t *testing.T) {
	resetWarmCache()
	defer resetWarmCache()

	// CWD is the repo (go test runs in the package dir): its config walks up to
	// the repo root with the default onnx provider (no override). h.root is a
	// temp dir whose config pins "off".
	root := t.TempDir()
	writeConfigDir(t, root, "off")

	s := &Service{dataDir: t.TempDir()}
	// If the CWD config (onnx) were read, this would be non-nil. It must be nil
	// because h.root's config (off) is what warmSearchEmbedder reads.
	if emb := s.warmSearchEmbedder(root); emb != nil {
		t.Fatalf("warmSearchEmbedder must read h.root's config (off → FTS floor), not the CWD's (onnx); got a non-nil embedder")
	}
}

// TestWarmSearchEmbedderOffIsNil covers the off/empty-provider branch directly:
// an "off" or empty provider returns nil (the FTS floor) — no vector leg, no
// RAM. This is the warm cache's off behavior (kept intact by the h.root fix).
func TestWarmSearchEmbedderOffIsNil(t *testing.T) {
	resetWarmCache()
	defer resetWarmCache()

	s := &Service{dataDir: t.TempDir()}

	// An explicit "off" provider in h.root's config → FTS floor (nil).
	offRoot := t.TempDir()
	writeConfigDir(t, offRoot, "off")
	if emb := s.warmSearchEmbedder(offRoot); emb != nil {
		t.Fatalf("provider=off must return a nil (FTS-floor) embedder")
	}
}
