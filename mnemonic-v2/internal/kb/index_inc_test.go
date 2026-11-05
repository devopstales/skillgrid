package kb

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestIndexIncrementalOnlyChanged(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	writeVaultPage(t, vault, "decisions/auth-model.md",
		"---\ntitle: Auth Model\ntype: Decision\nstatus: stable\n---\n\nWe use OAuth2.")
	writeVaultPage(t, vault, "gotchas/clock-skew.md",
		"---\ntitle: Clock Skew\ntype: Gotcha\n---\n\nNTP drift breaks token expiry.")
	st := openKbStore(t)

	res, err := Index(ctx, st, vault, "proj-a")
	if err != nil {
		t.Fatalf("index: %v", err)
	}
	if res.Scanned != 2 || res.Changed != 2 || res.Deleted != 0 {
		t.Fatalf("first: %+v, want scanned=2 changed=2 deleted=0", res)
	}

	// No drift → nothing to upsert.
	res, err = Index(ctx, st, vault, "proj-a")
	if err != nil {
		t.Fatalf("index again: %v", err)
	}
	if res.Scanned != 2 || res.Changed != 0 || res.Deleted != 0 {
		t.Fatalf("second: %+v, want scanned=2 changed=0 deleted=0", res)
	}

	// One page changed → only it is re-upserted.
	writeVaultPage(t, vault, "decisions/auth-model.md",
		"---\ntitle: Auth Model\ntype: Decision\nstatus: stable\n---\n\nWe use SAML.")
	res, err = Index(ctx, st, vault, "proj-a")
	if err != nil {
		t.Fatalf("index changed: %v", err)
	}
	if res.Scanned != 2 || res.Changed != 1 || res.Deleted != 0 {
		t.Fatalf("third: %+v, want scanned=2 changed=1 deleted=0", res)
	}

	rows, err := st.Query(ctx,
		`SELECT title, type, status, source_count FROM kb_pages WHERE project = 'proj-a' AND rel_path = 'decisions/auth-model.md'`)
	if err != nil || len(rows) != 1 {
		t.Fatalf("kb_pages row: %v %+v", err, rows)
	}
	if got := s0(rows[0], 0) + "/" + s0(rows[0], 1) + "/" + s0(rows[0], 2); got != "Auth Model/Decision/stable" {
		t.Fatalf("frontmatter parse: %q", got)
	}
}

