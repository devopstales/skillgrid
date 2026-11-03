package vectorstore

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
)

// VectorIndex is the ANN seam (briefing: "Seam: VectorIndex (adapters:
// BruteForce + IVF = 2, seam justified)"). A backend is anything that can
// return the top-K vector ids for a query, with the cosine distance (and
// derived similarity) so the durable leg can report Provenance.Sim.
//
// The contract is rank-approximate: BruteForce is exact, IVF is approximate
// (recall @ K degrades with nprobe). Callers that need exactness keep the
// BLOB table as ground truth and can call BruteForce directly.
//
// Deletion test (blueprint Task 4): deleting this interface collapses the
// search callers back to direct SQL — the interface earns its keep at 2+
// backends. HNSW (TICKET-05) is the 3rd adapter.
type VectorIndex interface {
	// Search returns the top-K vector ids for q, most-similar first.
	// limit <= 0 defaults to 10. A corrupt or missing backend must degrade
	// to brute force (no error) — see IVFIndex.Search.
	Search(ctx context.Context, q []float32, limit int) ([]SearchHit, error)

	// Upsert records a single vector in the backend. It must be safe under
	// the store's MaxOpenConns=1 invariant (no second connection from the
	// same *sql.DB while a tx is open).
	Upsert(ctx context.Context, id int64, vec []float32) error

	// Remove deletes a single vector. Missing ids are a no-op.
	Remove(ctx context.Context, id int64) error

	// Name identifies the backend ("brute-force" / "ivf") for logs and
	// provenance reporting.
	Name() string
}

// bruteSource is the read shape a brute-force backend needs: a table and the
// id column that maps the BLOB row to its vector id. It is the in-memory /
// in-Go reference implementation (no vtab, no ANN).
type bruteSource struct {
	table string // BLOB source-of-truth table (embeddings / chunk_embeddings)
	idCol string // FK column that matches the vec table rowid
}

// BruteForce is the exact VectorIndex adapter: a Go-side cosine scan over the
// BLOB source-of-truth table. It is the fallback for every ANN path (IVF
// corrupt/missing degrades here; HNSW warm-load failure degrades here).
//
// The scan decodes every row, scores it, and sorts descending by sim then
// ascending by id (deterministic tie-break). At 20k rows this is ~10ms warm;
// the bench (TICKET-01) gates the decision to add IVF / HNSW.
type BruteForce struct {
	db     *sql.DB
	source bruteSource
}

// NewBruteForce builds a brute-force index over the named BLOB table.
// table / idCol must be a valid pair (embeddings+symbol_id,
// chunk_embeddings+chunk_id); unknown pairs are a programming error.
func NewBruteForce(db *sql.DB, table, idCol string) *BruteForce {
	return &BruteForce{db: db, source: bruteSource{table: table, idCol: idCol}}
}

// Name implements VectorIndex.
func (b *BruteForce) Name() string { return "brute-force" }

// Search implements VectorIndex: decode every row, score, sort, take top-K.
func (b *BruteForce) Search(ctx context.Context, q []float32, limit int) ([]SearchHit, error) {
	if limit <= 0 {
		limit = 10
	}
	if b == nil || b.db == nil {
		return nil, fmt.Errorf("brute-force: no database")
	}
	rows, err := b.db.QueryContext(ctx,
		`SELECT `+b.source.idCol+`, vector FROM `+b.source.table)
	if err != nil {
		return nil, fmt.Errorf("brute-force %s scan: %w", b.source.table, err)
	}
	defer rows.Close()

	type scored struct {
		id  int64
		sim float64
	}
	all := make([]scored, 0, 1024)
	for rows.Next() {
		var id int64
		var blob []byte
		if err := rows.Scan(&id, &blob); err != nil {
			return nil, fmt.Errorf("brute-force scan row: %w", err)
		}
		v, derr := decodeVector(blob)
		if derr != nil {
			// Skip a corrupt BLOB row; the exactness check tolerates it.
			continue
		}
		all = append(all, scored{id: id, sim: cosine(q, v)})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("brute-force rows: %w", err)
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

// Upsert implements VectorIndex by mirroring the BLOB row (source of truth).
// A brute-force backend has no index to update, so the write is a
// INSERT OR REPLACE on the BLOB table; the caller passes a 0-length vecBytes
// (the BLOB write is owned by the indexer's dual-write).
//
// Note: BruteForce.Upsert is a no-op by design — the BLOB table IS the index.
// The method exists so callers can treat every VectorIndex uniformly; the
// ANN adapters (IVF/HNSW) use it to maintain their derived structures.
func (b *BruteForce) Upsert(ctx context.Context, id int64, _ []float32) error {
	_ = ctx
	_ = id
	return nil
}

// Remove implements VectorIndex as a no-op (see Upsert).
func (b *BruteForce) Remove(ctx context.Context, id int64) error {
	_ = ctx
	_ = id
	return nil
}

// decodeVector is the package-local little-endian float32 decoder (inverse of
// EncodeVector). Kept here so vectorstore has no dependency on the memory
// package; the production caller passes memory.EncodeVector output.
func decodeVector(blob []byte) ([]float32, error) {
	if len(blob)%4 != 0 {
		return nil, fmt.Errorf("embedding length %d is not a multiple of 4", len(blob))
	}
	v := make([]float32, len(blob)/4)
	for i := range v {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(blob[i*4:]))
	}
	return v, nil
}

// cosine is the dot-product / norm-of-each cosine similarity in [-1, 1].
// Zero-length or mismatched-length vectors score 0 (matches memory.CosineSimilarity).
func cosine(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		d := float64(a[i])
		e := float64(b[i])
		dot += d * e
		na += d * d
		nb += e * e
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
