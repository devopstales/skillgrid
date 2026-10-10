package vectorstore

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"sync"
)

// IVF (inverted file) flat-code ANN (briefing Task 4: "kmeans-lite centroids
// build, assignment on upsert, probe-then-rerank search; corrupt-IVF falls
// back to brute"). No new dependency (cgo-free, ADR-0006/0009 invariants).
//
// Layout (migration 056_ivf_store.sql):
//
//	ivf_centroids (k, dim, centroid, vector)  -- one row per centroid
//	ivf_assignments (vector_id, cluster, dim) -- one row per vector
//
// The IVF index is shared by vec_symbols / vec_chunks: vector_id is a
// global int64 owned by the indexer (see the migration comment). This ticket
// keeps the index on the BLOB source-of-truth tables (the indexer's
// dual-write is a later ticket; see index.go BruteForce).
//
// Build: kmeans-lite — K means are seeded from a deterministic sample of
// the corpus, then iterated to convergence (or max iters). Assignment on
// upsert: the new vector's nearest centroid under Euclidean distance (the
// standard IVF partition criterion; cosine similarity is only used for
// reranking because it is the metric the rest of the system reports).
//
// Search: pick the nprobe nearest centroids (by cosine), union their
// member ids, brute-force score them, return top-K. Any SQL error during
// the probe (missing table, corrupt blob, schema drift) degrades to the
// brute-force fallback — never a hard fail (briefing "corrupt ANN degrades
// to brute force, never fails").
type IVFIndex struct {
	db      *sql.DB
	source  bruteSource // BLOB table the IVF mirrors
	centK   int         // number of centroids (built; -1 until Build)
	centDim int         // centroid dimension (built; -1 until Build)
	probes  int         // nprobe for search (0 => default)

	mu       sync.RWMutex
	centIDs  []int64     // centroid ids, parallel to centroids
	centVecs [][]float32 // centroid vectors (dim = centDim)
}

// NewIVF builds an IVF index over the named BLOB table. probes is the
// nprobe for Search (0 => default 16). The index is empty until Build or
// a successful lazy rebuild; Search degrades to brute force until then.
func NewIVF(db *sql.DB, table, idCol string, probes int) *IVFIndex {
	if probes <= 0 {
		probes = 16
	}
	return &IVFIndex{
		db:      db,
		source:  bruteSource{table: table, idCol: idCol},
		probes:  probes,
		centK:   -1,
		centDim: -1,
	}
}

// Name implements VectorIndex.
func (v *IVFIndex) Name() string { return "ivf" }

