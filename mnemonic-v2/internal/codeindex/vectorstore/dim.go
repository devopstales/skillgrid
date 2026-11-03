package vectorstore

import (
	"context"
	"fmt"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/embedder"
)

// dim.go — the 768-dim pin (the v1 store.go:23-27 shape) + the model-swap
// guard (M9, the v1 indexer.go:1133-1160 shape, dim-change aware).
//
// The 768-dim gating: the ANN indexes (HNSW/vec0) activate only when the
// embedder's Dimensions() is the pinned ANN dimension (768). Below that (the
// hash-64 floor at 64-dim), the store uses brute-force over the BLOB.
//
// The guard (M9): compare emb.Model() + emb.Dimensions() against the stored
// embed_meta. A model-name change (same dim) → ClearVectors. A dimension
// change (64↔768) → the ANN tables are dim-shaped, so the guard DROPS them
// (DROP TABLE IF EXISTS) in addition to clearing vectors — a dim change is a
// re-index, not a re-embed-in-place.

// AnnDim is the pinned ANN dimension (the v1 store.go:23-27 Dim=768). The
// vec0/HNSW tables are created with this dimension; a vector of a different
// dimension is rejected by the vec_f32 constructor (dimension-strict).
const AnnDim = 768

// metaModelKey / metaDimKey are the embed_meta keys the guard reads/writes.
const (
	metaModelKey = "embedding_model"
	metaDimKey   = "embedding_dim"
)

// annTables are the dim-shaped ANN tables the guard drops on a dimension
// change (the vec0 tables + the HNSW nodes/links, SQLite-only). The PG
// adapter has no ANN tables (brute-force floor, TICKET-08c).
var annTables = []string{
	"vec_symbols", "vec_chunks",
	"hnsw_nodes", "hnsw_links",
}

// GuardModelSwap is the M9 model-swap + dim-change guard, called at the start
// of every code_index run (TICKET-07's CodeHandler.Index → before any vector
// write). It compares the embedder's Model() + Dimensions() against the
// stored embed_meta:
//
//   - model-name change (same dim) → ClearVectors (DELETE embeddings +
//     chunk_embeddings) + drop the ANN index state (the derived index must not
//     carry stale vectors across the swap).
//   - dimension change (64↔768) → the ANN tables are dim-shaped, so DROP them
//     (DROP TABLE IF EXISTS vec_symbols/vec_chunks/hnsw_nodes/hnsw_links) in
//     addition to ClearVectors — a re-index, not a re-embed-in-place.
//
// A first run (no embed_meta) records the model + dim. The guard is
// idempotent: a no-change run records nothing. The run registry's
// one-index-at-a-time guard (B3, TICKET-07) serializes a mid-in-flight swap.
func GuardModelSwap(ctx context.Context, st store.Store, emb embedder.Embedder) error {
	if emb == nil {
		// A nil embedder (absent — the vector leg does not run) has no model
		// to guard; the guard is a no-op (the floor is the FTS leg).
		return nil
	}
	model := emb.Model()
	dim := emb.Dimensions()

	meta := readEmbedMeta(ctx, st)
	storedModel, _ := meta[metaModelKey]
	storedDim, _ := meta[metaDimKey]

	// No stored model (first run) → record + return (nothing to clear).
	if storedModel == "" {
		return writeEmbedMeta(ctx, st, metaModelKey, model, metaDimKey, fmt.Sprintf("%d", dim))
	}

	// Same model + same dim → no change (the idempotent run records nothing).
	if storedModel == model && storedDim == fmt.Sprintf("%d", dim) {
		return nil
	}

	// A model-name OR dimension change.
	dimChanged := storedDim != fmt.Sprintf("%d", dim)
	if err := st.ClearVectors(ctx); err != nil {
		return fmt.Errorf("guard: clear vectors: %w", err)
	}
	if dimChanged {
		// The ANN tables are dim-shaped → DROP them (the M9 load-bearing fix:
		// a dim change is a re-index, not a re-embed-in-place).
		if err := dropANNTables(ctx, st); err != nil {
			return fmt.Errorf("guard: drop dim-shaped ANN tables: %w", err)
		}
	} else {
		// A model-name change (same dim): clear the ANN index state (the
		// v1 guard's DeleteSymbols/DeleteChunks — the derived index must not
		// carry stale vectors across the swap). The tables keep their dim
		// shape (a same-dim re-embed-in-place is valid); only their rows are
		// cleared.
		if err := clearANNIndexState(ctx, st); err != nil {
			return fmt.Errorf("guard: clear ANN index state: %w", err)
		}
	}
	return writeEmbedMeta(ctx, st, metaModelKey, model, metaDimKey, fmt.Sprintf("%d", dim))
}

