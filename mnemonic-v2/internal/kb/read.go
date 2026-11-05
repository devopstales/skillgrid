package kb

// read.go — stage-2 hydration (008 §3 kb_read): the full page (or one "##"
// section) + backlinks + the pin-check state for grounded pages.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// ReadResult is the stage-2 hydration result.
type ReadResult struct {
	RelPath     string      `json:"rel_path"`
	Title       string      `json:"title"`
	Type        string      `json:"type"`
	Status      string      `json:"status,omitempty"`
	Body        string      `json:"body"`
	Backlinks   []string    `json:"backlinks,omitempty"`
	InDegree    int64       `json:"in_degree"`
	PinCheck    *PageCheck  `json:"pin_check,omitempty"`
	NotGrounded bool        `json:"not_grounded,omitempty"`
	Frontmatter Frontmatter `json:"frontmatter,omitempty"`
}

// Read hydrates one page (008 §3 kb_read). section selects a single "## "
// section ("" = the full body). includeBacklinks walks the vault's wikilinks
// for the pages that link here. Grounded pages carry their pin-check state —
// a read-only ValidatePage (the no-op proof's layer 2 is exactly this read;
// the verified stamp is never written here, 006 §3).
func Read(ctx context.Context, st store.Store, vaultRoot, repoRoot, relPath, project, section string, includeBacklinks bool) (*ReadResult, error) {
	data, err := os.ReadFile(filepath.Join(vaultRoot, filepath.FromSlash(relPath)))
	if err != nil {
		return nil, fmt.Errorf("kb: read %s: page not found", relPath)
	}
	fm, body, err := SplitFrontmatter(string(data))
	if err != nil {
		return nil, fmt.Errorf("kb: read %s: %w", relPath, err)
	}
	res := &ReadResult{
		RelPath:     relPath,
		Title:       fm.Title,
		Type:        fm.Type,
		Status:      fm.Status,
		Frontmatter: fm,
		Backlinks:   []string{},
	}
	if section != "" {
		body = sectionBody(body, section)
		if body == "" {
			return nil, fmt.Errorf("kb: read %s: section %q not found", relPath, section)
		}
	}
	res.Body = strings.TrimSpace(body)
	rows, err := st.Query(ctx,
		`SELECT in_degree FROM kb_pages WHERE project = ? AND rel_path = ? AND deleted_at IS NULL`,
		project, relPath)
	if err != nil {
		return nil, err
	}
	if len(rows) > 0 {
		res.InDegree = i64At(rows[0], 0)
	}
	if IsGrounded(fm.Type) {
		pc, err := ValidatePage(ctx, st, repoRoot, relPath, fm)
		if err != nil {
			return nil, err
		}
		res.PinCheck = &pc
	} else {
		res.NotGrounded = true
	}
	if includeBacklinks {
		res.Backlinks = backlinksFor(vaultRoot, relPath)
	}
	return res, nil
}

// i64At reads one row cell as an int64 (0 when absent or another type).
func i64At(r []any, i int) int64 {
	if i >= len(r) {
		return 0
	}
	switch v := r[i].(type) {
	case int64:
		return v
	case int:
		return int64(v)
	}
	return 0
}

// sectionBody extracts one "## name" section (up to the next "## " heading or
// EOF). "" when the section is absent.
func sectionBody(body, section string) string {
	lines := strings.Split(body, "\n")
	var out []string
	in := false
	for _, ln := range lines {
		if strings.HasPrefix(ln, "## ") {
			name := strings.TrimSpace(strings.TrimPrefix(ln, "## "))
			if in {
				break
			}
			in = name == strings.TrimSpace(section)
			continue
		}
		if in {
			out = append(out, ln)
		}
	}
	if !in {
		return ""
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n")
}

// backlinksFor lists the vault pages whose wikilinks target relPath (the
// link side of the in_degree count). A wikilink target matches when it is
// the rel_path (with or without .md) or the page's file stem.
func backlinksFor(vaultRoot, relPath string) []string {
	targets := map[string]bool{}
	targets[relPath] = true
	if strings.HasSuffix(relPath, ".md") {
		targets[strings.TrimSuffix(relPath, ".md")] = true
	}
	stem := relPath
	stem = strings.TrimSuffix(stem, ".md")
	if i := strings.LastIndexByte(stem, '/'); i >= 0 {
		stem = stem[i+1:]
	}
	targets[stem] = true

	pages, err := collectPages(vaultRoot)
	if err != nil {
		return []string{}
	}
	var out []string
	for _, p := range pages {
		if p.RelPath == relPath {
			continue
		}
		for _, m := range wikilinkRe.FindAllStringSubmatch(p.Body, -1) {
			if targets[strings.TrimSpace(m[1])] {
				out = append(out, p.RelPath)
				break
			}
		}
	}
	return out
}
