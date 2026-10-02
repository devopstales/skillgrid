package vectorstore

import (
	"context"
	"database/sql"
	"encoding/binary"
	"math"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"

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
