package vectorstore

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// writeBlobRow inserts a vector into the BLOB source-of-truth table (the
// ground truth for the exactness check) mirroring the vec table's rowid. It
// writes on the given *sql.Tx (NOT st.DB) so it shares the transaction's
// connection - under the store's SetMaxOpenConns(1) invariant, a second
// connection from st.DB would deadlock against the open tx.
func writeBlobRow(t *testing.T, tx *sql.Tx, table, idCol string, id int64, vb []byte) {
	t.Helper()
	if _, err := tx.Exec(
		`INSERT OR REPLACE INTO `+table+` (`+idCol+`, model, dim, vector, updated_at, language)
		 VALUES (?,?,?,?,?,?)`,
		id, "m", 768, vb, "2026-01-01T00:00:00Z", ""); err != nil {
		t.Fatalf("write %s row %d: %v", table, id, err)
	}
}

func mustOpenStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(t.TempDir(), "proj-exact")
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// seedSymbol inserts a minimal files + symbols row so embeddings.symbol_id's
// FK (REFERENCES symbols(id) ON DELETE CASCADE) is satisfied. The exactness
// check only reads the BLOB vector column; the symbol row is incidental FK
// plumbing.
func seedSymbol(t *testing.T, tx *sql.Tx, id int64) {
	t.Helper()
	// files.id is AUTOINCREMENT; insert one and capture its id, then a symbol
	// row that references it. We use an explicit file id via the AUTOINCREMENT
	// sequence by inserting and reading back the last insert rowid.
	if _, err := tx.Exec(`INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES (?,?,?,?,?)`,
		fmt.Sprintf("seed-%d.go", id), int64(id), int64(10),
		fmt.Sprintf("hash-%d", id), "2026-01-01T00:00:00Z"); err != nil {
		t.Fatalf("seed file %d: %v", id, err)
	}
	var fileID int64
	if err := tx.QueryRow(`SELECT last_insert_rowid()`).Scan(&fileID); err != nil {
		t.Fatalf("last_insert_rowid: %v", err)
	}
	// symbols.id is AUTOINCREMENT; insert with an explicit id so the FK from
	// embeddings (REFERENCES symbols(id)) is satisfied by the id we want.
	if _, err := tx.Exec(`INSERT INTO symbols (id, file_id, name, kind, start_line, end_line, content_hash, uid)
		VALUES (?,?,?,?,?,?,?,?)`,
		id, fileID, fmt.Sprintf("sym%d", id), "func", 1, 1,
		fmt.Sprintf("hash-%d", id), fmt.Sprintf("uid-%d", id)); err != nil {
		t.Fatalf("seed symbol %d: %v", id, err)
	}
}

// seedChunk inserts a minimal files + chunks row so chunk_embeddings.chunk_id's
// FK is satisfied (analogous to seedSymbol for the chunk path).
func seedChunk(t *testing.T, tx *sql.Tx, id int64) {
	t.Helper()
	if _, err := tx.Exec(`INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES (?,?,?,?,?)`,
		fmt.Sprintf("cseed-%d.go", id), int64(id), int64(10),
		fmt.Sprintf("chash-%d", id), "2026-01-01T00:00:00Z"); err != nil {
		t.Fatalf("seed file %d: %v", id, err)
	}
	var fileID int64
	if err := tx.QueryRow(`SELECT last_insert_rowid()`).Scan(&fileID); err != nil {
		t.Fatalf("last_insert_rowid: %v", err)
	}
	if _, err := tx.Exec(`INSERT INTO chunks (id, file_id, start_line, end_line, text, content_hash)
		VALUES (?,?,?,?,?,?)`,
		id, fileID, 1, 1, "func seed() {}",
		fmt.Sprintf("chash-%d", id)); err != nil {
		t.Fatalf("seed chunk %d: %v", id, err)
	}
}