// Search implements VectorIndex: probe nprobe nearest centroids, union
// member ids, brute-force score, top-K. On any SQL/decode error it
// degrades to the brute-force fallback over the full corpus (no error).
func (v *IVFIndex) Search(ctx context.Context, q []float32, limit int) ([]SearchHit, error) {
	if limit <= 0 {
		limit = 10
	}
	if v == nil || v.db == nil {
		return nil, fmt.Errorf("ivf: no database")
	}

	// Ensure centroids are loaded (lazy build on first search).
	if err := v.ensureCentroids(ctx); err != nil {
		return v.bruteFallback(ctx, q, limit)
	}
	if v.centK <= 0 || len(v.centVecs) == 0 {
		// Empty corpus (no centroids) — brute force over an empty table is
		// correct (returns 0 hits).
		return v.bruteFallback(ctx, q, limit)
	}

	// 1. Pick the nprobe nearest centroids by cosine similarity.
	n := v.probes
	if n > v.centK {
		n = v.centK
	}
	ranked := make([]centScore, len(v.centVecs))
	for i, cv := range v.centVecs {
		ranked[i] = centScore{id: v.centIDs[i], sim: cosine(q, cv)}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].sim != ranked[j].sim {
			return ranked[i].sim > ranked[j].sim
		}
		return ranked[i].id < ranked[j].id
	})
	ranked = ranked[:n]

	// 2. Union member ids from the probed clusters (one indexed query per
	// cluster; the idx_ivf_assignments_cluster index makes this O(m)).
	idSet := make(map[int64]struct{}, 256)
	for _, cs := range ranked {
		rows, err := v.db.QueryContext(ctx,
			`SELECT vector_id FROM ivf_assignments WHERE cluster = ?`, cs.id)
		if err != nil {
			return v.bruteFallback(ctx, q, limit)
		}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return v.bruteFallback(ctx, q, limit)
			}
			idSet[id] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return v.bruteFallback(ctx, q, limit)
		}
		rows.Close()
	}
	if len(idSet) == 0 {
		// Nothing assigned to the probed clusters (e.g. a fresh corpus that
		// hasn't been assigned yet). Fall back to brute force so a query
		// never returns empty when the BLOB table is non-empty.
		return v.bruteFallback(ctx, q, limit)
	}

	// 3. Brute-force score the probed ids. Batch-load the BLOB rows in a
	// single indexed query (one round-trip instead of one per id) so the
	// probe stays fast even when nprobe is large.
	type scored struct {
		id  int64
		sim float64
	}
	all := make([]scored, 0, len(idSet))
	ids := make([]int64, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	// Chunk the IN-clause (SQLite's default max variables is 999; keep well
	// under it).
	const chunk = 200
	for off := 0; off < len(ids); off += chunk {
		end := off + chunk
		if end > len(ids) {
			end = len(ids)
		}
		sub := ids[off:end]
		ph := make([]string, len(sub))
		args := make([]interface{}, len(sub))
		for i, id := range sub {
			ph[i] = "?"
			args[i] = id
		}
		q2 := fmt.Sprintf(`SELECT %s, vector FROM %s WHERE %s IN (%s)`,
			v.source.idCol, v.source.table, v.source.idCol,
			strings.Join(ph, ","))
		rows2, err := v.db.QueryContext(ctx, q2, args...)
		if err != nil {
			return v.bruteFallback(ctx, q, limit)
		}
		for rows2.Next() {
			var id int64
			var blob []byte
			if err := rows2.Scan(&id, &blob); err != nil {
				rows2.Close()
				return v.bruteFallback(ctx, q, limit)
			}
			vv, derr := decodeVector(blob)
			if derr != nil {
				continue
			}
			all = append(all, scored{id: id, sim: cosine(q, vv)})
		}
		if err := rows2.Err(); err != nil {
			rows2.Close()
			return v.bruteFallback(ctx, q, limit)
		}
		rows2.Close()
	}
	if len(all) == 0 {
		return v.bruteFallback(ctx, q, limit)
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].sim != all[j].sim {
			return all[i].sim > all[j].sim
		}
		return all[i].id < all[j].id
	})
	if len(all) > limit {
		all = all[:limit]
	}
	hits := make([]SearchHit, len(all))
	for i, s := range all {
		hits[i] = SearchHit{ID: s.id, Dist: 1 - s.sim, Sim: s.sim}
	}
	return hits, nil
}

