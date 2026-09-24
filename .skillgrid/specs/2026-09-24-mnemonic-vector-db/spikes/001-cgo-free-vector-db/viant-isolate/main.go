package main

import (
	"fmt"
	"os"

	_ "modernc.org/sqlite"

	"github.com/viant/sqlite-vec/engine"
	"github.com/viant/sqlite-vec/vec"
)

func main() {
	f, _ := os.CreateTemp("", "vtab-isolate-*.sqlite")
	path := f.Name()
	f.Close()
	defer os.Remove(path)

	db, _ := engine.Open(path)
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := vec.Register(db); err != nil {
		fmt.Println("Register err:", err)
		os.Exit(1)
	}
	fmt.Println("Register OK")

	// step 1: vector_storage first
	_, e1 := db.Exec(`CREATE TABLE IF NOT EXISTS vector_storage (
		shadow_table_name TEXT NOT NULL, dataset_id TEXT NOT NULL DEFAULT '',
		"index" BLOB, PRIMARY KEY (shadow_table_name, dataset_id))`)
	fmt.Printf("vector_storage: %v\n", e1)

	// step 2: shadow table
	_, e2 := db.Exec(`CREATE TABLE IF NOT EXISTS _vec_vec_docs (
		dataset_id TEXT NOT NULL, id TEXT NOT NULL, content TEXT, meta TEXT,
		embedding BLOB, PRIMARY KEY(dataset_id, id))`)
	fmt.Printf("shadow: %v\n", e2)

	// step 3: virtual table
	_, e3 := db.Exec(`CREATE VIRTUAL TABLE vec_docs USING vec(doc_id)`)
	fmt.Printf("CREATE VIRTUAL TABLE: %v\n", e3)
}
