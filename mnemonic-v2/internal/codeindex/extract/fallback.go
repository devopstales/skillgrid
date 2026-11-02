package extract

import (
	"regexp"
	"strings"
)

// The regex battery (the deterministic floor, INV-3 — the tier the ladder
// falls to for unmapped extensions, malformed files, and quarantined
// grammars). Lifted from the v1 fallback.go: function/class declaration
// regexes + import regexes + a conservative next-function call chain. The
// v1 dead `fallbackRegexes` (the broad name-pattern list) is dropped.

// Confidence labels carried by every edge.
const (
	ConfidenceExtracted = "EXTRACTED"
	ConfidenceInferred  = "INFERRED"
	ConfidenceAmbiguous = "AMBIGUOUS"
)

// fallbackFuncRegexes identify function-like declarations by their name so the
// fallback emits symbols.
var fallbackFuncRegexes = []*regexp.Regexp{
	regexp.MustCompile(`(?m)(?:^|\s)(?:public\s+|private\s+|protected\s+|static\s+|final\s+|abstract\s+|synchronized\s+|native\s+|volatile\s+|transient\s+|strictfp\s+|pub\s+|async\s+|unsafe\s+|const\s+|export\s+|default\s+|def\s+|fn\s+|func\s+|sub\s+|fun\s+|proc\s+|defn\s+|defn-\s+|function\s+)([A-Za-z_$][\w$]*)\s*\(`),
	regexp.MustCompile(`(?m)(?:^|\s)(def|fn|func|sub|fun|proc|defn|defn-|function)\s+([A-Za-z_$][\w$]*)\s*\(`),
}

// fallbackClassRegex identifies class/interface declarations so the fallback
// emits class symbols (not just functions).
var fallbackClassRegex = regexp.MustCompile(`(?m)^\s*(?:export\s+)?(?:default\s+)?class\s+([A-Za-z_$][\w$]*)`)

// fallbackWordAny captures a plain identifier bound on its own line (a
// variable/const reference) so the fallback emits reference symbols.
var fallbackWordAny = regexp.MustCompile(`(?m)^\s*([A-Za-z_$][\w$]{2,})\s*[:=]`)

// fallbackImportRegexes capture import-ish lines for the fallback.
var fallbackImportRegexes = []*regexp.Regexp{
	regexp.MustCompile(`(?m)^\s*import\s+([^\n;]+)`),
	regexp.MustCompile(`(?m)^\s*use\s+([^\n;]+)`),
	regexp.MustCompile(`(?m)^\s*#include\s+([^\n]+)`),
	regexp.MustCompile(`(?m)^\s*require\s*[\'"]([^\'"]+)[\'"]`),
	regexp.MustCompile(`(?m)^\s*from\s+([\w./\\\'"\\-]+)\s+import`),
}

// Fallback extracts a FileGraph for path using only regexes. It always returns
// a non-nil, Indexed graph (possibly with zero symbols) and never errors.
func Fallback(path, lang string, src []byte) FileGraph {
	if lang == "" {
		lang = DetectLanguage(path)
	}
	text := string(src)
	g := FileGraph{Tier: TierRegex}

	type foundSym struct {
		name string
		line int
		kind string
	}
	var syms []foundSym
	seen := map[string]bool{}
	addSym := func(name, kind string, line int) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		syms = append(syms, foundSym{name: name, kind: kind, line: line})
	}

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lineNo := i + 1
		added := false
		for _, re := range fallbackFuncRegexes {
			m := re.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			name := ""
			for j := len(m) - 1; j >= 1; j-- {
				if m[j] != "" {
					name = m[j]
					break
				}
			}
			kind := "function"
			if cm := fallbackClassRegex.FindStringSubmatch(line); cm != nil {
				kind = "class"
				name = cm[1]
			}
			addSym(name, kind, lineNo)
			added = true
			break
		}
		if !added {
			if cm := fallbackClassRegex.FindStringSubmatch(line); cm != nil {
				addSym(cm[1], "class", lineNo)
				continue
			}
			if m := fallbackWordAny.FindStringSubmatch(line); m != nil {
				addSym(m[1], "reference", lineNo)
			}
		}
		for _, re := range fallbackImportRegexes {
			m := re.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			target := strings.TrimSpace(m[1])
			target = strings.TrimSuffix(target, ";")
			target = strings.TrimSpace(target)
			if target == "" {
				continue
			}
			// A name-only import (no statically-bound target) — to_id NULL.
			g.Edges = append(g.Edges, Edge{
				Kind:       "imports",
				ToID:       nil,
				ToName:     target,
				Confidence: ConfidenceAmbiguous,
				Line:       lineNo,
			})
			break
		}
	}

	for _, s := range syms {
		g.Symbols = append(g.Symbols, Symbol{
			Name:          s.name,
			QualifiedName: s.name,
			Kind:          s.kind,
			Language:      lang,
			StartLine:     s.line,
			EndLine:       s.line,
			ContentHash:   contentHash([]byte(s.name)),
			UID:           symbolUID(s.name, s.kind, s.line, s.line),
		})
	}

	// Connect each function symbol to the next function symbol as an AMBIGUOUS
	// call edge — a conservative, name-only link so the graph is non-trivial
	// even for unsupported languages (to_id NULL: not statically bound).
	for i := 0; i+1 < len(g.Symbols); i++ {
		if !functionKinds[g.Symbols[i].Kind] || !functionKinds[g.Symbols[i+1].Kind] {
			continue
		}
		g.Edges = append(g.Edges, Edge{
			Kind:       "calls",
			FromID:     0,
			ToID:       nil,
			ToName:     g.Symbols[i+1].Name,
			Confidence: ConfidenceAmbiguous,
			Line:       g.Symbols[i].EndLine,
		})
	}

	return g
}