func TestIndexSoftDeletesRemovedPages(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	writeVaultPage(t, vault, "decisions/auth-model.md",
		"---\ntitle: Auth Model\ntype: Decision\n---\n\nOAuth2 for external auth.")
	writeVaultPage(t, vault, "gotchas/zork-midrift.md",
		"---\ntitle: Zork Midrift\ntype: Gotcha\n---\n\nA rare zorkmidrift failure mode.")
	st := openKbStore(t)

	if _, err := Index(ctx, st, vault, "proj-a"); err != nil {
		t.Fatalf("index: %v", err)
	}
	if err := os.Remove(filepath.Join(vault, "gotchas", "zork-midrift.md")); err != nil {
		t.Fatalf("remove: %v", err)
	}
	res, err := Index(ctx, st, vault, "proj-a")
	if err != nil {
		t.Fatalf("index after remove: %v", err)
	}
	if res.Deleted != 1 || res.Changed != 0 {
		t.Fatalf("res: %+v, want deleted=1 changed=0", res)
	}

	// Soft-deleted in kb_pages (row survives, deleted_at set).
	rows, err := st.Query(ctx,
		`SELECT deleted_at IS NOT NULL FROM kb_pages WHERE project = 'proj-a' AND rel_path = 'gotchas/zork-midrift.md'`)
	if err != nil || len(rows) != 1 {
		t.Fatalf("soft-delete row: %v %+v", err, rows)
	}
	if n, _ := rows[0][0].(int64); n != 1 {
		t.Fatalf("deleted_at not set: %+v", rows[0])
	}
	// The deleted page is no longer queryable (the FTS row mirrors the
	// soft-deleted content row; the filter is on the join).
	hits, err := Query(ctx, st, "zorkmidrift", QueryFilters{Project: "proj-a"})
	if err != nil {
		t.Fatalf("query deleted: %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("deleted page still queryable: %+v", hits)
	}
	hits, err = Query(ctx, st, "OAuth2", QueryFilters{Project: "proj-a"})
	if err != nil {
		t.Fatalf("query live: %v", err)
	}
	if len(hits) != 1 || hits[0].RelPath != "decisions/auth-model.md" {
		t.Fatalf("live page query: %+v", hits)
	}
}

func TestIndexClaimsProjection(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	writeVaultPage(t, vault, "decisions/cache-ttl.md",
		`---
title: Cache TTL
type: Decision
anchors:
  - path: internal/cache/cache.go
    all_of:
      - ttl = 15 * time.Minute
claims:
  - subject: explicit site
    literal: ttl = 15 * time.Minute
    path: internal/cache/cache.go
  - subject: inherited site
    literal: maxEntries = 4096
---

We cache for 15 minutes.
`)
	st := openKbStore(t)
	if _, err := Index(ctx, st, vault, "proj-a"); err != nil {
		t.Fatalf("index: %v", err)
	}

	rows, err := st.Query(ctx,
		`SELECT c.subject, c.literal, c.anchor_path, c.anchor_selector
		 FROM kb_claims c JOIN kb_pages p ON p.id = c.page_id
		 WHERE p.project = 'proj-a' ORDER BY c.subject`)
	if err != nil || len(rows) != 2 {
		t.Fatalf("claims: %v %+v", err, rows)
	}
	if s0(rows[0], 0) != "explicit site" || s0(rows[0], 2) != "internal/cache/cache.go" || s0(rows[0], 3) != "{}" {
		t.Fatalf("explicit: %+v", rows[0])
	}
	// No explicit path + exactly one anchor → the anchor's path.
	if s0(rows[1], 0) != "inherited site" || s0(rows[1], 2) != "internal/cache/cache.go" {
		t.Fatalf("inherited: %+v", rows[1])
	}

	// Re-indexing the same page does not duplicate claim rows.
	if _, err := Index(ctx, st, vault, "proj-a"); err != nil {
		t.Fatalf("reindex: %v", err)
	}
	rows, err = st.Query(ctx,
		`SELECT count(*) FROM kb_claims c JOIN kb_pages p ON p.id = c.page_id WHERE p.project = 'proj-a'`)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n, _ := rows[0][0].(int64); n != 2 {
		t.Fatalf("claim rows after re-index: %d, want 2", n)
	}
}

func TestIndexInDegree(t *testing.T) {
	ctx := context.Background()
	vault := t.TempDir()
	writeVaultPage(t, vault, "decisions/alpha.md",
		"---\ntitle: Alpha\ntype: Decision\n---\n\nSee [[Beta]] and [[beta-alias]].")
	writeVaultPage(t, vault, "decisions/beta.md",
		"---\ntitle: Beta\ntype: Decision\naliases:\n  - beta-alias\n---\n\nBody of beta.")
	st := openKbStore(t)
	if _, err := Index(ctx, st, vault, "proj-a"); err != nil {
		t.Fatalf("index: %v", err)
	}

	rows, err := st.Query(ctx,
		`SELECT in_degree FROM kb_pages WHERE project = 'proj-a' AND rel_path = 'decisions/beta.md'`)
	if err != nil || len(rows) != 1 {
		t.Fatalf("beta row: %v %+v", err, rows)
	}
	if n, _ := rows[0][0].(int64); n != 2 {
		t.Fatalf("beta in_degree: %d, want 2 (title link + alias link)", n)
	}
	rows, err = st.Query(ctx,
		`SELECT in_degree FROM kb_pages WHERE project = 'proj-a' AND rel_path = 'decisions/alpha.md'`)
	if err != nil || len(rows) != 1 {
		t.Fatalf("alpha row: %v %+v", err, rows)
	}
	if n, _ := rows[0][0].(int64); n != 0 {
		t.Fatalf("alpha in_degree: %d, want 0", n)
	}
}
