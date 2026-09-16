package memory

import (
	"context"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// seedImporterEdgeWithWindow plants one importing file + its package symbol and
// an imports edge to hubSymID with EXPLICIT temporal bounds (valid_from /
// valid_to, UNIX seconds). The default seedImporterFile seeds valid_from = 0
// and valid_to = NULL (a perpetually-active edge); this helper drives the
// boundary cases that test — expired, not-yet-active, and exactly-active — so
// the temporal current-state filter is exercised at the edges, not just the
// "always active" middle.
func seedImporterEdgeWithWindow(t *testing.T, db *store.Store, path string, hubSymID int64, validFrom, validTo *int64) {
	t.Helper()
	now := time.Now().Unix()
	if validFrom == nil {
		z := now
		validFrom = &z
	}
	var fileID, symID int64
	if err := db.DB.QueryRow(`
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES (?, 1, 100, ?, '2026-01-01T00:00:00Z')
		RETURNING id`, path, "h-"+path).Scan(&fileID); err != nil {
		t.Fatalf("insert file %s: %v", path, err)
	}
	uid := "uid-tmp-" + path
	if err := db.DB.QueryRow(`
		INSERT INTO symbols (file_id, name, kind, start_line, end_line, content_hash, uid)
		VALUES (?, ?, 'package', 1, 1, ?, ?)
		RETURNING id`, fileID, path, "ch-tmp-"+path, uid).Scan(&symID); err != nil {
		t.Fatalf("insert symbol %s: %v", path, err)
	}
	var to any
	if validTo == nil {
		to = nil
	} else {
		to = *validTo
	}
	if _, err := db.DB.Exec(`
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, line, valid_from, valid_to)
		VALUES ('imports', ?, ?, ?, ?, ?, 'EXTRACTED', 1, ?, ?)`,
		symID, fileID, hubSymID, path, path, *validFrom, to); err != nil {
		t.Fatalf("insert imports edge %s: %v", path, err)
	}
}

// TestTemporalEdgesBoundary is 26.1 [RED] — the hub importer count applies the
// step-10 current-state temporal window `valid_from <= now AND (valid_to IS
// NULL OR valid_to > now)`. The default hub tests seed valid_from = 0 /
// valid_to = NULL (the "always active" middle); this one drives the BOUNDARIES:
// an edge whose valid_to is in the past (expired) and one whose valid_from is
// in the future (not yet active) must both be EXCLUDED from the importer
// count, while active edges (valid_to = NULL or valid_to in the future,
// valid_from in the past) are included. The hub is flagged only by the active
// importers, so the count reflects the window exactly.
func TestTemporalEdgesBoundary(t *testing.T) {
	st := openHubStore(t, "temporal-edge-bnd")
	svc := New(st, "temporal-edge-bnd")
	ctx := context.Background()

	now := time.Now().Unix()
	past := now - 3600  // 1h ago
	future := now + 3600 // 1h out

	hubSym := seedHubFile(t, st, "hub.go")

	// Three ACTIVE importers (valid_from = past, valid_to = NULL) → count 3,
	// just above the MinHubImporters=3 threshold.
	for _, name := range []string{"active1.go", "active2.go", "active3.go"} {
		seedImporterEdgeWithWindow(t, st, name, hubSym, &past, nil)
	}

	// One EXPIRED importer (valid_to in the past) → must be excluded.
	seedImporterEdgeWithWindow(t, st, "expired.go", hubSym, &past, &past)

	// One NOT-YET-ACTIVE importer (valid_from in the future) → must be excluded.
	seedImporterEdgeWithWindow(t, st, "future.go", hubSym, &future, nil)

	hubs, err := svc.IdentifyHubFiles(ctx)
	if err != nil {
		t.Fatalf("IdentifyHubFiles: %v", err)
	}
	if len(hubs) != 1 {
		t.Fatalf("expected exactly 1 hub, got %d: %+v", len(hubs), hubs)
	}
	if hubs[0].FilePath != "hub.go" {
		t.Fatalf("hub = %q, want hub.go", hubs[0].FilePath)
	}
	// Only the 3 active importers count; expired + future are filtered out.
	if hubs[0].ImporterCount != 3 {
		t.Fatalf("importer count = %d, want 3 (expired + not-yet-active edges excluded by the temporal window)", hubs[0].ImporterCount)
	}
}

// TestTemporalEdgeExactActiveBoundary is 26.1 [RED] — the window boundary
// itself: an edge whose valid_to == now is ACTIVE (the predicate is
// `valid_to > now`, so valid_to == now is strictly > now? No: now > now is
// false — valid_to == now is EXPIRED at the second granularity). This pins the
// inclusive/exclusive pairing at the exact second: valid_to in the past or
// equal to now is expired; valid_to one second into the future is active.
func TestTemporalEdgeExactActiveBoundary(t *testing.T) {
	st := openHubStore(t, "temporal-edge-exact")
	svc := New(st, "temporal-edge-exact")
	ctx := context.Background()

	now := time.Now().Unix()
	past := now - 3600
	nowPlus1 := now + 1

	hubSym := seedHubFile(t, st, "hub.go")

	// 4 importers so that either the future one being included or the expired
	// one being excluded changes the count by exactly one, making the boundary
	// observable.
	// active (valid_from past, valid_to NULL)        → included
	// expired (valid_to == now)                       → excluded (now > now is false)
	// future (valid_from past, valid_to = now + 1)     → included (now+1 > now)
	// notactive (valid_from = now + 1)                 → excluded (now+1 <= now is false)
	seedImporterEdgeWithWindow(t, st, "a1.go", hubSym, &past, nil)
	seedImporterEdgeWithWindow(t, st, "a2.go", hubSym, &past, nil)
	seedImporterEdgeWithWindow(t, st, "a3.go", hubSym, &past, nil)
	seedImporterEdgeWithWindow(t, st, "exactnow.go", hubSym, &past, &now)         // valid_to == now → expired
	seedImporterEdgeWithWindow(t, st, "future1s.go", hubSym, &past, &nowPlus1)    // valid_to = now+1 → active
	seedImporterEdgeWithWindow(t, st, "notactive1s.go", hubSym, &nowPlus1, nil)   // valid_from = now+1 → not active

	hubs, err := svc.IdentifyHubFiles(ctx)
	if err != nil {
		t.Fatalf("IdentifyHubFiles: %v", err)
	}
	if len(hubs) != 1 {
		t.Fatalf("expected exactly 1 hub, got %d: %+v", len(hubs), hubs)
	}
	// Included: a1, a2, a3, future1s = 4. Excluded: exactnow (valid_to==now),
	// notactive1s (valid_from in the future) = 2.
	if hubs[0].ImporterCount != 4 {
		t.Fatalf("importer count = %d, want 4 (valid_to==now is expired; valid_from in the future is not active; the +1s windows are active)", hubs[0].ImporterCount)
	}
}
