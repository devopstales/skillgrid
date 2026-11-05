package kb

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateUnknownPageFails(t *testing.T) {
	ctx := context.Background()
	st := openKbStore(t)
	res, err := Validate(ctx, st, t.TempDir(), t.TempDir(), "decisions/nope.md", "proj-a")
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
	// Grounded page, no anchors/claims → unattested → unresolved.
	writeVaultPage(t, vault, "decisions/auth-model.md",
		"---\ntitle: Auth Model\ntype: Decision\nstatus: stable\n---\n\nWe use OAuth2 token rotation.")
	st := openKbStore(t)
	res, err := Validate(ctx, st, vault, vault, "decisions/auth-model.md", "proj-a")
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if res.OK || res.Reason != "unresolved" {
		t.Fatalf("res = %+v, want ok=false reason=unresolved", res)
	}
}

func TestValidateFreshPagePasses(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	writeVaultPage(t, vault, "internal/auth/oauth.go", "package auth\n\nconst expiry = 15 * 60\n")
	writeVaultPage(t, vault, "decisions/auth-model.md",
		"---\n"+
			"title: Auth Model\n"+
			"type: Decision\n"+
			"anchors:\n"+
			"  - path: internal/auth/oauth.go\n"+
			"    all_of:\n"+
			"      - const expiry = 15 * 60\n"+
			"---\n\nWe use OAuth2 token rotation.")
	st := openKbStore(t)
	res, err := Validate(ctx, st, vault, vault, "decisions/auth-model.md", "proj-a")
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !res.OK || res.Reason != "fresh" {
		t.Fatalf("res = %+v, want ok=true reason=fresh", res)
	}
}

func TestValidateStaleClaimFails(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	// The claimed literal no longer holds in the anchor site.
	writeVaultPage(t, vault, "internal/cache/cache.go", "package cache\n\nvar ttl = 30 * time.Minute\n")
	writeVaultPage(t, vault, "decisions/cache-ttl.md",
		"---\n"+
			"title: Cache TTL\n"+
			"type: Decision\n"+
			"anchors:\n"+
			"  - path: internal/cache/cache.go\n"+
			"    all_of:\n"+
			"      - var ttl\n"+
			"claims:\n"+
			"  - subject: ttl value\n"+
			"    literal: 15 * time.Minute\n"+
			"---\n\nWe cache for 15 minutes.")
	st := openKbStore(t)
	res, err := Validate(ctx, st, vault, vault, "decisions/cache-ttl.md", "proj-a")
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if res.OK || res.Reason != "stale" {
		t.Fatalf("res = %+v, want ok=false reason=stale", res)
	}
}

func TestValidateBaselineFastPath(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	// A page whose pin would FAIL if re-checked (anchor file missing), but a
	// current baseline was recorded → the fast path proves it without
	// re-running the checks.
	writeVaultPage(t, vault, "decisions/auth-model.md",
		"---\n"+
			"title: Auth Model\n"+
			"type: Decision\n"+
			"anchors:\n"+
			"  - path: internal/auth/oauth.go\n"+
			"    all_of:\n"+
			"      - func NewOAuthClient\n"+
			"---\n\nWe use OAuth2 token rotation.")
	st := openKbStore(t)
	fp, err := SourceFingerprint(ctx, st, "proj-a", vault)
	if err != nil {
		t.Fatalf("fp: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(vault, "decisions", "auth-model.md"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := RecordBaseline(ctx, st, vault, "proj-a", "decisions/auth-model.md",
		fp, hashBody(string(body)), "test", "test"); err != nil {
		t.Fatalf("baseline: %v", err)
	}

	res, err := Validate(ctx, st, vault, vault, "decisions/auth-model.md", "proj-a")
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !res.OK || res.Reason != "baseline_current" {
		t.Fatalf("res = %+v, want ok=true reason=baseline_current", res)
	}

	// A changed page byte no longer matches the baseline → the re-check runs
	// and fails (anchor file missing → unresolved).
	body = append(body, []byte("edited\n")...)
	if err := os.WriteFile(filepath.Join(vault, "decisions", "auth-model.md"), body, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	res, err = Validate(ctx, st, vault, vault, "decisions/auth-model.md", "proj-a")
	if err != nil {
		t.Fatalf("validate 2: %v", err)
	}
	if res.OK || res.Reason != "unresolved" {
		t.Fatalf("res = %+v, want ok=false reason=unresolved (page drift past baseline)", res)
	}
}

func TestValidateNonGroundedPageSkipsPinCheck(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	writeVaultPage(t, vault, "gotchas/retry-storm.md",
		"---\ntitle: Retry Storm\ntype: Gotcha\n---\n\nRetrying on 502 amplifies the outage.")
	st := openKbStore(t)
	res, err := Validate(ctx, st, vault, vault, "gotchas/retry-storm.md", "proj-a")
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !res.OK || res.Reason != "not_grounded" {
		t.Fatalf("res = %+v, want ok=true reason=not_grounded", res)
	}
}
