package kb

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

func writeVaultPage(t *testing.T, root, relPath, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func openKbStore(t *testing.T) store.Store {
	t.Helper()
	st, err := store.Open(context.Background(), t.TempDir(), "proj-a", store.SQLite, "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestReindexRebuildsKbPages(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	writeVaultPage(t, vault, "decisions/auth-model.md",
		"---\ntitle: Auth Model\ntype: Decision\nstatus: stable\n---\n\nWe use OAuth2.")
	st := openKbStore(t)

	res, err := Reindex(ctx, st, vault, "proj-a")
	if err != nil {
		t.Fatalf("reindex: %v", err)
	}
	if res.Scanned != 1 {
		t.Fatalf("scanned = %d, want 1", res.Scanned)
	}
	if res.Changed != 1 {
		t.Fatalf("changed = %d, want 1", res.Changed)
	}
}

func TestReindexIncrementalOnlyChanged(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	writeVaultPage(t, vault, "decisions/auth-model.md",
		"---\ntitle: Auth Model\ntype: Decision\nstatus: stable\n---\n\nWe use OAuth2.")

	st := openKbStore(t)
	if _, err := Reindex(ctx, st, vault, "proj-a"); err != nil {
		t.Fatalf("reindex: %v", err)
	}

	writeVaultPage(t, vault, "decisions/auth-model.md",
		"---\ntitle: Auth Model\ntype: Decision\nstatus: stable\n---\n\nWe use SAML.")
	res, err := Reindex(ctx, st, vault, "proj-a")
	if err != nil {
		t.Fatalf("reindex: %v", err)
	}
	if res.Scanned != 1 || res.Changed != 1 {
		t.Fatalf("res = %+v, want scanned=1 changed=1", res)
	}
}
