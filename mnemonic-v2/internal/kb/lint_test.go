package kb

import (
	"context"
	"testing"
)

func TestLintMissingTitleAndFix(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	writeVaultPage(t, vault, "decisions/auth-model.md", "We use OAuth2 token rotation.")
	st := openKbStore(t)
	if _, err := Reindex(ctx, st, vault, "proj-a"); err != nil {
		t.Fatalf("reindex: %v", err)
	}

	issues, err := Lint(ctx, st, "proj-a")
	if err != nil {
		t.Fatalf("lint: %v", err)
	}
	found := false
	for _, i := range issues {
		if i.Code == "missing_title" {
			found = true
		}
	}
	if !found {
		t.Fatalf("lint issues = %+v, want missing_title", issues)
	}

	fixed, err := Fix(ctx, st, "proj-a", []string{"decisions/auth-model.md"})
	if err != nil {
		t.Fatalf("fix: %v", err)
	}
	if fixed != 1 {
		t.Fatalf("fixed = %d, want 1", fixed)
	}
}
