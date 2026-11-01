package codeindex

import (
	"strings"
	"unicode"
)

// nameSegments splits a symbol name into lowercase segments for the
// symbol_segments reverse index: on underscores, at lower->Upper case
// transitions, at runs of consecutive uppercase (a trailing capitalized word
// detaches: "HTTPServer" -> "http","server"), and at letter<->digit
// boundaries ("parse2" -> "parse","2"). Empty segments are skipped; segments
// are capped at 32 chars and deduped per name.
func nameSegments(name string) []string {
	seen := map[string]bool{}
	var out []string
	for _, seg := range rawNameSegments(name) {
		if seg == "" || len(seg) > 32 {
			continue
		}
		if !seen[seg] {
			seen[seg] = true
			out = append(out, seg)
		}
	}
	return out
}

func rawNameSegments(name string) []string {
	var out []string
	for _, part := range strings.Split(name, "_") {
		out = append(out, camelSegments(part)...)
	}
	return out
}

// camelSegments splits one underscore-free part at case and digit boundaries.
func camelSegments(s string) []string {
	var out []string
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			out = append(out, strings.ToLower(string(cur)))
			cur = cur[:0]
		}
	}
	for i, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()
			continue
		}
		if i > 0 {
			prev := rune(s[i-1])
			if unicode.IsLetter(prev) && unicode.IsDigit(r) ||
				unicode.IsDigit(prev) && unicode.IsLetter(r) ||
				unicode.IsLower(prev) && unicode.IsUpper(r) ||
				(unicode.IsUpper(prev) && unicode.IsUpper(r) && i+1 < len(s) && unicode.IsLower(rune(s[i+1]))) {
				flush()
			}
		}
		cur = append(cur, r)
	}
	flush()
	return out
}
