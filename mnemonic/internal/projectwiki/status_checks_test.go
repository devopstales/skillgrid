package projectwiki

import (
	"context"
	"testing"
)

// W006 (QA 2026-11-10): status reports the blueprint T7 breakdown —
// page counts by type, stale count, pending queue, orphan count.

func TestStatusBreakdown(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	writeVaultPage(t, vault, "decisions/auth-model.md",
		"---\ntitle: Auth Model\ntype: Decision\nstatus: stable\n---\n\nWe use OAuth2.")
	writeVaultPage(t, vault, "architecture/overview.md",
		"---\ntitle: Overview\ntype: Architecture\nstatus: stable\n---\n\nSee [[decisions/auth-model]].\n")
	st := openProjectWikiStore(t)
	if _, err := Reindex(ctx, st, vault, "proj-a"); err != nil {
		t.Fatalf("reindex: %v", err)
	}

	res, err := Status(ctx, st, vault, "proj-a")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if res.ByType["Decision"] != 1 || res.ByType["Architecture"] != 1 {
		t.Fatalf("by_type = %v, want Decision:1 Architecture:1", res.ByType)
	}
	// auth-model has 1 incoming link (<=1 → orphan); overview has 0.
	if res.Orphans != 2 {
		t.Fatalf("orphans = %d, want 2 (in_degree <= 1)", res.Orphans)
	}
	if res.Stale != 0 {
		t.Fatalf("stale = %d, want 0 right after reindex", res.Stale)
	}
	if res.Pending != 0 {
		t.Fatalf("pending = %d, want 0 (no active run)", res.Pending)
	}

	// Mutate one file without re-indexing → stale count rises.
	writeVaultPage(t, vault, "decisions/auth-model.md",
		"---\ntitle: Auth Model\ntype: Decision\nstatus: stable\n---\n\nWe use OAuth2 with rotation.")
	res, err = Status(ctx, st, vault, "proj-a")
	if err != nil {
		t.Fatalf("status 2: %v", err)
	}
	if res.Stale != 1 {
		t.Fatalf("stale = %d, want 1 after vault mutation", res.Stale)
	}
}
