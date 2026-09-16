package memory

import (
	"context"
	"math"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// openHubStore opens a fresh migrated store (all migrations applied, incl.
// 033_hub_score) for hub analysis tests.
func openHubStore(t *testing.T, project string) *store.Store {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// seedHubFile plants one file plus its package symbol (the hub target).
func seedHubFile(t *testing.T, db *store.Store, path string) int64 {
	t.Helper()
	var fileID int64
	if err := db.DB.QueryRow(`
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES (?, 1, 100, ?, '2026-01-01T00:00:00Z')
		RETURNING id`, path, "h-"+path).Scan(&fileID); err != nil {
		t.Fatalf("insert file %s: %v", path, err)
	}
	var symID int64
	uid := "uid-hub-" + path
	if err := db.DB.QueryRow(`
		INSERT INTO symbols (file_id, name, kind, start_line, end_line, content_hash, uid)
		VALUES (?, ?, 'package', 1, 1, ?, ?)
		RETURNING id`, fileID, path, "ch-hub-"+path, uid).Scan(&symID); err != nil {
		t.Fatalf("insert hub symbol %s: %v", path, err)
	}
	return symID
}

// seedImporterFile plants one importing file plus its package symbol and the
// imports edge to hubSymID.
func seedImporterFile(t *testing.T, db *store.Store, path string, hubSymID int64) {
	t.Helper()
	var fileID, symID int64
	if err := db.DB.QueryRow(`
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES (?, 1, 100, ?, '2026-01-01T00:00:00Z')
		RETURNING id`, path, "h-"+path).Scan(&fileID); err != nil {
		t.Fatalf("insert importer file %s: %v", path, err)
	}
	uid := "uid-imp-" + path
	if err := db.DB.QueryRow(`
		INSERT INTO symbols (file_id, name, kind, start_line, end_line, content_hash, uid)
		VALUES (?, ?, 'package', 1, 1, ?, ?)
		RETURNING id`, fileID, path, "ch-imp-"+path, uid).Scan(&symID); err != nil {
		t.Fatalf("insert importer symbol %s: %v", path, err)
	}
	if _, err := db.DB.Exec(`
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, line, valid_from)
		VALUES ('imports', ?, ?, ?, ?, ?, 'EXTRACTED', 1, 0)`,
		symID, fileID, hubSymID, path, path); err != nil {
		t.Fatalf("insert imports edge from %s: %v", path, err)
	}
}

// TestHubFileIdentification is 23.1 [RED]: hub files are identified by 3+
// importers, and the hub_score (importers / total_files) is computed and
// stored on the symbols.
func TestHubFileIdentification(t *testing.T) {
	st := openHubStore(t, "hub-id")
	svc := New(st, "hub-id")
	ctx := context.Background()

	// 10 files total: hub.go is imported by 5 files (the imp*.go files);
	// b.go is imported by 1 (ib.go); c.go is imported by 1 (ic.go).
	// Total: hub.go + 5 imp + b.go + ib.go + c.go + ic.go = 10 files.
	hubSym := seedHubFile(t, st, "hub.go")
	for i := 0; i < 5; i++ {
		seedImporterFile(t, st, "imp"+string(rune('a'+i))+".go", hubSym)
	}
	symB := seedHubFile(t, st, "b.go")
	seedImporterFile(t, st, "ib.go", symB) // 1 importer
	symC := seedHubFile(t, st, "c.go")
	seedImporterFile(t, st, "ic.go", symC) // 1 importer
	_ = symC

	hubs, err := svc.IdentifyHubFiles(ctx)
	if err != nil {
		t.Fatalf("IdentifyHubFiles: %v", err)
	}
	if len(hubs) != 1 {
		var names []string
		for _, h := range hubs {
			names = append(names, h.FilePath)
		}
		t.Fatalf("expected exactly 1 hub (3+ importers), got %d: %v", len(hubs), names)
	}
	if hubs[0].FilePath != "hub.go" {
		t.Fatalf("hub = %q, want hub.go", hubs[0].FilePath)
	}
	if hubs[0].ImporterCount != 5 {
		t.Fatalf("importer count = %d, want 5", hubs[0].ImporterCount)
	}
	if !hubs[0].IsHub {
		t.Fatalf("hub.go must be flagged IsHub")
	}
	// hub_score = importers / total_files = 5/10.
	if math.Abs(hubs[0].HubScore-0.5) > 1e-9 {
		t.Fatalf("hub_score = %v, want 0.5 (5 importers / 10 files)", hubs[0].HubScore)
	}

	// The hub_score is stored on the hub's symbol rows.
	var stored float64
	var storedNull bool
	err = st.DB.QueryRow(`
		SELECT hub_score IS NOT NULL, COALESCE(hub_score, 0)
		FROM symbols
		WHERE id = ?`, hubSym).Scan(&storedNull, &stored)
	if err != nil {
		t.Fatalf("read stored hub_score: %v", err)
	}
	if !storedNull {
		t.Fatalf("hub_score not stored on the hub symbol")
	}
	if math.Abs(stored-0.5) > 1e-9 {
		t.Fatalf("stored hub_score = %v, want 0.5", stored)
	}
	// A non-hub symbol carries 0 (stamped, not NULL).
	var bNull bool
	var bScore float64
	if err := st.DB.QueryRow(`SELECT hub_score IS NOT NULL, COALESCE(hub_score, 0) FROM symbols WHERE id = ?`, symB).
		Scan(&bNull, &bScore); err != nil {
		t.Fatalf("read b.go hub_score: %v", err)
	}
	if !bNull {
		t.Fatalf("non-hub symbol must carry a stamped hub_score (0), got NULL")
	}
	if bScore != 0 {
		t.Fatalf("non-hub hub_score = %v, want 0", bScore)
	}
}

// TestAnalyzeImpactHubFiles is 23.2 [RED]: AnalyzeImpact cross-references a
// change set with the hub file list, counts dependents, and classifies the
// impact level (high/medium/low).
func TestAnalyzeImpactHubFiles(t *testing.T) {
	st := openHubStore(t, "hub-impact")
	svc := New(st, "hub-impact")
	ctx := context.Background()

	// 10 files: hub1.go (6 importers), hub2.go (4 importers), 4 ordinary
	// files each with 0-2 importers.
	hub1 := seedHubFile(t, st, "hub1.go")
	for i := 0; i < 6; i++ {
		seedImporterFile(t, st, "i1"+string(rune('a'+i))+".go", hub1)
	}
	hub2 := seedHubFile(t, st, "hub2.go")
	for i := 0; i < 4; i++ {
		seedImporterFile(t, st, "i2"+string(rune('a'+i))+".go", hub2)
	}
	symA := seedHubFile(t, st, "a.go")
	seedImporterFile(t, st, "ia.go", symA) // 1 importer
	seedHubFile(t, st, "b.go")             // 0 importers

	// Change set: 5 files, 2 of them hubs.
	changed := []string{"hub1.go", "hub2.go", "a.go", "b.go", "i2a.go"}
	results, err := svc.AnalyzeImpact(ctx, changed)
	if err != nil {
		t.Fatalf("AnalyzeImpact: %v", err)
	}
	if len(results) != 5 {
		t.Fatalf("expected 5 impact results, got %d", len(results))
	}
	byFile := map[string]ImpactResult{}
	for _, r := range results {
		byFile[r.FilePath] = r
	}
	h1 := byFile["hub1.go"]
	if !h1.IsHub || h1.ImpactLevel != "high" {
		t.Fatalf("hub1.go: isHub=%v level=%q, want hub high", h1.IsHub, h1.ImpactLevel)
	}
	if h1.DependentCount != 6 {
		t.Fatalf("hub1.go dependents = %d, want 6", h1.DependentCount)
	}
	h2 := byFile["hub2.go"]
	if !h2.IsHub || h2.ImpactLevel != "medium" {
		t.Fatalf("hub2.go: isHub=%v level=%q, want hub medium", h2.IsHub, h2.ImpactLevel)
	}
	if h2.DependentCount != 4 {
		t.Fatalf("hub2.go dependents = %d, want 4", h2.DependentCount)
	}
	for _, f := range []string{"a.go", "b.go", "i2a.go"} {
		r := byFile[f]
		if r.IsHub {
			t.Fatalf("%s must not be a hub", f)
		}
		if r.ImpactLevel != "low" {
			t.Fatalf("%s level = %q, want low", f, r.ImpactLevel)
		}
		if r.DependentCount != 0 {
			t.Fatalf("%s dependents = %d, want 0 (non-hub)", f, r.DependentCount)
		}
	}
}

// seedRiskFile plants a file + symbol pair and returns the symbol id.
func seedRiskFile(t *testing.T, db *store.Store, path string) int64 {
	t.Helper()
	return seedHubFile(t, db, path)
}

// TestRiskScoreOnObservations is 23.3 [RED]: an observation referencing a hub
// file carries a risk_score proportional to the hub's hub_score; a non-hub
// observation scores 0; the score is re-stamped when the import fan-in
// changes.
func TestRiskScoreOnObservations(t *testing.T) {
	st := openHubStore(t, "hub-risk")
	svc := New(st, "hub-risk")
	ctx := context.Background()

	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s1', 'hub-risk', '/tmp', '2026-01-01T00:00:00Z', 'active')`); err != nil {
		t.Fatalf("insert session: %v", err)
	}

	// 9 files: hub.go imported by 5 (hub_score 5/9),
	// leaf.go + leaf2.go + leaf3.go are non-hub (0 importers).
	hubSym := seedRiskFile(t, st, "hub.go")
	for i := 0; i < 5; i++ {
		seedImporterFile(t, st, "rk"+string(rune('a'+i))+".go", hubSym)
	}
	seedRiskFile(t, st, "leaf.go")
	seedRiskFile(t, st, "leaf2.go")
	seedRiskFile(t, st, "leaf3.go")
	const totalFiles = 9

	if _, err := svc.IdentifyHubFiles(ctx); err != nil {
		t.Fatalf("IdentifyHubFiles: %v", err)
	}
	hubID, err := svc.Save(ctx, SaveInput{
		SessionID: "s1",
		Type:      "decision",
		Title:     "hub observation",
		Content:   "note about the hub file",
		Source:    "hub.go",
	})
	if err != nil {
		t.Fatalf("save hub obs: %v", err)
	}
	leafID, err := svc.Save(ctx, SaveInput{
		SessionID: "s1",
		Type:      "decision",
		Title:     "leaf observation",
		Content:   "note about the leaf file",
		Source:    "leaf.go",
	})
	if err != nil {
		t.Fatalf("save leaf obs: %v", err)
	}

	if err := svc.RecomputeRiskScores(ctx); err != nil {
		t.Fatalf("RecomputeRiskScores: %v", err)
	}
	var hubRisk float64
	if err := st.DB.QueryRow(`SELECT COALESCE(risk_score, -1) FROM observations WHERE id = ?`, hubID).
		Scan(&hubRisk); err != nil {
		t.Fatalf("read hub risk: %v", err)
	}
	wantHub := 5.0 / float64(totalFiles)
	if math.Abs(hubRisk-wantHub) > 1e-9 {
		t.Fatalf("hub observation risk_score = %v, want %v (5/%d, proportional to hub_score)", hubRisk, wantHub, totalFiles)
	}
	var leafRisk float64
	if err := st.DB.QueryRow(`SELECT COALESCE(risk_score, -1) FROM observations WHERE id = ?`, leafID).
		Scan(&leafRisk); err != nil {
		t.Fatalf("read leaf risk: %v", err)
	}
	if leafRisk != 0 {
		t.Fatalf("non-hub observation risk_score = %v, want 0", leafRisk)
	}

	// Add a 6th importer: total files becomes 10, hub_score 6/10 = 0.6,
	// and the re-stamp must pick the new value up.
	seedImporterFile(t, st, "rkz.go", hubSym)
	if _, err := svc.IdentifyHubFiles(ctx); err != nil {
		t.Fatalf("IdentifyHubFiles (re-stamp): %v", err)
	}
	if err := svc.RecomputeRiskScores(ctx); err != nil {
		t.Fatalf("RecomputeRiskScores (re-stamp): %v", err)
	}
	if err := st.DB.QueryRow(`SELECT COALESCE(risk_score, -1) FROM observations WHERE id = ?`, hubID).
		Scan(&hubRisk); err != nil {
		t.Fatalf("re-read hub risk: %v", err)
	}
	wantRe := 6.0 / 10.0
	if math.Abs(hubRisk-wantRe) > 1e-9 {
		t.Fatalf("after 6 importers, hub risk_score = %v, want %v (6/10)", hubRisk, wantRe)
	}
}
