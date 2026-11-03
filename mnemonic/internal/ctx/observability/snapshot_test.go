package observability

// [history-snapshot-durable] The durable JSON snapshot mirrors
// ctx_stats_history, survives a "re-index" (an external process rewriting
// other project files), and matches the DB rollups.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	ctxstore "github.com/devopstales/skillgrid/mnemonic/internal/ctx/store"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// openObsStore opens a migrated in-process store (migrations applied in
// Open) and wraps it in a ctx store. Returns the ctx store + cleanup.
func openObsStore(t *testing.T) (*ctxstore.Store, func()) {
	t.Helper()
	shared, err := store.Open(t.TempDir(), "obs-snapshot-test")
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = shared.Close() })
	return ctxstore.New(shared), func() {}
}

// TestSnapshotMatchesDBRollups verifies the written file equals the DB
// rollups: seed a bucket, write the snapshot, read the file, compare.
func TestSnapshotMatchesDBRollups(t *testing.T) {
	ts, cleanup := openObsStore(t)
	defer cleanup()

	// Seed one known bucket into ctx_stats_history.
	start := windowStart(Hourly, time.Now())
	seed := &ctxstore.StatsBucket{
		Granularity: "hourly",
		BucketID:    bucketKey(Hourly, start),
		BucketStart: start,
		TokensSaved: 1234,
		Requests:    7,
		Sessions:    2,
		CacheHits:   3,
	}
	if err := ts.PutStatsBucket(seed); err != nil {
		t.Fatalf("PutStatsBucket: %v", err)
	}

	path := filepath.Join(t.TempDir(), ".skillgrid", "ctx", "stats_history.json")
	if err := WriteHistorySnapshot(ts, path, nil); err != nil {
		t.Fatalf("WriteHistorySnapshot: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	var got History
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}
	buckets := got.Buckets[Hourly]
	if len(buckets) != 1 {
		t.Fatalf("hourly buckets = %d, want 1: %+v", len(buckets), buckets)
	}
	b := buckets[0]
	if b.TokensSaved != 1234 || b.Requests != 7 || b.Sessions != 2 || b.CacheHits != 3 {
		t.Fatalf("snapshot bucket = %+v, want the seeded values", b)
	}
}

// TestSnapshotSurvivesReIndex verifies the file is intact after an external
// "re-index" (other project files rewritten) and still matches the DB.
func TestSnapshotSurvivesReIndex(t *testing.T) {
	ts, cleanup := openObsStore(t)
	defer cleanup()

	start := windowStart(Hourly, time.Now())
	seed := &ctxstore.StatsBucket{
		Granularity: "hourly",
		BucketID:    bucketKey(Hourly, start),
		BucketStart: start,
		TokensSaved: 999,
		Requests:    1,
		Sessions:    1,
		CacheHits:   0,
	}
	if err := ts.PutStatsBucket(seed); err != nil {
		t.Fatalf("PutStatsBucket: %v", err)
	}

	dir := filepath.Join(t.TempDir(), ".skillgrid")
	path := filepath.Join(dir, "ctx", "stats_history.json")
	if err := WriteHistorySnapshot(ts, path, nil); err != nil {
		t.Fatalf("WriteHistorySnapshot: %v", err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}

	// Simulate a re-index: an external process rewrites sibling project files
	// (and even recreates the .skillgrid tree elsewhere) — the snapshot file
	// itself must remain intact.
	if err := os.MkdirAll(filepath.Join(dir, "archive"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "archive", "something.md"), []byte("re-indexed"), 0o644); err != nil {
		t.Fatal(err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("snapshot missing after re-index: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("snapshot changed during re-index:\nbefore: %s\nafter: %s", before, after)
	}
}

// TestSnapshotFailOpenNilStore verifies a nil store still yields a valid
// (all-empty) snapshot file — ADR-0016.
func TestSnapshotFailOpenNilStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats_history.json")
	if err := WriteHistorySnapshot(nil, path, nil); err != nil {
		t.Fatalf("WriteHistorySnapshot(nil): %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	var got History
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.Buckets) == 0 {
		t.Fatalf("expected the default granularities present (possibly empty), got %+v", got.Buckets)
	}
	for g, buckets := range got.Buckets {
		if len(buckets) != 0 {
			t.Fatalf("granularity %s should be empty on nil store, got %d", g, len(buckets))
		}
	}
}
