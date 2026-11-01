// Bench harness skeleton (TICKET-01): synthetic recall/latency baseline and
// observations caller audit. Door check for the HNSW YAGNI call (briefing R3,
// SATISFIES hnsw-recall-latency RED: this file was absent).
//
// Simplest rung (YAGNI): seeded 768-dim (Dim) gaussian generator + in-process
// brute-force cosine top-K timer. No DB, no BLOB leg, no ANN — the BLOB-scan
// vs cosine split and IVF/HNSW legs land with their own tickets. Stdlib only.
package vectorstore

import (
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// benchSeed pins the synthetic corpus so cold/warm numbers are comparable
// across runs and machines.
const benchSeed = 20261010

// synthCorpus builds n Dim-unit vectors (flat n*Dim) from gaussian noise.
// Deterministic under seed.
func synthCorpus(rng *rand.Rand, n int) []float32 {
	flat := make([]float32, n*Dim)
	for i := 0; i < n; i++ {
		var norm float64
		base := i * Dim
		for d := 0; d < Dim; d++ {
			v := rng.NormFloat64()
			flat[base+d] = float32(v)
			norm += v * v
		}
		norm = math.Sqrt(norm)
		for d := 0; d < Dim; d++ {
			flat[base+d] /= float32(norm)
		}
	}
	return flat
}

// rowCopy returns a copy of corpus row i as the query vector.
func rowCopy(flat []float32, i int) []float32 {
	q := make([]float32, Dim)
	copy(q, flat[i*Dim:(i+1)*Dim])
	return q
}

// cosineFlat scores corpus row base against q (both unit length, so the dot
// product is the cosine similarity).
func cosineFlat(flat []float32, base int, q []float32) float64 {
	var s float64
	for d := 0; d < Dim; d++ {
		s += float64(flat[base+d]) * float64(q[d])
	}
	return s
}

// bruteTopK returns the ids (row numbers) of the top-k rows by cosine
// similarity, most-similar first. Ties break by lower id (deterministic).
func bruteTopK(flat []float32, n int, q []float32, k int) []int {
	type scored struct {
		id  int
		sim float64
	}
	all := make([]scored, n)
	for i := 0; i < n; i++ {
		all[i] = scored{id: i, sim: cosineFlat(flat, i*Dim, q)}
	}
	sort.Slice(all, func(a, b int) bool {
		if all[a].sim != all[b].sim {
			return all[a].sim > all[b].sim
		}
		return all[a].id < all[b].id
	})
	if k > n {
		k = n
	}
	top := make([]int, k)
	for i := 0; i < k; i++ {
		top[i] = all[i].id
	}
	return top
}

// timeTopK runs one brute-force top-K and returns its wall time.
func timeTopK(flat []float32, n int, q []float32, k int) time.Duration {
	start := time.Now()
	_ = bruteTopK(flat, n, q, k)
	return time.Since(start)
}

// TestBenchHarnessExists is the door-check test: the recall harness exists and
// a corpus member query recalls itself top-1 (brute vs brute recall = 1.0).
// Triangulated over two member rows (mid + last).
func TestBenchHarnessExists(t *testing.T) {
	rng := rand.New(rand.NewSource(benchSeed))
	const n = 2000
	flat := synthCorpus(rng, n)
	for _, want := range []int{n / 2, n - 1} {
		got := bruteTopK(flat, n, rowCopy(flat, want), 10)
		if len(got) == 0 || got[0] != want {
			t.Fatalf("self-recall row %d: top-1 = %v, want [%d]", want, got, want)
		}
	}
	t.Logf("bench harness present: Dim=%d self-recall top-1 = 1.0 at n=%d (2 inputs)", Dim, n)
}

// baseline runs the cold query plus three warm queries and logs the numbers.
// It returns cold and mean-warm durations.
func baseline(t *testing.T, n, k int) (cold, warm time.Duration) {
	t.Helper()
	rng := rand.New(rand.NewSource(benchSeed))
	flat := synthCorpus(rng, n)
	q := rowCopy(flat, n/2)
	if got := bruteTopK(flat, n, q, k); got[0] != n/2 {
		t.Fatalf("n=%d: self-recall top-1 = %d, want %d", n, got[0], n/2)
	}
	cold = timeTopK(flat, n, q, k)
	var total time.Duration
	for i := 0; i < 3; i++ {
		total += timeTopK(flat, n, q, k)
	}
	warm = total / 3
	t.Logf("baseline n=%d dim=%d topK=%d cold=%s warm=%s (in-memory cosine scan; BLOB leg n/a in skeleton)",
		n, Dim, k, cold, warm)
	return cold, warm
}

// TestRecallLatencyBaseline prints the cold/warm baseline at 20k and 100k and
// records the IVF-only door decision when 100k warm is under 50ms
// (briefing R3: else IVF-only ships and HNSW stays out).
func TestRecallLatencyBaseline(t *testing.T) {
	if testing.Short() {
		t.Skip("baseline needs full corpus; run without -short")
	}
	baseline(t, 20000, 10)
	_, warm100k := baseline(t, 100000, 10)
	if warm100k < 50*time.Millisecond {
		t.Logf("DECISION: IVF-only (100k warm %s < 50ms; HNSW stays out)", warm100k)
	} else {
		t.Logf("DECISION: PENDING (100k warm %s >= 50ms; HNSW candidate, needs ADR-0037)", warm100k)
	}
}

// callerAuditCandidates lists where observations_callers.txt may live:
// repo-relative from this package dir, plus the absolute checkout path.
func callerAuditCandidates() []string {
	return []string{
		filepath.Join("..", "..", "..", ".skillgrid", "specs", "2026-10-10-obs-split-hnsw", "observations_callers.txt"),
		filepath.Join("/data", "git", "AI", "skillgrid", ".skillgrid", "specs", "2026-10-10-obs-split-hnsw", "observations_callers.txt"),
	}
}

// TestObservationsCallerAudit pins the checked-in FROM-observations caller
// list: the file must exist and name 7+ read sites (briefing R2: 7 read-site
// filters preserved across the split).
func TestObservationsCallerAudit(t *testing.T) {
	var raw []byte
	var hit string
	for _, p := range callerAuditCandidates() {
		if b, err := os.ReadFile(p); err == nil {
			raw, hit = b, p
			break
		}
	}
	if raw == nil {
		t.Fatalf("observations_callers.txt absent (tried %s)", strings.Join(callerAuditCandidates(), ", "))
	}
	count := 0
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if strings.Contains(line, "FROM observations") {
			count++
		}
	}
	t.Logf("caller audit %s: %d FROM-observations sites", hit, count)
	if count < 7 {
		t.Fatalf("caller audit lists %d sites, want 7+", count)
	}
}

// BenchmarkBruteForceTopK keeps `go test -bench` meaningful on the skeleton:
// one top-10 over a small fixed corpus (setup outside the timer).
func BenchmarkBruteForceTopK(b *testing.B) {
	rng := rand.New(rand.NewSource(benchSeed))
	const n = 2000
	flat := synthCorpus(rng, n)
	q := rowCopy(flat, n/2)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bruteTopK(flat, n, q, 10)
	}
}
