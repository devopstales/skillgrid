package main

import (
	"fmt"
	"os"

	_ "modernc.org/sqlite"

	"github.com/viant/sqlite-vec/engine"
	"github.com/viant/sqlite-vec/vec"
)

func main() {
	f, _ := os.CreateTemp("", "vtab-check-*.sqlite")
	path := f.Name()
	f.Close()
	defer os.Remove(path)

	db, err := engine.Open(path)
	if err != nil {
		fmt.Println("open err:", err)
		os.Exit(1)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := vec.Register(db); err != nil {
		fmt.Println("Register err:", err)
		os.Exit(1)
	}
	fmt.Println("Register OK")
	_, err = db.Exec(`CREATE VIRTUAL TABLE test_vec USING vec(doc_id)`)
	fmt.Printf("CREATE VIRTUAL TABLE: err=%v\n", err)
	if err == nil {
		var cnt int
		db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='test_vec'`).Scan(&cnt)
		fmt.Println("vtab exists:", cnt)
	}
	rows, _ := db.Query(`PRAGMA sqlite_module_list`)
	if rows != nil {
		for rows.Next() {
			var name, version, author string
			rows.Scan(&name, &version, &author)
			fmt.Printf("  module: %s\n", name)
		}
		rows.Close()
	}
}
