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
