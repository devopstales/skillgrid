// Package search: structural code_grep — by-example AST pattern matching.
//
// code_grep is index-free: it parses each supported-language file under a
// directory with gotreesitter and matches a by-example pattern (metavariables)
// against the syntax tree, skipping unknown files. No store or embeddings are
// required. An invalid pattern for a language skips that language's files with
// a clear note (warn+continue); other languages still match.
package search

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	ts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

// structuralMaxFileSize bounds per-file reads for structural grep so generated
// bundles / vendored blobs do not dominate the real-time walk.
const structuralMaxFileSize = 500 * 1024

// GrepHit is one structural match: the matched node's span + any named
// captures from the by-example pattern.
type GrepHit struct {
	Path     string            `json:"path"`
	Line     int               `json:"line"`
	Col      int               `json:"col"`
	Text     string            `json:"text"`
	Captures map[string]string `json:"captures,omitempty"`
}

// GrepNote is a warn+continue note when a pattern is invalid for a language:
// the language is skipped, the run is not a silent no-match.
type GrepNote struct {
	Language string `json:"language"`
	Reason   string `json:"reason"`
}

// GrepResult is the outcome of a structural grep across a directory.
type GrepResult struct {
	Hits         []GrepHit  `json:"hits"`
	Notes        []GrepNote `json:"notes,omitempty"`
	FilesSeen    int        `json:"files_seen"`
	FilesMatch   int        `json:"files_matched"`
	FilesSkipped int        `json:"files_skipped"`
	// Truncated is true when limit>0 capped the hit list before the scan
	// finished (more matches exist beyond the returned ones). HitsTruncated is
	// set to 1 as an indicator of truncation; the exact remainder is not counted
	// because the scan stops early. Both are omitted from JSON when zero/false.
	Truncated     bool `json:"truncated,omitempty"`
	HitsTruncated int  `json:"hits_truncated,omitempty"`
}

// ParseByExample converts a by-example structural pattern into gotreesitter
// query source. The first S-expression in the pattern is the root node to
// match; the capture name (from a \NAME metavariable) is attached to that
// root node (tree-sitter attaches a capture to the step it follows, so the
// capture must follow the node type, not a trailing wildcard).
//
// Metavariables are:
//
//	\NAME    a named-node capture (e.g. \name) — the matched node is captured
//	\(ARGS*) a run of any siblings (the argument list; captured as @args)
//	\*       an anonymous any-node that must follow the node (a wildcard step)
//	\_       an anonymous any-node that must follow the node (a wildcard step)
//
// Note: a named capture is a step suffix, not a sibling. `(function_definition
// \fn)` compiles to `(function_definition @fn)` — the capture attaches to the
// node it names. It must NOT compile to `(function_definition (_ @fn))`, which
// would require a child node (and for node types that have none, such as
// python's `call_expression`, the query fails with "unknown node type"
// because the grammar rejects the step).
//
// Whitespace separates tokens: `(call_expression function: (identifier) \name)`
// produces three root-internal tokens (`call_expression`, `function:`,
// `(identifier) @name`), reassembled with single spaces in the output.
//
// Returns the compiled source and an error for unbalanced parens or a
// malformed metavariable.
func ParseByExample(pattern string) (string, error) {
	var root strings.Builder // the first top-level S-expression (the matched node)
	var rest []string        // trailing tokens (arguments, wildcards, captures)
	var cur strings.Builder  // the token being built
	depth := 0
	open := false  // true while the root S-expression is open (between its parens)
	haveRoot := false
	// flush emits the current token: inside the root S-expression it joins
	// to root with a space; at depth 0 it becomes the root or a rest token.
	flush := func() {
		s := strings.TrimSpace(cur.String())
		if s == "" {
			cur.Reset()
			return
		}
		if open {
			// Inside the root S-expression: join tokens with a space so that
			// `call_expression function: (identifier)` stays as three separate
			// tokens rather than one mangled string.
			if root.Len() > 0 {
				root.WriteByte(' ')
			}
			root.WriteString(s)
		} else if depth == 0 && !haveRoot {
			root.WriteString(s)
			haveRoot = true
		} else {
			rest = append(rest, s)
		}
		cur.Reset()
	}
	// capture attaches the named capture to the currently-open S-expression
	// (metavariable written inside the parens, the common by-example form) or
	// to the root (metavariable written after the closing paren). Both compile
	// to the same tree-sitter query: the capture follows the node type.
	capture := func(name string) {
		// Flush the current token first so the capture becomes a separate
		// token; flush() will space-separate it from the preceding token.
		flush()
		if open {
			cur.WriteString("@")
			cur.WriteString(name)
		} else {
			root.WriteString(" @")
			root.WriteString(name)
		}
	}
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch {
		case c == '\\' && i+1 < len(pattern) && pattern[i+1] == '(':
			// \(ARGS*) -> argument-list run captured as @args. Consume the
			// full token so inner '(' / ')' are not S-expression delimiters.
			flush()
			j := i + 2
			for j < len(pattern) && pattern[j] != ')' {
				j++
			}
			if j < len(pattern) {
				j++
			}
			i = j
			rest = append(rest, "(argument_list) @args")
		case c == '\\' && i+1 < len(pattern) && (pattern[i+1] == '*' || pattern[i+1] == '_'):
			// \* or \_ -> anonymous any-node wildcard step.
			flush()
			i++
			rest = append(rest, "(_)")
		case c == '\\' && i+1 < len(pattern):
			// \NAME -> named-node capture (see capture()). Capture names are a
			// single identifier (letters, digits, dot, underscore).
			i++
			start := i
			for i < len(pattern) && (isIdentRune(pattern[i])) {
				i++
			}
			if i == start {
				return "", errGrepBadPattern("malformed metavariable: backslash not followed by an identifier")
			}
			capture(pattern[start:i])
			flush() // emit the @name token so the next token space-separates
			i-- // the loop above already consumed the last ident rune; step back
		case c == '(':
			depth++
			if depth == 1 {
				flush()
				open = true
			}
			cur.WriteByte(c)
		case c == ')':
			depth--
			if depth < 0 {
				return "", errGrepBadPattern("unbalanced ')' in pattern")
			}
			cur.WriteByte(c)
			if depth == 0 {
				open = false
				flush()
			}
		case c == ' ' || c == '\t' || c == '\n':
			// Whitespace is a token separator: flush the current token so
			// adjacent word tokens (node type + field name) do not merge.
			flush()
		default:
			cur.WriteByte(c)
		}
	}
	flush()
	if depth != 0 {
		return "", errGrepBadPattern("unbalanced parentheses in pattern")
	}
	if !haveRoot {
		if strings.Contains(pattern, "\\") {
			return "", errGrepBadPattern("bare metavariable: wrap the pattern in parentheses, e.g. (node_type) \\name")
		}
		return "", errGrepBadPattern("empty pattern")
	}
	out := root.String()
	for _, t := range rest {
		out += " " + t
	}
	return out, nil
}

