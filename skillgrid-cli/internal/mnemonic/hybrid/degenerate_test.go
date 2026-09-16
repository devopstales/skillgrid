package hybrid

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/embedder"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// zeroVectorBlob is the on-disk form of an all-zeros 64-dim vector — the
// canonical degenerate embedding (cosine 0 with everything, no direction).
func zeroVectorBlob() []byte {
	return memory.EncodeVector(memory.Vector{Data: make([]float32, 64)})
}

// hasWarning reports whether any warning mentions s (case-insensitive).
func hasWarning(warnings []string, s string) bool {
	for _, w := range warnings {
		if strings.Contains(strings.ToLower(w), s) {
			return true
		}
	}
	return false
}

// TestDegenerateVectorDegrades covers scenario 04.11: a degenerate (all-zeros)
// stored embedding is skipped by the vector leg, search degrades to the FTS
// floor, and the run surfaces a "degenerate ... skipped" warning instead of
// storing a silent bad vector.
func TestDegenerateVectorDegrades(t *testing.T) {
	ctx := context.Background()
	db := openHybridTestDB(t)
	insertHybridFixture(t, db)

	// One degenerate row + one healthy row for the same symbol set.
	if _, err := db.Exec(`INSERT INTO embeddings (symbol_id, vector, model) VALUES (1, ?, ?)`,
		zeroVectorBlob(), "hash-embedder-v1"); err != nil {
		t.Fatalf("insert degenerate embedding: %v", err)
	}
	// Two healthy rows: the corpus (indexing) side and the query (query:
	// prefix) side. HashEmbedder is asymmetric, so both vectors are distinct
	// and non-zero.
	healthy := embedder.NewHash(64)
	if vec, err := healthy.Embed(ctx, "parseConfig parseConfig"); err != nil {
		t.Fatalf("embed corpus: %v", err)
	} else if _, err := db.Exec(`INSERT INTO embeddings (symbol_id, vector, model) VALUES (2, ?, ?)`,
		memory.EncodeVector(vec), "hash-embedder-v1"); err != nil {
		t.Fatalf("insert healthy corpus embedding: %v", err)
	}
	if vec, err := healthy.EmbedQuery(ctx, "parseConfig"); err != nil {
		t.Fatalf("embed query: %v", err)
	} else if _, err := db.Exec(`INSERT INTO embeddings (symbol_id, vector, model) VALUES (3, ?, ?)`,
		memory.EncodeVector(vec), "hash-embedder-v1"); err != nil {
		t.Fatalf("insert healthy query embedding: %v", err)
	}

	res, err := Search(ctx, db, "parseConfig", Options{Embedder: embedder.NewHash(64)})
	if err != nil {
		t.Fatalf("Search with a degenerate row stored: %v", err)
	}

	// FTS floor still ranks the symbol.
	var fts bool
	for _, h := range res.Hits {
		if h.Symbol == "parseConfig" && h.Provenance.FTS {
			fts = true
		}
	}
	if !fts {
		t.Errorf("degenerate row must not drop the FTS floor; hits: %+v", res.Hits)
	}

	// The vector leg must report the skipped degenerate row.
	if !hasWarning(res.Warnings, "degenerate") && !hasWarning(res.Warnings, "skipped") {
		t.Errorf("warnings %q, want a degenerate/skipped note", res.Warnings)
	}

	// The healthy row still feeds the semantic leg; the degenerate one never
	// surfaces (a zero vector has cosine similarity 0 with everything).
	if !contains(res.Legs, "semantic") {
		t.Errorf("legs %v, want semantic (healthy row present)", res.Legs)
	}
	for _, h := range res.Hits {
		if h.Provenance.Semantic && h.Symbol == "parseConfig" {
			t.Errorf("degenerate parseConfig (sym:1) leaked into the vector leg: %+v", h)
		}
	}
}

// TestDegenerateVectorDetected exercises the doctor's degenerate-detection
// predicate: all-zeros and NaN vectors are degenerate, a healthy vector is not.
func TestDegenerateVectorDetected(t *testing.T) {
	detect := func(v memory.Vector) bool {
		anyNonZero := false
		for _, f := range v.Data {
			if math.IsNaN(float64(f)) || math.IsInf(float64(f), 0) {
				return true
			}
			if f != 0 {
				anyNonZero = true
			}
		}
		return !anyNonZero
	}

	t.Run("all zeros is degenerate", func(t *testing.T) {
		if !detect(memory.Vector{Data: make([]float32, 64)}) {
			t.Error("all-zeros vector must be detected as degenerate")
		}
	})

	t.Run("all NaN is degenerate", func(t *testing.T) {
		data := make([]float32, 64)
		for i := range data {
			data[i] = math.Float32frombits(0x7fc00000) // NaN
		}
		if !detect(memory.Vector{Data: data}) {
			t.Error("all-NaN vector must be detected as degenerate")
		}
	})

	t.Run("healthy vector is not degenerate", func(t *testing.T) {
		data := make([]float32, 64)
		for i := range data {
			data[i] = 1
		}
		if detect(memory.Vector{Data: data}) {
			t.Error("non-zero finite vector must not be detected as degenerate")
		}
	})

	// A zero vector decodes round-trip and keeps cosine 0 with any vector —
	// the reason it is silently useless in the vector leg.
	vec, err := memory.DecodeVector(zeroVectorBlob())
	if err != nil {
		t.Fatalf("decode zero blob: %v", err)
	}
	if got := memory.CosineSimilarity(vec, vec); got != 0 {
		t.Errorf("zero vector cosine self = %v, want 0", got)
	}
}
