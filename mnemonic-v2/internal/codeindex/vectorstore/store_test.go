package vectorstore

import (
	"context"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// newTestStore opens a fresh SQLite store for a test (the vectorstore's test
// fixture — the real SQLite store, the production path).
func newTestStore(t *testing.T) store.Store {
	t.Helper()
	st, err := store.Open(context.Background(), t.TempDir(), "vec-test", store.SQLite, "")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// a 768-dim embedder (the ANN-activating dim — the HNSW/vec0 path).
type emb768 struct {
	model string
}

func (e emb768) Model() string   { return e.model }
func (e emb768) Dimensions() int { return 768 }
func (e emb768) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	// A deterministic 768-dim vector (the test fixture — not the hash-64
	// floor; it activates the ANN path).
	out := make([][]float32, len(texts))
	for i, t := range texts {
		v := make([]float32, 768)
		for j, c := range t {
			if j < 768 {
				v[j] = float32(c)
			}
		}
		// L2-normalize (cosine well-defined).
		var norm float64
		for _, x := range v {
			norm += float64(x) * float64(x)
		}
		if norm > 0 {
			inv := float32(1 / float64(norm+1))
			for k := range v {
				v[k] *= inv
			}
		}
		out[i] = v
	}
	return out, nil
}

// a 64-dim embedder (the hash-64 floor — brute-force only).
type emb64 struct {
	model string
}

func (e emb64) Model() string   { return e.model }
func (e emb64) Dimensions() int { return 64 }
func (e emb64) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, t := range texts {
		v := make([]float32, 64)
		for j, c := range t {
			if j < 64 {
				v[j] = float32(c)
			}
		}
		out[i] = v
	}
	return out, nil
}

// TestSemanticStoreWired3Index verifies the 3-index semantic store (D-CI8):
// a 768-dim embedder → HNSW is preferred (SQLite); LSH/IVF are NOT in the
// preference order (dropped).
func TestSemanticStoreWired3Index(t *testing.T) {
	st := newTestStore(t)
	e := NewEngine(st, emb768{model: "test-768a"})
	ctx := context.Background()

	// Upsert a file + a symbol (the embedder's target).
	fid, err := st.UpsertFile(ctx, store.FileRow{Path: "pay.go", MtimeNS: 1, Size: 10, ContentHash: "c", Language: "go"})
	if err != nil {
		t.Fatalf("upsert file: %v", err)
	}
	_, err = st.UpsertSymbols(ctx, fid, []store.Symbol{{
		Name: "CapturePayment", QualifiedName: "x.CapturePayment", Kind: "function",
		Language: "go", UID: "pay.go:CapturePayment:function",
	}})
	if err != nil {
		t.Fatalf("upsert symbol: %v", err)
	}

	// Embed the symbol (the 768-dim vector → the ANN tables).
	if err := e.EmbedSymbol(ctx, fid, "pay.go:CapturePayment:function", "CapturePayment (sig)"); err != nil {
		t.Fatalf("embed symbol: %v", err)
	}

	// Search (the query vector is the same as the stored one → top-1 hit).
	queryVec := make([]float64, 768)
	queryVec[0] = 1
	results, idx, err := e.Search(ctx, queryVec, 5)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	// HNSW is preferred for a 768-dim embedder (SQLite) — NOT LSH/IVF.
	if idx != "hnsw" {
		t.Fatalf("768-dim embedder → HNSW is preferred (SQLite), got %s", idx)
	}
	if idx == "lsh" || idx == "ivf" {
		t.Fatalf("LSH/IVF must not be shipped (D-CI8): got %s", idx)
	}
	if len(results) == 0 {
		t.Fatalf("semantic search must return hits, got none (index %s)", idx)
	}
}

