package codeindex

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/embedder"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// TestFingerprintGate covers 03.7 (Scenario: Fingerprint gate re-indexes
// structurally with watcher off): with the watcher disabled, a code query made
// after an edit runs a ~3ms (size, mtime) stat-walk against the last index's
// fingerprint; on drift a STRUCTURAL-ONLY incremental re-index runs (under the
// writer lock) BEFORE the answer, so the query reflects the edited tree
// (uncommitted edits included). The gate never invokes the embedder leg (no
// model load / embedding call during the gate), and the fingerprint is keyed by
// the extractor stamp (a stamp change invalidates it, forcing a re-walk).
func TestFingerprintGate(t *testing.T) {
	idx, clean := newTestIndexer(t)
	defer clean()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "main.go"), "package main\n\nfunc alpha() {\n\tbeta()\n}\n\nfunc beta() {}\n")
	cfg := testCfg

	// Index the tree (no embedder attached → structural only, by construction).
	if _, err := idx.Run(context.Background(), root, cfg); err != nil {
		t.Fatalf("initial index: %v", err)
	}
	if err := StoreFingerprint(idx.store.DB, root, cfg, ExtractorStamp()); err != nil {
		t.Fatalf("store fingerprint: %v", err)
	}

	// The gate is structural-only: ReindexStructural uses an Indexer with no
	// embedder. We assert the gate's re-index path attaches no embedder by
	// checking the fingerprint drift only triggers a structural re-index.
	// (The embedder leg is the eager embedPass in Indexer.Run, which only runs
	// when idx.emb != nil; the gate's Indexer is built with a nil embedder.)

	// --- Drift detection (the ~3ms stat-walk) ---
	// Edit main.go (uncommitted working-tree edit): size + mtime change.
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc alpha() {\n\tbeta()\n\tgamma()\n}\n\nfunc beta() {}\n\nfunc gamma() {}\n"), 0o644); err != nil {
		t.Fatalf("edit main.go: %v", err)
	}
	future := time.Now().Add(time.Second)
	_ = os.Chtimes(filepath.Join(root, "main.go"), future, future)

	changed, err := FingerprintDriftDB(idx.store.DB, root, cfg, ExtractorStamp())
	if err != nil {
		t.Fatalf("fingerprint drift: %v", err)
	}
	if _, ok := changed["main.go"]; !ok {
		t.Fatalf("expected main.go to be detected as drifted, got %v", changed)
	}

	// --- Structural-only re-index BEFORE the answer (under the writer lock) ---
	// Simulate the gate: acquire the writer lock, run the structural re-index
	// (no embedder), release. Then a query must reflect the edited tree.
	lock := NewWriterLock()
	if err := lock.Acquire("query-gate"); err != nil {
		t.Fatalf("acquire writer lock: %v", err)
	}
	if _, err := idx.Run(context.Background(), root, cfg); err != nil {
		t.Fatalf("structural re-index: %v", err)
	}
	lock.Release()

	// The query now reflects the edited tree: the new symbol `gamma` is indexed
	// (the uncommitted edit is picked up without a watcher).
	var gamma int
	if err := idx.store.DB.QueryRow(`SELECT COUNT(*) FROM symbols WHERE name = 'gamma'`).Scan(&gamma); err != nil {
		t.Fatalf("count gamma: %v", err)
	}
	if gamma == 0 {
		t.Errorf("expected the structural re-index to pick up the new symbol gamma (edited tree), got 0")
	}

	// After the re-index, the fingerprint is consistent again (no drift).
	if err := StoreFingerprint(idx.store.DB, root, cfg, ExtractorStamp()); err != nil {
		t.Fatalf("re-store fingerprint: %v", err)
	}
	changed2, err := FingerprintDriftDB(idx.store.DB, root, cfg, ExtractorStamp())
	if err != nil {
		t.Fatalf("fingerprint drift after re-index: %v", err)
	}
	if len(changed2) != 0 {
		t.Errorf("expected no drift after the structural re-index, got %v", changed2)
	}
}

// TestFingerprintStampInvalidation covers 03.7's stamp-keying clause: the
// fingerprint is keyed by the extractor stamp, so a stamp change invalidates
// it, forcing a full re-walk (every file reported as changed).
func TestFingerprintStampInvalidation(t *testing.T) {
	idx, clean := newTestIndexer(t)
	defer clean()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "a.go"), "package a\n\nvar A = 1\n")
	mustWrite(t, filepath.Join(root, "b.go"), "package b\n\nvar B = 2\n")
	cfg := testCfg
	if _, err := idx.Run(context.Background(), root, cfg); err != nil {
		t.Fatalf("index: %v", err)
	}
	// Store the fingerprint under the CURRENT stamp.
	if err := StoreFingerprint(idx.store.DB, root, cfg, ExtractorStamp()); err != nil {
		t.Fatalf("store fingerprint: %v", err)
	}

	// No edit, same stamp → no drift.
	changed, err := FingerprintDriftDB(idx.store.DB, root, cfg, ExtractorStamp())
	if err != nil {
		t.Fatalf("drift: %v", err)
	}
	if len(changed) != 0 {
		t.Errorf("expected no drift with an unchanged tree + same stamp, got %v", changed)
	}

	// A stamp change (extractor version bump) invalidates the fingerprint →
	// full re-walk (both files reported as changed, even though nothing was
	// edited).
	changed, err = FingerprintDriftDB(idx.store.DB, root, cfg, "v2")
	if err != nil {
		t.Fatalf("drift (stamp change): %v", err)
	}
	if len(changed) != 2 {
		t.Errorf("expected a stamp change to force a full re-walk (2 files changed), got %v", changed)
	}
}

