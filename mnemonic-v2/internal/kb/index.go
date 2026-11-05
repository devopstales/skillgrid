package kb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// Page is one knowledge vault page (a markdown file under the vault root).
type Page struct {
	RelPath  string
	Title    string
	Type     string
	Status   string
	Body     string
	BodyHash string
}

// IndexResult is the outcome of one index run.
type IndexResult struct {
	Scanned int
	Changed int
	Deleted int
}

// Reindex rebuilds the derived index (kb_pages + kb_pages_fts + kb_claims)
// from the vault markdown at root. Full rebuild: clear kb_pages for the
// project, then re-insert every page (the FTS external-content triggers
// keep kb_pages_fts in sync).
func Reindex(ctx context.Context, st store.Store, root string, project string) (*IndexResult, error) {
	res := &IndexResult{}
	pages, err := collectPages(root)
	if err != nil {
		return nil, err
	}
	res.Scanned = len(pages)
	if err := st.ExecDDL(ctx, []string{`DELETE FROM kb_pages WHERE project = ` + sqlLit(project)}); err != nil {
		return nil, fmt.Errorf("kb: delete kb_pages: %w", err)
	}
	for _, p := range pages {
		if err := upsertPage(ctx, st, p, project); err != nil {
			return nil, err
		}
	}
	res.Changed = len(pages)
	return res, nil
}

// upsertPage inserts one page row into kb_pages (the source of truth).
func upsertPage(ctx context.Context, st store.Store, p Page, project string) error {
	if err := st.ExecDDL(ctx, []string{
		`INSERT INTO kb_pages
			(sync_id, project, rel_path, title, type, status, aliases, in_degree, body, body_hash, anchor_count, claim_count, source_count, updated_at, deleted_at)
		VALUES (` + sqlLit(project) + `, ` + sqlLit(project) + `, ` + sqlLit(p.RelPath) + `, ` + sqlLit(p.Title) + `, ` + sqlLit(p.Type) + `, ` + sqlLit(p.Status) + `, '` + "[]" + "', 0, " + sqlLit(p.Body) + ", " + sqlLit(p.BodyHash) + ", 0, 0, 0, " + sqlLit(time.Now().UTC().Format(time.RFC3339)) + ", NULL)",
	}); err != nil {
		return fmt.Errorf("kb: upsert %s: %w", p.RelPath, err)
	}
	return nil
}

// collectPages walks the vault root and returns every markdown page.
func collectPages(root string) ([]Page, error) {
	var pages []Page
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		pages = append(pages, Page{
			RelPath:  rel,
			Body:     string(body),
			BodyHash: hashBody(string(body)),
			Title:    pageTitle(string(body)),
			Type:     frontmatterType(string(body)),
			Status:   frontmatterStatus(string(body)),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return pages, nil
}

func frontmatterTitle(b string) string  { return frontmatterField(b, "title") }
func frontmatterType(b string) string   { return frontmatterField(b, "type") }
func frontmatterStatus(b string) string { return frontmatterField(b, "status") }

// pageTitle derives the display title: the frontmatter title if present,
// else the first H1 heading, else "" (lint flags it as missing_title).
func pageTitle(b string) string {
	if t := frontmatterTitle(b); t != "" {
		return t
	}
	for _, ln := range strings.Split(b, "\n") {
		ln = strings.TrimLeft(ln, " \t")
		if strings.HasPrefix(ln, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(ln, "# "))
		}
	}
	return ""
}

func frontmatterField(b, key string) string {
	content := string(b)
	if i := strings.Index(content, "---\n"); i >= 0 {
		rest := content[i+4:]
		if j := strings.Index(rest, "\n---"); j >= 0 {
			for _, ln := range strings.Split(rest[:j], "\n") {
				k, v, ok := strings.Cut(ln, ":")
				if !ok {
					continue
				}
				if strings.TrimSpace(k) == key {
					return strings.TrimSpace(v)
				}
			}
		}
	}
	return ""
}

func hashBody(b string) string {
	h := sha256.Sum256([]byte(b))
	return hex.EncodeToString(h[:])
}

func sqlLit(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
