package projectwiki

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/core/store"
)

// Lint severities (spec 007 §2): SeverityError blocks under --strict/CI;
// SeverityWarn is reported, non-blocking; SeverityInfo is advisory. Named
// so the CLI boundary compares against the vocabulary, not raw strings.
const (
	SeverityError = "error"
	SeverityWarn  = "warn"
	SeverityInfo  = "info"
)

// orphanInDegreeMax is spec 007 §2's orphan threshold ("0–1 incoming
// backlinks") — the single definition of the rule; status.go's orphan
// count embeds it in its SQL.
const orphanInDegreeMax = 1

// LintIssue is one deterministic health finding (spec 007 §2).
// Severity is one of the Severity* constants above.
type LintIssue struct {
	RelPath  string
	Code     string
	Severity string
	Message  string
}

func issue(rel, code, severity, msg string) LintIssue {
	return LintIssue{RelPath: rel, Code: code, Severity: severity, Message: msg}
}

// Lint scans a project's indexed pages for the deterministic health
// checks of spec 007 §2: missing title, stale index (body_hash
// mismatch against the stored body), orphans (in_degree <= 1), broken
// wikilinks, duplicate aliases, missing sources (error on grounded
// pages, warn on contextual), ungrounded grounded pages, weak pins
// (bare line: with no version: token), past stale_after, weak
// cross-references (< 2 outgoing wikilinks), contradiction markers,
// broken code:/external: citations, and coverage gaps (top-level Go
// packages under repo that no page cites). repo is the working-tree
// root for code: citation and
// coverage resolution — empty repo skips those two checks. The
// external: neuledge fingerprint re-resolution is a network check:
// offline it is skipped with no finding (007 §3), never an error.
// The checks read the indexed rows (bodies, aliases, in_degree), never
// the vault files — the handler syncs the index before linting.
func Lint(ctx context.Context, st store.Store, repo, project string) ([]LintIssue, error) {
	rows, err := st.Query(ctx,
		`SELECT rel_path, title, type, body, body_hash, aliases, in_degree FROM projectwiki_pages WHERE project = ? AND deleted_at IS NULL ORDER BY rel_path`,
		project)
	if err != nil {
		return nil, err
	}

	type page struct {
		rel      string
		typ      string
		body     string
		fm       Frontmatter
		outLinks []string
	}
	var pages []page
	var infos []pageLinkInfo
	aliasOwner := map[string]string{}
	var out []LintIssue
	for _, r := range rows {
		rel, title, body, bh := strVal(r, 0), strVal(r, 1), strVal(r, 3), strVal(r, 4)
		fm, _, _ := SplitFrontmatter(body)
		p := page{rel: rel, typ: strVal(r, 2), body: body, fm: fm}
		for _, m := range wikilinkRe.FindAllStringSubmatch(body, -1) {
			p.outLinks = append(p.outLinks, strings.TrimSpace(m[1]))
		}
		pages = append(pages, p)
		infos = append(infos, pageLinkInfo{rel, body})

		if strings.TrimSpace(title) == "" {
			out = append(out, issue(rel, "missing_title", SeverityWarn, "title empty"))
			continue
		}
		h := sha256.Sum256([]byte(body))
		if hex.EncodeToString(h[:]) != bh {
			out = append(out, issue(rel, "stale_index", SeverityWarn, "body_hash mismatch"))
		}
		var aliases []string
		// The aliases payload is engine-written (upsertPageFull's
		// json.Marshal) — on a corrupt row the duplicate_alias check is
		// skipped for that page, never the whole lint.
		_ = json.Unmarshal([]byte(strVal(r, 5)), &aliases)
		for _, a := range aliases {
			key := strings.ToLower(strings.TrimSpace(a))
			if key == "" {
				continue
			}
			if owner, dup := aliasOwner[key]; dup && owner != rel {
				out = append(out, issue(rel, "duplicate_alias", SeverityWarn,
					"alias "+a+" also claimed by "+owner))
			} else {
				aliasOwner[key] = rel
			}
		}
		if deg, ok := r[6].(int64); ok && deg <= orphanInDegreeMax {
			out = append(out, issue(rel, "orphan", SeverityWarn, "in_degree <= 1 (no page links here)"))
		}
	}

	targets := buildLinkTargets(infos)
	citedPkgs := map[string]bool{}
	for _, p := range pages {
		grounded := IsGrounded(p.typ)
		if len(p.outLinks) < 2 {
			out = append(out, issue(p.rel, "weak_cross_reference", SeverityWarn, "fewer than 2 outgoing wikilinks"))
		}
		for _, link := range p.outLinks {
			if _, ok := targets[strings.ToLower(link)]; !ok {
				out = append(out, issue(p.rel, "broken_wikilink", SeverityError, "unresolved wikilink [["+link+"]]"))
			}
		}
		if len(p.fm.Sources) == 0 {
			sev := SeverityWarn
			if grounded {
				sev = SeverityError
			}
			out = append(out, issue(p.rel, "missing_sources", sev, "empty sources: (no provenance)"))
		}
		if grounded && len(p.fm.Anchors) == 0 {
			out = append(out, issue(p.rel, "ungrounded_grounded", SeverityError, "grounded page with no anchors:"))
		}
		for _, a := range p.fm.Anchors {
			if a.Line > 0 && a.Version == "" {
				out = append(out, issue(p.rel, "weak_pin", SeverityWarn, "span pinned by bare line: with no version: token"))
			}
		}
		if p.fm.StaleAfter != "" {
			// time.Now() is a deliberate hidden dependency (code-standards
			// exception, named in review.md): stale_after is a wall-clock
			// horizon (007 §2) and both test sides are pinned (2020-01-01
			// past / no fixture uses a future date).
			if t, perr := time.Parse("2006-01-02", p.fm.StaleAfter); perr == nil && t.Before(time.Now()) {
				out = append(out, issue(p.rel, "past_stale_after", SeverityWarn, "stale_after "+p.fm.StaleAfter+" has passed"))
			}
		}
		if strings.Contains(p.body, "## Contradiction") {
			out = append(out, issue(p.rel, "contradiction", SeverityWarn, "unresolved ## Contradiction marker"))
		}
		for _, s := range p.fm.Sources {
			switch {
			case strings.HasPrefix(s.Inline, "code:"):
				ref := strings.TrimPrefix(s.Inline, "code:")
				if i := strings.Index(ref, "#"); i >= 0 {
					ref = ref[:i]
				}
				citedPkgs[pkgOf(ref)] = true
				if repo != "" {
					joined := filepath.Join(repo, filepath.FromSlash(ref))
					if joined != filepath.Clean(repo) && !strings.HasPrefix(joined, filepath.Clean(repo)+string(filepath.Separator)) {
						// A vault-authored ref is data, not a path: refuse
						// to stat outside the repo root (CWE-22 — the
						// finding is the only leak).
						out = append(out, issue(p.rel, "broken_citation", SeverityWarn, "code: citation escapes repo root: "+ref))
						continue
					}
					if _, serr := os.Stat(joined); serr != nil {
						out = append(out, issue(p.rel, "broken_citation", SeverityWarn, "code: citation does not resolve: "+ref))
					}
				}
			case s.External != "":
				if s.Version == "" || s.SourceURL == "" {
					out = append(out, issue(p.rel, "broken_citation", SeverityWarn,
						"external: citation missing version/source_url: "+s.External))
				}
				// Fingerprint re-resolution is network-bound; skipped
				// offline with no finding (007 §3), never an error.
			}
		}
		for _, a := range p.fm.Anchors {
			citedPkgs[pkgOf(a.Path)] = true
		}
	}

	if repo != "" {
		for _, pkg := range goPackages(repo) {
			if !citedPkgs[pkg] {
				out = append(out, issue("", "coverage_gap", SeverityInfo, "no page cites package "+pkg))
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].RelPath != out[j].RelPath {
			return out[i].RelPath < out[j].RelPath
		}
		return out[i].Code < out[j].Code
	})
	return out, nil
}

