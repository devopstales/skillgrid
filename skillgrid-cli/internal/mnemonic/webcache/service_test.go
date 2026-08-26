package webcache_test

import (
	"context"
	"testing"

	"skillgrid-cli/internal/mnemonic/config"
	"skillgrid-cli/internal/mnemonic/store"
	"skillgrid-cli/internal/mnemonic/webcache"
)

const testProject = "test-project"

func openTestService(t *testing.T) *webcache.Service {
	t.Helper()

	dir := t.TempDir()
	st, err := store.Open(dir, testProject)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })

	cfg := config.DefaultIndexing().WebCache
	return webcache.New(st, testProject, cfg)
}

func TestWebCacheContext7RoundTrip(t *testing.T) {
	svc := openTestService(t)
	ctx := context.Background()

	id, err := svc.Save(ctx, webcache.SaveWebInput{
		Source:     "context7",
		LibraryID:  "/vercel/next.js",
		VersionTag: "v15",
		Query:      "middleware",
		Title:      "Next.js middleware",
		Content:    "Middleware runs before routes and can rewrite requests.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id <= 0 {
		t.Fatalf("expected positive id, got %d", id)
	}

	lookup, err := svc.Lookup(ctx, webcache.LookupInput{
		Source:     "context7",
		LibraryID:  "/vercel/next.js",
		VersionTag: "v15",
		Query:      "middleware",
	})
	if err != nil {
		t.Fatal(err)
	}
	if lookup.Status != "hit" {
		t.Fatalf("status=%q want hit", lookup.Status)
	}
	if !lookup.Fresh {
		t.Fatal("expected fresh=true on new entry")
	}
	if lookup.ID != id {
		t.Fatalf("lookup id=%d save id=%d", lookup.ID, id)
	}

	hits, err := svc.Search(ctx, "middleware", "", false, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	if hits[0].Query != "middleware" {
		t.Fatalf("query=%q", hits[0].Query)
	}
}
