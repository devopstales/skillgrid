package webcache

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/config"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

func newFixture(t *testing.T, ttl map[string]time.Duration) *Service {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir, "web-test")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	wc := config.WebCache{
		Enabled:       true,
		MaxEntryBytes: 262144,
		TTL:           ttl,
		Sources:       []string{"context7", "exa", "deepwiki", "fetch", "manual"},
	}
	return New(st, "web-test", wc)
}

// TestSaveAndLookup round-trips a save and a lookup hit.
func TestSaveAndLookup(t *testing.T) {
	svc := newFixture(t, map[string]time.Duration{"context7": 30 * 24 * time.Hour})
	ctx := context.Background()
	in := SaveWebInput{
		Source:    "context7",
		Content:   "docs body",
		LibraryID: "/org/proj",
		Query:     "auth setup",
	}
	id, err := svc.Save(ctx, in)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive id, got %d", id)
	}
	res, err := svc.Lookup(ctx, LookupInput{
		Source:    "context7",
		LibraryID: "/org/proj",
		Query:     "auth setup",
	})
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if res.Status != "hit" {
		t.Errorf("expected hit, got %q", res.Status)
	}
	if !res.Fresh {
		t.Errorf("expected fresh, got stale")
	}
}

// TestLookupMiss returns a miss status when nothing is cached.
func TestLookupMiss(t *testing.T) {
	svc := newFixture(t, map[string]time.Duration{"context7": 72 * time.Hour})
	res, err := svc.Lookup(context.Background(), LookupInput{
		Source:    "context7",
		LibraryID: "/nope",
		Query:     "nope",
	})
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if res.Status != "miss" {
		t.Errorf("expected miss, got %q", res.Status)
	}
}