// Upsert implements VectorIndex: write the BLOB row (source of truth), then
// assign it to its nearest centroid. If the index isn't built yet the
// assignment is a no-op (the lazy rebuild at Search time will assign it).
func (v *IVFIndex) Upsert(ctx context.Context, id int64, vec []float32) error {
	if v == nil || v.db == nil {
		return fmt.Errorf("ivf: no database")
	}
	if len(vec) == 0 {
		return fmt.Errorf("ivf upsert %d: empty vector", id)
	}
	vb := encodeF32(vec)
	// 2. Assignment (only if the index is built; otherwise the lazy rebuild
	// at Search time assigns it). Read the centroid state before opening the
	// tx so the lock isn't held across the write.
	v.mu.RLock()
	built := v.centK > 0 && len(v.centVecs) > 0
	dim := v.centDim
	centIDs := v.centIDs
	centVecs := v.centVecs
	v.mu.RUnlock()
	// Both writes run in one short tx on a single connection (the store's
	// MaxOpenConns=1 invariant: a second connection from v.db while a tx is
	// open would deadlock).
	tx, err := v.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ivf upsert %d begin: %w", id, err)
	}
	defer tx.Rollback()
	// 1. BLOB row (source of truth). INSERT OR REPLACE so a re-embed is an
	// overwrite. model / updated_at / language are the indexer's columns; we
	// pass placeholder values so the row is valid and the dual-write can fill
	// them.
	if _, err := tx.ExecContext(ctx,
		`INSERT OR REPLACE INTO `+v.source.table+`
			(`+v.source.idCol+`, model, dim, vector, updated_at, language)
		 VALUES (?,?,?,?,?,?)`,
		id, "ivf", len(vec), vb, "1970-01-01T00:00:00Z", ""); err != nil {
		return fmt.Errorf("ivf upsert %d blob: %w", id, err)
	}
	if built {
		cluster := nearestCentroid(centIDs, centVecs, vec)
		if _, err := tx.ExecContext(ctx,
			`INSERT OR REPLACE INTO ivf_assignments (vector_id, cluster, dim) VALUES (?,?,?)`,
			id, cluster, dim); err != nil {
			return fmt.Errorf("ivf upsert %d assign: %w", id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ivf upsert %d commit: %w", id, err)
	}
	return nil
}

// Remove implements VectorIndex: delete the BLOB row and the assignment.
// Both are no-ops when the rows are absent (the BLOB row's own FK cascade
// handles the assignment if the parent is gone, but the assignment table
// has no FK, so we delete it explicitly).
func (v *IVFIndex) Remove(ctx context.Context, id int64) error {
	if v == nil || v.db == nil {
		return fmt.Errorf("ivf: no database")
	}
	if _, err := v.db.ExecContext(ctx,
		`DELETE FROM `+v.source.table+` WHERE `+v.source.idCol+` = ?`, id); err != nil {
		return fmt.Errorf("ivf remove %d blob: %w", id, err)
	}
	if _, err := v.db.ExecContext(ctx,
		`DELETE FROM ivf_assignments WHERE vector_id = ?`, id); err != nil {
		return fmt.Errorf("ivf remove %d assign: %w", id, err)
	}
	return nil
}

// Build runs the kmeans-lite centroid build over the corpus and assigns
// every vector to its nearest centroid. It is idempotent: re-running with
// the same corpus produces the same centroids (deterministic seed). A
// re-run after the corpus changes rebuilds from scratch (the old centroids
// and assignments are dropped).
//
// k is the number of centroids; 0 => default 64 (a small default gives
// denser clusters that a modest nprobe covers well on high-dim data).
func (v *IVFIndex) Build(ctx context.Context, k int) error {
	if v == nil || v.db == nil {
		return fmt.Errorf("ivf: no database")
	}
	// 1. Read the whole corpus from the BLOB table.
	rows, err := v.db.QueryContext(ctx,
		`SELECT `+v.source.idCol+`, vector FROM `+v.source.table)
	if err != nil {
		return fmt.Errorf("ivf build scan: %w", err)
	}
	corpus := make([]ivfRow, 0, 1024)
	for rows.Next() {
		var id int64
		var blob []byte
		if err := rows.Scan(&id, &blob); err != nil {
			rows.Close()
			return fmt.Errorf("ivf build scan row: %w", err)
		}
		vv, derr := decodeVector(blob)
		if derr != nil {
			continue
		}
		corpus = append(corpus, ivfRow{id: id, vec: vv})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("ivf build rows: %w", err)
	}
	rows.Close()
	if len(corpus) == 0 {
		// Empty corpus: nothing to build. Clear any stale centroids so a
		// later Search falls back to brute force (which is also empty).
		if err := v.clear(ctx); err != nil {
			return err
		}
		v.mu.Lock()
		v.centK = 0
		v.centDim = 0
		v.centIDs = nil
		v.centVecs = nil
		v.mu.Unlock()
		return nil
	}
	dim := len(corpus[0].vec)
	if k <= 0 {
		// For high-dim (768) gaussian data, cosine similarity is flat and
		// too many centroids fragment each region (the nprobe probe can't
		// cover the true top-K). A smaller k gives denser clusters that a
		// modest nprobe covers well. k=64 is the default; callers can
		// override for their corpus.
		k = 64
	}
	if k > len(corpus) {
		k = len(corpus)
	}
	// 2. K-means-lite: seed from a deterministic sample, iterate to
	// convergence (max 20 iters or 1% movement).
	centroids := kmeansLite(corpus, k, dim)
	// 3. Assign every corpus vector to its nearest centroid.
	centIDs := make([]int64, k)
	for i := range centroids {
		centIDs[i] = int64(i)
	}
	assign := make([]int, len(corpus))
	for i, r := range corpus {
		assign[i] = nearestCentroidIdx(centroids, r.vec)
	}
	// 4. Persist: drop old centroids + assignments, write new ones.
	if err := v.clear(ctx); err != nil {
		return err
	}
	tx, err := v.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("ivf build begin: %w", err)
	}
	defer tx.Rollback()
	for i := 0; i < k; i++ {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO ivf_centroids (k, dim, centroid, vector) VALUES (?,?,?,vec_f32(?))`,
			k, dim, i, encodeF32(centroids[i])); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("ivf build write c%d: %w", i, err)
		}
	}
	for i, r := range corpus {
		if _, err := tx.ExecContext(ctx,
			`INSERT OR REPLACE INTO ivf_assignments (vector_id, cluster, dim) VALUES (?,?,?)`,
			r.id, assign[i], dim); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("ivf build assign %d: %w", r.id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("ivf build commit: %w", err)
	}
	// 5. Update in-memory state.
	v.mu.Lock()
	v.centK = k
	v.centDim = dim
	v.centIDs = centIDs
	v.centVecs = centroids
	v.mu.Unlock()
	return nil
}

// ensureCentroids lazy-builds the IVF index on first Search if it hasn't
// been built yet (or if the on-disk state is missing). It is idempotent and
// safe under concurrency (the RWMutex + a per-call Build).
func (v *IVFIndex) ensureCentroids(ctx context.Context) error {
	v.mu.RLock()
	built := v.centK > 0 && len(v.centVecs) > 0
	v.mu.RUnlock()
	if built {
		// Verify the on-disk centroids still match (a rebuild by another
		// connection would have changed them; detect by count).
		var n int
		if err := v.db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM ivf_centroids`).Scan(&n); err != nil {
			return err
		}
		if n == v.centK {
			return nil
		}
	}
	return v.Build(ctx, 0)
}

