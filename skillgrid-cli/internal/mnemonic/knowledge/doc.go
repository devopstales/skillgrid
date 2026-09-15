// Package knowledge extracts the code's surrounding knowledge — markdown
// docs, config files, and SQL DDL — into the 005 graph as first-class nodes
// (doc_nodes / config_nodes / sql_schema_nodes) with confidence-labeled
// edges (references / configures / reads / writes). It is additive: it never
// rewrites a 005 symbol or edge, and a malformed file yields zero knowledge
// rows (the index continues, the bad part is skipped, the rest is indexed).
package knowledge

import (
	"hash/fnv"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// Edge kinds emitted by the knowledge extractors (distinct from 005's
// call/heritage kinds; references is shared with 010's route references but
// scoped here to doc node -> doc node).
const (
	KindReferences = "references"
	KindConfigures = "configures"
	KindReads      = "reads"
	KindWrites     = "writes"
)

// Confidence labels (mirrored from extract/route/graph).
//
//	EXTRACTED  — explicit syntax (a literal doc link, a config key naming a
//	            symbol by explicit spec, an SQL statement naming a table).
//	INFERRED   — convention-derived (a wikilink resolved by file convention).
//	AMBIGUOUS  — a resolved-but-inferred guess (an unresolved config reference
//	            kept, not dropped, per 03.5).
const (
	ConfidenceExtracted = "EXTRACTED"
	ConfidenceInferred  = "INFERRED"
	ConfidenceAmbiguous = "AMBIGUOUS"
)

// KindDoc / KindConfig / KindTable / KindColumn are the knowledge node kinds.
const (
	KindDoc    = "doc"
	KindConfig = "config"
	KindTable  = "table"
	KindColumn = "column"
)

// mdLink matches a markdown link [text](target). The target is a relative
// path (or a URL fragment); anchors (#...) are stripped before resolution.
var mdLink = regexp.MustCompile(`\[[^\]]*\]\(([^)\s]+)\)`)

// mdWiki matches a wikilink [[target]].
var mdWiki = regexp.MustCompile(`\[\[([^\[\]]+)\]\]`)

// DocLink is one markdown link / wikilink in a doc file: its target as
// written, the line it appears on, and its confidence.
type DocLink struct {
	Target     string
	Line       int
	Confidence string
}

// SymbolRef is a code-symbol identifier mentioned in a doc's plain text: the
// name as written and its first line. Resolution (doc node -> symbol node)
// happens in the store (doc.go has no DB access); a name that does not resolve
// to exactly one indexed symbol is dropped (drop-not-guess).
type SymbolRef struct {
	Name string
	Line int
}

// DocResult is the extraction result for one markdown file: the doc node
// (its title + path), its outgoing references links, and the code-symbol
// identifiers its text mentions (for doc->symbol reference edges).
type DocResult struct {
	Path       string
	Title      string
	Links      []DocLink
	SymbolRefs []SymbolRef
	IsDoc      bool
}

// DocTitle derives the doc node's title from the first H1 heading, falling
// back to the file name (without extension).
func DocTitle(path string, src []byte) string {
	for _, line := range strings.Split(string(src), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
		}
	}
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}

// mdLinkSpan is one [text](target) or [[target]] match: the text span to
// blank out (so the link's target text is not mistaken for a symbol mention)
// and its link span, in 1-based line offsets.
type mdLinkSpan struct {
	blank  [2]int // [text] span (text + brackets), 1-based offsets
	linkLo int    // link span start, 1-based
	linkHi int    // link span end (exclusive), 1-based
}

// rawSpan is one markdown link/wikilink match in 0-based line offsets.
type rawSpan struct {
	lo, hi, textLo, textHi int
}

// linkSpansInLine returns the blankable spans for a line's markdown links and
// wikilinks, in document order.
func linkSpansInLine(line string) []mdLinkSpan {
	var spans []rawSpan
	for _, m := range mdLink.FindAllStringSubmatchIndex(line, -1) {
		spans = append(spans, rawSpan{lo: m[0], hi: m[1], textLo: m[0], textHi: m[2]})
	}
	for _, m := range mdWiki.FindAllStringSubmatchIndex(line, -1) {
		spans = append(spans, rawSpan{lo: m[0], hi: m[1], textLo: m[0] + 1, textHi: m[2]})
	}
	sortSpans(spans)
	out := make([]mdLinkSpan, 0, len(spans))
	for _, s := range spans {
		out = append(out, mdLinkSpan{
			blank:  [2]int{s.textLo + 1, s.textHi + 1},
			linkLo: s.lo + 1,
			linkHi: s.hi + 1,
		})
	}
	return out
}

func sortSpans(spans []rawSpan) {
	for i := 1; i < len(spans); i++ {
		for j := i; j > 0 && spans[j].lo < spans[j-1].lo; j-- {
			spans[j], spans[j-1] = spans[j-1], spans[j]
		}
	}
}