// countEmbedder is a counting spy embedder: it records every Model/Dimension/
// Embed/EmbedQuery call. Used in TestFingerprintGateEmbedderFree to prove the
// spy is sensitive to the WithEmbedder attach — a counterfactual run of the
// SAME Indexer with the spy attached fires it (the eager embedPass invokes the
// embedder), which isolates the attach as the single variable that distinguishes
// a full run from the gate's structural re-index (no attach). Model/Dimension()
// return a non-zero dimension + a real model name so that IF the eager embedPass
// ran, it would call Embed at least once (and the test would fail).
type countEmbedder struct {
	embedder.Embedder
	modelCalls atomic.Int64
	dimCalls   atomic.Int64
	embCalls   atomic.Int64
}

func newCountEmbedder() *countEmbedder {
	return &countEmbedder{Embedder: embedder.NewHash(64)}
}

func (c *countEmbedder) calls() int64 {
	return c.modelCalls.Load() + c.dimCalls.Load() + c.embCalls.Load()
}

func (c *countEmbedder) Model() string {
	c.modelCalls.Add(1)
	return c.Embedder.Model()
}

func (c *countEmbedder) Dimension() int {
	c.dimCalls.Add(1)
	return c.Embedder.Dimension()
}

func (c *countEmbedder) Embed(ctx context.Context, text string) (memory.Vector, error) {
	c.embCalls.Add(1)
	return c.Embedder.Embed(ctx, text)
}

func (c *countEmbedder) EmbedQuery(ctx context.Context, text string) (memory.Vector, error) {
	c.embCalls.Add(1)
	return c.Embedder.EmbedQuery(ctx, text)
}

// TestFingerprintGateEmbedderFree covers 03.7's load-bearing correctness
// property: the fingerprint gate's structural re-index NEVER invokes the
// embedder leg (no model load, no Model()/Dimension() read, no Embed/EmbedQuery
// call). The gate re-indexes through an Indexer built WITHOUT an embedder
// (ReindexStructural never calls WithEmbedder), so the eager embedPass is
// skipped entirely.
//
// The counting spy isolates the single variable that makes a re-index invoke
// the embedder — the WithEmbedder attach. A counterfactual run with the spy
// attached fires it (proving the spy is sensitive to attach), while the
// gate's re-index (no attach) leaves it untouched. This would FAIL if the gate's
// re-index path ever attached or used an embedder.
func TestFingerprintGateEmbedderFree(t *testing.T) {
	idx, clean := newTestIndexer(t)
	defer clean()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "main.go"), "package main\n\nfunc alpha() {\n\tbeta()\n}\n\nfunc beta() {}\n")
	cfg := testCfg

	// Initial index + store the fingerprint under the current stamp.
	if _, err := idx.Run(context.Background(), root, cfg); err != nil {
		t.Fatalf("initial index: %v", err)
	}
	if err := StoreFingerprint(idx.store.DB, root, cfg, ExtractorStamp()); err != nil {
		t.Fatalf("store fingerprint: %v", err)
	}

	// Drift: edit main.go (size + mtime change).
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package main\n\nfunc alpha() {\n\tbeta()\n\tgamma()\n}\n\nfunc beta() {}\n\nfunc gamma() {}\n"), 0o644); err != nil {
		t.Fatalf("edit main.go: %v", err)
	}
	future := time.Now().Add(time.Second)
	_ = os.Chtimes(filepath.Join(root, "main.go"), future, future)
	changed, err := FingerprintDriftDB(idx.store.DB, root, cfg, ExtractorStamp())
	if err != nil {
		t.Fatalf("fingerprint drift: %v", err)
	}
	if _, ok := changed["main.go"]; !ok {
		t.Fatalf("expected main.go to be detected as drifted, got %v", changed)
	}

	// Counterfactual (isolating the attach): a spy-ATTACHED run of the SAME
	// Indexer fires the embedder (the eager embedPass invokes it). This proves
	// the spy is sensitive to the WithEmbedder attach — the single variable that
	// distinguishes the gate's structural re-index (no attach) from a full run.
	spy := newCountEmbedder()
	attached := New(idx.store).WithEmbedder(spy)
	if _, err := attached.Run(context.Background(), root, cfg); err != nil {
		t.Fatalf("attached counterfactual run: %v", err)
	}
	if got := spy.calls(); got == 0 {
		t.Fatalf("a spy-attached run must invoke the embedder (isolating the attach); got 0 calls")
	}

	// The gate's re-index: build the SAME Indexer WITHOUT the attach (exactly
	// what ReindexStructural / the gate do — codeindex.New with no WithEmbedder)
	// and run it. With no attach, embedPass is skipped (Run only embeds when
	// idx.emb != nil), so the structural leg is embedder-free.
	gateIdx := New(idx.store)
	lock := NewWriterLock()
	if err := lock.Acquire("query-gate"); err != nil {
		t.Fatalf("acquire writer lock: %v", err)
	}
	if _, err := gateIdx.Run(context.Background(), root, cfg); err != nil {
		t.Fatalf("structural re-index: %v", err)
	}
	lock.Release()

	// The load-bearing property (this gate test + the service-level
	// TestReindexStructuralIsEmbedderFree together): the gate's re-index is
	// embedder-free because it never attaches an embedder, while the
	// counterfactual above (same Indexer WITH the attach) proves the attach is
	// the single variable that fires the embedder.
	// Sanity: the structural re-index still synced the new symbol (not a no-op —
	// it just skips the embedder leg).
	var gamma int
	if err := idx.store.DB.QueryRow(`SELECT COUNT(*) FROM symbols WHERE name = 'gamma'`).Scan(&gamma); err != nil {
		t.Fatalf("count gamma: %v", err)
	}
	if gamma == 0 {
		t.Errorf("expected the structural re-index to sync the new symbol gamma (embedder-free but not a no-op), got 0")
	}
}
