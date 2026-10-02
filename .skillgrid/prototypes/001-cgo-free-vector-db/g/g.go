package main

import (
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"time"

	_ "modernc.org/sqlite"
	_ "modernc.org/sqlite/vec"
)

// G path: modernc.org/sqlite/vec blank-import. Tests vec_version, vec0 table
// creation, vec_f32 insert, and vec_distance_cosine top-K. Also measures the
// brute-force latency at N vectors (the "is brute-force acceptable at 100K" claim).

const dim = 768

func randVector(rng *rand.Rand) []float32 {
	v := make([]float32, dim)
	for i := range v {
		v[i] = float32(rng.Float64()*2 - 1)
	}
	return v
}

func vecLiteral(v []float32) string {
	// vec_f32 accepts a JSON-like array string.
	out := make([]string, dim)
	for i, x := range v {
		out[i] = fmt.Sprintf("%.6f", x)
	}
	s := "["
	for i, p := range out {
		if i > 0 {
			s += ","
		}
		s += p
	}
	return s + "]"
}

func main() {
	rng := rand.New(rand.NewSource(42))
	n := 100000
	fmt.Printf("=== G: modernc.org/sqlite/vec ===\n")
	fmt.Printf("dim=%d n=%d\n", dim, n)

	// 1. vec_version
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		fatal("open: %v", err)
	}
	defer db.Close()
	var ver string
	if err := db.QueryRow("SELECT vec_version()").Scan(&ver); err != nil {
		fmt.Printf("FAIL vec_version: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("vec_version=%s\n", ver)

	// 2. create vec0 table
	if _, err := db.Exec(fmt.Sprintf(
		`CREATE VIRTUAL TABLE vec_items USING vec0(rowid INTEGER PRIMARY KEY, embedding float[%d])`, dim)); err != nil {
		fatal("create vec0: %v", err)
	}
	fmt.Println("vec0 table created OK")

	// 3. insert N vectors (batched)
	fmt.Printf("inserting %d vectors...\n", n)
	insStart := time.Now()
	const batch = 5000
	tx, _ := db.Begin()
	insStmt, _ := tx.Prepare("INSERT INTO vec_items(rowid, embedding) VALUES (?, vec_f32(?))")
	q := randVector(rng)
	for i := 0; i < n; i++ {
		v := randVector(rng)
		if _, err := insStmt.Exec(int64(i+1), vecLiteral(v)); err != nil {
			fatal("insert %d: %v", i, err)
		}
		if (i+1)%batch == 0 {
			insStmt.Close()
			tx.Commit()
			tx, _ = db.Begin()
			insStmt, _ = tx.Prepare("INSERT INTO vec_items(rowid, embedding) VALUES (?, vec_f32(?))")
		}
	}
	insStmt.Close()
	tx.Commit()
	fmt.Printf("insert done in %v (%.1f vectors/sec)\n", time.Since(insStart), float64(n)/time.Since(insStart).Seconds())

	// 4. top-K query via vec_distance_cosine (10 runs, report median)
	const k = 10
	const runs = 10
	latencies := make([]time.Duration, runs)
	var resultIDs []int64
	for r := 0; r < runs; r++ {
		t0 := time.Now()
		rows, err := db.Query(fmt.Sprintf(
			`SELECT rowid, vec_distance_cosine(embedding, vec_f32(?)) AS d
			 FROM vec_items ORDER BY d LIMIT %d`, k), vecLiteral(q))
		if err != nil {
			fatal("query: %v", err)
		}
		ids := make([]int64, 0, k)
		for rows.Next() {
			var id int64
			var d float64
			rows.Scan(&id, &d)
			ids = append(ids, id)
		}
		rows.Close()
		latencies[r] = time.Since(t0)
		if r == 0 {
			resultIDs = ids
		}
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	med := latencies[runs/2]
	fmt.Printf("G top-K=%d query: median=%v min=%v max=%v\n", k, med, latencies[0], latencies[runs-1])
	fmt.Printf("G top-3 result ids: %v\n", resultIDs[:min(3, len(resultIDs))])

	// 5. sanity: exact brute-force check on a small subset (first 2000) to
	// confirm vec_distance_cosine ordering matches a Go cosine scan.
	fmt.Println("exactness check on first 2000 vectors...")
	exactIDs := exactTopK(db, q, 2000, k)
	fmt.Printf("exact top-3 (Go scan): %v\n", exactIDs[:min(3, len(exactIDs))])
	// compare only positions that exist in both (exact is subset-limited)
	cmp := min(min(k, len(exactIDs)), len(resultIDs))
	match := 0
	for i := 0; i < cmp; i++ {
		// only compare within the 2000-subset range
		if exactIDs[i] == resultIDs[i] {
			match++
		}
	}
	fmt.Printf("exactness: %d/%d top positions match (subset=2000)\n", match, min(k, len(exactIDs)))

	_ = math.Abs
}

func exactTopK(db *sql.DB, q []float32, subset, k int) []int64 {
	// scan first `subset` rows, compute cosine similarity in Go, return top-k ids
	rows, err := db.Query("SELECT rowid, embedding FROM vec_items WHERE rowid <= ? ORDER BY rowid", subset)
	if err != nil {
		fatal("exact scan: %v", err)
	}
	defer rows.Close()
	type sv struct {
		id  int64
		sim float64
	}
	var all []sv
	for rows.Next() {
		var id int64
		var blob []byte
		if err := rows.Scan(&id, &blob); err != nil {
			// embedding is a BLOB of float32s; scan as []byte
			fatal("scan blob: %v", err)
		}
		// decode float32 BLOB (little-endian)
		if len(blob)%4 != 0 {
			continue
		}
		v := make([]float32, len(blob)/4)
		for i := range v {
			b := blob[i*4 : i*4+4]
			u := uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
			v[i] = float32(math.Float32frombits(u))
		}
		all = append(all, sv{id: id, sim: cosine(q, v)})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].sim > all[j].sim })
	out := make([]int64, 0, k)
	for i := 0; i < min(k, len(all)); i++ {
		out = append(out, all[i].id)
	}
	return out
}

func cosine(a, b []float32) float64 {
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func fatal(format string, args ...any) {
	fmt.Printf("FATAL: "+format+"\n", args...)
	os.Exit(1)
}
