package wiki

import (
	"strings"
	"time"
)

// Emit serializes a Concept to an OKF v0.2 markdown document: a YAML
// frontmatter block (fixed key order, no null keys, absolute-UTC RFC3339
// timestamps) followed by a blank line and the body verbatim.
//
// Emit is a pure serializer: it never validates, never mutates its input, and
// is deterministic — the same Concept always produces the same byte sequence.
// Empty/zero fields are omitted (no null keys). The error return is part of
// the signature for forward compatibility; the current implementation always
// returns a nil error.
func Emit(c Concept) (string, error) {
	var b strings.Builder
	b.WriteString("---\n")

	writeKey(&b, "type", c.Type)
	writeKey(&b, "title", c.Title)
	if c.Description != "" {
		writeKey(&b, "description", c.Description)
	}
	if c.Resource != "" {
		writeKey(&b, "resource", c.Resource)
	}
	if len(c.Tags) > 0 {
		writeFlowList(&b, "tags", c.Tags)
	}
	if len(c.Sources) > 0 {
		writeSources(&b, c.Sources)
	}
	if c.Generated.By != "" || !c.Generated.At.IsZero() {
		writeVerifierMap(&b, "generated", "  ", c.Generated)
	}
	switch len(c.Verified) {
	case 0:
	case 1:
		writeVerifierMap(&b, "verified", "  ", c.Verified[0])
	default:
		writeVerifierList(&b, "verified", c.Verified)
	}
	if c.Status != "" {
		writeKey(&b, "status", c.Status)
	}
	if !c.StaleAfter.IsZero() {
		writeTime(&b, "stale_after", c.StaleAfter)
	}
	if c.SourcePath != "" {
		writeKey(&b, "source_path", c.SourcePath)
	}

	b.WriteString("---\n")
	b.WriteString("\n")
	b.WriteString(c.Body)
	return b.String(), nil
}

// writeKey writes a scalar "key: value" line, omitting empty values.
func writeKey(b *strings.Builder, key, value string) {
	if value == "" {
		return
	}
	b.WriteString(key)
	b.WriteString(": ")
	b.WriteString(quoteYAML(value))
	b.WriteString("\n")
}

// writeFlowList writes "key: [a, b, c]" for a list of simple strings.
func writeFlowList(b *strings.Builder, key string, items []string) {
	parts := make([]string, len(items))
	for i, s := range items {
		parts[i] = quoteYAML(s)
	}
	b.WriteString(key)
	b.WriteString(": [")
	b.WriteString(strings.Join(parts, ", "))
	b.WriteString("]\n")
}

// writeSources writes the "sources:" block as a YAML list of maps. Each entry
// carries resource (always), author, title, and last_modified (omitted when
// zero).
func writeSources(b *strings.Builder, sources []SourceRef) {
	b.WriteString("sources:\n")
	for _, s := range sources {
		b.WriteString("  - resource: ")
		b.WriteString(quoteYAML(s.Resource))
		b.WriteString("\n")
		if s.Author != "" {
			b.WriteString("    author: ")
			b.WriteString(quoteYAML(s.Author))
			b.WriteString("\n")
		}
		if s.Title != "" {
			b.WriteString("    title: ")
			b.WriteString(quoteYAML(s.Title))
			b.WriteString("\n")
		}
		if !s.LastModified.IsZero() {
			b.WriteString("    last_modified: ")
			b.WriteString(formatTime(s.LastModified))
			b.WriteString("\n")
		}
	}
}

// writeVerifierMap writes a single verifier as a nested two-line block:
//
//	key:
//	  by: ...
//	  at: ...
func writeVerifierMap(b *strings.Builder, key, indent string, v Verifier) {
	b.WriteString(key)
	b.WriteString(":\n")
	b.WriteString(indent)
	b.WriteString("by: ")
	b.WriteString(quoteYAML(v.By))
	b.WriteString("\n")
	b.WriteString(indent)
	b.WriteString("at: ")
	b.WriteString(formatTime(v.At))
	b.WriteString("\n")
}

// writeVerifierList writes multiple verifiers as a YAML list of {by, at} maps.
func writeVerifierList(b *strings.Builder, key string, verifiers []Verifier) {
	b.WriteString(key)
	b.WriteString(":\n")
	for _, v := range verifiers {
		b.WriteString("  - by: ")
		b.WriteString(quoteYAML(v.By))
		b.WriteString("\n")
		b.WriteString("    at: ")
		b.WriteString(formatTime(v.At))
		b.WriteString("\n")
	}
}

// formatTime renders a time as absolute-UTC RFC3339 ("2006-01-02T15:04:05Z").
// Zero times are formatted as the zero instant ("0001-01-01T00:00:00Z") but
// callers omit zero-valued fields before reaching here.
//
// The value is written as a YAML plain scalar (no quotes): the RFC3339
// character set ([0-9TZ:.] plus the '-' between date and time) contains no
// YAML indicator, so the unquoted form is valid YAML. Keeping it unquoted
// makes the emitted `at:` / `last_modified:` / `stale_after:` lines match the
// plain-scalar form conform.go's unquoteYAML expects and keeps emitted pages
// stable under substring assertions.
func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// writeTime writes a "key: <rfc3339>" line as a plain (unquoted) scalar.
func writeTime(b *strings.Builder, key string, t time.Time) {
	b.WriteString(key)
	b.WriteString(": ")
	b.WriteString(t.UTC().Format(time.RFC3339))
	b.WriteString("\n")
}

// yamlSpecial reports whether a string contains any character that requires
// double-quoting in this emitter's YAML subset: the indicator characters from
// the requirements, backslashes (escaped in a double-quoted scalar), newlines,
// tabs (unsafe in a plain scalar), or leading/trailing whitespace.
func yamlSpecial(s string) bool {
	if strings.ContainsAny(s, ":#><&*[]{}\",'%`\n\t\\") {
		return true
	}
	// Leading/trailing whitespace or a leading indicator char needs quoting.
	if s == "" {
		return false
	}
	if r := s[0]; r == '-' || r == '?' || r == '!' || r == '|' || r == '>' || r == '@' || r == '&' || r == '%' {
		return true
	}
	return s[0] == ' ' || s[len(s)-1] == ' '
}

// quoteYAML double-quotes a string when it contains YAML-special characters
// (escaping embedded backslashes and double quotes); simple alphanumeric (and
// other safe) strings are returned unquoted.
func quoteYAML(s string) string {
	if !yamlSpecial(s) {
		return s
	}
	escaped := strings.ReplaceAll(s, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}
