// Package extract turns source files into a queryable graph of symbols and
// edges. The Extractor interface is backed by a gotreesitter adapter (30
// supported languages) with a regex fallback for unknown languages, malformed
// files, and quarantined grammars.
package extract

import (
	"fmt"
	"hash/fnv"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Confidence labels carried by every Edge.
const (
	ConfidenceExtracted = "EXTRACTED"
	ConfidenceInferred  = "INFERRED"
	ConfidenceAmbiguous = "AMBIGUOUS"
)

// ConfidenceScore maps a categorical confidence label to a numeric score for
// edge weighting (graphify-style): EXTRACTED=1.0, INFERRED=0.85, AMBIGUOUS=0.5.
func ConfidenceScore(label string) float64 {
	switch label {
	case ConfidenceInferred:
		return 0.85
	case ConfidenceAmbiguous:
		return 0.5
	default:
		return 1.0
	}
}

// Symbol is one indexed code unit (function, method, type, ...).
type Symbol struct {
	Name          string
	QualifiedName string
	Kind          string
	Language      string
	Signature     string
	StartLine     int
	EndLine       int
	ContentHash   string
	UID           string

	// Type/visibility enrichment, derived from the source span (the
	// gotreesitter DefinitionSpan carries only name/kind/range). Empty means
	// "not derivable / not a function", never "unknown-but-present".
	// ReturnType is the function return type, ParamTypes is the ordered
	// parameter-type list, Visibility is export/private/unexported/unknown,
	// IsExported is the boolean form of Visibility for fast filtering.
	ReturnType string
	ParamTypes string
	Visibility string
	IsExported bool
}

// UnresolvedMember is a receiver-qualified call site (obj.Scan(...)) whose
// receiver the extractor cannot statically bind. It is logged to the
// unresolved_members table so the call-resolution backlog is auditable: an
// agent can see "these N call sites are unresolvable, here is the shape of
// why" instead of a bare missing-edge.
type UnresolvedMember struct {
	FilePath string
	Language string
	// Member is the called name (Scan, QueryRow, ...); Receiver is the
	// qualifier (rows, x, ...). External marks members that resolve to
	// external/stdlib APIs (e.g. a .catch on a JS promise) rather than an
	// in-repo symbol.
	Member   string
	Receiver string
	External bool
	Line     int
}

// AuditCounts is the per-run, per-language call-resolution ledger: how many
// call sites were extracted and how many receiver-qualified members were left
// unresolved. It quantifies the drop-not-guess policy so code_status can show
// the resolver's health ("across N Go calls, M members were unresolved").
type AuditCounts struct {
	Lang               string
	CallSites          int
	Unresolved         int
	ExternalUnresolved int
}

// Edge connects two symbols (or a symbol and an unresolved target name).
type Edge struct {
	Kind            string
	FromUID         string
	ToUID           string
	ToName          string
	TargetPath      string
	Confidence      string
	Context         string
	ConfidenceScore float64
	Line            int
}

// FileGraph is the extraction result for one file.
type FileGraph struct {
	Path       string
	Language   string
	Symbols    []Symbol
	Edges      []Edge
	Rationales []Rationale
	Extractor  string // "treesitter" | "regex"
	Error      string // set when the primary extractor failed and fallback was used

	// UnresolvedMembers are receiver-qualified call sites the extractor could
	// not statically bind (logged to the unresolved_members table).
	UnresolvedMembers []UnresolvedMember
	// Audit is the per-language call-resolution ledger (call sites vs
	// unresolved receiver members) for the drop-not-guess health signal.
	Audit AuditCounts
}

// Extractor produces a FileGraph for a single source file. Implementations
// must never abort the index run: per-file failures fall back to regex and
// return a well-formed (possibly empty) FileGraph.
type Extractor interface {
	ExtractFile(path string, src []byte) (*FileGraph, error)
}

// SupportedLanguages is the 30-language scope for step 01.
var SupportedLanguages = []string{
	"go", "typescript", "tsx", "javascript", "python", "rust", "java", "c",
	"cpp", "c_sharp", "php", "ruby", "kotlin", "swift", "scala", "dart",
	"lua", "r", "matlab", "perl", "elixir", "haskell", "clojure", "zig",
	"nim", "groovy", "objective_c", "bash", "sql", "css",
}

// LanguageSet is the fast-lookup form of SupportedLanguages.
var LanguageSet = func() map[string]bool {
	m := make(map[string]bool, len(SupportedLanguages))
	for _, l := range SupportedLanguages {
		m[l] = true
	}
	return m
}()

// grammarLoaderFn loads a gotreesitter language for a language name. The
// default implementation resolves via the gotreesitter grammar registry;
// tests may override it to inject crashing grammars.
type grammarLoaderFn func(name string) (func() any, bool)

// Pool bounds worker self-healing: a crashing grammar quarantines the file
// (regex fallback), the worker respawns up to a bound per slot, and a circuit
// breaker trips if one grammar dies repeatedly.
type Pool struct {
	mu          sync.Mutex
	quarantined map[string]map[string]bool // lang -> path
	breaker     map[string]bool            // lang
	respawns    map[string]int             // lang
	deaths      map[string]int             // lang
	MaxRespawns int
	TripAfter   int
}

// NewPool returns a self-healing pool with default bounds.
func NewPool() *Pool {
	return &Pool{
		quarantined: map[string]map[string]bool{},
		breaker:     map[string]bool{},
		respawns:    map[string]int{},
		deaths:      map[string]int{},
		MaxRespawns: 3,
		TripAfter:   3,
	}
}

// IsQuarantined reports whether path for lang is quarantined.
func (p *Pool) IsQuarantined(lang, path string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.quarantined[lang][path]
}

// IsBreakerTripped reports whether the circuit breaker for lang has tripped.
func (p *Pool) IsBreakerTripped(lang string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.breaker[lang]
}

// RespawnsUsed returns how many respawns were recorded for lang.
func (p *Pool) RespawnsUsed(lang string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.respawns[lang]
}

// Quarantine marks path for lang as quarantined (regex fallback).
func (p *Pool) Quarantine(lang, path string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.quarantined[lang] == nil {
		p.quarantined[lang] = map[string]bool{}
	}
	p.quarantined[lang][path] = true
}

// recordDeath counts a grammar death, respawns the worker up to the bound,
// and trips the breaker on repeated deaths.
func (p *Pool) recordDeath(lang string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.deaths[lang]++
	if p.deaths[lang] <= p.MaxRespawns {
		p.respawns[lang]++
	}
	if p.deaths[lang] >= p.TripAfter {
		p.breaker[lang] = true
	}
}

// extractor is the default Extractor: gotreesitter adapter + regex fallback.
type extractor struct {
	pool    *Pool
	grammar grammarLoaderFn
}

// Default returns the shared Extractor backed by gotreesitter with the
// self-healing pool.
func Default() *extractor {
	return &extractor{pool: NewPool()}
}

// resolveGrammarFor returns a lazy loader for the gotreesitter language named
// langName. The grammar field (override hook) is used by tests to inject
// crashes; the production path uses lookupGrammar.
func (e *extractor) resolveGrammarFor(langName string) (func() any, bool) {
	if e.grammar != nil {
		return e.grammar(langName)
	}
	return lookupGrammar(langName)
}

// IsQuarantined delegates to the pool.
func (e *extractor) IsQuarantined(lang, path string) bool {
	return e.pool.IsQuarantined(lang, path)
}

// IsBreakerTripped delegates to the pool.
func (e *extractor) IsBreakerTripped(lang string) bool {
	return e.pool.IsBreakerTripped(lang)
}

// RespawnsUsed delegates to the pool.
func (e *extractor) RespawnsUsed(lang string) int {
	return e.pool.RespawnsUsed(lang)
}

// extToLang maps file extensions to their supported language. Grammar reuse
// is applied here (e.g. .mts/.cts -> typescript, .cu/.cuh -> cpp).
var extToLang = map[string]string{
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
	".hh":     "cpp",
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

// DetectLanguage returns the supported language for a file path, or "" if the
// extension is not in the 30-language scope. Grammar reuse maps are applied
// (e.g. .mts -> typescript, .cu -> cpp).
func DetectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		return ""
	}
	lang, ok := extToLang[ext]
	if !ok {
		return ""
	}
	if !LanguageSet[lang] {
		return ""
	}
	return lang
}

