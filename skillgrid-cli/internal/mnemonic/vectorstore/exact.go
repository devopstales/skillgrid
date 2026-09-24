package vectorstore

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
)

// ExactnessResult is the per-ID agreement report between the vec table's
// top-K and a Go-side brute-force cosine scan over the BLOB table (ground
// truth). The spike's exactness-check methodology: the BLOB scan is exact
// (deterministic cosine), the vec table is the candidate.
type ExactnessResult struct {
	Table         string
	QueryDim      int
	Limit         int
	Agree         bool
	Disagreements []Disagreement
}

// Disagreement is one rank where the vec table's top-K ID differs from the
// brute-force scan's top-K ID.
type Disagreement struct {
	Rank    int
	VecID   int64
	BlobID  int64
	VecSim  float64
	BlobSim float64
}

// blobTableFor maps a vec table name to its BLOB source-of-truth table.
func blobTableFor(vecTable string) string {
	switch vecTable {
	case "vec_symbols":
		return "embeddings"
	case "vec_chunks":
		return "chunk_embeddings"
	default:
		return ""
	}
}

// idColumnFor maps a vec table to the BLOB table's FK column (symbol_id /
// chunk_id) that matches the vec table's rowid.
func idColumnFor(vecTable string) string {
	switch vecTable {
	case "vec_symbols":
		return "symbol_id"
	case "vec_chunks":
		return "chunk_id"
	default:
		return ""
	}
}

// ExactnessCheck compares the vec table's top-K against a brute-force Go
// cosine scan over the BLOB table for the same query vector.
func ExactnessCheck(ctx context.Context, db *sql.DB, vecTable string, queryVec []float32, limit int) (*ExactnessResult, error) {
	blobTable := blobTableFor(vecTable)
	idCol := idColumnFor(vecTable)
	if blobTable == "" || idCol == "" {
		return nil, fmt.Errorf("unknown vec table %q", vecTable)
	}
	if limit <= 0 {
		limit = 10
	}

	// 1. Vec table's top-K (candidate).
	vecIDs, err := search(ctx, db, vecTable, queryVec, limit)
	if err != nil {
		return nil, fmt.Errorf("vec search: %w", err)
	}

	// 2. Brute-force ground truth: decode every BLOB vector, cosine-score
	// against the query, sort descending by sim then ascending by id, take
	// top-K. This is the exact reference (deterministic, no vtab).
	rows, err := db.QueryContext(ctx,
		`SELECT `+idCol+`, vector FROM `+blobTable)
	if err != nil {
		return nil, fmt.Errorf("blob scan: %w", err)
	}
	type scored struct {
		id  int64
		sim float64
	}
	var all []scored
	qv := memory.Vector{Data: queryVec}
	for rows.Next() {
		var id int64
		var blob []byte
		if err := rows.Scan(&id, &blob); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan blob row: %w", err)
		}
		v, derr := memory.DecodeVector(blob)
		if derr != nil {
			continue
		}
		all = append(all, scored{id: id, sim: memory.CosineSimilarity(qv, v)})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("rows: %w", err)
	}
	rows.Close()
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].sim != all[j].sim {
			return all[i].sim > all[j].sim
		}
		return all[i].id < all[j].id
	})
	if len(all) > limit {
		all = all[:limit]
	}

	// 3. Compare per rank.
	res := &ExactnessResult{
		Table:    vecTable,
		QueryDim: len(queryVec),
		Limit:    limit,
		Agree:    true,
	}
	for r := 0; r < limit; r++ {
		var vecID, blobID int64
		var vecSim, blobSim float64
		if r < len(vecIDs) {
			vecID = vecIDs[r]
		}
		if r < len(all) {
			blobID = all[r].id
			blobSim = all[r].sim
		}
		// vecSim: decode the vec table's vector for this rowid (best effort).
		if r < len(vecIDs) {
			var vb []byte
			_ = db.QueryRowContext(ctx,
				fmt.Sprintf(`SELECT embedding FROM %s WHERE rowid=?`, vecTable), vecID).Scan(&vb)
			if vv, derr := memory.DecodeVector(vb); derr == nil {
				vecSim = memory.CosineSimilarity(qv, vv)
			}
		}
		if vecID != blobID {
			res.Agree = false
			res.Disagreements = append(res.Disagreements, Disagreement{
				Rank: r + 1, VecID: vecID, BlobID: blobID, VecSim: vecSim, BlobSim: blobSim,
			})
		}
	}
	return res, nil
}
