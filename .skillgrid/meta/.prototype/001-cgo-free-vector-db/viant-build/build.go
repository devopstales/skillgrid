package main

// viant build: inserts vectors + builds the cover-tree index. The index
// persists in the vector_storage table, so a separate query process can use it.

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	_ "modernc.org/sqlite"

	"github.com/viant/sqlite-vec/engine"
	"github.com/viant/sqlite-vec/vec"
	"github.com/viant/sqlite-vec/vector"
)

const dim = 768

func randVector(rng *rand.Rand) []float32 {
	v := make([]float32, dim)
	for i := range v {
		v[i] = float32(rng.Float64()*2 - 1)
	}
	return v
}

func itoa(i int) string { return fmt.Sprintf("%d", i) }

func fatal(format string, args ...any) {
	fmt.Printf("FATAL: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	dbPath := os.Args[1]
	rng := rand.New(rand.NewSource(42))
	n := 100000
	fmt.Printf("=== viant build: %s ===\n", dbPath)

	db, err := engine.Open(dbPath)
	if err != nil {
		fatal("open: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := vec.Register(db); err != nil {
		fatal("vec.Register: %v", err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000;`); err != nil {
		fatal("pragma: %v", err)
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS _vec_vec_docs (
		dataset_id TEXT NOT NULL, id TEXT NOT NULL, content TEXT, meta TEXT,
		embedding BLOB, PRIMARY KEY(dataset_id, id))`); err != nil {
		fatal("shadow: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS vector_storage (
		shadow_table_name TEXT NOT NULL, dataset_id TEXT NOT NULL DEFAULT '',
		"index" BLOB, PRIMARY KEY (shadow_table_name, dataset_id))`); err != nil {
		fatal("vector_storage: %v", err)
	}
	if _, err := db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS vec_docs USING vec(doc_id)`); err != nil {
		fatal("create vec vtab: %v", err)
	}
	fmt.Println("schema OK (vec only, lazy index)")

	fmt.Printf("inserting %d vectors...\n", n)
	insStart := time.Now()
	const batch = 5000
	tx, _ := db.Begin()
	insStmt, _ := tx.Prepare(`INSERT INTO _vec_vec_docs(dataset_id, id, embedding) VALUES ('demo', ?, ?)`)
	for i := 0; i < n; i++ {
		v := randVector(rng)
		e, _ := vector.EncodeEmbedding(v)
		if _, err := insStmt.Exec(itoa(i+1), e); err != nil {
			fatal("insert %d: %v", i, err)
		}
		if (i+1)%batch == 0 {
			insStmt.Close()
			tx.Commit()
			tx, _ = db.Begin()
			insStmt, _ = tx.Prepare(`INSERT INTO _vec_vec_docs(dataset_id, id, embedding) VALUES ('demo', ?, ?)`)
		}
	}
	insStmt.Close()
	tx.Commit()
	fmt.Printf("insert done in %v (%.1f vectors/sec)\n", time.Since(insStart), float64(n)/time.Since(insStart).Seconds())
	fmt.Println("BUILD COMPLETE (index will build lazily on first query)")
}
