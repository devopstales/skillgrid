package kb

import (
	"context"
	"testing"
)

func TestStatusReportsIndexedPages(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	writeVaultPage(t, vault, "decisions/auth-model.md",
		"---\ntitle: Auth Model\ntype: Decision\nstatus: stable\n---\n\nWe use OAuth2 token rotation.")
	st := openKbStore(t)
	if _, err := Reindex(ctx, st, vault, "proj-a"); err != nil {
		t.Fatalf("reindex: %v", err)
	}
	res, err := Status(ctx, st, "proj-a")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if res.Indexed != 1 {
		t.Fatalf("indexed = %d, want 1", res.Indexed)
	}
	if res.Phase == "" {
		t.Fatalf("phase = %q, want non-empty", res.Phase)
	}
}
