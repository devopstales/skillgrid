package kb

import (
	"context"
	"testing"
)

func TestQueryReturnsRankedKbPages(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	writeVaultPage(t, vault, "decisions/auth-model.md",
		"---\ntitle: Auth Model\ntype: Decision\nstatus: stable\n---\n\nWe use OAuth2 token rotation.")
	st := openKbStore(t)
	if _, err := Reindex(ctx, st, vault, "proj-a"); err != nil {
		t.Fatalf("reindex: %v", err)
	}
	res, err := Query(ctx, st, "OAuth2", QueryFilters{Project: "proj-a", Limit: 10})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res) != 1 {
		t.Fatalf("query results = %d, want 1", len(res))
	}
	if res[0].RelPath != "decisions/auth-model.md" {
		t.Fatalf("rel_path = %q, want decisions/auth-model.md", res[0].RelPath)
	}
}

func TestQueryNoMatchReturnsEmpty(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	writeVaultPage(t, vault, "decisions/auth-model.md",
		"---\ntitle: Auth Model\ntype: Decision\nstatus: stable\n---\n\nWe use OAuth2.")
	st := openKbStore(t)
	if _, err := Reindex(ctx, st, vault, "proj-a"); err != nil {
		t.Fatalf("reindex: %v", err)
	}
	res, err := Query(ctx, st, "zzzznotathing", QueryFilters{Project: "proj-a", Limit: 10})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(res) != 0 {
		t.Fatalf("query results = %d, want 0", len(res))
	}
}
