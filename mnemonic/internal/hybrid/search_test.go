package hybrid

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/embedder"
	"github.com/devopstales/skillgrid/mnemonic/internal/facts"
	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic/internal/skills"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(t.TempDir(), "testproj")
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if _, err := st.DB.Exec(`INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('sess-1', 'testproj', '/tmp', datetime('now'), 'active')`); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	return st
}

func seedFactsAndSkills(t *testing.T, st *store.Store) {
	t.Helper()
	ctx := context.Background()
	fs := facts.New(st.DB, "testproj")
	_, err := fs.Add(ctx, "sess-1", "the database connection pool max size is 50")
	if err != nil {
		t.Fatalf("fact add: %v", err)
	}
	_, err = fs.Add(ctx, "sess-1", "use bcrypt for password hashing, never md5")
	if err != nil {
		t.Fatalf("fact add: %v", err)
	}
	ss := skills.New(st.DB, t.TempDir(), "testproj")
	if _, err := ss.Write(ctx, "go-testing", "go", "write table-driven go tests with t.Run subtests", "func TestX(t *testing.T){}", true); err != nil {
		t.Fatalf("skill write: %v", err)
	}
}

func TestSearchMemoryBM25OnlyNoEmbedder(t *testing.T) {
	st := openTestStore(t)
	seedFactsAndSkills(t, st)
	fs := facts.New(st.DB, "testproj")
	ss := skills.New(st.DB, t.TempDir(), "testproj")

	res, err := SearchMemory(context.Background(), fs, ss, "connection pool", 10, MemoryOptions{})
	if err != nil {
		t.Fatalf("SearchMemory: %v", err)
	}
	if len(res.Legs) != 1 || res.Legs[0] != "fts" {
		t.Fatalf("legs = %v, want [fts] (BM25-only, no embedder)", res.Legs)
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("warnings = %v, want none (BM25-only is not a warning)", res.Warnings)
	}
	if len(res.Facts) == 0 {
		t.Fatalf("facts = 0, want the pool fact via BM25")
	}
	if res.Facts[0].Content != "the database connection pool max size is 50" {
		t.Fatalf("top fact = %q, want the pool fact", res.Facts[0].Content)
	}
	if !res.Facts[0].Provenance.FTS {
		t.Fatalf("top fact provenance FTS = false, want true")
	}
}

func TestSearchMemoryVectorLegActive(t *testing.T) {
	st := openTestStore(t)
	seedFactsAndSkills(t, st)
	fs := facts.New(st.DB, "testproj")
	ss := skills.New(st.DB, t.TempDir(), "testproj")
	emb := embedder.NewHash(32)

	res, err := SearchMemory(context.Background(), fs, ss, "password hashing", 10, MemoryOptions{Embedder: emb})
	if err != nil {
		t.Fatalf("SearchMemory: %v", err)
	}
	foundSemantic := false
	for _, l := range res.Legs {
		if l == "semantic" {
			foundSemantic = true
		}
	}
	if !foundSemantic {
		t.Fatalf("legs = %v, want it to include semantic (embedder active)", res.Legs)
	}
	if len(res.Facts) == 0 {
		t.Fatalf("facts = 0, want the bcrypt fact")
	}
}

func TestSearchMemoryDegradesWhenEmbedderDown(t *testing.T) {
	st := openTestStore(t)
	seedFactsAndSkills(t, st)
	fs := facts.New(st.DB, "testproj")
	ss := skills.New(st.DB, t.TempDir(), "testproj")

	// A zero-dim HashEmbedder whose Model() is non-empty but whose vectors are
	// effectively degenerate still returns a vector; use a stub that errors.
	emb := errorEmbedder{err: errBoom}
	res, err := SearchMemory(context.Background(), fs, ss, "connection pool", 10, MemoryOptions{Embedder: emb})
	if err != nil {
		t.Fatalf("SearchMemory must not hard-fail on a down embedder, got: %v", err)
	}
	if len(res.Warnings) == 0 {
		t.Fatalf("warnings = 0, want an embedder-down warning")
	}
	joined := strings.Join(res.Warnings, "; ")
	if !strings.Contains(joined, "embedder down") {
		t.Fatalf("warnings = %v, want to mention embedder down", res.Warnings)
	}
	if len(res.Facts) == 0 {
		t.Fatalf("facts = 0 after degradation, want BM25-only results")
	}
}

func TestSearchMemoryRequiresQuery(t *testing.T) {
	st := openTestStore(t)
	fs := facts.New(st.DB, "testproj")
	ss := skills.New(st.DB, t.TempDir(), "testproj")
	if _, err := SearchMemory(context.Background(), fs, ss, "   ", 10, MemoryOptions{}); err == nil {
		t.Fatalf("empty query must error")
	}
}

func TestRankRRFOrdering(t *testing.T) {
	hits := map[string]Hit{
		"a": {Path: "a"}, "b": {Path: "b"}, "c": {Path: "c"},
	}
	fts := map[string]int{"b": 0, "a": 1, "c": 2}
	sem := map[string]int{"c": 0, "a": 1, "b": 2}
	ranked := Rank(hits, fts, nil, sem, RRFK)
	if len(ranked) != 3 {
		t.Fatalf("ranked = %d, want 3", len(ranked))
	}
	// a appears in both legs at rank 1 each: 1/61+1/61. b: fts rank0 1/60 + sem rank2 1/62.
	// a: fts rank1 + sem rank1 = 2/(60+2) = 0.03226
	// b: fts rank0 + sem rank2 = 1/60 + 1/62 = 0.03280  (top)
	// c: fts rank2 + sem rank0 = 1/62 + 1/60 = 0.03280  (tied with b; id tie-break b < c)
	// So order must be: b, c, a. c (last in FTS) is lifted out of last place
	// by its semantic rank0 — that is the fusion working.
	if ranked[0].Path != "b" {
		t.Fatalf("rank[0] = %q, want b (both-legs top scorer)", ranked[0].Path)
	}
	if ranked[2].Path != "a" {
		t.Fatalf("rank[2] = %q, want a (weakest fusion score)", ranked[2].Path)
	}
}

type errorEmbedder struct{ err error }

func (e errorEmbedder) Embed(ctx context.Context, text string) (memory.Vector, error) {
	return memory.Vector{}, e.err
}
func (e errorEmbedder) EmbedQuery(ctx context.Context, text string) (memory.Vector, error) {
	return memory.Vector{}, e.err
}
func (e errorEmbedder) Model() string  { return "error-model" }
func (e errorEmbedder) Dimension() int { return 8 }

var errBoom = errors.New("model unavailable")