// lookupGrammar resolves a gotreesitter language loader by name via the
// registry. It is the production grammarFor.
var lookupGrammar grammarLoaderFn = resolveGrammar

// Resolve returns the gotreesitter registry entry name for a detected
// language. This is a thin alias kept for adapter clarity.
func Resolve(lang string) string {
	return lang
}

// qualifiedName builds the symbol's qualified name from its enclosing scope.
func qualifiedName(parts []string, name string) string {
	parts = append(parts, name)
	return strings.Join(parts, ".")
}

// symbolUID derives a stable, path-independent identity for a symbol from its
// qualified name, kind, and span. Content is excluded so a moved function
// keeps its identity across files (edges re-target by UID).
func symbolUID(qualified, kind string, startLine, endLine int) string {
	h := fnv.New64a()
	fmt.Fprintf(h, "%s\x00%s\x00%d:%d", qualified, kind, startLine, endLine)
	return fmt.Sprintf("%x", h.Sum64())
}

// contentHash hashes a symbol's source span for re-extraction detection.
func contentHash(text []byte) string {
	h := fnv.New64a()
	h.Write(text)
	return fmt.Sprintf("%x", h.Sum64())
}

// lineOf converts a byte offset to a 1-based line number.
func lineOf(src []byte, offset uint32) int {
	if int(offset) >= len(src) {
		offset = uint32(len(src))
	}
	n := 1
	for i := uint32(0); i < offset; i++ {
		if src[i] == '\n' {
			n++
		}
	}
	return n
}

