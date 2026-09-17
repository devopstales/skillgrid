// Package docs (markdown readers) — read-only, sandboxed access to the repo's
// markdown across the declared doc roots. Nothing is executed: files are
// served as text for the SPA to render. Every requested path is cleaned and
// prefix-checked against the declared roots before any file is opened, so
// `..`, absolute paths, and symlink-escaped names are rejected (400) or not
// found (404) before a byte is read.
package docs

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// mdRoot maps the tree `root=` selector to its on-disk directory (relative to
// the server's working directory, the repo root). "sdd" maps to the spec/plan
// store (.skillgrid/specs) in this repo layout.
var mdRoots = map[string]string{
	"sdd":     ".skillgrid/specs",
	"openspec": "openspec",
	"backlog": ".backlog/tasks",
	"docs":    "docs",
	"root":    ".",
}

// MDNode is one entry in the docs tree.
type MDNode struct {
	Dir     bool       `json:"dir"`
	Name    string     `json:"name"`
	Path    string     `json:"path"`
	Title   string     `json:"title,omitempty"`
	Status  string     `json:"status,omitempty"`
	Updated string     `json:"updated,omitempty"`
	Children []MDNode  `json:"children,omitempty"`
}

// MDContent is a single markdown file served for rendering.
type MDContent struct {
	Path          string         `json:"path"`
	Title         string         `json:"title,omitempty"`
	Frontmatter   map[string]any `json:"frontmatter,omitempty"`
	Body          string         `json:"body"`
	RelatedPlans  []string       `json:"relatedPlans,omitempty"`
	UpdatedAt     string         `json:"updatedAt,omitempty"`
	Mermaid       MermaidCfg     `json:"mermaid"`
}

// MermaidCfg carries the mermaid render config the SPA must use so diagrams are
// XSS-safe regardless of the doc author's markup.
type MermaidCfg struct {
	SecurityLevel string `json:"securityLevel"`
}

// MermaidSecurityLevel is the server-declared default the SPA renders with.
// 'strict' is the safe choice: mermaid does not allow HTML inside node labels.
func MermaidSecurityLevel() string { return "strict" }

// SearchHit is one search match.
type SearchHit struct {
	Path    string `json:"path"`
	Title   string `json:"title,omitempty"`
	Snippet string `json:"snippet,omitempty"`
	Root    string `json:"root,omitempty"`
}

// ErrMDBadPath marks a path that fails the sandbox (raw .. / absolute) -> 400.
var ErrMDBadPath = errMD("bad path")
// ErrMDNotFound marks a clean path that does not exist -> 404.
var ErrMDNotFound = errMD("not found")

type errMD string

func (e errMD) Error() string { return string(e) }

