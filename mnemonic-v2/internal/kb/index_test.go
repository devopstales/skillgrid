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

func TestPageTitleFallback(t *testing.T) {
	if got := pageTitle("---\ntitle: FM Title\n---\n# H1\n"); got != "FM Title" {
		t.Fatalf("frontmatter: got %q", got)
	}
	if got := pageTitle("# Heading One\n\nbody\n"); got != "Heading One" {
		t.Fatalf("h1: got %q", got)
	}
	if got := pageTitle("---\ntype: Decision\n---\n# H1 Title\n"); got != "H1 Title" {
		t.Fatalf("h1 after fm: got %q", got)
	}
	if got := pageTitle("no heading here\n"); got != "" {
		t.Fatalf("empty: got %q", got)
	}
}

func TestCollectPagesWhitelistScope(t *testing.T) {
	vault := t.TempDir()
	writeVaultPage(t, vault, "decisions/auth-model.md", "body a")
	writeVaultPage(t, vault, "invariants/grace.md", "body b")
	writeVaultPage(t, vault, "index/catalog.md", "derived projection")
	writeVaultPage(t, vault, "templates/decision.md", "a template")
	writeVaultPage(t, vault, "ARCHITECTURE.md", "vault-root doc")
	writeVaultPage(t, vault, "KNOWLEDGE_SCHEMA.md", "schema")
	writeVaultPage(t, vault, "trajectories/TRJ-2026-01-01-001/extracted.md", "packet")
	writeVaultPage(t, vault, ".obsidian/app.md", "obsidian state")
	writeVaultPage(t, vault, "gotchas/skipme.md", "ignored by .knowledgeignore")
	if err := os.WriteFile(filepath.Join(vault, ".knowledgeignore"),
		[]byte("# boundary\ngotchas/skipme.md\n"), 0o644); err != nil {
		t.Fatalf("write ignore: %v", err)
	}

	pages, err := collectPages(vault)
	if err != nil {
		t.Fatalf("collectPages: %v", err)
	}
	got := map[string]bool{}
	for _, p := range pages {
		got[p.RelPath] = true
	}
	if len(pages) != 2 {
		t.Fatalf("collected %d pages, want 2: %v", len(pages), got)
	}
	if !got["decisions/auth-model.md"] || !got["invariants/grace.md"] {
		t.Fatalf("in-scope pages missing: %v", got)
	}
}
