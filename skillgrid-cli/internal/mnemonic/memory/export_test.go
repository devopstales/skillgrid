package memory

import (
	"context"
	"database/sql"
	"encoding/base64"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// seedGraphForExport plants a file, three symbols, two edges (one with a
// temporal window), and two symbol embeddings so ExportProject has graph data
// to include. It returns the symbol IDs in insertion order.
func seedGraphForExport(t *testing.T, db *sql.DB) []int64 {
	t.Helper()
	var fileID int64
	if err := db.QueryRow(`
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES ('/tmp/export.go', 1, 100, 'h-export', '2026-01-01T00:00:00Z')
		RETURNING id`).Scan(&fileID); err != nil {
		t.Fatalf("insert file: %v", err)
	}
	var ids []int64
	for i, name := range []string{"expAlpha", "expBeta", "expGamma"} {
		var id int64
		if err := db.QueryRow(`
			INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
			VALUES (?, ?, ?, 'function', 'go', 'func '+?+'()', ?, ?, 'ch-exp-'+?+'-'+? , ?)
			RETURNING id`,
			fileID, name, name, name, i*10+1, i*10+9, "ch-exp-"+name, "uid-exp-"+name, name).Scan(&id); err != nil {
			t.Fatalf("insert symbol %s: %v", name, err)
		}
		ids = append(ids, id)
	}
	now := time.Now().Unix()
	if _, err := db.Exec(`
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, line, valid_from, valid_to)
		VALUES ('calls', ?, ?, ?, 'expBeta', '/tmp/export.go', 'EXTRACTED', 1, ?, NULL),
		       ('references', ?, ?, NULL, 'someLib', NULL, 'EXTRACTED', 2, ?, ?)`,
		ids[0], fileID, ids[1], now-1000,
		ids[1], fileID, now-500, now-10); err != nil {
		t.Fatalf("insert edges: %v", err)
	}
	vectors := [][]byte{
		float32VectorBytes(4, 0.1, 0.2, 0.3, 0.4),
		float32VectorBytes(8, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.1, 1.2),
	}
	for i, symID := range []int64{ids[0], ids[2]} {
		if _, err := db.Exec(`
			INSERT INTO embeddings (symbol_id, model, dim, vector, updated_at)
			VALUES (?, 'export-model-v1', ?, ?, '2026-01-02T00:00:00Z')`,
			symID, len(vectors[i])/4, vectors[i]); err != nil {
			t.Fatalf("insert embedding %d: %v", symID, err)
		}
	}
	return ids
}

// openExportTestStore opens a store over a temp data dir and returns a
// multi-connection *sql.DB over the same file. The store.Open pool is
// single-connection (SetMaxOpenConns(1)), which deadlocks when ExportProject
// re-enters the same pool from a nested per-observation embedding lookup (the
// same class of issue graph.temporal_test documents). The export tests build
// their Service on the multi-conn DB so the nested lookup can take a second
// connection.
func openExportTestStore(t *testing.T, project string) (*store.Store, *sql.DB) {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	db, err := sql.Open("sqlite", st.Path())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	db.SetMaxOpenConns(8)
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=10000",
	} {
		if _, err := db.Exec(pragma); err != nil {
			t.Fatalf("pragma %s: %v", pragma, err)
		}
	}
	return st, db
}



// float32VectorBytes packs floats into the little-endian float32 BLOB layout
// the embeddings table uses (step 07 convention).
func float32VectorBytes(values ...float32) []byte {
	buf := make([]byte, 4*len(values))
	for i, v := range values {
		u := math.Float32bits(v)
		copy(buf[4*i:], []byte{byte(u), byte(u >> 8), byte(u >> 16), byte(u >> 24)})
	}
	return buf
}

