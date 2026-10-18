package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConceptGraph is the read-only concept graph of a knowledge bundle,
// derived PER CALL from the current Markdown (a disposable view: it is
// never stored, so a bundle edit is visible on the next derive).
//
// A bundle is the project's post-R6 knowledge zone — one .md file per
// concept (YAML frontmatter: type/title/description/tags/status) plus
// catalog files (index.md, log.md, reverse-index.md) that list the
// concepts but are not concepts themselves.
//
// Bundle-size assumption (documented cap): this derive is sized for tens of
// files, not thousands. One os.Walk + one parse per file with no caching —
// at OKF bundle scale (a handful of identity folders, one file per concept)
// that is well under a second. If a bundle ever grows into the hundreds of
// files, paginate or cache the derive at the route layer instead of
// complicating this pure function.
type ConceptGraph struct {
	Nodes     []ConceptNode       `json:"nodes"`
	Edges     []ConceptEdge       `json:"edges"`
	Backlinks map[string][]string `json:"backlinks"`
}

// ConceptNode is one concept page: its frontmatter fields plus the two
// flags the graph view styles distinctly (orphan: no inbound links;
// deprecated: status: deprecated).
type ConceptNode struct {
	ID          string   `json:"id"` // path relative to root, minus .md (slash separators)
	Title       string   `json:"title"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Status      string   `json:"status"`
	Orphan      bool     `json:"orphan"`
	Deprecated  bool     `json:"deprecated"`
}

// ConceptEdge is one Markdown link that RESOLVED to another concept.
// Broken links (target outside the bundle or not a concept) are tolerated:
// they simply produce no edge.
type ConceptEdge struct {
	Source string `json:"source"` // concept id
	Target string `json:"target"` // concept id
}

// conceptCatalogBases are the catalog files at the bundle root: they index
// the concepts but are not concepts themselves, so they are excluded from
// nodes (and their links produce no edges).
var conceptCatalogBases = map[string]bool{
	"index.md":         true,
	"log.md":           true,
	"reverse-index.md": true,
}

// conceptFrontmatter is the YAML frontmatter of a concept page. All fields
// are optional: a page missing one carries the zero value (a missing
// title falls back to the file name at the derive layer, not here).
type conceptFrontmatter struct {
	Type        string   `yaml:"type"`
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
	Status      string   `yaml:"status"`
}

// DeriveConceptGraph derives the concept graph of the knowledge bundle at
// root, reading the CURRENT Markdown on every call (per-call, disposable:
// never stored). It walks the bundle recursively, builds one node per
// concept page (root-relative path minus .md as id), and one edge per
// Markdown link [text](target.md) that resolves to another concept
// relative to the LINKING file's directory — broken links (outside the
// bundle, a non-concept file, a URL) are tolerated and produce no edge.
// Catalog files (index.md / log.md / reverse-index.md at the root) are
// excluded from nodes. Backlinks are the reverse of edges.
//
// An empty-but-present bundle derives an empty graph (not an error); a
// missing root is an error (the caller named a directory that is not
// there). Nodes and edges are sorted by id for deterministic output.
func DeriveConceptGraph(root string) (ConceptGraph, error) {
	info, err := os.Stat(root)
	if err != nil {
		return ConceptGraph{}, err
	}
	if !info.IsDir() {
		return ConceptGraph{}, fmt.Errorf("knowledge bundle %s is not a directory", root)
	}

	// Pass 1: collect the concept pages.
	type page struct {
		id  string // root-relative path, slash-separated, minus .md
		src []byte
	}
	pages := []page{}
	walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// A file that disappears mid-walk is skipped (the bundle is
			// disposable and live); a directory that cannot be entered
			// skips its subtree. Walk-level failures are surfaced below
			// via the return value, not here.
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(d.Name()), ".md") {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		relSlash := filepath.ToSlash(rel)
		// Catalog files live at the bundle root (no path separator).
		if !strings.Contains(relSlash, "/") && conceptCatalogBases[relSlash] {
			return nil
		}
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			// A file that disappears mid-derive is skipped, not fatal
			// (mirrors the extractor tolerance: the bad part is skipped).
			return nil
		}
		pages = append(pages, page{id: strings.TrimSuffix(relSlash, ".md"), src: src})
		return nil
	})
	if walkErr != nil {
		return ConceptGraph{}, walkErr
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].id < pages[j].id })

	concepts := make(map[string]bool, len(pages))
	nodes := make([]ConceptNode, 0, len(pages))
	for _, p := range pages {
		concepts[p.id] = true
		fm := parseConceptFrontmatter(p.id, p.src)
		nodes = append(nodes, ConceptNode{
			ID:          p.id,
			Title:       fm.Title,
			Type:        fm.Type,
			Description: fm.Description,
			Tags:        fm.Tags,
			Status:      fm.Status,
			Deprecated:  fm.Status == "deprecated",
		})
	}

	// Pass 2: edges — Markdown links that resolve to another concept.
	seen := make(map[[2]string]bool)
	edges := []ConceptEdge{}
	for _, p := range pages {
		base := strings.TrimSuffix(p.id, "/"+filepath.Base(p.id)) // parent dir, "" at root
		for _, target := range extractConceptLinkTargets(string(p.src)) {
			resolved := resolveConceptLinkTarget(base, target)
			if resolved == "" || !concepts[resolved] {
				continue // broken link: tolerated, no edge
			}
			if p.id == resolved {
				continue // self-link: no edge
			}
			key := [2]string{p.id, resolved}
			if seen[key] {
				continue
			}
			seen[key] = true
			edges = append(edges, ConceptEdge{Source: p.id, Target: resolved})
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Source != edges[j].Source {
			return edges[i].Source < edges[j].Source
		}
		return edges[i].Target < edges[j].Target
	})

	// Backlinks: the reverse of edges (sorted for determinism).
	backlinks := make(map[string][]string, len(edges))
	for _, e := range edges {
		backlinks[e.Target] = append(backlinks[e.Target], e.Source)
	}
	for k := range backlinks {
		sort.Strings(backlinks[k])
	}

	// Orphans: nodes with no inbound edges.
	for i := range nodes {
		if len(backlinks[nodes[i].ID]) == 0 {
			nodes[i].Orphan = true
		}
	}

	return ConceptGraph{Nodes: nodes, Edges: edges, Backlinks: backlinks}, nil
}

// parseConceptFrontmatter parses the YAML frontmatter block (leading
// ---...---) of one concept page. A missing or malformed block yields the
// zero frontmatter; a missing title falls back to the file name (the id
// base) so every node carries a display title.
func parseConceptFrontmatter(id string, src []byte) conceptFrontmatter {
	var fm conceptFrontmatter
	b := src
	if len(b) > 0 && b[0] == '\r' {
		b = b[1:]
	}
	if strings.HasPrefix(string(b), "---\n") {
		rest := b[len("---\n"):]
		if end := bytesIndex(rest, []byte("\n---")); end >= 0 {
			var err error
			_ = yaml.Unmarshal(rest[:end], &fm)
			_ = err
		}
	}
	if strings.TrimSpace(fm.Title) == "" {
		fm.Title = id
	}
	return fm
}

func bytesIndex(b, sub []byte) int {
	for i := 0; i+len(sub) <= len(b); i++ {
		if string(b[i:i+len(sub)]) == string(sub) {
			return i
		}
	}
	return -1
}

// conceptLink matches a markdown link [text](target) whose target is a
// plain .md path. It is the concept-graph variant of mdLink: an in-page
// anchor (#...) is tolerated but EXCLUDED from the captured target, so an
// anchor-suffixed link (./b.md#sec) resolves to the doc, not the fragment.
// mdLink itself cannot be reused because its capture keeps the anchor. The
// closing paren is not part of the pattern: an in-page anchor sits between
// the path and the paren (./b.md#sec), so the path must terminate at the
// ".md". The path character class excludes ")" and whitespace only — a "."
// is a legal path byte (./b.md, ../out.md, a.md).
var conceptLink = regexp.MustCompile(`\[[^\]]*\]\(\s*([^)\s]*\.md)`)

// extractConceptLinkTargets returns the Markdown link targets in src that
// look like doc references (relative .md paths), each with its in-page
// anchor stripped. URLs, anchors, and images are not doc references.
func extractConceptLinkTargets(src string) []string {
	var out []string
	for _, m := range conceptLink.FindAllStringSubmatch(src, -1) {
		out = append(out, m[1])
	}
	return out
}

// resolveConceptLinkTarget resolves one link target against the linking
// file's directory and returns the root-relative concept id (path minus
// .md) when it points inside the bundle; "" when it points outside (a
// ../ escape, an absolute path, or an unresolvable path). The result is
// not checked against the concept set — that is the caller's job.
func resolveConceptLinkTarget(baseDir, target string) string {
	if strings.HasPrefix(target, "/") || strings.HasPrefix(target, "\\\\") {
		return "" // absolute: outside the bundle
	}
	joined := target
	if baseDir != "" {
		joined = baseDir + "/" + target
	}
	// Collapse "." and ".." segments without touching the filesystem: a
	// single ".." above the root means "outside the bundle".
	segments := strings.Split(joined, "/")
	stack := []string{}
	for _, seg := range segments {
		switch seg {
		case "", ".":
		case "..":
			if len(stack) == 0 {
				return ""
			}
			stack = stack[:len(stack)-1]
		default:
			stack = append(stack, seg)
		}
	}
	id := strings.Join(stack, "/")
	if id == "" || !strings.HasSuffix(id, ".md") {
		return ""
	}
	return strings.TrimSuffix(id, ".md")
}
