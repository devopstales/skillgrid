package codeindex

import "testing"

// TestRecordUnresolvedStoresCountWithoutPrinting: the count is for
// tools/stats. An "unresolved" line on stderr reads as a failed run.
func TestRecordUnresolvedStoresCountWithoutPrinting(t *testing.T) {
	idx, clean := newTestIndexer(t)
	defer clean()
	var stats Stats
	recordUnresolvedMembers(idx.store.DB, &stats)
	if !stats.UnresolvedKnown || stats.UnresolvedMembers != 0 {
		t.Fatalf("stats = %+v, want a known count of 0", stats)
	}
}