// TestExportProjectStructure is 11.1 [RED]: ExportProject returns a bundle
// with non-empty observations, graph edges, and embeddings, and every
// observation record carries id/type/content/metadata/embeddings(base64)/
// graph_ref.
func TestExportProjectStructure(t *testing.T) {
	_, db := openExportTestStore(t, "export-struct")
	if _, err := db.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s1', 'export-struct', '/tmp', '2026-01-01T00:00:00Z', 'active')`); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	st := &store.Store{DB: db}
	svc := New(st, "export-struct")
	ctx := context.Background()

	symIDs := seedGraphForExport(t, db)

	obsLinked, err := svc.Save(ctx, SaveInput{
		SessionID: "s1",
		Type:      "decision",
		Title:     "linked observation",
		Content:   "observation bound to an indexed symbol",
		Source:    "/tmp/export.go",
	})
	if err != nil {
		t.Fatalf("save linked: %v", err)
	}
	obsPlain, err := svc.Save(ctx, SaveInput{
		SessionID: "s1",
		Type:      "discovery",
		Title:     "plain observation",
		Content:   "observation with no graph binding",
	})
	if err != nil {
		t.Fatalf("save plain: %v", err)
	}

	bundle, err := svc.ExportProject(ctx)
	if err != nil {
		t.Fatalf("ExportProject: %v", err)
	}
	if len(bundle.Observations) == 0 {
		t.Fatalf("bundle.Observations is empty, want non-empty")
	}
	if len(bundle.GraphEdges) == 0 {
		t.Fatalf("bundle.GraphEdges is empty, want non-empty")
	}
	if len(bundle.Embeddings) == 0 {
		t.Fatalf("bundle.Embeddings is empty, want non-empty")
	}

	// Per-record field contract.
	recs := map[int64]*ExportRecord{}
	for i := range bundle.Observations {
		r := &bundle.Observations[i]
		recs[r.ID] = r
		if r.ID == 0 {
			t.Errorf("observation record %d has id 0", i)
		}
		if r.Type == "" {
			t.Errorf("observation %d has empty type", r.ID)
		}
		if r.Content == "" {
			t.Errorf("observation %d has empty content", r.ID)
		}
		if r.Metadata == nil {
			t.Errorf("observation %d has nil metadata", r.ID)
		} else {
			if r.Metadata["title"] == "" {
				t.Errorf("observation %d metadata missing title", r.ID)
			}
		}
	}
	linked := recs[obsLinked]
	if linked == nil {
		t.Fatalf("linked observation %d missing from bundle", obsLinked)
	}
	if !strings.Contains(linked.Content, "indexed symbol") {
		t.Errorf("linked observation content = %q", linked.Content)
	}
	if linked.Embeddings == nil {
		t.Errorf("linked observation has nil embeddings (should reference the symbol bridge)")
	} else {
		raw, err := base64.StdEncoding.DecodeString(linked.Embeddings["vector"])
		if err != nil {
			t.Fatalf("embeddings vector is not base64: %v", err)
		}
		if len(raw) == 0 {
			t.Errorf("embedded vector is empty")
		}
	}
	if linked.GraphRef == nil {
		t.Fatalf("linked observation has no graph_ref, want symbol %d", symIDs[0])
	}
	if *linked.GraphRef != symIDs[0] {
		t.Errorf("graph_ref = %d, want %d", *linked.GraphRef, symIDs[0])
	}
	plain := recs[obsPlain]
	if plain == nil {
		t.Fatalf("plain observation %d missing from bundle", obsPlain)
	}
	if plain.GraphRef != nil {
		t.Errorf("plain observation has graph_ref %v, want nil", *plain.GraphRef)
	}

	// Edge records preserve the temporal window.
	var windowed *EdgeRecord
	for i := range bundle.GraphEdges {
		e := &bundle.GraphEdges[i]
		if e.Kind == "references" {
			windowed = e
		}
	}
	if windowed == nil {
		t.Fatalf("references edge missing from bundle")
	}
	if windowed.ValidTo == nil {
		t.Errorf("references edge lost its valid_to window")
	}

	// Embedding records carry symbol/model/dim/base64 vector.
	var embs []EmbeddingRecord
	embs = bundle.Embeddings
	if len(embs) != 2 {
		t.Fatalf("want 2 embedding records, got %d", len(embs))
	}
	for _, em := range embs {
		if em.SymbolID == 0 || em.Model == "" || em.Dim <= 0 {
			t.Errorf("embedding record incomplete: %+v", em)
		}
		raw, err := base64.StdEncoding.DecodeString(em.Vector)
		if err != nil {
			t.Fatalf("embedding vector not base64: %v", err)
		}
		if len(raw) != 4*em.Dim {
			t.Errorf("vector len %d != 4*dim %d", len(raw), em.Dim)
		}
	}
}

