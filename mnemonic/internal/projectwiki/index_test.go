package projectwiki

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/core/store"
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

func openProjectWikiStore(t *testing.T) store.Store {
	t.Helper()
	st, err := store.Open(context.Background(), t.TempDir(), "proj-a", store.SQLite, "")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestReindexRebuildsProjectWikiPages(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	// Two pages: the second links to the first and carries an alias, so
	// the rebuild's derived state (in_degree, aliases) is observable.
	writeVaultPage(t, vault, "decisions/auth-model.md",
		"---\ntitle: Auth Model\ntype: Decision\nstatus: stable\n---\n\nWe use OAuth2.")
	writeVaultPage(t, vault, "architecture/flows.md",
		"---\ntitle: Flows\ntype: Architecture\nstatus: stable\naliases: [flow-map]\n"+
			"---\n\nSee [[decisions/auth-model]].\n")
	st := openProjectWikiStore(t)

	res, err := Reindex(ctx, st, vault, "proj-a")
	if err != nil {
		t.Fatalf("reindex: %v", err)
	}
	if res.Scanned != 2 {
		t.Fatalf("scanned = %d, want 2", res.Scanned)
	}
	if res.Changed != 2 {
		t.Fatalf("changed = %d, want 2", res.Changed)
	}

	// W003 root fix: Reindex must write the same derived state as the
	// incremental Index (upsertPageFull + recomputeInDegree) — not the
	// primitive writer's aliases=[]/in_degree=0.
	rows, err := st.Query(ctx,
		`SELECT body, body_hash, aliases, in_degree FROM projectwiki_pages WHERE project = 'proj-a' AND deleted_at IS NULL ORDER BY rel_path`,
	)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("indexed rows = %d, want 2", len(rows))
	}
	for _, r := range rows {
		body := strVal(r, 0)
		if body == "" {
			t.Fatalf("stored body empty for %v", r)
		}
		sum := sha256.Sum256([]byte(body))
		if strVal(r, 1) != hex.EncodeToString(sum[:]) {
			t.Fatalf("body_hash %q != sha256(body)", strVal(r, 1))
		}
	}
	// rows are ordered by rel_path: architecture/flows.md first,
	// decisions/auth-model.md second (the link target).
	aliases := strVal(rows[0], 2)
	if !strings.Contains(aliases, "flow-map") {
		t.Fatalf("reindex lost the alias: %q", aliases)
	}
	if deg, ok := rows[1][3].(int64); !ok || deg < 1 {
		t.Fatalf("link target in_degree = %v, want >= 1 (recomputeInDegree)", rows[1][3])
	}
}

func TestReindexIncrementalOnlyChanged(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	writeVaultPage(t, vault, "decisions/auth-model.md",
		"---\ntitle: Auth Model\ntype: Decision\nstatus: stable\n---\n\nWe use OAuth2.")

	st := openProjectWikiStore(t)
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
	writeVaultPage(t, vault, "PROJECTWIKI_SCHEMA.md", "schema")
	writeVaultPage(t, vault, "trajectories/TRJ-2026-01-01-001/extracted.md", "packet")
	writeVaultPage(t, vault, ".obsidian/app.md", "obsidian state")
	writeVaultPage(t, vault, "gotchas/skipme.md", "ignored by .ignore")
	writeVaultPage(t, vault, "sources/context7-embed-rs-v0.4.md", "source summary")
	writeVaultPage(t, vault, "concepts/fence-sizing.md", "a concept")
	writeVaultPage(t, vault, "syntheses/mcp-seriality.md", "a result hub")
	writeVaultPage(t, vault, "preferences/tooling.md", "a preference")
	if err := os.WriteFile(filepath.Join(vault, ".ignore"),
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
	if len(pages) != 6 {
		t.Fatalf("collected %d pages, want 6: %v", len(pages), got)
	}
	if !got["decisions/auth-model.md"] || !got["invariants/grace.md"] {
		t.Fatalf("in-scope pages missing: %v", got)
	}
	// 013 (ADR-0060): the four contextual type folders are in scope.
	for _, p := range []string{
		"sources/context7-embed-rs-v0.4.md",
		"concepts/fence-sizing.md",
		"syntheses/mcp-seriality.md",
		"preferences/tooling.md",
	} {
		if !got[p] {
			t.Fatalf("013 contextual page %s not collected: %v", p, got)
		}
	}
}
