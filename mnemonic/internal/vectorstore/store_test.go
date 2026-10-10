package vectorstore

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"

	_ "modernc.org/sqlite"
	_ "modernc.org/sqlite/vec"
)

// mustOpen opens a fresh store (which runs the migrations, including 042's
// vec0 tables) and registers cleanup.
func mustOpen(t *testing.T) *sql.DB {
	t.Helper()
	s, err := store.Open(t.TempDir(), "proj-vstore")
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s.DB
}

// makeVec builds a 768-dim unit vector with the hot-th component = 1
// (orthogonal basis), so cosine similarity is exactly 1 for a matching vector
// and 0 for a different basis vector. Deterministic, no float noise.
func makeVec(t *testing.T, hot int) []float32 {
	t.Helper()
	v := make([]float32, 768)
	v[hot] = 1
	return v
}

// encodeFloat32s is a test-only little-endian float32 encoder (same layout as
// memory.EncodeVector). Kept here so the test does not import the memory
// package just to encode a vector.
func encodeFloat32s(v []float32) []byte {
	buf := make([]byte, len(v)*4)
	for i, f := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

func TestTableExists(t *testing.T) {
	db := mustOpen(t)
	ctx := context.Background()
	for _, table := range []string{"vec_symbols", "vec_chunks"} {
		ok, err := TableExists(ctx, db, table)
		if err != nil {
			t.Fatalf("TableExists(%s): %v", table, err)
		}
		if !ok {
			t.Fatalf("TableExists(%s)=false, want true (migration 042)", table)
		}
	}
	ok, err := TableExists(ctx, db, "vec_nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("TableExists(vec_nonexistent)=true, want false")
	}
}

func TestUpsertAndSearchSymbols(t *testing.T) {
	db := mustOpen(t)
	ctx := context.Background()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	// Insert 3 orthogonal basis vectors at symbol_id 1, 2, 3 (hot = 0, 1, 2).
	for hot, id := range []int64{1, 2, 3} {
		if err := UpsertSymbol(ctx, tx, id, "test-model", 768, encodeFloat32s(makeVec(t, hot))); err != nil {
			t.Fatalf("UpsertSymbol(%d): %v", id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	// Query for basis vector hot=1: top-1 must be symbol_id 2.
	ids, err := SearchSymbols(ctx, db, makeVec(t, 1), 1)
	if err != nil {
		t.Fatalf("SearchSymbols: %v", err)
	}
	if len(ids) != 1 || ids[0].ID != 2 {
		t.Fatalf("SearchSymbols top-1 = %v, want [2]", ids)
	}
	// Query top-3 for basis hot=0: all three, with symbol_id 1 (hot=0) first.
	ids3, err := SearchSymbols(ctx, db, makeVec(t, 0), 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids3) != 3 || ids3[0].ID != 1 {
		t.Fatalf("SearchSymbols top-3 = %v, want id 1 first", ids3)
	}
	// The in-SQL cosine similarity of a query with itself must be ~1.
	if ids3[0].Sim < 0.999 {
		t.Fatalf("SearchSymbols self-similarity = %f, want ~1", ids3[0].Sim)
	}
}

func TestUpsertAndSearchChunks(t *testing.T) {
	db := mustOpen(t)
	ctx := context.Background()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	for hot, id := range []int64{10, 20} {
		if err := UpsertChunk(ctx, tx, id, "m", 768, encodeFloat32s(makeVec(t, hot))); err != nil {
			t.Fatalf("UpsertChunk(%d): %v", id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	ids, err := SearchChunks(ctx, db, makeVec(t, 1), 1)
	if err != nil {
		t.Fatalf("SearchChunks: %v", err)
	}
	if len(ids) != 1 || ids[0].ID != 20 {
		t.Fatalf("SearchChunks top-1 = %v, want [20]", ids)
	}
}

func TestDeleteSymbolsAndChunks(t *testing.T) {
	db := mustOpen(t)
	ctx := context.Background()
	tx, _ := db.Begin()
	for i, id := range []int64{1, 2, 3} {
		_ = UpsertSymbol(ctx, tx, id, "m", 768, encodeFloat32s(makeVec(t, i)))
		_ = UpsertChunk(ctx, tx, id, "m", 768, encodeFloat32s(makeVec(t, i)))
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	nS, err := Count(ctx, db, "vec_symbols")
	if err != nil || nS != 3 {
		t.Fatalf("Count(vec_symbols) before delete = %d (err %v), want 3", nS, err)
	}
	nC, err := Count(ctx, db, "vec_chunks")
	if err != nil || nC != 3 {
		t.Fatalf("Count(vec_chunks) before delete = %d (err %v), want 3", nC, err)
	}
	tx2, _ := db.Begin()
	if err := DeleteSymbols(ctx, tx2); err != nil {
		t.Fatal(err)
	}
	if err := DeleteChunks(ctx, tx2); err != nil {
		t.Fatal(err)
	}
	if err := tx2.Commit(); err != nil {
		t.Fatal(err)
	}
	nS2, err := Count(ctx, db, "vec_symbols")
	if err != nil || nS2 != 0 {
		t.Fatalf("Count(vec_symbols) after delete = %d (err %v), want 0", nS2, err)
	}
	nC2, err := Count(ctx, db, "vec_chunks")
	if err != nil || nC2 != 0 {
		t.Fatalf("Count(vec_chunks) after delete = %d (err %v), want 0", nC2, err)
	}
}

// TestMaxOpenConnsOne is the prototype's viant-deadlock guard: the vec tables must
// register and query under the store's hard SetMaxOpenConns(1) invariant. The
// prototype showed viant deadlocks exactly here (ensureIndex re-enters db.Exec from
// inside the vtab Filter callback). G has no such re-entrancy, but this test
// pins it.
func TestMaxOpenConnsOne(t *testing.T) {
	db := mustOpen(t)
	if db.Stats().MaxOpenConnections != 1 {
		t.Fatalf("MaxOpenConns = %d, want 1 (store invariant)", db.Stats().MaxOpenConnections)
	}
	ctx := context.Background()
	tx, _ := db.Begin()
	if err := UpsertSymbol(ctx, tx, 1, "m", 768, encodeFloat32s(makeVec(t, 0))); err != nil {
		t.Fatalf("UpsertSymbol under MaxOpenConns=1: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	// A query that would deadlock if the vtab re-enters db.Exec.
	ids, err := SearchSymbols(ctx, db, makeVec(t, 0), 1)
	if err != nil {
		t.Fatalf("SearchSymbols under MaxOpenConns=1 (deadlock?): %v", err)
	}
	if len(ids) != 1 || ids[0].ID != 1 {
		t.Fatalf("SearchSymbols = %v, want [1]", ids)
	}
}

// ---- IVF (TICKET-04, SATISFIES ivf-recall) ----

// ivfSeedFile is the files+symbols row plumbing needed so an embeddings row's
// symbol_id FK is satisfied. It mirrors seedSymbol in exact_test.go but lives
// here so the IVF tests are self-contained.
func ivfSeedSymbol(t *testing.T, tx *sql.Tx, id int64) {
	t.Helper()
	if _, err := tx.Exec(`INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES (?,?,?,?,?)`,
		fmt.Sprintf("ivf-%d.go", id), int64(id), int64(10),
		fmt.Sprintf("ivf-hash-%d", id), "2026-01-01T00:00:00Z"); err != nil {
		t.Fatalf("ivf seed file %d: %v", id, err)
	}
	var fileID int64
	if err := tx.QueryRow(`SELECT last_insert_rowid()`).Scan(&fileID); err != nil {
		t.Fatalf("ivf last_insert_rowid: %v", err)
	}
	if _, err := tx.Exec(`INSERT INTO symbols (id, file_id, name, kind, start_line, end_line, content_hash, uid)
		VALUES (?,?,?,?,?,?,?,?)`,
		id, fileID, fmt.Sprintf("ivfsym%d", id), "func", 1, 1,
		fmt.Sprintf("ivf-hash-%d", id), fmt.Sprintf("ivf-uid-%d", id)); err != nil {
		t.Fatalf("ivf seed symbol %d: %v", id, err)
	}
}

// writeEmbeddingBlob writes a vector into the embeddings BLOB table (the
// IVF's source of truth) for symbol id on the given tx. It writes on the tx
// (NOT the shared *sql.DB) so it shares the transaction's connection — under
// the store's SetMaxOpenConns(1) invariant, a second connection from the
// shared DB would deadlock against the open tx.
func writeEmbeddingBlob(t *testing.T, tx *sql.Tx, id int64, vb []byte) {
	t.Helper()
	if _, err := tx.Exec(
		`INSERT OR REPLACE INTO embeddings (symbol_id, model, dim, vector, updated_at, language)
		 VALUES (?,?,?,?,?,?)`,
		id, "ivf-model", len(vb)/4, vb, "2026-01-01T00:00:00Z", ""); err != nil {
		t.Fatalf("write embedding blob %d: %v", id, err)
	}
}

// TestIVFRecallVsBrute is the recall@10 gate (SATISFIES ivf-recall): the IVF
// probe-then-rerank must recall the brute-force top-K at >= 0.90 on 20k
// synthetic 768-dim vectors. The corpus is seeded (deterministic) so the
// number is comparable across runs.
func TestIVFRecallVsBrute(t *testing.T) {
	const n = 20000
	const k = 10
	db := mustOpen(t)
	ctx := context.Background()

	// Build the corpus: n unit gaussian vectors (seeded).
	rng := rand.New(rand.NewSource(20261010))
	corpus := make([][]float32, n)
	for i := 0; i < n; i++ {
		var norm float64
		v := make([]float32, Dim)
		for d := 0; d < Dim; d++ {
			x := rng.NormFloat64()
			v[d] = float32(x)
			norm += x * x
		}
		norm = math.Sqrt(norm)
		for d := 0; d < Dim; d++ {
			v[d] /= float32(norm)
		}
		corpus[i] = v
	}

	// Write the corpus to the embeddings BLOB table (IVF source of truth).
	// Symbol ids are 1..n (files + symbols plumbing per id).
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < n; i++ {
		ivfSeedSymbol(t, tx, int64(i+1))
		writeEmbeddingBlob(t, tx, int64(i+1), encodeFloat32s(corpus[i]))
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	// Brute-force reference (the exact top-K over the BLOB table).
	bf := NewBruteForce(db, "embeddings", "symbol_id")
	// IVF index (probe nprobe=64 over the built centroids — for high-dim
	// gaussian data the true top-K straddles many clusters, so a larger
	// nprobe is needed to reach the recall gate).
	ivf := NewIVF(db, "embeddings", "symbol_id", 64)
	if err := ivf.Build(ctx, 0); err != nil {
		t.Fatalf("IVF Build: %v", err)
	}

	// Measure recall@10 over a sample of queries (deterministic rows).
	const qCount = 100
	hits := 0
	for qi := 0; qi < qCount; qi++ {
		qIdx := (qi * n) / qCount
		q := corpus[qIdx]
		want, err := bf.Search(ctx, q, k)
		if err != nil {
			t.Fatalf("brute search %d: %v", qi, err)
		}
		got, err := ivf.Search(ctx, q, k)
		if err != nil {
			t.Fatalf("ivf search %d: %v", qi, err)
		}
		wantSet := make(map[int64]struct{}, len(want))
		for _, w := range want {
			wantSet[w.ID] = struct{}{}
		}
		for _, g := range got {
			if _, ok := wantSet[g.ID]; ok {
				hits++
			}
		}
	}
	recall := float64(hits) / float64(qCount*k)
	t.Logf("IVF recall@%d over %d queries = %f (n=%d, k=64, nprobe=64)", k, qCount, recall, n)
	if recall < 0.90 {
		t.Fatalf("IVF recall@%d = %f, want >= 0.90", k, recall)
	}
}

// TestIVFBruteFallbackOnCorrupt asserts that a corrupt / missing IVF table
// degrades to the brute-force fallback without error (SATISFIES ivf-recall:
// "a corrupt IVF table degrades to brute force without error").
func TestIVFBruteFallbackOnCorrupt(t *testing.T) {
	db := mustOpen(t)
	ctx := context.Background()
	// Write 3 orthogonal vectors to the BLOB table.
	tx, _ := db.Begin()
	for hot, id := range []int64{1, 2, 3} {
		ivfSeedSymbol(t, tx, id)
		writeEmbeddingBlob(t, tx, id, encodeFloat32s(makeVec(t, hot)))
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	// Corrupt the IVF state: drop the assignments table entirely (the probe
	// query will fail with "no such table").
	if _, err := db.Exec(`DROP TABLE IF EXISTS ivf_assignments`); err != nil {
		t.Fatal(err)
	}
	ivf := NewIVF(db, "embeddings", "symbol_id", 4)
	// Query basis-1: the fallback must return the brute-force top-1 = id 2.
	hits, err := ivf.Search(ctx, makeVec(t, 1), 1)
	if err != nil {
		t.Fatalf("IVF Search after corrupt (want fallback, no error): %v", err)
	}
	if len(hits) != 1 || hits[0].ID != 2 {
		t.Fatalf("IVF fallback top-1 = %v, want [2]", hits)
	}
}

// TestIVFBuildThenSearchRoundTrip asserts the basic round-trip: build an IVF
// index over a small corpus, then probe-search for a basis vector and verify
// the top-1 is the matching vector. This pins the probe-then-rerank path on
// a deterministic (non-gaussian) corpus so the recall test above is the
// statistical gate, not the correctness gate.
func TestIVFBuildThenSearchRoundTrip(t *testing.T) {
	db := mustOpen(t)
	ctx := context.Background()
	tx, _ := db.Begin()
	// 8 orthogonal basis vectors at ids 1..8.
	for hot, id := range []int64{1, 2, 3, 4, 5, 6, 7, 8} {
		ivfSeedSymbol(t, tx, id)
		writeEmbeddingBlob(t, tx, id, encodeFloat32s(makeVec(t, hot)))
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	ivf := NewIVF(db, "embeddings", "symbol_id", 4)
	if err := ivf.Build(ctx, 2); err != nil {
		t.Fatalf("IVF Build: %v", err)
	}
	// Query basis-3: top-1 must be id 4 (the vector with hot=3).
	hits, err := ivf.Search(ctx, makeVec(t, 3), 1)
	if err != nil {
		t.Fatalf("IVF Search: %v", err)
	}
	if len(hits) != 1 || hits[0].ID != 4 {
		t.Fatalf("IVF round-trip top-1 = %v, want [4]", hits)
	}
}

// TestIVFUpsertAssignsCluster asserts that an Upsert after Build assigns the
// new vector to a cluster (an ivf_assignments row appears).
func TestIVFUpsertAssignsCluster(t *testing.T) {
	db := mustOpen(t)
	ctx := context.Background()
	tx, _ := db.Begin()
	ivfSeedSymbol(t, tx, 1)
	writeEmbeddingBlob(t, tx, 1, encodeFloat32s(makeVec(t, 0)))
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	ivf := NewIVF(db, "embeddings", "symbol_id", 4)
	if err := ivf.Build(ctx, 1); err != nil {
		t.Fatalf("IVF Build: %v", err)
	}
	// Seed the parent symbols row for id=2 (embeddings.symbol_id has an FK to
	// symbols(id); the upsert's BLOB write requires the parent row to exist).
	tx2, _ := db.Begin()
	ivfSeedSymbol(t, tx2, 2)
	if err := tx2.Commit(); err != nil {
		t.Fatal(err)
	}
	// Upsert a new vector (basis-1) at id 2. It must land in the BLOB table
	// and get an assignment row.
	if err := ivf.Upsert(ctx, 2, makeVec(t, 1)); err != nil {
		t.Fatalf("IVF Upsert: %v", err)
	}
	var assignCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ivf_assignments WHERE vector_id = 2`).
		Scan(&assignCount); err != nil {
		t.Fatal(err)
	}
	if assignCount != 1 {
		t.Fatalf("ivf_assignments rows for id 2 = %d, want 1", assignCount)
	}
	// And a search for basis-1 must find id 2.
	hits, err := ivf.Search(ctx, makeVec(t, 1), 1)
	if err != nil {
		t.Fatalf("IVF Search after upsert: %v", err)
	}
	if len(hits) != 1 || hits[0].ID != 2 {
		t.Fatalf("IVF search after upsert top-1 = %v, want [2]", hits)
	}
}