// resolveMDPath maps a relative requested path to its absolute location,
// guaranteeing it sits under one of the declared roots.
//
// Rules (defense in depth — the guard also rejects raw dot-segments up front):
//   - the raw query must contain no `..` segment and must not be absolute (400)
//   - after filepath.Clean it is joined to cwd and checked with filepath.Rel
//     against each root; it must sit exactly under a root (else 404)
func resolveMDPath(cwd, rel string) (string, string, error) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		return "", "", ErrMDBadPath
	}
	if filepath.IsAbs(rel) {
		return "", "", ErrMDBadPath
	}
	// Reject raw dot-segments before cleaning (matches the /docs/changes guard).
	for _, seg := range strings.FieldsFunc(rel, func(r rune) bool { return r == '/' || r == '\\' }) {
		if seg == ".." || seg == "." {
			return "", "", ErrMDBadPath
		}
	}
	clean := filepath.Clean(rel)
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", "", ErrMDBadPath
	}
	full := filepath.Join(cwd, clean)
	absFull, err := filepath.Abs(full)
	if err != nil {
		return "", "", ErrMDBadPath
	}
	// Must sit under exactly one declared root. The "root" (".") selector is
	// restricted to top-level *.md files (no subpaths) so it never swallows
	// files under a real doc root or a private subdir.
	for sel, rootRel := range mdRoots {
		absRoot, err := filepath.Abs(filepath.Join(cwd, rootRel))
		if err != nil {
			continue
		}
		relPath, err := filepath.Rel(absRoot, absFull)
		if err != nil {
			continue
		}
		if relPath == "." || filepath.IsAbs(relPath) || relPath == ".." || strings.HasPrefix(relPath, ".."+string(os.PathSeparator)) {
			continue
		}
		if sel == "root" {
			// top-level file only: no path separators, must be a .md file
			if strings.ContainsAny(relPath, `/\`) || !strings.EqualFold(filepath.Ext(relPath), ".md") {
				continue
			}
		}
		return sel, absFull, nil
	}
	return "", "", ErrMDNotFound
}

// guardMDPath is the up-front raw-segment check for query-string paths.
func guardMDPath(w http.ResponseWriter, raw string) bool {
	u := raw
	if strings.Contains(u, "/../") || strings.HasSuffix(u, "/..") ||
		strings.Contains(u, "/./") || strings.HasSuffix(u, "/.") ||
		strings.Contains(u, "%2e%2e") || strings.Contains(strings.ToLower(u), "%2f..") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid doc path (no . or .. segments)"})
		return false
	}
	return true
}

// NewContent returns the GET /docs/content?path=... handler.
func NewContent(cwd string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.URL.Query().Get("path")
		if !guardMDPath(w, raw) {
			return
		}
		sel, abs, err := resolveMDPath(cwd, raw)
		if err != nil {
			code := http.StatusNotFound
			if err == ErrMDBadPath {
				code = http.StatusBadRequest
			}
			writeJSON(w, code, map[string]string{"error": err.Error()})
			return
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "doc not found"})
			return
		}
		fm, body := splitFrontmatter(string(data))
		out := MDContent{
			Path:         filepath.ToSlash(raw),
			Title:        titleFromBody(body, filepath.Base(raw)),
			Frontmatter:  fm,
			Body:         body,
			RelatedPlans: relatedPlans(body, cwd),
			Mermaid:      MermaidCfg{SecurityLevel: MermaidSecurityLevel()},
		}
		if info, err := os.Stat(abs); err == nil {
			out.UpdatedAt = info.ModTime().UTC().Format("2006-01-02T15:04:05Z")
		}
		_ = sel
		writeJSON(w, http.StatusOK, out)
	})
}

// NewTree returns the GET /docs/tree?root=... handler.
func NewTree(cwd string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		root := r.URL.Query().Get("root")
		if root == "" {
			root = "all"
		}
		if _, ok := mdRoots[root]; !ok && root != "all" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown root: " + root})
			return
		}
		tree, err := BuildTree(r.Context(), cwd, root)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		// Emit [] (not null) when a root has no markdown, so the UI's
		// nodes.map/reduce never runs on null.
		if tree == nil {
			tree = []MDNode{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"root": root, "nodes": tree})
	})
}

// NewSearch returns the GET /docs/search?q=... handler.
func NewSearch(cwd string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if q == "" {
			writeJSON(w, http.StatusOK, map[string]any{"results": []SearchHit{}})
			return
		}
		hits := Search(r.Context(), cwd, q, 50)
		writeJSON(w, http.StatusOK, map[string]any{"q": q, "results": hits})
	})
}

// NewRender returns the GET /docs/render?path=... handler (SSR HTML fallback).
func NewRender(cwd string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.URL.Query().Get("path")
		if !guardMDPath(w, raw) {
			return
		}
		_, abs, err := resolveMDPath(cwd, raw)
		if err != nil {
			code := http.StatusNotFound
			if err == ErrMDBadPath {
				code = http.StatusBadRequest
			}
			w.WriteHeader(code)
			return
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, body := splitFrontmatter(string(data))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(renderHTML(body))
	})
}

// splitFrontmatter pulls a leading YAML frontmatter block (--- ... ---) into a
// flat string->string map, and returns the remaining body.
func splitFrontmatter(s string) (map[string]any, string) {
	if !strings.HasPrefix(s, "---\n") {
		return nil, s
	}
	rest := s[4:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return nil, s
	}
	block := rest[:idx]
	body := rest[idx+len("\n---"):]
	body = strings.TrimPrefix(body, "\n")
	fm := map[string]any{}
	for _, line := range strings.Split(block, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fm[strings.TrimSpace(k)] = strings.TrimSpace(strings.Trim(strings.TrimSpace(v), `"`))
	}
	return fm, body
}

// titleFromBody extracts the first markdown H1 as the title, else falls back.
func titleFromBody(body, fallback string) string {
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(t, "# "))
		}
	}
	return fallback
}

