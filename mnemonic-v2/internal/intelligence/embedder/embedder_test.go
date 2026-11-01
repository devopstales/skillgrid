package embedder_test

import (
	"context"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/degrade"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/embedder"
)

// Null is the "absent" adapter (INV-3): it is not an erroring seam — Model is
// empty, Dimensions is 0, and the vector leg simply does not run.
func TestNullAdapterIsAbsentNotFailing(t *testing.T) {
	var e embedder.Embedder = embedder.NewNull()
	if e.Model() != "" {
		t.Fatalf("null Model() = %q, want empty", e.Model())
	}
	if e.Dimensions() != 0 {
		t.Fatalf("null Dimensions() = %d, want 0", e.Dimensions())
	}
	if _, err := e.Embed(context.Background(), []string{"x"}); err != nil {
		t.Fatalf("null Embed must not error (absent, not failing): %v", err)
	}
}

// [degraded-explicit] (embedder form) — a nil (absent) embedder seam runs the
// floor and flags the result degraded.
func TestDegradeNilEmbedderRunsFloor(t *testing.T) {
	var e embedder.Embedder // nil
	res, degraded := degrade.Run(e, func() any {
		return [][]float32(nil) // floor: no vector written
	}, "embed")
	got, ok := res.([][]float32)
	if !ok {
		t.Fatalf("floor result type = %T, want [][]float32", res)
	}
	if got != nil {
		t.Fatalf("floor must return nil vectors, got %v", got)
	}
	if !degraded {
		t.Fatal("absent embedder must be degraded")
	}
}

// [degraded-bounded-retry] (embedder form) — a present-but-failing embedder
// gets exactly one retry, then the floor. A failing Embedder never degrades
// other seams (per-call policy), and vice versa.
func TestDegradeFailingEmbedderRetriesOnce(t *testing.T) {
	calls := 0
	emb := failingEmbedder{fn: func() ([][]float32, error) {
		calls++
		return nil, errBoom
	}}
	res, degraded := degrade.Run(emb, func() any { return [][]float32(nil) }, "embed")
	if _, ok := res.([][]float32); !ok {
		t.Fatalf("floor result type = %T, want [][]float32", res)
	}
	if calls != 2 {
		t.Fatalf("expected exactly one retry (2 calls), got %d", calls)
	}
	if !degraded {
		t.Fatal("failing embedder must be degraded")
	}
}

// A successful seam must NOT be flagged degraded and must NOT run the floor.
func TestDegradeSuccessfulSeamNotDegraded(t *testing.T) {
	emb := okEmbedder{fn: func() ([][]float32, error) {
		return [][]float32{{0.1, 0.2}}, nil
	}}
	floorRan := false
	res, degraded := degrade.Run(emb, func() any {
		floorRan = true
		return nil
	}, "embed")
	got, ok := res.([][]float32)
	if !ok {
		t.Fatalf("result type = %T, want [][]float32", res)
	}
	if degraded {
		t.Fatal("successful seam must not be degraded")
	}
	if floorRan {
		t.Fatal("floor must not run on success")
	}
	if len(got) != 1 || len(got[0]) != 2 || got[0][0] != 0.1 {
		t.Fatalf("got %v, want the seam's vectors", got)
	}
}

var errBoom = &seamError{"boom"}

type seamError struct{ msg string }

func (e *seamError) Error() string { return e.msg }

type failingEmbedder struct{ fn func() ([][]float32, error) }

func (e failingEmbedder) Call(ctx context.Context, a, b any) (any, error) { return e.fn() }

func (failingEmbedder) Model() string   { return "failing" }
func (failingEmbedder) Dimensions() int { return 8 }
func (f failingEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return f.fn()
}

type okEmbedder struct{ fn func() ([][]float32, error) }

func (e okEmbedder) Call(ctx context.Context, a, b any) (any, error) { return e.fn() }

func (okEmbedder) Model() string   { return "ok-model" }
func (okEmbedder) Dimensions() int { return 2 }
func (e okEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return e.fn()
}