// readEmbedMeta reads the whole embed_meta table (key → value). A missing
// table (pre-migration) returns an empty map (the guard treats it as a first
// run).
func readEmbedMeta(ctx context.Context, st store.Store) map[string]string {
	out := map[string]string{}
	rq, ok := st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return out
	}
	rows, err := rq.RawQuery(`SELECT key, value FROM embed_meta`)
	if err != nil {
		return out
	}
	for _, r := range rows {
		k, _ := r[0].(string)
		v, _ := r[1].(string)
		if k != "" {
			out[k] = v
		}
	}
	return out
}

// writeEmbedMeta upserts the model + dim keys into embed_meta (the guard's
// record step).
func writeEmbedMeta(ctx context.Context, st store.Store, modelKey, model, dimKey, dim string) error {
	rq, ok := st.(interface {
		RawExec(query string, args ...any) error
	})
	if !ok {
		return fmt.Errorf("guard: store %T does not expose the RawExec seam", st)
	}
	if err := rq.RawExec(`INSERT OR REPLACE INTO embed_meta (key, value) VALUES (?, ?)`, modelKey, model); err != nil {
		return fmt.Errorf("guard: record model: %w", err)
	}
	return rq.RawExec(`INSERT OR REPLACE INTO embed_meta (key, value) VALUES (?, ?)`, dimKey, dim)
}

// dropANNTables drops the dim-shaped ANN tables (the M9 dim-change branch).
// SQLite-only (the PG adapter has no ANN tables — brute-force floor).
// DROP TABLE IF EXISTS is idempotent (a missing table is a no-op).
func dropANNTables(ctx context.Context, st store.Store) error {
	rq, ok := st.(interface {
		RawExec(query string, args ...any) error
	})
	if !ok {
		return fmt.Errorf("guard: store %T does not expose the RawExec seam", st)
	}
	for _, t := range annTables {
		if err := rq.RawExec(`DROP TABLE IF EXISTS ` + t); err != nil {
			return fmt.Errorf("drop %s: %w", t, err)
		}
	}
	return nil
}

// clearANNIndexState clears the ANN index rows (the model-name-change branch,
// the v1 guard's DeleteSymbols/DeleteChunks): the vec0 tables + the HNSW
// graph nodes/links. The tables keep their dim shape (a same-dim re-embed-in-
// place is valid); only their derived rows are cleared so the index does not
// carry stale vectors across the swap.
func clearANNIndexState(ctx context.Context, st store.Store) error {
	rq, ok := st.(interface {
		RawExec(query string, args ...any) error
	})
	if !ok {
		return fmt.Errorf("guard: store %T does not expose the RawExec seam", st)
	}
	for _, q := range []string{
		`DELETE FROM vec_symbols`, `DELETE FROM vec_chunks`,
		`DELETE FROM hnsw_nodes`, `DELETE FROM hnsw_links`,
	} {
		if err := rq.RawExec(q); err != nil {
			// A missing table (a pre-ANN store) is a no-op (the table is
			// absent, there is nothing to clear). The guard never hard-fails
			// on the ANN state (the BLOB ClearVectors is the load-bearing
			// clear).
			continue
		}
	}
	return nil
}

// annActive reports whether the ANN indexes activate for the embedder's dim
// (the 768-dim gating): true only when Dimensions() == AnnDim. Below that
// (the hash-64 floor), brute-force.
func annActive(emb embedder.Embedder) bool {
	if emb == nil {
		return false
	}
	return emb.Dimensions() == AnnDim
}
