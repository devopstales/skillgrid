package kb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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

// pageDirs is the index-scope whitelist (001 §7): only the page folders are
// scanned. Everything else under the vault root — index/ (derived
// projections), templates/, dot-dirs (.obsidian/), trajectories/, and the
// vault-root docs — is out of scope by construction.
var pageDirs = map[string]struct{}{
	"decisions":    {},
	"invariants":   {},
	"workflows":    {},
	"architecture": {},
	"gotchas":      {},
	"glossary":     {},
	"skills":       {}, // opt-in (012)
	"cases":        {}, // opt-in (012)
}

// loadIgnore reads the vault's .knowledgeignore (gitignore-like, 001 §7) and
// returns a predicate reporting whether a slash-separated vault-relative path
// (optionally with a trailing "/" for dirs) is excluded from indexing. The
// ignore file can only narrow the page-folder whitelist, never widen it.
func loadIgnore(root string) (func(string) bool, error) {
	data, err := os.ReadFile(filepath.Join(root, VaultIgnoreFile))
	if errors.Is(err, os.ErrNotExist) {
		return func(string) bool { return false }, nil
	}
	if err != nil {
		return nil, err
	}
	var pats []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		pats = append(pats, line)
	}
	return func(rel string) bool {
		for _, p := range pats {
			switch {
			case strings.HasSuffix(p, "/"):
				d := strings.TrimSuffix(p, "/")
				if rel == d || strings.HasPrefix(rel, d+"/") {
					return true
				}
			case rel == p || strings.HasPrefix(rel, p+"/"):
				return true
			case strings.ContainsAny(p, "*?"):
				if ok, _ := filepath.Match(p, filepath.Base(rel)); ok {
					return true
				}
			}
		}
		return false
	}, nil
}

// collectPages walks the vault root and returns the in-scope markdown pages
// (the page-folder whitelist of 001 §7, further narrowed by
// .knowledgeignore).
func collectPages(root string) ([]Page, error) {
	ignore, err := loadIgnore(root)
	if err != nil {
		return nil, err
	}
	var pages []Page
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return rerr
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel == "." {
				return nil
			}
			if _, ok := pageDirs[d.Name()]; !ok {
				return filepath.SkipDir
			}
			if ignore(rel + "/") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}
		if !strings.Contains(rel, "/") {
			return nil // vault-root files (docs, schema) are out of index scope (001 §7)
		}
		if ignore(rel) {
			return nil
		}
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