// TestVectorPreferenceOrderFallback verifies the preference order
// HNSW → vec0 → brute-force (never a hard fail — the brute-force floor
// always answers).
func TestVectorPreferenceOrderFallback(t *testing.T) {
	st := newTestStore(t)
	e := NewEngine(st, emb768{model: "test-768b"})
	ctx := context.Background()

	fid, _ := st.UpsertFile(ctx, store.FileRow{Path: "a.go", MtimeNS: 1, Size: 10, ContentHash: "c", Language: "go"})
	_, _ = st.UpsertSymbols(ctx, fid, []store.Symbol{{
		Name: "Foo", QualifiedName: "x.Foo", Kind: "function", Language: "go",
		UID: "a.go:Foo:function",
	}})
	if err := e.EmbedSymbol(ctx, fid, "a.go:Foo:function", "Foo (sig)"); err != nil {
		t.Fatalf("embed symbol: %v", err)
	}

	queryVec := make([]float64, 768)
	queryVec[0] = 1

	// HNSW missing → vec0.
	e.DisableIndexForTest("hnsw")
	_, idx, err := e.Search(ctx, queryVec, 5)
	if err != nil {
		t.Fatalf("search (hnsw disabled): %v", err)
	}
	if idx != "vec0" {
		t.Fatalf("HNSW missing → vec0, got %s", idx)
	}

	// vec0 missing → brute-force (the floor — always answers).
	e.DisableIndexForTest("vec0")
	results, idx2, err := e.Search(ctx, queryVec, 5)
	if err != nil {
		t.Fatalf("search (vec0 disabled): %v", err)
	}
	if idx2 != "brute_force" {
		t.Fatalf("vec0 missing → brute-force floor, got %s", idx2)
	}
	if len(results) == 0 {
		t.Fatalf("the brute-force floor must answer (never a hard fail)")
	}
}

// TestModelSwapGuard verifies the M9 model-swap + dim-change guard:
// a model-name change (same dim) → vectors + indexes cleared; a dimension
// change (64↔768) → the dim-shaped ANN tables DROPPED + vectors cleared.
func TestModelSwapGuard(t *testing.T) {
	st := newTestStore(t)
	ctx := context.Background()
	e := NewEngine(st, emb768{model: "model-A"})

	fid, _ := st.UpsertFile(ctx, store.FileRow{Path: "b.go", MtimeNS: 1, Size: 10, ContentHash: "c", Language: "go"})
	_, _ = st.UpsertSymbols(ctx, fid, []store.Symbol{{
		Name: "Bar", QualifiedName: "x.Bar", Kind: "function", Language: "go",
		UID: "b.go:Bar:function",
	}})
	if err := e.EmbedSymbol(ctx, fid, "b.go:Bar:function", "Bar (sig)"); err != nil {
		t.Fatalf("embed symbol: %v", err)
	}
	// A second symbol (so CountVectors > 0).
	_, _ = st.UpsertSymbols(ctx, fid, []store.Symbol{{
		Name: "Baz", QualifiedName: "x.Baz", Kind: "function", Language: "go",
		UID: "b.go:Baz:function",
	}})
	if err := e.EmbedSymbol(ctx, fid, "b.go:Baz:function", "Baz (sig)"); err != nil {
		t.Fatalf("embed symbol 2: %v", err)
	}
	// Record the model (the guard at index start — the production call site
	// runs before any vector write, but here the embeddings are the corpus so
	// the guard records them on the first call).
	if err := GuardModelSwap(ctx, st, emb768{model: "model-A"}); err != nil {
		t.Fatalf("guard (first run): %v", err)
	}

	// A model-name change (same 768-dim) → vectors + indexes cleared.
	if err := GuardModelSwap(ctx, st, emb768{model: "model-B"}); err != nil {
		t.Fatalf("guard (model swap): %v", err)
	}
	if e.CountVectors() != 0 {
		t.Fatalf("a model swap must clear vectors, got %d", e.CountVectors())
	}

	// A DIMENSION change (768→64→768) → the dim-shaped ANN tables are
	// DROPPED (the M9 load-bearing fix: a dim change is a re-index, not a
	// re-embed-in-place). The model swap above cleared the ANN *state* (rows)
	// but kept the tables' dim shape (768). A dim change DROPS the tables.
	e2 := NewEngine(st, emb64{model: "model-64"})
	if err := e2.EmbedSymbol(ctx, fid, "b.go:Bar:function", "Bar (sig)"); err != nil {
		t.Fatalf("embed symbol (64-dim): %v", err)
	}
	if e2.CountVectors() == 0 {
		t.Fatalf("a 64-dim embed must write a vector (the BLOB floor)")
	}
	// The 768→64 dim change → the dim-shaped ANN tables (768) are DROPPED.
	if err := GuardModelSwap(ctx, st, emb64{model: "model-64"}); err != nil {
		t.Fatalf("guard (dim change 768→64): %v", err)
	}
	if n := e2.ANNTablesDropped(); n == 0 {
		t.Fatalf("a dim change (768→64) must DROP the dim-shaped ANN tables, none dropped")
	}
	if e2.CountVectors() != 0 {
		t.Fatalf("a dim change must clear vectors, got %d", e2.CountVectors())
	}
}
