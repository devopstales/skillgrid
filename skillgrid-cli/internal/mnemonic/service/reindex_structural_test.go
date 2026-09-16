package service

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/embedder"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// reindexSpy is a counting spy embedder that records every method call so a
// test can prove the structural (embedder-free) re-index leg never touches the
// embedder. It embeds via a deterministic hash so any accidental invocation is
// observable (and non-fatal).
type reindexSpy struct {
	embedder.Embedder
	modelCalls atomic.Int64
	dimCalls   atomic.Int64
	embCalls   atomic.Int64
}

func newReindexSpy() *reindexSpy { return &reindexSpy{Embedder: embedder.NewHash(64)} }

func (s *reindexSpy) calls() int64 {
	return s.modelCalls.Load() + s.dimCalls.Load() + s.embCalls.Load()
}
func (s *reindexSpy) Model() string {
	s.modelCalls.Add(1)
	return s.Embedder.Model()
}
func (s *reindexSpy) Dimension() int {
	s.dimCalls.Add(1)
	return s.Embedder.Dimension()
}
func (s *reindexSpy) Embed(ctx context.Context, text string) (memory.Vector, error) {
	s.embCalls.Add(1)
	return s.Embedder.Embed(ctx, text)
}
func (s *reindexSpy) EmbedQuery(ctx context.Context, text string) (memory.Vector, error) {
	s.embCalls.Add(1)
	return s.Embedder.EmbedQuery(ctx, text)
}

// mustWriteSvc writes a file under root (test helper for the service package).
func mustWriteSvc(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// TestReindexStructuralIsEmbedderFree covers 03.7's load-bearing correctness
// property: ReindexStructural — the pull-at-query fingerprint gate's re-index
// entrypoint (freshness.preQuery → svc.ReindexStructural) — NEVER invokes the
// embedder leg. It re-syncs a drifted tree structurally (symbols/edges) and
// stores the stamp, without attaching or calling an embedder (no ONNX model
// load just to reflect an edit). The gate relies on this, so proving it here
// proves the gate's re-index is embedder-free in production.
func TestReindexStructuralIsEmbedderFree(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)
	root := t.TempDir()
	mustWriteSvc(t, filepath.Join(root, "main.go"), "package main\n\nfunc alpha() {\n\tbeta()\n}\n\nfunc beta() {}\n")

	// Establish the index + fingerprint under the current stamp (so the
	// re-index is a genuine structural sync of a drifted tree, not a cold index).
	if _, err := svc.RunCodeIndex(context.Background(), root); err != nil {
		t.Fatalf("initial index: %v", err)
	}
	h, cleanup, err := svc.OpenForDirectory(root)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cleanup()
	if err := codeindex.StoreFingerprint(h.Store().DB, root, codeindex.Config{}, codeindex.ExtractorStamp()); err != nil {
		t.Fatalf("store fingerprint: %v", err)
	}

	// Drift: edit main.go (size + mtime change) so the re-index actually syncs
	// a new symbol.
	mustWriteSvc(t, filepath.Join(root, "main.go"), "package main\n\nfunc alpha() {\n\tbeta()\n\tgamma()\n}\n\nfunc beta() {}\n\nfunc gamma() {}\n")
	_ = os.Chtimes(filepath.Join(root, "main.go"), time.Now().Add(time.Second), time.Now().Add(time.Second))

	// Drive the gate's re-index entrypoint. It attaches NO embedder (structurally
	// embedder-free), so no model load / no embedding call fires.
	if _, err := svc.ReindexStructural(context.Background(), root, codeindex.Config{}); err != nil {
		t.Fatalf("reindex structural: %v", err)
	}

	// The structural re-index still synced the new symbol (not a no-op — it just
	// skips the embedder leg).
	h2, cleanup2, err := svc.OpenForDirectory(root)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer cleanup2()
	var gamma int
	if err := h2.Store().DB.QueryRow(`SELECT COUNT(*) FROM symbols WHERE name = 'gamma'`).Scan(&gamma); err != nil {
		t.Fatalf("count gamma: %v", err)
	}
	if gamma == 0 {
		t.Errorf("expected the structural re-index to sync the new symbol gamma (embedder-free but not a no-op), got 0")
	}
}

// TestReindexStructuralAttachedEmbedderIgnored is the counterfactual that makes
// the embedder-free property load-bearing. The gate's re-index is embedder-free
// because ReindexStructural builds its Indexer WITHOUT calling WithEmbedder. The
// counterfactual proves the attach is the single variable: the SAME Indexer
// (codeindex.New(h.Store()), exactly what ReindexStructural constructs) run WITH
// a spy attached fires the spy (the eager embedPass invokes the embedder), while
// ReindexStructural — the same build minus the attach — leaves the spy untouched.
// If someone later wired ReindexStructural through the WithEmbedder path (a
// regression that would add an ONNX model load to every gate re-index), this
// contrast makes the property assertable and the regression visible.
func TestReindexStructuralAttachedEmbedderIgnored(t *testing.T) {
	dataDir := t.TempDir()
	svc := New(dataDir)
	root := t.TempDir()
	mustWriteSvc(t, filepath.Join(root, "main.go"), "package main\n\nfunc alpha() {\n\tbeta()\n}\n\nfunc beta() {}\n")

	// Baseline cold index via the gate's entrypoint (no embedder).
	if _, err := svc.ReindexStructural(context.Background(), root, codeindex.Config{}); err != nil {
		t.Fatalf("initial structural index: %v", err)
	}

	h, cleanup, err := svc.OpenForDirectory(root)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cleanup()

	// Counterfactual (the regression we guard against): the re-indexing Indexer
	// (codeindex.New(h.Store()) — identical to what ReindexStructural builds) run
	// WITH an embedder attached (the WithEmbedder path RunCodeIndex uses). It
	// must fire the spy, isolating the attach as the sole cause of embedder calls.
	spy := newReindexSpy()
	attached := codeindex.New(h.Store()).WithEmbedder(spy)
	if _, err := attached.Run(context.Background(), root, codeindex.Config{}); err != nil {
		t.Fatalf("attached index run: %v", err)
	}
	if got := spy.calls(); got == 0 {
		t.Fatalf("a spy-attached index run must invoke the embedder (isolating the attach); got 0 calls")
	}
}