// TestExportImportRoundtrip is 11.2 [RED]: export a project (5 observations,
// 3 edges, 2 embeddings) to a JSON file, import it into a FRESH empty store,
// and verify every record round-trips with matching IDs and content.
func TestExportImportRoundtrip(t *testing.T) {
	_, srcDB := openExportTestStore(t, "rt-src")
	if _, err := srcDB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s1', 'rt-src', '/tmp', '2026-01-01T00:00:00Z', 'active')`); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	srcSvc := New(&store.Store{DB: srcDB}, "rt-src")
	ctx := context.Background()
	symIDs := seedGraphForExport(t, srcDB)

	// A second edge so the graph has 3.
	fileID := fileIDOf(t, srcDB, "/tmp/export.go")
	if _, err := srcDB.Exec(`
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, line, valid_from, valid_to)
		VALUES ('imports', ?, ?, ?, 'expGamma', '/tmp/export.go', 'EXTRACTED', 3, ?, NULL)`,
		symIDs[0], fileID, symIDs[2], time.Now().Unix()); err != nil {
		t.Fatalf("insert third edge: %v", err)
	}

	wantObs := map[int64]string{}
	linkedID, err := srcSvc.Save(ctx, SaveInput{
		SessionID: "s1",
		Type:      "decision",
		Title:     "roundtrip linked",
		Content:   "bound to the exported symbol",
		Source:    "/tmp/export.go",
	})
	if err != nil {
		t.Fatalf("save linked: %v", err)
	}
	wantObs[linkedID] = "bound to the exported symbol"
	for i, typ := range []string{"discovery", "bugfix", "pattern", "config"} {
		id, err := srcSvc.Save(ctx, SaveInput{
			SessionID: "s1",
			Type:      typ,
			Title:     "roundtrip note " + typ,
			Content:   "content " + typ + " body for roundtrip",
		})
		if err != nil {
			t.Fatalf("save %d: %v", i, err)
		}
		wantObs[id] = "content " + typ + " body for roundtrip"
	}
	if len(wantObs) != 5 {
		t.Fatalf("want 5 distinct observations, got %d", len(wantObs))
	}

	bundle, err := srcSvc.ExportProject(ctx)
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if len(bundle.Observations) != 5 {
		t.Fatalf("bundle has %d observations, want 5", len(bundle.Observations))
	}
	if len(bundle.GraphEdges) != 3 {
		t.Fatalf("bundle has %d edges, want 3", len(bundle.GraphEdges))
	}
	if len(bundle.Embeddings) != 2 {
		t.Fatalf("bundle has %d embeddings, want 2", len(bundle.Embeddings))
	}

	// Write the bundle to a JSON file (the portable artifact).
	out := filepath.Join(t.TempDir(), "bundle.json")
	f, err := os.Create(out)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := srcSvc.WriteExport(ctx, f); err != nil {
		t.Fatalf("write export: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Fresh, empty store under a DIFFERENT project id (another instance).
	dst, err := openStoreFor(t, "rt-dst")
	if err != nil {
		t.Fatalf("open dst: %v", err)
	}
	if _, err := dst.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s1', 'rt-dst', '/tmp', '2026-01-01T00:00:00Z', 'active')`); err != nil {
		t.Fatalf("dst session: %v", err)
	}
	in, err := os.Open(out)
	if err != nil {
		t.Fatalf("open bundle: %v", err)
	}
	defer in.Close()
	dstSvc := New(dst, "rt-dst")
	if err := dstSvc.ImportProject(ctx, in); err != nil {
		t.Fatalf("import: %v", err)
	}

	// Observations: all 5 present, same IDs, same content.
	for id, content := range wantObs {
		var gotID int64
		var gotContent, gotType string
		err := dst.DB.QueryRow(`
			SELECT id, content, type FROM observations WHERE id = ?`, id).
			Scan(&gotID, &gotContent, &gotType)
		if err != nil {
			t.Fatalf("observation %d missing after import: %v", id, err)
		}
		if gotContent != content {
			t.Errorf("observation %d content = %q, want %q", id, gotContent, content)
		}
	}

	// Edges: all 3 present with the same endpoints and window.
	var edgeCount int
	if err := dst.DB.QueryRow(`SELECT COUNT(*) FROM edges`).Scan(&edgeCount); err != nil {
		t.Fatalf("count edges: %v", err)
	}
	if edgeCount != 3 {
		t.Fatalf("dst has %d edges, want 3", edgeCount)
	}
	var windowedTo sql.NullInt64
	if err := dst.DB.QueryRow(`
		SELECT valid_to FROM edges WHERE kind = 'references'`).Scan(&windowedTo); err != nil {
		t.Fatalf("read windowed edge: %v", err)
	}
	if !windowedTo.Valid {
		t.Errorf("windowed edge lost valid_to after import")
	}

	// Embeddings: both present, vectors byte-identical.
	for i, symID := range []int64{symIDs[0], symIDs[2]} {
		var blob []byte
		var model string
		if err := dst.DB.QueryRow(`
			SELECT vector, model FROM embeddings WHERE symbol_id = ?`, symID).
			Scan(&blob, &model); err != nil {
			t.Fatalf("embedding for symbol %d missing: %v", symID, err)
		}
		if model == "" {
			t.Errorf("embedding %d lost its model", symID)
		}
		var srcBlob []byte
		if err := srcDB.QueryRow(`
			SELECT vector FROM embeddings WHERE symbol_id = ?`, symID).Scan(&srcBlob); err != nil {
			t.Fatalf("src embedding %d: %v", symID, err)
		}
		if string(blob) != string(srcBlob) {
			t.Errorf("embedding %d vector changed on roundtrip", symID)
		}
		_ = i
	}

	// The linked observation kept its graph_ref binding.
	var ref sql.NullInt64
	if err := dst.DB.QueryRow(`SELECT graph_ref FROM observations WHERE id = ?`, linkedID).Scan(&ref); err != nil {
		t.Fatalf("read dst graph_ref: %v", err)
	}
	if !ref.Valid || ref.Int64 != symIDs[0] {
		t.Errorf("dst graph_ref = %v, want %d", ref, symIDs[0])
	}
}

// fileIDOf looks up a files row by path.
func fileIDOf(t *testing.T, db *sql.DB, path string) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(`SELECT id FROM files WHERE path = ?`, path).Scan(&id); err != nil {
		t.Fatalf("lookup file %s: %v", path, err)
	}
	return id
}
