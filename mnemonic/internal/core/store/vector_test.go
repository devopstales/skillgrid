package store

import (
	"context"
	"encoding/binary"
	"math"
	"testing"
)

// mathBits is the IEEE-754 bit pattern of a float32 (the stored-vector
// encoding VectorSearch reads back).
func mathBits(f float32) uint32 { return math.Float32bits(f) }

// [vector-blend-degraded] the vector leg runs over the stored
// observations.embedding blobs with cosine distance. A query vector close to
// a stored vector ranks that row first; a query far from every row returns an
// empty (degraded) result. No embedder is needed here — the seam is the
// stored vector, and the leg degrades when there are no vectors at all.
func TestVectorSearchReturnsClosest(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()

	// Two rows with orthogonal-ish 2-D embeddings.
	a := addVecObs(t, st, "near", 0.99, 0.01)
	_ = addVecObs(t, st, "far", 0.01, 0.99)

	// A query close to a's vector must rank a first.
	q := []float32{0.98, 0.02}
	res, err := st.VectorSearch(ctx, VecQuery{QueryVector: q, Project: "proj-a", TopK: 2})
	if err != nil {
		t.Fatalf("vector search: %v", err)
	}
	if res.Degraded {
		t.Fatalf("vectors present → the leg must not degrade: %+v", res)
	}
	if len(res.Observations) < 1 {
		t.Fatalf("expected at least one vector result: %+v", res.Observations)
	}
	if res.Observations[0].ID != a {
		t.Fatalf("closest row must rank first, got %d (want %d)", res.Observations[0].ID, a)
	}
}

// [vector-blend-degraded] an empty store (no vectors) is an absent leg: the
// result is degraded, never an error (INV-3 — explicit degraded, never a
// silent fusion).
func TestVectorSearchDegradedWhenNoVectors(t *testing.T) {
	st := openTestStore(t)
	ctx := context.Background()
	res, err := st.VectorSearch(ctx, VecQuery{QueryVector: []float32{1, 0}, Project: "proj-a", TopK: 5})
	if err != nil {
		t.Fatalf("vector search (no vectors) must not error: %v", err)
	}
	if !res.Degraded {
		t.Fatalf("no vectors → the leg must degrade: %+v", res)
	}
}

// addVecObs inserts an observation with a 2-D embedding blob and returns its
// local id.
func addVecObs(t *testing.T, st Store, title string, v ...float32) int64 {
	t.Helper()
	blob := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(blob[i*4:], mathBits(f))
	}
	raw := st.(interface {
		RawExec(query string, args ...any) error
	})
	// AddObservation has no embedding field; reach the row through RawExec
	// (the same write seam the memory package uses) to set the vector.
	id, err := addObs(t, st, title)
	if err != nil {
		t.Fatalf("add obs: %v", err)
	}
	if err := raw.RawExec(`UPDATE observations SET embedding = ? WHERE id = ?`, blob, id); err != nil {
		t.Fatalf("set embedding: %v", err)
	}
	return id
}

// addObs inserts a single observation and returns its id (0 on error).
func addObs(t *testing.T, st Store, title string) (int64, error) {
	t.Helper()
	res, err := st.AddObservation(context.Background(), AddObservationParams{
		SessionID: "s", Type: "discovery", Title: title, Content: "content " + title,
		Project: "proj-a", Scope: "project",
	})
	if err != nil {
		return 0, err
	}
	return res.ID, nil
}