// TestExactnessCheckAgreement writes the same 3 orthogonal vectors to BOTH the
// vec table and the BLOB table; the vec top-K and the brute-force scan must
// agree exactly.
func TestExactnessCheckAgreement(t *testing.T) {
	st := mustOpenStore(t)
	ctx := context.Background()
	tx, err := st.DB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	for hot, id := range []int64{1, 2, 3} {
		seedSymbol(t, tx, id)
		vb := encodeFloat32s(makeVec(t, hot))
		if err := UpsertSymbol(ctx, tx, id, "m", 768, vb); err != nil {
			t.Fatal(err)
		}
		writeBlobRow(t, tx, "embeddings", "symbol_id", id, vb)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	res, err := ExactnessCheck(ctx, st.DB, "vec_symbols", makeVec(t, 0), 3)
	if err != nil {
		t.Fatalf("ExactnessCheck: %v", err)
	}
	if !res.Agree {
		t.Fatalf("ExactnessCheck.Agree=false, want true (orthogonal basis is exact). Disagreements: %+v", res.Disagreements)
	}
	if res.Limit != 3 {
		t.Fatalf("res.Limit=%d, want 3", res.Limit)
	}
}

// TestExactnessCheckDisagreement writes basis-0 to the vec table at id 1 but
// basis-1 to the BLOB table at id 1 (deliberate drift); the top-1 sets diverge.
func TestExactnessCheckDisagreement(t *testing.T) {
	st := mustOpenStore(t)
	ctx := context.Background()
	tx, err := st.DB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	vb0 := encodeFloat32s(makeVec(t, 0))
	vb1 := encodeFloat32s(makeVec(t, 1))
	seedSymbol(t, tx, 1)
	// vec table: basis-0 at id 1.
	if err := UpsertSymbol(ctx, tx, 1, "m", 768, vb0); err != nil {
		t.Fatal(err)
	}
	// BLOB table: basis-1 at id 1 (deliberate drift from the vec table).
	writeBlobRow(t, tx, "embeddings", "symbol_id", 1, vb1)
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	// Query basis-1: the BLOB ground truth scores id 1 = 1.0 (basis-1), but the
	// vec table scores id 1 = 0.0 (basis-0). With only one row, both top-1s are
	// id 1 (the same id), so to force a true ID disagreement add a second row
	// where the two tables rank differently.
	// vec table: id 2 = basis-1 (scores 1.0 for the basis-1 query).
	vb1b := encodeFloat32s(makeVec(t, 1))
	tx2, _ := st.DB.Begin()
	seedSymbol(t, tx2, 2)
	if err := UpsertSymbol(ctx, tx2, 2, "m", 768, vb1b); err != nil {
		t.Fatal(err)
	}
	// BLOB table: id 2 = basis-2 (scores 0.0 for the basis-1 query).
	vb2 := encodeFloat32s(makeVec(t, 2))
	if _, err := tx2.Exec(
		`INSERT OR REPLACE INTO embeddings (symbol_id, model, dim, vector, updated_at, language)
		 VALUES (?,?,?,?,?,?)`, 2, "m", 768, vb2, "2026-01-01T00:00:00Z", ""); err != nil {
		t.Fatal(err)
	}
	if err := tx2.Commit(); err != nil {
		t.Fatal(err)
	}
	// Query basis-1, limit 1:
	//   vec table top-1 = id 2 (basis-1, sim 1.0); id 1 (basis-0, sim 0.0).
	//   BLOB  top-1     = id 1 (basis-1, sim 1.0); id 2 (basis-2, sim 0.0).
	// The top-1 IDs differ (2 vs 1) -> Agree=false.
	res, err := ExactnessCheck(ctx, st.DB, "vec_symbols", makeVec(t, 1), 1)
	if err != nil {
		t.Fatal(err)
	}
	if res.Agree {
		t.Fatal("ExactnessCheck.Agree=true, want false (vec and BLOB rank differently for the basis-1 query)")
	}
	if len(res.Disagreements) == 0 {
		t.Fatal("no disagreements reported, want >= 1")
	}
}