// clear drops the IVF tables' rows (the tables themselves are created by
// migration 056 and are NOT dropped — they are the index's home).
func (v *IVFIndex) clear(ctx context.Context) error {
	if _, err := v.db.ExecContext(ctx, `DELETE FROM ivf_centroids`); err != nil {
		return fmt.Errorf("ivf clear centroids: %w", err)
	}
	if _, err := v.db.ExecContext(ctx, `DELETE FROM ivf_assignments`); err != nil {
		return fmt.Errorf("ivf clear assignments: %w", err)
	}
	return nil
}

// bruteFallback is the brute-force fallback used when the IVF probe fails
// (missing table, corrupt blob, schema drift). It scans the BLOB table
// directly, so it is correct even when the IVF state is broken.
func (v *IVFIndex) bruteFallback(ctx context.Context, q []float32, limit int) ([]SearchHit, error) {
	bf := NewBruteForce(v.db, v.source.table, v.source.idCol)
	return bf.Search(ctx, q, limit)
}

// ivfRow is a (vector id, vector) pair read from the BLOB table during Build.
type ivfRow struct {
	id  int64
	vec []float32
}

// centScore is a (centroid id, cosine sim to query) pair for the probe
// ranking.
type centScore struct {
	id  int64
	sim float64
}

// kmeansLite runs the standard k-means loop with a deterministic seed.
// maxIters = 20; convergence = sub-1e-6 total centroid movement. The seed is
// a fixed farthest-point selection (seeded RNG shuffle) so a re-build with
// the same corpus produces the same centroids.
func kmeansLite(corpus []ivfRow, k, dim int) [][]float32 {
	// Seed: pick k well-spread distinct rows (deterministic farthest-point
	// style: the first point is row 0, each subsequent point maximizes the
	// minimum distance to already-chosen points). This is much better for
	// high-dim gaussian data than a strided seed. A seeded RNG reorders the
	// rows first (fixed seed) so the seed picks spread points across the
	// corpus rather than the first k rows.
	order := make([]int, len(corpus))
	for i := range order {
		order[i] = i
	}
	rng := rand.New(rand.NewSource(42))
	rng.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })

	idx := make([]int, 0, k)
	idx = append(idx, order[0])
	minDist := make([]float64, len(corpus))
	for i := range minDist {
		minDist[i] = math.MaxFloat64
	}
	for len(idx) < k {
		bestI := -1
		bestD := -1.0
		for i := 0; i < len(corpus); i++ {
			if minDist[i] > bestD {
				bestD = minDist[i]
				bestI = i
			}
		}
		if bestI < 0 || bestD <= 0 {
			break
		}
		idx = append(idx, bestI)
		nv := corpus[bestI].vec
		for i := 0; i < len(corpus); i++ {
			d := euqDist2(corpus[i].vec, nv)
			if d < minDist[i] {
				minDist[i] = d
			}
		}
	}
	// If the corpus is smaller than k (degenerate), fill with the first rows.
	for i := 0; i < len(corpus) && len(idx) < k; i++ {
		dup := false
		for _, j := range idx {
			if j == i {
				dup = true
				break
			}
		}
		if !dup {
			idx = append(idx, i)
		}
	}
	centroids := make([][]float32, k)
	for i, ci := range idx {
		centroids[i] = append([]float32(nil), corpus[ci].vec...)
	}
	for it := 0; it < 20; it++ {
		// Assign.
		assign := make([]int, len(corpus))
		counts := make([]int, k)
		sums := make([][]float32, k)
		for i, r := range corpus {
			c := nearestCentroidIdx(centroids, r.vec)
			assign[i] = c
			counts[c]++
			if sums[c] == nil {
				sums[c] = make([]float32, dim)
			}
			for d := 0; d < dim; d++ {
				sums[c][d] += r.vec[d]
			}
		}
		// Update: recompute means; empty clusters keep their old mean.
		var movement float64
		for c := 0; c < k; c++ {
			if counts[c] == 0 {
				continue
			}
			newMean := make([]float32, dim)
			for d := 0; d < dim; d++ {
				newMean[d] = sums[c][d] / float32(counts[c])
			}
			for d := 0; d < dim; d++ {
				dv := float64(newMean[d] - centroids[c][d])
				movement += dv * dv
			}
			centroids[c] = newMean
		}
		if movement < 1e-6 {
			break
		}
	}
	return centroids
}

// nearestCentroidIdx returns the index of the centroid nearest to vec under
// Euclidean distance (the standard IVF partition criterion).
func nearestCentroidIdx(centroids [][]float32, vec []float32) int {
	best := 0
	bestD := math.MaxFloat64
	for i, c := range centroids {
		d := euqDist2(c, vec)
		if d < bestD {
			bestD = d
			best = i
		}
	}
	return best
}

// nearestCentroid is the nearest centroid under the same criterion, returning
// the centroid's id (the in-memory centroid index, which equals the id we
// persist in Build).
func nearestCentroid(centIDs []int64, centVecs [][]float32, vec []float32) int64 {
	i := nearestCentroidIdx(centVecs, vec)
	return centIDs[i]
}

// euqDist2 is the squared Euclidean distance (no sqrt — monotonic, so argmin
// is the same).
func euqDist2(a, b []float32) float64 {
	var s float64
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		d := float64(a[i] - b[i])
		s += d * d
	}
	return s
}
