package main

// viant query: opens the pre-built DB, runs MATCH (cover-tree ANN), measures
// latency and recall@K vs exact Go cosine.

import (
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
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

func itoa(i int) string { return fmt.Sprintf("%d", i) }

func fatal(format string, args ...any) {
	fmt.Printf("FATAL: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	dbPath := os.Args[1]
	fmt.Printf("=== viant query: %s ===\n", dbPath)

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

	// the query uses the SAME vector that the build used for its rng seed —
	// but we generate a fresh query vector here (seed 42, first vector)
	rng := rand.New(rand.NewSource(42))
	q := randVector(rng)
	qe, _ := vector.EncodeEmbedding(q)

	const k = 10
	const runs = 10
	latencies := make([]time.Duration, runs)
	var annIDs []string
	for r := 0; r < runs; r++ {
		t0 := time.Now()
		rows, err := db.Query(`SELECT doc_id, match_score FROM vec_docs
			WHERE dataset_id = 'demo' AND doc_id MATCH ? LIMIT `+itoa(k), qe)
		if err != nil {
			fatal("query: %v", err)
		}
		ids := make([]string, 0, k)
		for rows.Next() {
			var id string
			var score float64
			rows.Scan(&id, &score)
			ids = append(ids, id)
		}
		rows.Close()
		latencies[r] = time.Since(t0)
		if r == 0 {
			annIDs = ids
		}
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	fmt.Printf("viant top-K=%d query: median=%v min=%v max=%v (results=%d)\n",
		k, latencies[runs/2], latencies[0], latencies[runs-1], len(annIDs))
	fmt.Printf("viant ANN top-3: %v\n", annIDs[:min(3, len(annIDs))])

	// recall@K vs exact
	n := 100000
	fmt.Printf("computing exact top-%d (Go cosine over %d vectors)...\n", k, n)
	exactStart := time.Now()
	exactIDs := exactTopK(db, q, n, k)
	fmt.Printf("  exact scan: %v\n", time.Since(exactStart))
	fmt.Printf("exact top-3: %v\n", exactIDs[:min(3, len(exactIDs))])

	hit := 0
	exactSet := map[string]bool{}
	for _, id := range exactIDs {
		exactSet[id] = true
	}
	for _, id := range annIDs {
		if exactSet[id] {
			hit++
		}
	}
	if len(annIDs) > 0 {
		recall := float64(hit) / float64(k)
		fmt.Printf("RECALL@%d = %d/%d = %.2f\n", k, hit, k, recall)
	} else {
		fmt.Printf("RECALL@%d = UNDEFINED (ANN returned 0 results)\n", k)
	}
}

func exactTopK(db *sql.DB, q []float32, subset, k int) []string {
	rows, err := db.Query(`SELECT id, embedding FROM _vec_vec_docs WHERE dataset_id='demo' ORDER BY CAST(id AS INTEGER) LIMIT ` + itoa(subset))
	if err != nil {
		fatal("exact scan: %v", err)
	}
	defer rows.Close()
	type sv struct {
		id  string
		sim float64
	}
	var all []sv
	for rows.Next() {
		var id string
		var blob []byte
		if err := rows.Scan(&id, &blob); err != nil {
			fatal("scan blob: %v", err)
		}
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
	out := make([]string, 0, k)
	for i := 0; i < min(k, len(all)); i++ {
		out = append(out, all[i].id)
	}
	return out
}