// ExtractDoc parses one markdown file into doc node + references links +
// symbol mentions. It never errors and never returns nil: a malformed file
// (unparseable link) skips the bad link and keeps the rest (03.6). A
// non-markdown path returns an empty (IsDoc=false) result.
func ExtractDoc(path string, src []byte) *DocResult {
	if !isMarkdown(path) {
		return &DocResult{Path: path, Title: filepath.Base(path)}
	}
	res := &DocResult{Path: path, Title: DocTitle(path, src), IsDoc: true}
	seen := map[string]bool{}
	for lineNo, line := range strings.Split(string(src), "\n") {
		for _, m := range mdLink.FindAllStringSubmatchIndex(line, -1) {
			target := line[m[2]:m[3]]
			if !isDocLinkTarget(target) {
				continue
			}
			res.Links = append(res.Links, DocLink{
				Target:     normalizeLinkTarget(target),
				Line:       lineNo + 1,
				Confidence: ConfidenceExtracted,
			})
		}
		for _, m := range mdWiki.FindAllStringSubmatchIndex(line, -1) {
			target := line[m[2]:m[3]]
			res.Links = append(res.Links, DocLink{
				Target:     normalizeWikiTarget(target),
				Line:       lineNo + 1,
				Confidence: ConfidenceInferred,
			})
		}
		// Symbol mentions: scan the line with link targets blanked (their text
		// names a DOC, not a code symbol) for identifier-looking names.
		for _, name := range symbolMentionsInLine(line, linkSpansInLine(line)) {
			if !seen[name] {
				seen[name] = true
				res.SymbolRefs = append(res.SymbolRefs, SymbolRef{Name: name, Line: lineNo + 1})
			}
		}
	}
	return res
}

// symbolMentionsInLine extracts symbol-name-looking identifiers from a line:
// len>=4, matches [A-Za-z_][A-Za-z0-9_]*, has an internal camel hump
// (lowercase→uppercase) or an underscore. Link/wikilink spans are skipped
// (their text names a DOC, not a code symbol). This is a cheap heuristic;
// resolution is the store's job.
func symbolMentionsInLine(line string, spans []mdLinkSpan) []string {
	n := len(line)
	skip := make([]bool, n)
	for _, sp := range spans {
		for p := sp.linkLo - 1; p < sp.linkHi-1 && p < n; p++ {
			skip[p] = true
		}
	}
	var out []string
	i := 0
	for i < n {
		if skip[i] {
			i++
			continue
		}
		c := line[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_') {
			i++
			continue
		}
		j := i
		for j < n && !skip[j] && isIdentByte(line[j]) {
			j++
		}
		w := strings.Trim(line[i:j], "._")
		if len(w) >= 4 && looksLikeSymbolName(w) {
			out = append(out, w)
		}
		i = j
	}
	return out
}

func isIdentByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_'
}

// looksLikeSymbolName reports whether a candidate identifier looks like a code
// symbol name: it contains a letter and has an internal camel hump or an
// underscore (plain lowercase words are English prose, not symbols).
func looksLikeSymbolName(s string) bool {
	hasLetter := false
	for _, c := range s {
		if unicode.IsLetter(c) {
			hasLetter = true
			break
		}
	}
	if !hasLetter {
		return false
	}
	for i := 1; i < len(s); i++ {
		if s[i] == '_' {
			return true
		}
		if s[i-1] >= 'a' && s[i-1] <= 'z' && s[i] >= 'A' && s[i] <= 'Z' {
			return true
		}
	}
	return false
}

// isMarkdown reports whether path is a markdown file.
func isMarkdown(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown":
		return true
	}
	return false
}

// isDocLinkTarget reports whether a markdown link target is a doc reference
// (not an external URL, a bare anchor, or an image) — only doc links become
// references edges (03.2: links between doc nodes).
func isDocLinkTarget(target string) bool {
	if target == "" {
		return false
	}
	if strings.Contains(target, "://") {
		return false // external URL
	}
	if strings.HasPrefix(target, "#") {
		return false // in-page anchor, not a doc reference
	}
	return strings.HasSuffix(target, ".md") || strings.HasSuffix(target, ".markdown")
}

// normalizeLinkTarget strips an in-page anchor (#...) from a markdown link
// target so the reference resolves to the doc, not the fragment.
func normalizeLinkTarget(target string) string {
	if i := strings.Index(target, "#"); i >= 0 {
		target = target[:i]
	}
	return strings.TrimSpace(target)
}

// normalizeWikiTarget strips an alias ([[page|alias]] -> page) and an anchor
// from a wikilink target. Wikilinks name a doc by title/path; a bare name
// (no path separator, no .md suffix) is resolved to <name>.md by file
// convention so it links to the same-named doc node (03.2: wikilinks between
// doc nodes).
func normalizeWikiTarget(target string) string {
	if i := strings.Index(target, "|"); i >= 0 {
		target = target[:i]
	}
	if i := strings.Index(target, "#"); i >= 0 {
		target = target[:i]
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return target
	}
	if strings.Contains(target, "/") || strings.Contains(target, "\\") {
		if !strings.HasSuffix(target, ".md") && !strings.HasSuffix(target, ".markdown") {
			target += ".md"
		}
		return target
	}
	if !strings.HasSuffix(target, ".md") && !strings.HasSuffix(target, ".markdown") {
		target += ".md"
	}
	return target
}

// docNodeID derives the doc node's deterministic id-key (the file path,
// normalized to slashes) — a doc node is per-file.
func docNodeKey(path string) string {
	return filepath.ToSlash(path)
}

// fnvHex returns the hex FNV-1a digest of s (pure Go, CGo-free) — used for
// deterministic knowledge node uids.
func fnvHex(s string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return strings.ToLower(strings.TrimLeft(
		func() string { return hex64(h.Sum64()) }(), "0"))
}

func hex64(v uint64) string {
	const digits = "0123456789abcdef"
	var b [16]byte
	for i := 15; i >= 0; i-- {
		b[i] = digits[v&0xf]
		v >>= 4
	}
	return string(b[:])
}
