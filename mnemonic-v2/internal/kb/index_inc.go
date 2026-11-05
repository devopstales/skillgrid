package kb

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// index_inc.go — the incremental index path (011 deliverable 6): a
// body_hash diff against kb_pages. Changed pages are upserted with the
// frontmatter re-parsed (title/type/status/aliases + anchor/claim/source
// counts + the kb_claims projection); removed pages are soft-deleted
// (deleted_at — the FTS triggers keep kb_pages_fts consistent); wikilink
// in_degree is recomputed across the live pages. Reindex stays the
// full-rebuild path.

var wikilinkRe = regexp.MustCompile(`\[\[([^\]|]+)(?:\|[^\]]+)?\]\]`)

// Index rebuilds only what changed (the incremental path).
func Index(ctx context.Context, st store.Store, root, project string) (*IndexResult, error) {
	pages, err := collectPages(root)
	if err != nil {
		return nil, err
	}
	res := &IndexResult{Scanned: len(pages)}

	existing, err := st.Query(ctx,
		`SELECT rel_path, body_hash FROM kb_pages WHERE project = ? AND deleted_at IS NULL`, project)
	if err != nil {
		return nil, fmt.Errorf("kb: index existing: %w", err)
	}
	ex := map[string]string{}
	for _, r := range existing {
		ex[s0(r, 0)] = s0(r, 1)
	}

	seen := map[string]bool{}
	for _, p := range pages {
		seen[p.RelPath] = true
		if h, ok := ex[p.RelPath]; ok && h == p.BodyHash {
			continue
		}
		if err := upsertPageFull(ctx, st, p, project); err != nil {
			return nil, err
		}
		res.Changed++
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for rel := range ex {
		if seen[rel] {
			continue
		}
		if err := st.Exec(ctx,
			`UPDATE kb_pages SET deleted_at = ? WHERE project = ? AND rel_path = ? AND deleted_at IS NULL`,
			now, project, rel); err != nil {
			return nil, fmt.Errorf("kb: soft-delete %s: %w", rel, err)
		}
		res.Deleted++
	}

	if err := recomputeInDegree(ctx, st, root, project); err != nil {
		return nil, err
	}
	return res, nil
}

// upsertPageFull writes one page row (insert or update) with the frontmatter
// re-parsed: title/type/status/aliases + anchor/claim/source counts, and the
// kb_claims projection rebuilt for the page (011 deliverable 6: claims parse).
func upsertPageFull(ctx context.Context, st store.Store, p Page, project string) error {
	fm, _, _ := SplitFrontmatter(p.Body)
	title := fm.Title
	if title == "" {
		title = pageTitle(p.Body)
	}
	typ := fm.Type
	status := fm.Status
	if status == "" {
		status = "draft"
	}
	aliases, _ := json.Marshal(fm.Aliases)
	now := time.Now().UTC().Format(time.RFC3339)

	var pageID int64
	rows, err := st.Query(ctx, `SELECT id FROM kb_pages WHERE project = ? AND rel_path = ?`, project, p.RelPath)
	if err != nil {
		return fmt.Errorf("kb: upsert %s: %w", p.RelPath, err)
	}
	if len(rows) > 0 {
		if v, ok := rows[0][0].(int64); ok {
			pageID = v
		}
		if err := st.Exec(ctx,
			`UPDATE kb_pages SET title = ?, type = ?, status = ?, aliases = ?, body = ?, body_hash = ?,
			 anchor_count = ?, claim_count = ?, source_count = ?, updated_at = ?, deleted_at = NULL
			 WHERE project = ? AND rel_path = ?`,
			title, typ, status, string(aliases), p.Body, p.BodyHash,
			len(fm.Anchors), len(fm.Claims), len(fm.Sources), now, project, p.RelPath); err != nil {
			return fmt.Errorf("kb: update %s: %w", p.RelPath, err)
		}
	} else {
		if err := st.Exec(ctx,
			`INSERT INTO kb_pages
				(sync_id, project, rel_path, title, type, status, aliases, in_degree, body, body_hash,
				 anchor_count, claim_count, source_count, updated_at, deleted_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?, ?, ?, NULL)`,
			project, project, p.RelPath, title, typ, status, string(aliases),
			p.Body, p.BodyHash, len(fm.Anchors), len(fm.Claims), len(fm.Sources), now); err != nil {
			return fmt.Errorf("kb: insert %s: %w", p.RelPath, err)
		}
		rows, err = st.Query(ctx, `SELECT id FROM kb_pages WHERE project = ? AND rel_path = ?`, project, p.RelPath)
		if err != nil {
			return fmt.Errorf("kb: upsert %s: %w", p.RelPath, err)
		}
		if len(rows) > 0 {
			if v, ok := rows[0][0].(int64); ok {
				pageID = v
			}
		}
	}

	if pageID == 0 {
		return nil
	}
	if err := st.Exec(ctx, `DELETE FROM kb_claims WHERE page_id = ?`, pageID); err != nil {
		return fmt.Errorf("kb: claims %s: %w", p.RelPath, err)
	}
	for _, c := range fm.Claims {
		site := c.Path
		if site == "" && len(fm.Anchors) == 1 {
			site = fm.Anchors[0].Path
		}
		if err := st.Exec(ctx,
			`INSERT INTO kb_claims (page_id, subject, literal, anchor_path, anchor_selector, created_at)
			 VALUES (?, ?, ?, ?, '{}', ?)`,
			pageID, c.Subject, c.Literal, site, now); err != nil {
			return fmt.Errorf("kb: claims %s: %w", p.RelPath, err)
		}
	}
	return nil
}

// recomputeInDegree recomputes wikilink in-degree across the live pages
// (011 deliverable 6: in_degree). [[Target]] resolves against page title,
// aliases, and slug (basename without .md), case-insensitively.
func recomputeInDegree(ctx context.Context, st store.Store, root, project string) error {
	rows, err := st.Query(ctx,
		`SELECT rel_path, body FROM kb_pages WHERE project = ? AND deleted_at IS NULL ORDER BY rel_path`, project)
	if err != nil {
		return fmt.Errorf("kb: in_degree: %w", err)
	}

	targets := map[string]string{} // normalized target -> rel_path
	type pageInfo struct {
		rel  string
		body string
	}
	var infos []pageInfo
	for _, r := range rows {
		rel, body := s0(r, 0), s0(r, 1)
		infos = append(infos, pageInfo{rel, body})
		fm, _, _ := SplitFrontmatter(body)
		register := func(t string) {
			if t != "" {
				targets[strings.ToLower(strings.TrimSpace(t))] = rel
			}
		}
		register(fm.Title)
		for _, a := range fm.Aliases {
			register(a)
		}
		register(slugOf(rel))
		register(rel)
	}

	degs := map[string]int{}
	for _, pi := range infos {
		for _, m := range wikilinkRe.FindAllStringSubmatch(pi.body, -1) {
			t, ok := targets[strings.ToLower(strings.TrimSpace(m[1]))]
			if !ok || t == pi.rel {
				continue
			}
			degs[t]++
		}
	}
	if err := st.Exec(ctx,
		`UPDATE kb_pages SET in_degree = 0 WHERE project = ? AND deleted_at IS NULL`, project); err != nil {
		return err
	}
	for rel, d := range degs {
		if err := st.Exec(ctx,
			`UPDATE kb_pages SET in_degree = ? WHERE project = ? AND rel_path = ?`, d, project, rel); err != nil {
			return err
		}
	}
	return nil
}

// slugOf derives a page slug from a rel_path or topic_key: lowercase,
// non-alphanumeric runs collapsed to "-", trimmed.
func slugOf(s string) string {
	s = strings.ToLower(filepath.Base(s))
	s = strings.TrimSuffix(s, ".md")
	var b strings.Builder
	dash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
			continue
		}
		if !dash && b.Len() > 0 {
			b.WriteRune('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// sortedRelPaths returns the live page rel_paths (used by full runs).
func sortedRelPaths(ctx context.Context, st store.Store, project string) ([]string, error) {
	rows, err := st.Query(ctx,
		`SELECT rel_path FROM kb_pages WHERE project = ? AND deleted_at IS NULL ORDER BY rel_path`, project)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, s0(r, 0))
	}
	sort.Strings(out)
	return out, nil
}
