package hybrid

import (
	"regexp"
	"strings"
)

// elisionToken marks a collapsed (unrelated) body in a skeletonized snippet.
const elisionToken = "  … elided"

// isSignatureLine reports whether a line is a declaration/signature (a
// func/method/type/class/def/interface/const/let/var statement or a
// closing/opening brace at statement level).
func isSignatureLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	for _, prefix := range []string{"func ", "func(", "type ", "class ", "struct ", "interface ", "def ", "package ", "import ", "const ", "let ", "var ", "export ", "public ", "private "} {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

// isImportLine reports whether a line is an import/package statement.
func isImportLine(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, "import ") || strings.HasPrefix(t, "package ") || strings.HasPrefix(t, "#include")
}

// isMatchedLine reports whether the line contains the query (case-insensitive).
func isMatchedLine(line, query string) bool {
	if query == "" {
		return false
	}
	return strings.Contains(strings.ToLower(line), strings.ToLower(query))
}

// isBlank reports whether a line is whitespace-only.
func isBlank(line string) bool {
	return strings.TrimSpace(line) == ""
}

// Skeletonize collapses unrelated bodies in a code snippet while preserving
// imports, signatures, the matched line, and the exact read range (01.11).
// Lines outside [startLine, endLine] are dropped entirely. Consecutive
// non-preserved lines (the unrelated body) collapse to a single elision marker
// (with an original-line-count annotation), so the snippet stays short but the
// structural skeleton (imports → signature → matched line) survives.
func Skeletonize(lines []string, query string, startLine, endLine int) []string {
	if startLine < 0 {
		startLine = 0
	}
	if endLine < 0 || endLine >= len(lines) {
		endLine = len(lines) - 1
	}
	if startLine > endLine {
		return nil
	}

	var out []string
	var run []string // current run of elided lines
	flush := func() {
		if len(run) == 0 {
			return
		}
		out = append(out, elisionToken+" ("+itoa(len(run))+" lines)")
		run = nil
	}
	for i := startLine; i <= endLine && i < len(lines); i++ {
		line := lines[i]
		if isImportLine(line) || isSignatureLine(line) || isMatchedLine(line, query) || !isBlank(line) && isStructuralLine(line) {
			flush()
			out = append(out, line)
			continue
		}
		// Blank lines are kept (they separate the skeleton), not elided.
		if isBlank(line) {
			flush()
			out = append(out, line)
			continue
		}
		// Unrelated body line → accumulate for elision.
		run = append(run, line)
	}
	flush()
	return out
}

// isStructuralLine reports whether a line is structural (a brace boundary or a
// return/break/continue at the top of a block) — kept in the skeleton even if
// it is not a signature.
func isStructuralLine(line string) bool {
	t := strings.TrimSpace(line)
	if t == "{" || t == "}" || t == ");" || t == "} else" {
		return true
	}
	if strings.HasPrefix(t, "return") || strings.HasPrefix(t, "break") || strings.HasPrefix(t, "continue") {
		// Only count a return as structural if it is short (the signature's
		// return type, not a deep body statement).
		return len(t) <= 30
	}
	return false
}

// Snippet is a (path, text) pair for SimHash near-duplicate suppression.
type Snippet struct {
	Path string
	Text string
}

// simhashBits is the number of bits in the SimHash fingerprint (pure Go, no CGo).
const simhashBits = 64
// simhashDupThreshold is the Hamming distance at/below which two snippets are
// near-duplicates.
const simhashDupThreshold = 6

// simHash computes a 64-bit SimHash of the whitespace-tokenized text.
func simHash(text string) uint64 {
	tokens := strings.Fields(text)
	if len(tokens) == 0 {
		return 0
	}
	// Per-bit vote counts.
	var pos, neg [simhashBits]int
	for _, tok := range tokens {
		h := fnv64(tok)
		for b := 0; b < simhashBits; b++ {
			if h&(uint64(1)<<uint(b)) != 0 {
				pos[b]++
			} else {
				neg[b]++
			}
		}
	}
	var fingerprint uint64
	for b := 0; b < simhashBits; b++ {
		if pos[b] >= neg[b] {
			fingerprint |= uint64(1) << uint(b)
		}
	}
	return fingerprint
}

// fnv64 is a pure-Go FNV-1a 64-bit hash.
func fnv64(s string) uint64 {
	const (
		offset uint64 = 14695981039346656037
		prime  uint64 = 1099511628211
	)
	h := offset
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime
	}
	return h
}

// popcount is a pure-Go Hamming-distance helper.
func popcount(x uint64) int {
	c := 0
	for x != 0 {
		x &= x - 1
		c++
	}
	return c
}

// SimHashDedupe suppresses near-duplicate snippets: it walks the list in order
// and drops any snippet whose SimHash is within the threshold of an already-
// kept snippet (01.11). It cuts dup% WITHOUT moving a ranking metric because it
// operates on the already-ranked output, not the candidate set.
func SimHashDedupe(snippets []Snippet) []Snippet {
	if len(snippets) == 0 {
		return nil
	}
	kept := []Snippet{snippets[0]}
	keptHashes := []uint64{simHash(snippets[0].Text)}
	for i := 1; i < len(snippets); i++ {
		h := simHash(snippets[i].Text)
		dup := false
		for _, kh := range keptHashes {
			if popcount(h^kh) <= simhashDupThreshold {
				dup = true
				break
			}
		}
		if !dup {
			kept = append(kept, snippets[i])
			keptHashes = append(keptHashes, h)
		}
	}
	return kept
}

// Secret patterns replaced at output time (01.3). Index-time exclusion alone
// does not count — these are applied to the response text itself so a
// secret-like string in an indexed file is never emitted raw.
var secretPatterns = []*regexp.Regexp{
	// AWS access key ids.
	regexp.MustCompile(`\b(?:A3T[A-Z0-9]|AKIA|ASIA|AGPA|AIDA|AROA|ANPA|ANVA)[A-Z0-9]{16}\b`),
	// GitHub tokens (classic + fine-grained).
	regexp.MustCompile(`\bghp_[A-Za-z0-9]{20,}\b`),
	regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{20,}\b`),
	// Generic Bearer / secret / token / api-key assignments.
	regexp.MustCompile(`(?i)\b(password|passwd|secret|api[_-]?key|access[_-]?key|token|auth)\b\s*[:=]\s*["'][A-Za-z0-9_\-./+]{8,}["']`),
	regexp.MustCompile(`(?i)\bauthorization\s*:\s*bearer\s+[A-Za-z0-9_\-./+]{16,}`),
	// Private key blocks (replace the whole key material).
	regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`),
}

// secretReplacement is the placeholder a secret is replaced with.
const secretReplacement = "••••"

// RedactSecrets replaces every secret-like pattern in s with a placeholder so
// the value is never emitted raw in code_search/code_read response text (01.3).
// Non-secret text is returned unchanged.
func RedactSecrets(s string) string {
	if s == "" {
		return s
	}
	out := s
	for _, re := range secretPatterns {
		out = re.ReplaceAllString(out, secretReplacement)
	}
	return out
}

// RedactSnippet applies RedactSecrets to a snippet's text (search leg).
func RedactSnippet(s Snippet) Snippet {
	s.Text = RedactSecrets(s.Text)
	return s
}
