package webcache_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"skillgrid-cli/internal/mnemonic/config"
	"skillgrid-cli/internal/mnemonic/store"
	"skillgrid-cli/internal/mnemonic/webcache"
)

const testProject = "test-project"

type testEnv struct {
	svc *webcache.Service
	st  *store.Store
}

func openTestEnv(t *testing.T) testEnv {
	t.Helper()

	dir := t.TempDir()
	st, err := store.Open(dir, testProject)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })

	cfg := config.DefaultIndexing().WebCache
	return testEnv{svc: webcache.New(st, testProject, cfg), st: st}
}

func openTestService(t *testing.T) *webcache.Service {
	return openTestEnv(t).svc
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

func TestWebCacheStaleLookup(t *testing.T) {
	env := openTestEnv(t)
	ctx := context.Background()

	id, err := env.svc.Save(ctx, webcache.SaveWebInput{
		Source:     "context7",
		LibraryID:  "/vercel/next.js",
		VersionTag: "v15",
		Query:      "stale-test",
		Title:      "Stale entry",
		Content:    "Content that should become stale.",
	})
	if err != nil {
		t.Fatal(err)
	}

	expiredAt := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	if _, err := env.st.DB.ExecContext(ctx, `
		UPDATE web_cache SET expires_at = ? WHERE id = ?`,
		expiredAt, id,
	); err != nil {
		t.Fatal(err)
	}

	lookup, err := env.svc.Lookup(ctx, webcache.LookupInput{
		Source:     "context7",
		LibraryID:  "/vercel/next.js",
		VersionTag: "v15",
		Query:      "stale-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if lookup.Status != "stale" {
		t.Fatalf("status=%q want stale", lookup.Status)
	}
	if lookup.Fresh {
		t.Fatal("expected fresh=false for expired entry")
	}
	if lookup.ID != id {
		t.Fatalf("lookup id=%d save id=%d", lookup.ID, id)
	}
}

func TestWebCacheContentTooLarge(t *testing.T) {
	svc := openTestService(t)
	ctx := context.Background()

	maxBytes := config.DefaultIndexing().WebCache.MaxEntryBytes
	oversized := strings.Repeat("x", maxBytes+1)

	_, err := svc.Save(ctx, webcache.SaveWebInput{
		Source:  "manual",
		Title:   "Too large",
		Content: oversized,
	})
	if err == nil {
		t.Fatal("expected error for oversized content")
	}
	if !errors.Is(err, webcache.ErrContentTooLarge) {
		t.Fatalf("err=%v want ErrContentTooLarge", err)
	}
}