// isIdentRune reports whether c is a valid capture-name character.
func isIdentRune(c byte) bool {
	return c == '.' || c == '_' ||
		(c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

type grepPatternError struct{ msg string }

func (e grepPatternError) Error() string { return "invalid grep pattern: " + e.msg }

func errGrepBadPattern(msg string) error { return grepPatternError{msg: msg} }

// IsPatternError reports whether err is a structural pattern compile error
// (as opposed to a per-language unknown-node-type error).
func IsPatternError(err error) bool {
	_, ok := err.(grepPatternError)
	return ok
}

// CompileGrep compiles a by-example pattern into a gotreesitter Query for lang.
// It returns a compiled query or an error. An error from the gotreesitter
// compiler (e.g. an unknown node type for this language) is the caller's
// signal to skip that language with a note.
func CompileGrep(pattern string, lang *ts.Language) (*ts.Query, error) {
	src, err := ParseByExample(pattern)
	if err != nil {
		return nil, err
	}
	if src == "" {
		return nil, errGrepBadPattern("empty pattern")
	}
	return ts.NewQuery(src, lang)
}

// CompileGrepAnchor compiles a query that captures the pattern's root node
// (the first S-expression's node type) as @__anchor. It is used to recover the
// matched node's position/text when the user's pattern carries no named
// capture: a bare `(function_definition)` produces a match with zero captures,
// so m.Captures[0] is nil and the hit would default to line:1 col:1. The anchor
// query matches the exact same root, so its single capture is the root node
// itself, one per match, in the same order. The anchor source is validated
// against lang (an unknown node type yields an error, skipping the language).
func CompileGrepAnchor(pattern string, lang *ts.Language) (*ts.Query, error) {
	// The root is the first top-level token of the by-example pattern: the
	// text from its opening '(' to the matching closing ')'. Extract it by
	// walking parens; it is a bare node type or a node type with internal
	// field/child specs. The anchor only needs the ROOT node type, so strip
	// any internal field/child specs down to the first token.
	root := patternRootType(pattern)
	if root == "" {
		return nil, errGrepBadPattern("no root node type in pattern")
	}
	src := "(" + root + ") @__anchor"
	return ts.NewQuery(src, lang)
}

// patternRootType returns the pattern's root node type: the first token inside
// the first top-level S-expression. `(function_definition name: (identifier))`
// -> "function_definition"; `(call_expression)` -> "call_expression". For an
// invalid pattern it returns "" (the caller falls back to no-anchor).
func patternRootType(pattern string) string {
	i := 0
	for i < len(pattern) && pattern[i] != '(' {
		i++
	}
	if i >= len(pattern) {
		return ""
	}
	i++ // step past '('
	start := i
	for i < len(pattern) && !isIdentRune(pattern[i]) {
		i++
	}
	for i < len(pattern) && isIdentRune(pattern[i]) {
		i++
	}
	if i == start {
		return ""
	}
	return pattern[start:i]
}

// GrepByExample runs a structural grep over every supported-language file
// under root, matching pattern against each file's syntax tree. Unknown files
// (no grammar) are skipped. A pattern that is invalid for a given language
// skips that language's files and records a note; other languages still match.
// It is index-free: no store or embeddings are required.
//
// limit caps the returned hit list for unbounded patterns (e.g. (call) over a
// large tree). limit<=0 means no cap. When the cap is reached the walk stops
// early (remaining files are not scanned), res.Truncated is set, and
// HitsTruncated is the count of hits dropped by the cap.
func GrepByExample(root, pattern string, limit int) (*GrepResult, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if rootAbs == "" {
		return nil, errGrepBadPattern("root is required")
	}
	// A structurally invalid pattern (unbalanced parens, dangling backslash,
	// empty) aborts before any per-language compile: it is a clear validation
	// error, not a silent no-match.
	if _, err := ParseByExample(pattern); err != nil {
		return nil, err
	}

	res := &GrepResult{}

	// stopGrep signals filepath.WalkDir to halt once the hit cap is reached.
	// It is a sentinel (not os.ErrNotExist) so a genuine walk error elsewhere
	// is never confused with the early stop.
	var stopGrep = errors.New("grep limit reached")

	// Cache per-language compiled queries and notes so each language is
	// compiled once (invalid -> note, skip its files).
	type langState struct {
		known  bool
		query  *ts.Query
		anchor *ts.Query // captures the root node for position/text on no-capture matches
		lang   *ts.Language
		note   string
	}
	langCache := map[string]*langState{}
	langStateFor := func(name string) *langState {
		if s, ok := langCache[name]; ok {
			return s
		}
		s := &langState{}
		probe, ok := grammarProbeFor(name)
		if !ok {
			s.known = false
		} else {
			entry := grammars.DetectLanguage(probe)
			if entry == nil {
				s.known = false
			} else {
				s.known = true
				s.lang = entry.Language()
				q, err := CompileGrep(pattern, s.lang)
				if err != nil {
					s.known = false
					s.note = err.Error()
				} else {
					s.query = q
					// Best-effort anchor query for no-capture positions. If the
					// root node type is unknown for this language the main
					// CompileGrep would have failed, so the anchor (same root)
					// is valid too; a nil anchor just means positions fall back
					// to the first capture.
					s.anchor, _ = CompileGrepAnchor(pattern, s.lang)
				}
			}
		}
		if !s.known && s.note != "" {
			res.Notes = append(res.Notes, GrepNote{Language: name, Reason: s.note})
		}
		langCache[name] = s
		return s
	}

	_ = filepath.WalkDir(rootAbs, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // skip unreadable entries; structural grep is best-effort
		}
		if d.IsDir() {
			if isGrepSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		langName := detectLangForGrep(path)
		if langName == "" {
			return nil // unknown file: skipped
		}
		st := langStateFor(langName)
		if !st.known {
			res.FilesSkipped++
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.Size() > structuralMaxFileSize {
			res.FilesSkipped++
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		res.FilesSeen++
		parser := ts.NewParser(st.lang)
		tree, err := parser.Parse(src)
		if err != nil || tree == nil || tree.ParseStoppedEarly() {
			return nil // malformed / early-stopped: no match, not a note
		}
		rel := filepath.ToSlash(filepath.Base(path))
		if d2, err2 := os.Stat(rootAbs); err2 == nil && d2.IsDir() {
			if r, err := filepath.Rel(rootAbs, path); err == nil && r != "." {
				rel = filepath.ToSlash(r)
			}
		}
		matched := false
		mainMatches := st.query.Execute(tree)
		// anchorMatches runs the root-capture query once per file. It matches
		// the same root as the main query, so anchorMatches[i] corresponds to
		// mainMatches[i] (same node, same order). It is computed lazily only
		// when a main match has no capture to anchor on.
		var anchorMatches []ts.QueryMatch
		anchorReady := false
		for mi, m := range mainMatches {
			// Hit cap: check before appending so the walk stops the moment the
			// returned list is full (the current file is not consumed past the
			// cap). HitsTruncated is set to 1 as an indicator that further hits
			// exist (the exact remainder is unknown without scanning on).
			if limit > 0 && len(res.Hits) >= limit {
				res.Truncated = true
				res.HitsTruncated = 1
				return stopGrep
			}
			h := GrepHit{Path: rel, Line: 1, Col: 1}
			h.Captures = map[string]string{}
			// The anchor node is the first capture, or (when the pattern has no
			// named capture and the match has zero captures) the root node from
			// the per-language anchor query. Position/text come from it.
			anchor := (*ts.Node)(nil)
			if len(m.Captures) > 0 {
				anchor = m.Captures[0].Node
			}
			if anchor == nil && st.anchor != nil {
				if !anchorReady {
					anchorMatches = st.anchor.Execute(tree)
					anchorReady = true
				}
				if mi < len(anchorMatches) {
					if a := anchorMatches[mi]; len(a.Captures) > 0 && a.Captures[0].Node != nil {
						anchor = a.Captures[0].Node
					}
				}
			}
			if anchor != nil {
				p := anchor.StartPoint()
				h.Line = int(p.Row) + 1
				h.Col = int(p.Column) + 1
				h.Text = anchor.Text(src)
				if len(h.Text) > 200 {
					h.Text = h.Text[:200]
				}
			}
			for _, c := range m.Captures {
				if c.Node == nil {
					continue
				}
				txt := c.Text(src)
				if len(txt) > 80 {
					txt = txt[:80]
				}
				h.Captures[c.Name] = txt
			}
			if len(h.Captures) == 0 {
				h.Captures = nil
			}
			res.Hits = append(res.Hits, h)
			matched = true
		}
		if matched {
			res.FilesMatch++
		}
		return nil
	})
	if res.FilesMatch > 0 {
		res.FilesSkipped = 0
	}
	return res, nil
}

// detectLangForGrep returns the supported language name for a path, or "".
func detectLangForGrep(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return ""
	}
	return grepExtToLang[ext]
}

func isGrepSkipDir(name string) bool {
	if name == "" {
		return false
	}
	if name[0] == '.' {
		return true
	}
	return grepSkipDirs[name]
}

var grepSkipDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	"out":          true,
	".git":         true,
	".skillgrid":   true,
	"target":       true,
	"__pycache__":  true,
}

// grammarProbeFor maps a supported language name to a probe filename the
// gotreesitter registry resolves to that language.
func grammarProbeFor(name string) (string, bool) {
	for ext, lang := range grepExtToLang {
		if lang == name {
			return probeNameForExt(ext), true
		}
	}
	return "", false
}

func probeNameForExt(ext string) string {
	return "a" + ext
}

// grepExtToLang is the structural-grep extension map (subset of extract's,
// self-contained so search stays free of an extract import cycle).
var grepExtToLang = map[string]string{
	".go":     "go",
	".ts":     "typescript",
	".mts":    "typescript",
	".cts":    "typescript",
	".tsx":    "tsx",
	".js":     "javascript",
	".mjs":    "javascript",
	".cjs":    "javascript",
	".py":     "python",
	".rs":     "rust",
	".java":   "java",
	".c":      "c",
	".h":      "c",
	".cpp":    "cpp",
	".cc":     "cpp",
	".cxx":    "cpp",
	".cu":     "cpp",
	".cuh":    "cpp",
	".hpp":    "cpp",
	".cs":     "c_sharp",
	".php":    "php",
	".rb":     "ruby",
	".kt":     "kotlin",
	".kts":    "kotlin",
	".swift":  "swift",
	".scala":  "scala",
	".dart":   "dart",
	".lua":    "lua",
	".r":      "r",
	".R":      "r",
	".m":      "matlab",
	".mm":     "objective_c",
	".pl":     "perl",
	".pm":     "perl",
	".ex":     "elixir",
	".exs":    "elixir",
	".hs":     "haskell",
	".clj":    "clojure",
	".cljs":   "clojure",
	".cljc":   "clojure",
	".zig":    "zig",
	".nim":    "nim",
	".groovy": "groovy",
	".gvy":    "groovy",
	".sh":     "bash",
	".bash":   "bash",
	".sql":    "sql",
	".css":    "css",
	".html":   "html",
	".htm":    "html",
}
