// Rationale extraction: # NOTE: / # WHY: / // WHY: / ADR-RFC citations →
// rationale nodes linked to the nearest enclosing symbol. A rationale is the
// *why* behind code: a comment explaining a decision or citing a spec/ADR/RFC.
package extract

import (
	"regexp"
	"strings"
)

// Rationale is one extracted rationale node.
type Rationale struct {
	Text string `json:"text"`
	Kind string `json:"kind"` // "note" | "why" | "adr"
	Line int    `json:"line"`
	// SymbolUID is the nearest enclosing symbol's UID (empty if no enclosing
	// symbol exists, e.g. a file-level comment before the first definition).
	SymbolUID string `json:"symbol_uid,omitempty"`
}

// rationalePatterns are the comment markers that signal a rationale. The text
// is the portion after the marker (trimmed). ADR/RFC citations are detected
// separately on the comment line.
var (
	// NOTE/WHY markers with a colon: "# NOTE:", "# WHY:", "// WHY:", "/* NOTE: */".
	markerNote = regexp.MustCompile(`(?m)^[ \t]*(?:#|//|/\*)[ \t]*NOTE[ \t]*:`)
	markerWhy  = regexp.MustCompile(`(?m)^[ \t]*(?:#|//|/\*)[ \t]*WHY[ \t]*:`)
	// ADR / RFC / spec citations anywhere in a comment line.
	adrRef = regexp.MustCompile(`(?i)\b(ADR|RFC|spec)\b\s*[-–#]?\s*[0-9]+`)
	// A comment line start (any of the common comment markers).
	commentLine = regexp.MustCompile(`(?m)^[ \t]*(?:#|//|/\*)`)
)

// ExtractRationale scans src for rationale comments and links each to the
// nearest enclosing symbol (the definition whose line-range contains the
// comment line; otherwise the closest preceding definition). It is language-
// agnostic: it works on raw comment text, not the AST, so it covers every
// supported language uniformly.
func ExtractRationale(src []byte, symbols []Symbol) []Rationale {
	lines := strings.Split(string(src), "\n")
	var out []Rationale

	// nearestEnclosing finds the symbol whose span contains line, else the
	// closest preceding symbol (its end <= line). Returns "" if none.
	nearest := func(line int) string {
		bestUID, bestEnd := "", -1
		for _, s := range symbols {
			if s.StartLine <= line && line <= s.EndLine && s.EndLine > bestEnd {
				bestEnd = s.EndLine
				bestUID = s.UID
				continue
			}
			// Closest preceding: end <= line, pick the largest end.
			if s.EndLine <= line && s.EndLine > bestEnd {
				bestEnd = s.EndLine
				bestUID = s.UID
			}
		}
		return bestUID
	}

	for i, line := range lines {
		lineNo := i + 1
		if !commentLine.MatchString(line) {
			continue
		}
		var kind string
		switch {
		case markerNote.MatchString(line):
			kind = "note"
		case markerWhy.MatchString(line):
			kind = "why"
		case adrRef.MatchString(line):
			kind = "adr"
		default:
			continue
		}
		text := strings.TrimSpace(line)
		// Strip a leading comment marker for cleaner rationale text.
		text = strings.TrimLeft(text, "#/ \t")
		if text == "" {
			continue
		}
		out = append(out, Rationale{
			Text:      text,
			Kind:      kind,
			Line:      lineNo,
			SymbolUID: nearest(lineNo),
		})
	}
	return out
}
