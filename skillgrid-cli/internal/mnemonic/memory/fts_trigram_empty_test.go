package memory

import (
	"context"
	"testing"
)

// TestFTSTrigramEmptyResultsFallback is 26.1 [RED] — a trigram-mode FTS search
// whose trigram groups match NO stored observation returns an empty result set
// (`len == 0`), not an error. The trigram query builder turns the term into
// OR-joined double-quoted trigram groups (see
// TestBuildFTSQueryTrigramMode); when none of those groups match any row,
// FTS5's MATCH yields zero rows and the read path must surface that as the
// graceful empty floor the agent relies on. A regression that treats an empty
// FTS match as an error (e.g. a missing-row scan failure) or that
// short-circuits before the query would break the "search returns nothing, not
// a failure" contract.
func TestFTSTrigramEmptyResultsFallback(t *testing.T) {
	_, svc := newTestStore(t, "fts-trigram-empty")
	sid := newSession(t, svc)
	ctx := context.Background()

	// Seed an observation that contains a real word (for the healthy-path
	// sanity check below).
	if _, err := svc.Save(ctx, SaveInput{
		Title: "trigram seed", Type: "decision", Content: "we always run gofmt before commit",
		SessionID: sid, Scope: "project",
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	// The default (phrase) mode still finds the seeded row — this proves the
	// store is seeded and the read path is healthy, so a trigram query that
	// returns 0 rows below is a genuine "no FTS match", not a cold store.
	positive, err := svc.Search(ctx, "commit", "", 10)
	if err != nil {
		t.Fatalf("positive phrase search: %v", err)
	}
	if len(positive) == 0 {
		t.Fatalf("expected the seeded row for a phrase query, got 0")
	}

	// A term absent from every observation: FTS matches nothing → empty, not an
	// error.
	empty, err := svc.Search(ctx, "zzqqxy", "trigram", 10)
	if err != nil {
		t.Fatalf("empty trigram search must return nil error, got: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected 0 hits for a non-matching trigram term, got %d: %+v", len(empty), empty)
	}

	// The owner-scoped seam (the mem_search production path) must honor the same
	// empty floor.
	emptyOwner, err := svc.SearchOwnerScoped(ctx, "owner-a", "agent", "zzqqxy", "trigram", "project", 10)
	if err != nil {
		t.Fatalf("empty owner-scoped trigram search must return nil error, got: %v", err)
	}
	if len(emptyOwner) != 0 {
		t.Fatalf("owner-scoped: expected 0 hits for a non-matching trigram term, got %d", len(emptyOwner))
	}
}