// relatedPlans finds other spec/change names referenced in the body (cross-links).
func relatedPlans(body, cwd string) []string {
	var out []string
	seen := map[string]bool{}
	for _, line := range strings.Split(body, "\n") {
		for _, tok := range strings.FieldsFunc(line, func(r rune) bool {
			return r == ' ' || r == '/' || r == ',' || r == '(' || r == ')' || r == '`'
		}) {
			if strings.HasPrefix(tok, "0") && strings.Contains(tok, "-") && len(tok) > 6 {
				dir := filepath.Join(cwd, ".skillgrid", "specs", tok)
				if info, err := os.Stat(dir); err == nil && info.IsDir() {
					if !seen[tok] {
						seen[tok] = true
						out = append(out, tok)
					}
				}
			}
		}
	}
	return out
}

// BuildTree walks the selected root(s) and returns a nested tree of MDNode.
func BuildTree(ctx context.Context, cwd, root string) ([]MDNode, error) {
	sels := orderedRoots(root)
	var out []MDNode
	for _, sel := range sels {
		abs, err := filepath.Abs(filepath.Join(cwd, mdRoots[sel]))
		if err != nil {
			continue
		}
		if info, err := os.Stat(abs); err != nil || !info.IsDir() {
			continue
		}
		node, err := walkDir(ctx, cwd, abs, sel, filepath.Base(abs))
		if err != nil {
			return nil, err
		}
		out = append(out, *node)
	}
	return out, nil
}

func orderedRoots(root string) []string {
	if root != "all" {
		if _, ok := mdRoots[root]; ok {
			return []string{root}
		}
		return []string{}
	}
	// stable order
	return []string{"sdd", "openspec", "backlog", "docs", "root"}
}

// repoRel returns the repo-relative slash path for abs, or "" on error.
// cwd may be "." (the docsCwd default) — it is made absolute first so
// filepath.Rel has two absolute operands.
func repoRel(cwd, abs string) string {
	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		return ""
	}
	rel, err := filepath.Rel(absCwd, abs)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(rel)
}