// pkgOf returns the directory portion of a repo-relative path.
func pkgOf(ref string) string {
	if i := strings.LastIndex(ref, "/"); i >= 0 {
		return ref[:i]
	}
	return ""
}

// goPackages lists repo-relative directories (depth <= 4) containing
// Go files — the shallow package scan of spec 007 §3, used when the
// code-index graph is not consulted.
func goPackages(repo string) []string {
	seen := map[string]bool{}
	_ = filepath.WalkDir(repo, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, rerr := filepath.Rel(repo, p)
		if rerr != nil || rel == "." {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata" || name == "node_modules" {
				return filepath.SkipDir
			}
			if strings.Count(filepath.ToSlash(rel), "/") >= 4 {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".go") {
			if dir := filepath.ToSlash(filepath.Dir(rel)); dir != "." && dir != "" {
				seen[dir] = true
			}
		}
		return nil
	})
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Fix applies the safe subset of lint fixes: backfill empty titles from the
// page's rel_path. Returns the number of fixes applied.
func Fix(ctx context.Context, st store.Store, project string, relPaths []string) (int, error) {
	fixed := 0
	for _, rel := range relPaths {
		rows, err := st.Query(ctx,
			`SELECT title FROM projectwiki_pages WHERE project = ? AND rel_path = ? AND deleted_at IS NULL`, project, rel)
		if err != nil {
			return fixed, err
		}
		empty := false
		if len(rows) > 0 {
			empty = strings.TrimSpace(strVal(rows[0], 0)) == ""
		}
		if !empty {
			continue
		}
		if err := st.Exec(ctx, `UPDATE projectwiki_pages SET title = ? WHERE project = ? AND rel_path = ?`, rel, project, rel); err != nil {
			return fixed, err
		}
		fixed++
	}
	return fixed, nil
}