// signatureOf extracts the declaration text of a symbol span, trimmed to a
// single line for display.
func signatureOf(src []byte, start, end uint32) string {
	if start > end || int(end) > len(src) {
		return ""
	}
	text := string(src[start:end])
	if idx := strings.IndexByte(text, '\n'); idx >= 0 {
		text = text[:idx]
	}
	text = strings.TrimSpace(text)
	if len(text) > 200 {
		text = text[:200]
	}
	return text
}

// typeInfo derives a function's return type and parameter types from its
// signature span. The gotreesitter DefinitionSpan is name/kind/range only, so
// this is a Go/TS heuristic over the source text, not type inference: it
// returns "" for non-functions or anything it cannot parse (never
// "unknown-but-present"). It is deliberately conservative — a wrong type is
// worse than an absent one for rename/impact reasoning.
type typeInfo struct {
	ReturnType string
	ParamTypes string
}

func deriveTypeInfo(lang, name, sig string) typeInfo {
	var ti typeInfo
	if sig == "" || name == "" {
		return ti
	}
	switch lang {
	case "go":
		// A Go signature: [func ]Name(params) [results]. The callee name
		// anchors the parameter list so we never mistake an assignment.
		anchored := sig
		if i := strings.Index(anchored, name+"("); i >= 0 {
			anchored = anchored[i:]
		} else {
			return ti
		}
		open := strings.Index(anchored, "(")
		if open < 0 {
			return ti
		}
		closeParen := matchingParen(anchored, open)
		if closeParen < 0 {
			return ti
		}
		inner := strings.TrimSpace(anchored[open+1 : closeParen])
		if inner != "" {
			ti.ParamTypes = extractParamTypes(inner)
		}
		ti.ReturnType = cleanSingleType(anchored[closeParen+1:])
	case "typescript", "tsx", "javascript":
		// A TS signature: Name(params): ReturnType.
		anchored := sig
		if i := strings.Index(anchored, name+"("); i >= 0 {
			anchored = anchored[i:]
		} else {
			return ti
		}
		open := strings.Index(anchored, "(")
		if open < 0 {
			return ti
		}
		closeParen := matchingParen(anchored, open)
		if closeParen < 0 {
			return ti
		}
		inner := strings.TrimSpace(anchored[open+1 : closeParen])
		if inner != "" {
			ti.ParamTypes = extractParamTypes(inner)
		}
		rest := anchored[closeParen+1:]
		if i := strings.IndexByte(rest, ':'); i >= 0 {
			ti.ReturnType = cleanSingleType(rest[i+1:])
		}
	}
	return ti
}

// extractParamTypes reduces a raw parameter list (names + types) to just the
// types, dropping Go parameter names and TS parameter names.
func extractParamTypes(inner string) string {
	parts := splitTopLevel(inner)
	var types []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// A Go/TS parameter is either `name type` or `type` (bare). Heuristic:
		// if the last token is a known type-ish token, keep from it; otherwise
		// keep the whole thing (it is already type-only).
		fields := strings.Fields(p)
		if len(fields) >= 2 {
			// Keep the trailing type tokens (everything after the name). The
			// name is the first field; the rest is the type.
			types = append(types, strings.Join(fields[1:], " "))
		} else {
			types = append(types, p)
		}
	}
	return strings.Join(types, ", ")
}

// cleanSingleType reduces a raw return-type tail to a single clean type: it
// strips trailing braces/semicolons, unwraps a Go multi-return paren
// (`(int, error)`), and keeps only the first top-level comma-separated type.
func cleanSingleType(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, " {;")
	s = strings.TrimSpace(s)
	// A Go multi-return is wrapped in parens: `(*T, error)`. Unwrap.
	if strings.HasPrefix(s, "(") {
		if end := matchingParen(s, 0); end >= 0 {
			s = strings.TrimSpace(s[1:end])
		}
	}
	// Keep only the first top-level comma-separated type (drop `, error`).
	if i := strings.IndexByte(s, ','); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return s
}

// matchingParen returns the index of the ')' matching the '(' at open, or -1.
func matchingParen(s string, open int) int {
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// splitTopLevel splits on commas that are not nested in (), [], {}, or a
// string/character literal.
func splitTopLevel(s string) []string {
	var parts []string
	depth := 0
	start := 0
	var quote rune
	inStr := false
	for i, r := range s {
		switch {
		case inStr:
			if r == quote && s[i-1] != '\\' {
				inStr = false
			}
		case r == '"' || r == '\'' || r == '`':
			quote = r
			inStr = true
		case r == '(' || r == '[' || r == '{':
			depth++
		case r == ')' || r == ']' || r == '}':
			depth--
		case r == ',' && depth == 0:
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// sortedNames returns the sorted unique names (for deterministic output).
func sortedNames(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}
