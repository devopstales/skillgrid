package kb

import (
	"context"
	"testing"
)

func TestValidateUnknownPageFails(t *testing.T) {
	ctx := context.Background()
	st := openKbStore(t)
	res, err := Validate(ctx, st, "decisions/nope.md", "proj-a")
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if res.OK || res.Reason != "not_found" {
		t.Fatalf("res = %+v, want ok=false reason=not_found", res)
	}
}

func TestValidateUnresolvedPinFails(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	writeVaultPage(t, vault, "decisions/auth-model.md",
		"---\ntitle: Auth Model\ntype: Decision\nstatus: stable\n---\n\nWe use OAuth2 token rotation.")
	st := openKbStore(t)
	if _, err := Reindex(ctx, st, vault, "proj-a"); err != nil {
		t.Fatalf("reindex: %v", err)
	}
	res, err := Validate(ctx, st, "decisions/auth-model.md", "proj-a")
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if res.OK {
		t.Fatalf("validate ok = true, want false (no baseline -> unresolved)")
	}
}
