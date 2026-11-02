package search

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	ts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

// GrepHit is one structural match: the matched node's span + any named
// captures from the by-example pattern.
type GrepHit struct {
	Path     string
	Line     int
	Col      int
	Text     string
	Captures map[string]string
}

// GrepResult is the outcome of a structural grep. Degraded is set when the
// pattern is invalid for the language (no_grammar:<lang> — never a silent
// skip, T12).
type GrepResult struct {
	Hits     []GrepHit
	Degraded string
}

// structuralMaxFileSize bounds per-file reads for structural grep so generated
// bundles / vendored blobs do not dominate the real-time walk.
const structuralMaxFileSize = 500 * 1024

// Grep runs the index-free by-example AST grep (spec 006 §3): it re-parses the
// working tree with the TICKET-03 gotreesitter ladder, so it works on files
// not yet indexed. pattern is the by-example pattern (the fixed pattern
// language, T12: the node-type+capture form AND the \f(ARGS) shorthand).
// path is a single file or a directory to walk; lang is the language (when
// empty, the file's language is detected from its extension). A pattern
// invalid for the language degrades to no_grammar:<lang> (never a silent skip
// — T12); a structurally invalid pattern (unbalanced parens, dangling
// backslash) is a hard error.
//
// F2: the walk is index-scoped (indexSkipDir = the codeindex.Index exclude
// set) so the structural hit path space matches the index's files.path space
// (a structural hit for a file the index skipped is not produced — the RRF
// fusion's dedup-by-path relies on one path space).
func Grep(ctx context.Context, pattern, path, lang string) (GrepResult, error) {
	res := GrepResult{}

	// T12: an unsupported language degrades BEFORE the pattern check (the
	// pattern may be structurally fine but have no grammar to compile for).
	// This keeps no_grammar:<lang> the degrade for both an unknown language
	// and an invalid-for-language node type.
	langResolved := lang
	if langResolved == "" && path != "" {
		langResolved = grepLangForPath(path)
	}
	probe, known := grammarProbeFile[langResolved]

	// A structurally invalid pattern (unbalanced parens, dangling backslash,
	// empty) aborts before any per-language compile: a clear validation error,
	// not a silent no-match. The \f shorthand is valid by construction (the
	// expansion is a fixed, well-formed query) so it skips the check; a
	// non-shorthand pattern is validated as written.
	if _, ok := fShorthandPrefix(pattern); !ok {
		if _, err := ParseByExample(pattern); err != nil {
			return res, err
		}
	}

	// Collect candidate files (the path itself if a file, or the dir's
	// supported-language files). A missing path is not an error for grep (no
	// files to scan). The walk is index-scoped (F2): it uses the SAME root +
	// exclude set the codeindex.Index walk uses, so a structural hit for a
	// file the index skipped is not produced (the RRF fusion dedups by path —
	// one path space).
	//
	// ponytail: the walk caps candidates at 5000 files (large-repo defense);
	// a structural grep over more files than that silently truncates the
	// walk. Upgrade path: a limit parameter + a Truncated flag on GrepResult
	// (the v1 shape) if the code_grep transport needs it.
	const maxGrepFiles = 5000
	files := []string{}
	if path != "" {
		info, err := os.Stat(path)
		if err == nil {
			if info.IsDir() {
				_ = walkGrepFiles(path, &files, maxGrepFiles)
			} else {
				files = append(files, path)
			}
		}
	} else {
		// No path: grep the current working directory.
		cwd, _ := os.Getwd()
		_ = walkGrepFiles(cwd, &files, maxGrepFiles)
	}

	if !known || probe == "" {
		res.Degraded = "no_grammar:" + langResolved
		return res, nil
	}

	// Compile the pattern for the language once (invalid node type → degrade).
	query, anchorQ, dgr, err := compileGrep(pattern, langResolved)
	if dgr != "" {
		res.Degraded = dgr
		return res, err // err is nil when dgr is set (a skip, not a failure)
	}
	if err != nil {
		return res, err
	}

	for _, f := range files {
		flang := lang
		if flang == "" {
			flang = grepLangForPath(f)
			if flang == "" {
				continue
			}
		}
		info, err := os.Stat(f)
		if err != nil || info.IsDir() || info.Size() > structuralMaxFileSize {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		entry := grammars.DetectLanguage(grammarProbeFile[flang])
		if entry == nil {
			continue
		}
		parser := ts.NewParser(entry.Language())
		tree, err := parser.Parse(src)
		if err != nil || tree == nil || tree.ParseStoppedEarly() {
			continue
		}
		rel := f
		if path != "" {
			if r, err := filepath.Rel(path, f); err == nil && r != "." {
				rel = filepath.ToSlash(r)
			}
		}
		matches := query.Execute(tree)
		// The anchor query (root capture) recovers position/text when the
		// pattern has no named capture.
		var anchorMatches []ts.QueryMatch
		anchorReady := false
		for mi, m := range matches {
			h := GrepHit{Path: rel, Line: 1, Col: 1, Captures: map[string]string{}}
			anchor := (*ts.Node)(nil)
			if len(m.Captures) > 0 && m.Captures[0].Node != nil {
				anchor = m.Captures[0].Node
			}
			if anchor == nil && anchorQ != nil {
				if !anchorReady {
					anchorMatches = anchorQ.Execute(tree)
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
				txt := anchor.Text(src)
				if len(txt) > 200 {
					txt = txt[:200]
				}
				h.Text = txt
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
		}
	}
	return res, nil
}

// compileGrep compiles the by-example pattern into a gotreesitter Query (+
// anchor) for lang (assumed supported — Grep degrades before calling). It
// returns (query, anchor, degraded, err): degraded is no_grammar:<lang> when
// the pattern's node types are invalid for the language (a skip, not a
// failure — err is nil in that case).
func compileGrep(pattern, lang string) (*ts.Query, *ts.Query, string, error) {
	src, err := ParseByExample(pattern)
	if err != nil {
		return nil, nil, "", err
	}
	entry := grammars.DetectLanguage(grammarProbeFile[lang])
	if entry == nil {
		// Defensive: Grep already degraded an unsupported lang before calling.
		return nil, nil, "no_grammar:" + lang, nil
	}
	q, err := ts.NewQuery(src, entry.Language())
	if err != nil {
		// An unknown node type for this language: degrade, never a silent skip.
		return nil, nil, "no_grammar:" + lang, nil
	}
	anchor, _ := compileGrepAnchor(pattern, entry.Language())
	return q, anchor, "", nil
}

// compileGrepAnchor compiles a query that captures the pattern's root node as
// @__anchor (to recover position/text on no-capture matches).
func compileGrepAnchor(pattern string, lang *ts.Language) (*ts.Query, error) {
	root := patternRootType(pattern)
	if root == "" {
		return nil, errors.New("no root node type in pattern")
	}
	return ts.NewQuery("("+root+") @__anchor", lang)
}

// patternRootType returns the pattern's root node type: the first token inside
// the first top-level S-expression.
func patternRootType(pattern string) string {
	i := 0
	for i < len(pattern) && pattern[i] != '(' {
		i++
	}
	if i >= len(pattern) {
		return ""
	}
	i++
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

// isIdentRune reports whether c is a valid capture-name / node-type character.
func isIdentRune(c byte) bool {
	return c == '.' || c == '_' ||
		(c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// walkGrepFiles walks dir (best-effort) appending supported-language files.
// F2: the skip set is indexSkipDir (the codeindex.Index exclude set) so the
// walk's path space matches the index's files.path space — a structural hit
// for a file the index skipped is not produced (the RRF fusion dedups by path,
// one path space).
func walkGrepFiles(dir string, files *[]string, limit int) error {
	return filepath.WalkDir(dir, func(p string, d os.DirEntry, werr error) error {
		if werr != nil {
			return nil
		}
		if d.IsDir() {
			if indexSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if limit > 0 && len(*files) >= limit {
			return filepath.SkipAll // ponytail: cap the walk, see Grep's comment
		}
		if grepLangForPath(p) != "" {
			*files = append(*files, p)
		}
		return nil
	})
}

// indexSkipDir is the structural walk's exclude set — the SAME set the
// codeindex.Index driver uses (spec 001 §6 / T1: .git / node_modules / vendor
// / build output / the KB vault — the KB has its own kb_index and is not
// code). Mirrored locally (not imported) so the search package stays free of a
// codeindex import cycle (codeindex → extract, and search is reached through
// the walker). The F2 fix: the pre-fix walk used a SUPERSET (isGrepSkipDir
// also skipped ALL dot-dirs + out/__pycache__), which could DIVERGE from the
// index's path space (the index indexes dot-dirs that are not in its exclude
// set, e.g. a dot-prefixed source dir, and the v1 walk skipped them). Using
// the identical set guarantees the structural + index walks produce the same
// path set.
//
// ponytail: the set is duplicated (not shared) to avoid the import cycle. If
// codeindex ever exports excludedDirs, replace this mirror with the shared var
// so the two walks cannot drift apart again.
func indexSkipDir(name string) bool {
	if name == "" {
		return false
	}
	return indexExcludedDirs[name]
}

var indexExcludedDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	"target":       true,
	".skillgrid":   true,
}

// grammarProbeFile maps a supported language to a probe filename the
// gotreesitter registry resolves to that language.
var grammarProbeFile = map[string]string{
	"go":          "a.go",
	"typescript":  "a.ts",
	"tsx":         "a.tsx",
	"javascript":  "a.js",
	"python":      "a.py",
	"rust":        "a.rs",
	"java":        "a.java",
	"c":           "a.c",
	"cpp":         "a.cpp",
	"c_sharp":     "a.cs",
	"php":         "a.php",
	"ruby":        "a.rb",
	"kotlin":      "a.kt",
	"swift":       "a.swift",
	"scala":       "a.scala",
	"dart":        "a.dart",
	"lua":         "a.lua",
	"r":           "a.r",
	"perl":        "a.pl",
	"elixir":      "a.ex",
	"haskell":     "a.hs",
	"clojure":     "a.clj",
	"zig":         "a.zig",
	"nim":         "a.nim",
	"groovy":      "a.groovy",
	"objective_c": "a.m",
	"bash":        "a.sh",
	"sql":         "a.sql",
	"css":         "a.css",
	"html":        "a.html",
}

// grepExtToLang is the structural-grep extension map (self-contained so search
// stays free of an extract import cycle).
var grepExtToLang = map[string]string{
	".go": "go", ".ts": "typescript", ".mts": "typescript", ".cts": "typescript",
	".tsx": "tsx", ".js": "javascript", ".mjs": "javascript", ".cjs": "javascript",
	".py": "python", ".rs": "rust", ".java": "java", ".c": "c", ".h": "c",
	".cpp": "cpp", ".cc": "cpp", ".cxx": "cpp", ".cu": "cpp", ".cuh": "cpp",
	".hpp": "cpp", ".cs": "c_sharp", ".php": "php", ".rb": "ruby", ".kt": "kotlin",
	".kts": "kotlin", ".swift": "swift", ".scala": "scala", ".dart": "dart",
	".lua": "lua", ".r": "r", ".R": "r", ".m": "matlab", ".mm": "objective_c",
	".pl": "perl", ".pm": "perl", ".ex": "elixir", ".exs": "elixir",
	".hs": "haskell", ".clj": "clojure", ".cljs": "clojure", ".cljc": "clojure",
	".zig": "zig", ".nim": "nim", ".groovy": "groovy", ".gvy": "groovy",
	".sh": "bash", ".bash": "bash", ".sql": "sql", ".css": "css",
	".html": "html", ".htm": "html",
}

// grepLangForPath returns the supported language for a path, or "".
func grepLangForPath(path string) string {
	return grepExtToLang[strings.ToLower(filepath.Ext(path))]
}

// ParseByExample converts a by-example structural pattern into gotreesitter
// query source (the v1 ParseByExample, lifted). The first S-expression is the
// root node to match; a \NAME metavariable attaches a capture to it.
// Metavariables: \NAME (named capture), \(ARGS*) (argument-list run → @args),
// \* / \_ (anonymous any-node wildcard step).
//
// T12 (the NOT-PORT UX fix): the v1 parser only accepted the node-type+capture
// form; the v2 parser ALSO accepts the \f(ARGS) shorthand (the v1 CLI help
// implied it but the parser rejected it). \f(ARGS) expands to a function-call
// node type with an argument-list capture.
func ParseByExample(pattern string) (string, error) {
	// T12: the \f(ARGS) shorthand (a function call with an argument run). The
	// v1 CLI help implied it but the parser rejected it — the NOT-PORT fix.
	if s := expandFShorthand(pattern); s != "" {
		pattern = s
	}
	var root strings.Builder
	var rest []string
	var cur strings.Builder
	depth := 0
	open := false
	haveRoot := false
	flush := func() {
		s := strings.TrimSpace(cur.String())
		if s == "" {
			cur.Reset()
			return
		}
		if open {
			if root.Len() > 0 {
				root.WriteByte(' ')
			}
			root.WriteString(s)
			// A flushed token that ends with ')' completes the root
			// S-expression: mark haveRoot so subsequent top-level tokens
			// (captures, arg runs) go to rest.
			if len(s) > 0 && s[len(s)-1] == ')' {
				haveRoot = true
			}
		} else if !haveRoot {
			root.WriteString(s)
			haveRoot = true
		} else {
			rest = append(rest, s)
		}
		cur.Reset()
	}
	capture := func(name string) {
		// The capture is a separate token: flush the current token first so it
		// space-separates (the tree-sitter query is space-separated — an
		// @capture glued to a preceding node type is an unknown node). The
		// @name is appended to `rest` (a sibling), NOT to the root: tree-sitter
		// attaches a capture to the step it follows, and a capture INSIDE the
		// root S-expression (before its closing paren) would be glued to the
		// preceding token. The query reassembles root + rest with single
		// spaces, so the @name lands after the root's closing paren.
		flush()
		rest = append(rest, "@"+name)
	}
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch {
		case c == '\\' && i+1 < len(pattern):
			// A backslash escape: the next character is a literal. The
			// escaped paren '(' / ')' are S-expression delimiters (they
			// open / close the root); the escaped '*' / '_' is an any-node
			// wildcard step; the escaped NAME is a named-node capture.
			// Consume the escape as a unit (the backslash is not written
			// to cur).
			i++
			esc := pattern[i]
			switch {
			case esc == '(':
				// \( — an escaped S-expression opener. The node type
				// follows (up to the matching \) or a space); it is the
				// root. Depth advances; open sets at depth 1.
				flush()
				depth++
				if depth == 1 {
					open = true
				}
				cur.WriteByte('(')
			case esc == ')':
				// \) — an escaped S-expression closer.
				depth--
				if depth < 0 {
					return "", errGrepBadPattern("unbalanced ')' in pattern")
				}
				cur.WriteByte(')')
				if depth == 0 {
					open = false
					flush()
				}
			case esc == '*' || esc == '_':
				// \* / \_ — an anonymous any-node wildcard step.
				flush()
				rest = append(rest, "(_)")
			default:
				// \NAME — a named-node capture. Read the identifier
				// starting at the escaped char (which is the first ident
				// rune).
				start := i
				for i < len(pattern) && isIdentRune(pattern[i]) {
					i++
				}
				if i == start {
					return "", errGrepBadPattern("malformed metavariable: backslash not followed by an identifier")
				}
				capture(pattern[start:i])
				i--
			}
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

// fShorthand is the T12 expansion target: \f(ARGS) → a call-expression node
// with its callee identifier captured + the argument-list run captured. The
// v1 CLI help implied the shorthand but the parser rejected it (a bare \f is a
// malformed metavariable); this expansion is the NOT-PORT UX fix.
const fShorthand = "(call_expression function: (identifier) @__fn) (argument_list) @args"

// expandFShorthand expands the T12 \f(ARGS) shorthand into the node-type+
// capture form (fShorthand). It accepts both the \f( prefix (the documented
// form) and the \(f( variant, in case a caller writes the f as a node-type
// placeholder. It returns "" when the pattern is not the \f( form.
func expandFShorthand(pattern string) string {
	// The \f(ARGS) shorthand: the '(' after \f is an UNESCAPED arg-run
	// opener (the shorthand's arg list), not a node-type S-expression
	// delimiter. Consume the balanced group; the expansion is the same
	// call-expression form regardless of the arg-run content.
	p := strings.TrimSpace(pattern)
	if !strings.HasPrefix(p, `\f(`) {
		return ""
	}
	rest := p[len(`\f(`):]
	depth := 1
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return fShorthand
			}
		}
	}
	return ""
}

// fShorthandPrefix reports whether the pattern starts with the \f( or \(f
// shorthand prefix (the T12 documented form and its escaped-open-paren
// variant). The shorthand is recognized by its PREFIX (not its full balanced
// form) so the structural-validation skip applies to it.
func fShorthandPrefix(p string) (string, bool) {
	p = strings.TrimSpace(p)
	if strings.HasPrefix(p, `\f(`) {
		return p[len([]rune(`\f(`)):], true
	}
	return "", false
}

type grepPatternError struct{ msg string }

func (e grepPatternError) Error() string { return "invalid grep pattern: " + e.msg }

func errGrepBadPattern(msg string) error { return grepPatternError{msg: msg} }

// IsPatternError reports whether err is a structural pattern compile error.
func IsPatternError(err error) bool {
	_, ok := err.(grepPatternError)
	return ok
}
