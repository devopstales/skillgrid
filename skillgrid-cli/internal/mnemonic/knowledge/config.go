package knowledge

import (
	"path/filepath"
	"regexp"
	"strings"
)

// ConfigRef is one reference from a config file to the code it configures:
// the literal value that names the code, the line, and the confidence.
//   - EXTRACTED: the value names the code by explicit syntax (a quoted
//     identifier that is unambiguously a symbol reference, e.g. a Go struct
//     tag or a handler field naming a function).
//   - INFERRED:  the value names the code by convention (a bare key that
//     resolves to a symbol by a naming heuristic).
//   - AMBIGUOUS: the value resolves to no known symbol — kept (not dropped)
//     so an unresolvable config reference is surfaced, per 03.5.
type ConfigRef struct {
	Value      string
	Line       int
	Confidence string
}

// ConfigResult is the extraction result for one config file: the config node
// (title + path) and its outgoing configures references.
type ConfigResult struct {
	Path     string
	Title    string
	Refs     []ConfigRef
	IsConfig bool
}

// isConfig reports whether path is a config file (yaml/toml/json).
func isConfig(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml", ".toml", ".json":
		return true
	}
	return false
}

// configValue matches a config key/value pair whose value could name code:
// yaml `key: value`, toml `key = value`, json `"key": value`. The value is
// either a quoted string or a bare identifier. A comma follows JSON array /
// object separators and is tolerated.
var configValue = regexp.MustCompile(`(?m)^\s*["']?([A-Za-z_][A-Za-z0-9_.\-]*)["']?\s*[:=]\s*("?[A-Za-z_][A-Za-z0-9_.\-]*"?)\s*,?\s*$`)

// unquote strips a single pair of surrounding quotes from a config value.
func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// ExtractConfig parses one yaml/toml/json file into a config node + its
// configures references. It never errors and never returns nil: a malformed
// file yields zero refs (the index continues, the rest is indexed). A
// non-config path returns an empty (IsConfig=false) result.
//
// Confidence policy (03.3 / 03.5):
//   - a value matching an indexed symbol by explicit syntax is EXTRACTED
//     (resolved at persistence time by SaveConfig);
//   - a value that is a conventional reference (bare identifier, no path
//     separator) is INFERRED when it resolves;
//   - a value that resolves to nothing is AMBIGUOUS (kept, not dropped).
func ExtractConfig(path string, src []byte) *ConfigResult {
	if !isConfig(path) {
		return &ConfigResult{Path: path, Title: filepath.Base(path)}
	}
	res := &ConfigResult{Path: path, Title: filepath.Base(path), IsConfig: true}
	lines := strings.Split(string(src), "\n")
	for lineNo, line := range lines {
		if isCommentLine(line) {
			continue
		}
		m := configValue.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		value := unquote(m[2])
		res.Refs = append(res.Refs, ConfigRef{
			Value:      value,
			Line:       lineNo + 1,
			Confidence: classifyConfigRef(value),
		})
	}
	return res
}

// isCommentLine reports whether a config line is a comment (yaml #, toml #,
// or blank). JSON has no comments; a blank line is skipped.
func isCommentLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return trimmed == "" || strings.HasPrefix(trimmed, "#")
}

// isExplicitRefValue reports whether a config value is an explicit symbol
// reference (a function/struct name): a CamelCase identifier (at least two
// case-runs, e.g. `serverConfig`). A lowercase or single-case value (a
// filename, a domain name, a plain word) is a conventional reference, not an
// explicit one.
func isExplicitRefValue(v string) bool {
	if v == "" {
		return false
	}
	first := v[0]
	if first >= 'A' && first <= 'Z' {
		// Starts uppercase: explicit only when it has an internal camel hump
		// (a lowercase followed by an uppercase) — a plain PascalCase word is
		// still treated as explicit (it names a type/constructor).
		for i := 1; i < len(v); i++ {
			if v[i] >= 'A' && v[i] <= 'Z' {
				return true
			}
		}
		return false
	}
	// Starts lowercase: explicit when it contains a camel hump (lowercase
	// followed by uppercase), e.g. serverConfig.
	for i := 1; i < len(v); i++ {
		if v[i] >= 'A' && v[i] <= 'Z' {
			return true
		}
	}
	return false
}

// classifyConfigRef assigns the base confidence for a config reference value
// BEFORE symbol resolution (SaveConfig finalizes EXTRACTED/INFERRED/AMBIGUOUS
// against the indexed symbols). An explicit symbol reference (a CamelCase
// identifier or a path/qualified name) is EXTRACTED; a plain value is a
// conventional reference (INFERRED).
func classifyConfigRef(value string) string {
	if isExplicitRefValue(value) || strings.ContainsAny(value, "/.") {
		return ConfidenceExtracted
	}
	return ConfidenceInferred
}
