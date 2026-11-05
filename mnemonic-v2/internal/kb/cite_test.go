package kb

import (
	"context"
	"testing"
)

func TestCiteSourceReturnsContent(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	writeVaultPage(t, vault, "decisions/auth-model.md",
		"---\ntitle: Auth Model\ntype: Decision\nstatus: stable\n---\n\nWe use OAuth2 token rotation.")
	st := openKbStore(t)
	if _, err := Reindex(ctx, st, vault, "proj-a"); err != nil {
		t.Fatalf("reindex: %v", err)
	}
	res, err := CiteSource(ctx, st, "proj-a", "decisions/auth-model.md")
	if err != nil {
		t.Fatalf("cite: %v", err)
	}
	if res.Source != "decisions/auth-model.md" {
		t.Fatalf("source = %q", res.Source)
	}
	if res.Content == "" {
		t.Fatalf("content = %q, want non-empty", res.Content)
	}
}
