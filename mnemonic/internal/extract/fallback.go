package extract

import (
	"regexp"
	"strings"
)

// fallbackRegexes are the name patterns used by the regex fallback extractor
// for unknown languages, malformed files, and quarantined grammars.
var fallbackRegexes = []*regexp.Regexp{
	regexp.MustCompile(`(?m)^\s*(?:public\s+|private\s+|protected\s+|static\s+|final\s+|abstract\s+|synchronized\s+|native\s+|volatile\s+|transient\s+|strictfp\s+)*[A-Za-z_][\w<>\[\],\s]*\s+([A-Za-z_]\w*)\s*\([^)]*\)\s*(?:throws\s+[\w\s,.]+)?\s*\{`),
	regexp.MustCompile(`(?m)^\s*(?:export\s+)?(?:default\s+)?(?:async\s+)?function\s*\*?\s*([A-Za-z_$][\w$]*)`),
	regexp.MustCompile(`(?m)^\s*(?:export\s+)?(?:default\s+)?class\s+([A-Za-z_$][\w$]*)`),
	regexp.MustCompile(`(?m)^\s*(?:export\s+)?(?:async\s+)?function\s+([A-Za-z_$][\w$]*)`),
	regexp.MustCompile(`(?m)^\s*def\s+([A-Za-z_]\w*)`),
	regexp.MustCompile(`(?m)^\s*(?:pub\s+|async\s+|unsafe\s+|const\s+)*fn\s+([A-Za-z_]\w*)`),
	regexp.MustCompile(`(?m)^\s*(?:pub\s+|async\s+|unsafe\s+|extern\s+|static\s+)*func\s+([A-Za-z_]\w*)`),
	regexp.MustCompile(`(?m)^\s*sub\s+([A-Za-z_]\w*)`),
	regexp.MustCompile(`(?m)^\s*(?:defn|defn-)\s+([A-Za-z_?!\-][\w?!\-]*)`),
	regexp.MustCompile(`(?m)^\s*(?:const|let|var)\s+([A-Za-z_$][\w$]*)\s*[:=]`),
	regexp.MustCompile(`(?m)^\s*function\s+([A-Za-z_]\w*)`),
	regexp.MustCompile(`(?m)^\s*(?:@|/|!)?[A-Za-z_]\w*\s*\(\)`),
}

// fallbackWordRegex captures a plain identifier on its own line (or after a
// common declaration keyword) so the fallback emits symbols even for languages
// with no recognizable declaration shape.
var fallbackWordRegex = regexp.MustCompile(`(?m)^\s*([A-Za-z_$][\w$]{2,})\s*[:=(]?\s*$`)
var fallbackWordAny = regexp.MustCompile(`(?m)^\s*([A-Za-z_$][\w$]{2,})\s*[:=]`)

// fallbackFuncRegexes identify function-like declarations by their name so the
// fallback can emit symbols.
var fallbackFuncRegexes = []*regexp.Regexp{
	regexp.MustCompile(`(?m)(?:^|\s)(?:public\s+|private\s+|protected\s+|static\s+|final\s+|abstract\s+|synchronized\s+|native\s+|volatile\s+|transient\s+|strictfp\s+|pub\s+|async\s+|unsafe\s+|const\s+|export\s+|default\s+|def\s+|fn\s+|func\s+|sub\s+|fun\s+|proc\s+|defn\s+|defn-\s+|function\s+)([A-Za-z_$][\w$]*)\s*\(`),
	regexp.MustCompile(`(?m)(?:^|\s)(def|fn|func|sub|fun|proc|defn|defn-|function)\s+([A-Za-z_$][\w$]*)\s*\(`),
}

// fallbackImportRegexes capture import-ish lines for the fallback.
var fallbackImportRegexes = []*regexp.Regexp{
	regexp.MustCompile(`(?m)^\s*import\s+([^\n;]+)`),
	regexp.MustCompile(`(?m)^\s*use\s+([^\n;]+)`),
	regexp.MustCompile(`(?m)^\s*#include\s+([^\n]+)`),
	regexp.MustCompile(`(?m)^\s*require\s*[\'"]([^\'"]+)[\'"]`),
	regexp.MustCompile(`(?m)^\s*from\s+([\w./\\\'"\\-]+)\s+import`),
}

// Fallback extracts a FileGraph for path using only regexes. It is used for
// unknown languages, malformed files, and quarantined grammars. It always
// returns a non-nil graph (possibly with zero symbols) and never errors.
func Fallback(path string, src []byte) *FileGraph {
	lang := ""
	if l := DetectLanguage(path); l != "" {
		lang = l
	}
	text := string(src)
	g := &FileGraph{
		Path:      path,
		Language:  lang,
		Extractor: "regex",
	}

	type foundSym struct {
		name      string
		line      int
		kind      string
		qualified string
	}
	var syms []foundSym
	seen := map[string]bool{}
	addSym := func(name, kind string, line int) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		syms = append(syms, foundSym{name: name, kind: kind, line: line, qualified: name})
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
			// The name is the last capture that is non-empty.
			name := ""
			for j := len(m) - 1; j >= 1; j-- {
				if m[j] != "" {
					name = m[j]
					break
				}
			}
			kind := "function"
			if strings.Contains(line, "class ") {
				kind = "class"
			}
			addSym(name, kind, lineNo)
			added = true
			break
		}
		if !added {
			if m := fallbackWordAny.FindStringSubmatch(line); m != nil {
				addSym(m[1], "reference", lineNo)
				continue
			}
			if m := fallbackWordRegex.FindStringSubmatch(line); m != nil {
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
			g.Edges = append(g.Edges, Edge{
				Kind:       "imports",
				ToName:     target,
				TargetPath: target,
				Confidence: ConfidenceAmbiguous,
				Line:       lineNo,
			})
			break
		}
	}

	for _, s := range syms {
		g.Symbols = append(g.Symbols, Symbol{
			Name:          s.name,
			QualifiedName: s.qualified,
			Kind:          s.kind,
			Language:      lang,
			StartLine:     s.line,
			EndLine:       s.line,
			ContentHash:   contentHash([]byte(s.name)),
			UID:           symbolUID(s.qualified, s.kind, s.line, s.line),
		})
	}

	// Connect each function symbol to the next function symbol as an
	// AMBIGUOUS call edge — a conservative, name-only link so the graph is
	// non-trivial even for unsupported languages.
	for i := 0; i+1 < len(g.Symbols); i++ {
		g.Edges = append(g.Edges, Edge{
			Kind:       "calls",
			FromUID:    g.Symbols[i].UID,
			ToUID:      g.Symbols[i+1].UID,
			ToName:     g.Symbols[i+1].Name,
			Confidence: ConfidenceAmbiguous,
			Line:       g.Symbols[i].EndLine,
		})
	}

	return g
}