// TestStaleDetection returns stale when an entry is past TTL.
func TestStaleDetection(t *testing.T) {
	// TTL of 1 second: expired ~1s after save.
	svc := newFixture(t, map[string]time.Duration{"context7": time.Second})
	ctx := context.Background()
	if _, err := svc.Save(ctx, SaveWebInput{Source: "context7", Content: "x", LibraryID: "/org", Query: "q"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	time.Sleep(1100 * time.Millisecond)
	res, err := svc.Lookup(ctx, LookupInput{Source: "context7", LibraryID: "/org", Query: "q"})
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if res.Status != "stale" {
		t.Errorf("expected stale, got %q", res.Status)
	}
	if res.Fresh {
		t.Errorf("expected fresh=false for stale entry")
	}
}

// TestDedup verifies saving the same (project, source, cache_key) twice returns
// the same id (upsert).
func TestDedup(t *testing.T) {
	svc := newFixture(t, map[string]time.Duration{"exa": 7 * 24 * time.Hour})
	ctx := context.Background()
	in := SaveWebInput{Source: "exa", Content: "first", Query: "search-q"}
	id1, err := svc.Save(ctx, in)
	if err != nil {
		t.Fatalf("save1: %v", err)
	}
	in2 := SaveWebInput{Source: "exa", Content: "second", Query: "search-q"}
	id2, err := svc.Save(ctx, in2)
	if err != nil {
		t.Fatalf("save2: %v", err)
	}
	if id1 != id2 {
		t.Errorf("expected same id for same cache_key, got %d and %d", id1, id2)
	}
	// Content should be updated to the latest.
	entry, err := svc.Get(ctx, id1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if entry.Content != "second" {
		t.Errorf("expected content 'second', got %q", entry.Content)
	}
}

// TestEntrySizeCap rejects content over the 256KB cap.
func TestEntrySizeCap(t *testing.T) {
	svc := newFixture(t, map[string]time.Duration{"fetch": 7 * 24 * time.Hour})
	ctx := context.Background()
	big := strings.Repeat("a", 262145) // one byte over the cap
	_, err := svc.Save(ctx, SaveWebInput{Source: "fetch", Content: big, URL: "https://example.com"})
	if err == nil {
		t.Fatalf("expected error for oversized content")
	}
	if !strings.Contains(err.Error(), "max entry size") {
		t.Errorf("error should name the size cap: %v", err)
	}
}

// TestCacheKeyPerSource verifies each source requires its key fields and
// produces distinct keys for different sources.
func TestCacheKeyPerSource(t *testing.T) {
	cases := []struct {
		name    string
		source  string
		in      KeyInput
		wantErr bool
	}{
		{"fetch needs url", "fetch", KeyInput{}, true},
		{"fetch with url", "fetch", KeyInput{URL: "https://x.com/a"}, false},
		{"exa needs query", "exa", KeyInput{}, true},
		{"context7 needs lib+query", "context7", KeyInput{LibraryID: "/o/p"}, true},
		{"context7 ok", "context7", KeyInput{LibraryID: "/o/p", Query: "q"}, false},
		{"deepwiki needs repo+q", "deepwiki", KeyInput{RepoName: "o/r"}, true},
		{"deepwiki ok", "deepwiki", KeyInput{RepoName: "o/r", Question: "q"}, false},
		{"manual needs title+hash", "manual", KeyInput{Title: "t"}, true},
		{"manual ok", "manual", KeyInput{Title: "t", ContentHash: "h"}, false},
		{"unknown source", "banana", KeyInput{URL: "https://x"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			k, err := CacheKey(c.source, c.in)
			if c.wantErr && err == nil {
				t.Errorf("expected error, got key %q", k)
			}
			if !c.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !c.wantErr && k == "" {
				t.Errorf("expected non-empty key")
			}
		})
	}
}

// TestSearchOverCached runs FTS search over cached entries with + without
// fresh_only filter.
func TestSearchOverCached(t *testing.T) {
	// context7 has a 1s TTL (goes stale after the wait); exa is long-lived.
	svc := newFixture(t, map[string]time.Duration{
		"context7": time.Second,
		"exa":      7 * 24 * time.Hour,
	})
	ctx := context.Background()
	if _, err := svc.Save(ctx, SaveWebInput{Source: "context7", Content: "react auth docs", LibraryID: "/v", Query: "react auth"}); err != nil {
		t.Fatalf("save ctx7: %v", err)
	}
	if _, err := svc.Save(ctx, SaveWebInput{Source: "exa", Content: "express auth article", Query: "express auth"}); err != nil {
		t.Fatalf("save exa: %v", err)
	}
	time.Sleep(1100 * time.Millisecond) // let the context7 entry expire
	all, err := svc.Search(ctx, "auth", "", false, 20)
	if err != nil {
		t.Fatalf("search all: %v", err)
	}
	if len(all) < 2 {
		t.Errorf("expected at least 2 results (stale included), got %d", len(all))
	}
	fresh, err := svc.Search(ctx, "auth", "", true, 20)
	if err != nil {
		t.Fatalf("search fresh: %v", err)
	}
	if len(fresh) >= len(all) {
		t.Errorf("expected fresh_only to drop the stale entry, got %d >= %d", len(fresh), len(all))
	}
}

// TestGetReturnsFullEntry verifies the full content body is returned.
func TestGetReturnsFullEntry(t *testing.T) {
	svc := newFixture(t, map[string]time.Duration{"manual": 0})
	ctx := context.Background()
	body := "full cached snapshot body"
	id, err := svc.Save(ctx, SaveWebInput{Source: "manual", Content: body, Title: "my-note"})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	entry, err := svc.Get(ctx, id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if entry.Content != body {
		t.Errorf("content mismatch: %q", entry.Content)
	}
	if entry.Source != "manual" {
		t.Errorf("source mismatch: %q", entry.Source)
	}
}

// TestCacheStatus returns aggregate stats.
func TestCacheStatus(t *testing.T) {
	svc := newFixture(t, map[string]time.Duration{"exa": time.Second, "fetch": 7 * 24 * time.Hour})
	ctx := context.Background()
	if _, err := svc.Save(ctx, SaveWebInput{Source: "exa", Content: "a", Query: "q1"}); err != nil {
		t.Fatalf("save1: %v", err)
	}
	if _, err := svc.Save(ctx, SaveWebInput{Source: "fetch", Content: "b", URL: "https://x.com"}); err != nil {
		t.Fatalf("save2: %v", err)
	}
	time.Sleep(1100 * time.Millisecond) // let the exa entry expire
	st, err := svc.CacheStatus(ctx)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if st.TotalEntries != 2 {
		t.Errorf("expected TotalEntries 2, got %d", st.TotalEntries)
	}
	if st.ExpiredEntries != 1 {
		t.Errorf("expected 1 expired, got %d", st.ExpiredEntries)
	}
}
