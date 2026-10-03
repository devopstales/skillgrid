package codeindex

import (
	"bytes"
	"testing"
)

// TestRecordUnresolvedDefersPrintWhenProgress: a live progress callback means
// a TUI owns the terminal. The unresolved-member line must not be written
// then; the count is stored for the caller to print after restore.
func TestRecordUnresolvedDefersPrintWhenProgress(t *testing.T) {
	idx, clean := newTestIndexer(t)
	defer clean()
	var stats Stats
	var buf bytes.Buffer
	recordUnresolvedMembers(idx.store.DB, func(Event) {}, &stats, &buf)
	if buf.Len() != 0 {
		t.Fatalf("wrote %q while the progress UI owns the terminal", buf.String())
	}
	if !stats.UnresolvedKnown || stats.UnresolvedMembers != 0 {
		t.Fatalf("stats = %+v, want a known count of 0", stats)
	}
}

// TestRecordUnresolvedPrintsWhenNoProgress: without a progress UI the line
// is the operator status and is written immediately, including a zero count.
func TestRecordUnresolvedPrintsWhenNoProgress(t *testing.T) {
	idx, clean := newTestIndexer(t)
	defer clean()
	var stats Stats
	var buf bytes.Buffer
	recordUnresolvedMembers(idx.store.DB, nil, &stats, &buf)
	if got := buf.String(); got != "index: unresolved member calls: 0\n" {
		t.Fatalf("stderr = %q", got)
	}
	if !stats.UnresolvedKnown || stats.UnresolvedMembers != 0 {
		t.Fatalf("stats = %+v, want a known count of 0", stats)
	}
}