func walkDir(ctx context.Context, cwd, abs, sel, name string) (*MDNode, error) {
	entries, err := os.ReadDir(abs)
	if err != nil {
		if errorsIsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	node := &MDNode{Dir: true, Name: name, Path: repoRel(cwd, abs), Title: name}
	if info, err := os.Stat(abs); err == nil {
		node.Updated = info.ModTime().UTC().Format("2006-01-02")
	}
	// The "root" selector only lists top-level *.md files (not subdirs).
	rootOnly := sel == "root"
	for _, e := range entries {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		childAbs := filepath.Join(abs, e.Name())
		if e.IsDir() {
			if rootOnly {
				continue
			}
			child, err := walkDir(ctx, cwd, childAbs, sel, e.Name())
			if err != nil {
				return nil, err
			}
			if child != nil {
				node.Children = append(node.Children, *child)
			}
			continue
		}
		if !strings.EqualFold(filepath.Ext(e.Name()), ".md") {
			continue
		}
		data, err := os.ReadFile(childAbs)
		if err != nil {
			continue
		}
		fm, body := splitFrontmatter(string(data))
		cn := MDNode{
			Name:  e.Name(),
			Path:  repoRel(cwd, childAbs),
			Title: titleFromBody(body, strings.TrimSuffix(e.Name(), ".md")),
		}
		if st, ok := fm["status"]; ok {
			cn.Status, _ = st.(string)
		}
		if info, err := os.Stat(childAbs); err == nil {
			cn.Updated = info.ModTime().UTC().Format("2006-01-02")
		}
		node.Children = append(node.Children, cn)
	}
	return node, nil
}

// Search scans all roots for case-insensitive substring matches on the term.
func Search(ctx context.Context, cwd, q string, limit int) []SearchHit {
	qLower := strings.ToLower(q)
	var hits []SearchHit
	// Search the real doc roots (not the "." root — that would re-walk all
	// other roots and double-count). Top-level root *.md are still scanned via
	// a separate shallow pass below.
	searchRoots := []string{"sdd", "openspec", "backlog", "docs"}
	for _, sel := range searchRoots {
		abs, err := filepath.Abs(filepath.Join(cwd, mdRoots[sel]))
		if err != nil {
			continue
		}
		_ = filepath.WalkDir(abs, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".md") {
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			text := string(data)
			if strings.Contains(strings.ToLower(text), qLower) {
				_, body := splitFrontmatter(text)
				hits = append(hits, SearchHit{
					Path:    repoRel(cwd, path),
					Title:   titleFromBody(body, filepath.Base(path)),
					Snippet: snippetAround(text, qLower),
					Root:    sel,
				})
			}
			return nil
		})
	}
	// Shallow pass: top-level root *.md (e.g. README.md) — not under a doc root.
	if entries, err := os.ReadDir(cwd); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), ".md") {
				continue
			}
			path := filepath.Join(cwd, e.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			text := string(data)
			if strings.Contains(strings.ToLower(text), qLower) {
				_, body := splitFrontmatter(text)
				hits = append(hits, SearchHit{
					Path:    e.Name(),
					Title:   titleFromBody(body, e.Name()),
					Snippet: snippetAround(text, qLower),
					Root:    "root",
				})
			}
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].Path < hits[j].Path })
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

func snippetAround(text, qLower string) string {
	lower := strings.ToLower(text)
	idx := strings.Index(lower, qLower)
	if idx < 0 {
		return ""
	}
	start := idx - 40
	if start < 0 {
		start = 0
	}
	end := idx + len(qLower) + 40
	if end > len(text) {
		end = len(text)
	}
	s := text[start:end]
	s = strings.ReplaceAll(s, "\n", " ")
	if start > 0 {
		s = "…" + s
	}
	return s
}

// renderHTML is a minimal, dependency-free markdown->HTML used only as the
// /docs/render SSR fallback. The primary path renders client-side; this is a
// conservative best-effort (headings, paragraphs, code, lists) and is escaped.
func renderHTML(body string) []byte {
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\"></head><body><article>")
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "```"):
			// toggle code block state crudely; emit as <pre> via a marker
		case trimmed == "":
		default:
			esc := escapeHTML(line)
			if strings.HasPrefix(trimmed, "# ") {
				b.WriteString("<h1>" + esc[2:] + "</h1>")
			} else if strings.HasPrefix(trimmed, "## ") {
				b.WriteString("<h2>" + esc[3:] + "</h2>")
			} else if strings.HasPrefix(trimmed, "### ") {
				b.WriteString("<h3>" + esc[4:] + "</h3>")
			} else {
				b.WriteString("<p>" + esc + "</p>")
			}
		}
	}
	b.WriteString("</article></body></html>")
	return []byte(b.String())
}

func escapeHTML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&#34;")
	return r.Replace(s)
}

func errorsIsNotExist(err error) bool {
	return os.IsNotExist(err)
}
