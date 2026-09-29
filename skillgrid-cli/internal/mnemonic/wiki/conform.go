package wiki

import (
	"fmt"
	"strings"
	"time"
)

// Conform validates the OKF v0.2 reserved frontmatter surface of a markdown
// page and returns one error per violation (an empty slice means the page is
// conformant).
//
// The checks are:
//
//   - the page has a well-formed `---` frontmatter block
//   - `type` is present and non-empty (R2.1)
//   - every `sources[]` entry has a non-empty `resource` (R2.2)
//   - every timestamp value (generated.at, verified[].at,
//     sources[].last_modified, stale_after) parses as absolute-UTC RFC3339
//     (…T…Z)
//   - `status`, when present, is one of draft|stable|deprecated (R2.3)
//
// Unknown keys are ignored: the gate checks the reserved OKF surface only.
// The frontmatter is parsed with a small line-based parser — no YAML library.
func Conform(page string) []error {
	lines, ok := splitFrontmatter(page)
	if !ok {
		return []error{fmt.Errorf("okf: no frontmatter: page must start with a --- delimited YAML frontmatter block")}
	}

	var errs []error
	var (
		typeVal     string
		typePresent bool
		statusVal   string
		statusSet   bool
	)

	i := 0
	for i < len(lines) {
		line := lines[i]
		// Top-level keys are unindented "key:" lines; anything indented
		// belongs to the enclosing block and is handled by the block
		// parsers below (or ignored when it is not a reserved key).
		if isIndented(line) {
			i++
			continue
		}
		key, value, ok := splitKeyValue(line)
		if !ok {
			i++
			continue
		}

		switch key {
		case "type":
			typeVal = value
			typePresent = true
			i++
		case "status":
			statusVal = value
			statusSet = true
			i++
		case "stale_after":
			errs = append(errs, checkTimestamp("stale_after", value)...)
			i++
		case "sources":
			var block []string
			i, block = collectBlock(lines, i+1)
			for n, entry := range parseEntries(block) {
				if r := entry["resource"]; r == "" {
					errs = append(errs, fmt.Errorf("okf: sources[%d].resource: must be a non-empty resource", n))
				}
				if lm, ok := entry["last_modified"]; ok {
					errs = append(errs, checkTimestamp(fmt.Sprintf("sources[%d].last_modified", n), lm)...)
				}
			}
		case "generated":
			var block []string
			i, block = collectBlock(lines, i+1)
			entry, ok := parseFlatMap(block)
			if !ok {
				break
			}
			if at, ok := entry["at"]; ok {
				errs = append(errs, checkTimestamp("generated.at", at)...)
			}
		case "verified":
			var block []string
			i, block = collectBlock(lines, i+1)
			if isListBlock(block) {
				for n, entry := range parseEntries(block) {
					if at, ok := entry["at"]; ok {
						errs = append(errs, checkTimestamp(fmt.Sprintf("verified[%d].at", n), at)...)
					}
				}
			} else if entry, ok := parseFlatMap(block); ok {
				if at, ok := entry["at"]; ok {
					errs = append(errs, checkTimestamp("verified.at", at)...)
				}
			}
		default:
			// Unknown top-level key: consume its block (if any) and ignore.
			i, _ = collectBlock(lines, i+1)
		}
	}

	if !typePresent || strings.TrimSpace(typeVal) == "" {
		errs = append(errs, fmt.Errorf("okf: type: must be present and non-empty"))
	}
	if statusSet && !validStatus(statusVal) {
		errs = append(errs, fmt.Errorf("okf: status: %q is not one of draft|stable|deprecated", statusVal))
	}
	return errs
}

// validStatus reports whether s is a reserved OKF v0.2 status value.
func validStatus(s string) bool {
	switch s {
	case "draft", "stable", "deprecated":
		return true
	}
	return false
}

// checkTimestamp validates that v parses as absolute-UTC RFC3339 (ending in Z)
// and returns a descriptive error when it does not. An empty value is a
// missing timestamp and is not validated.
func checkTimestamp(path, v string) []error {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	if _, err := time.Parse(time.RFC3339, v); err != nil {
		return []error{fmt.Errorf("okf: %s: %q is not absolute UTC ISO 8601 (RFC3339, ending in Z)", path, v)}
	}
	if !strings.HasSuffix(v, "Z") {
		return []error{fmt.Errorf("okf: %s: %q is not absolute UTC (must end in Z)", path, v)}
	}
	return nil
}

// splitFrontmatter extracts the lines between the opening `---` (which must be
// the first line) and the next line that is exactly `---`. ok is false when
// the page has no well-formed frontmatter block.
func splitFrontmatter(page string) (lines []string, ok bool) {
	all := strings.Split(page, "\n")
	if len(all) == 0 || all[0] != "---" {
		return nil, false
	}
	for i := 1; i < len(all); i++ {
		if all[i] == "---" {
			return all[1:i], true
		}
	}
	return nil, false
}

// isIndented reports whether the line is a continuation of a block value
// (leading whitespace) rather than a top-level key.
func isIndented(line string) bool {
	if line == "" {
		return false
	}
	return line[0] == ' ' || line[0] == '\t'
}

// splitKeyValue splits an unindented "key: value" line into key and the raw
// (unquoted) value. ok is false for lines that are not key lines (e.g. bare
// `- item` entries or blank lines).
func splitKeyValue(line string) (key, value string, ok bool) {
	colon := strings.IndexByte(line, ':')
	if colon <= 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:colon])
	if key == "" {
		return "", "", false
	}
	value = unquoteYAML(strings.TrimSpace(line[colon+1:]))
	return key, value, true
}

// collectBlock consumes the indented lines belonging to the key at lines[i-1],
// starting at start. It returns the next index to process and the collected
// block lines (without their indentation). A top-level key with no value and
// no indented continuation (e.g. a bare `tags:`) yields an empty block.
func collectBlock(lines []string, start int) (next int, block []string) {
	for start < len(lines) {
		line := lines[start]
		if !isIndented(line) {
			break
		}
		block = append(block, line)
		start++
	}
	return start, block
}

// isListBlock reports whether a block is a YAML list of maps (lines starting
// with a list dash) rather than a flat map.
func isListBlock(block []string) bool {
	for _, line := range block {
		if line == "" {
			continue
		}
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "- ") || trimmed == "-" {
			return true
		}
		return false
	}
	return false
}

// parseEntries parses a block that is a list of maps into per-entry key/value
// maps. The first `key: value` after each `- ` starts a new entry; subsequent
// indented lines until the next dash belong to the current entry.
func parseEntries(block []string) []map[string]string {
	var entries []map[string]string
	var current map[string]string
	for _, line := range block {
		if line == "" {
			continue
		}
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "- ") {
			current = map[string]string{}
			entries = append(entries, current)
			rest := strings.TrimPrefix(trimmed, "- ")
			if k, v, ok := splitKeyValue(rest); ok {
				current[k] = v
			}
			continue
		}
		if current == nil {
			continue
		}
		if k, v, ok := splitKeyValue(trimmed); ok {
			current[k] = v
		}
	}
	return entries
}

// parseFlatMap parses a block of "key: value" lines into a single map. ok is
// false when the block is empty (no value for the key).
func parseFlatMap(block []string) (map[string]string, bool) {
	if len(block) == 0 {
		return nil, false
	}
	m := map[string]string{}
	for _, line := range block {
		if line == "" {
			continue
		}
		trimmed := strings.TrimLeft(line, " \t")
		if k, v, ok := splitKeyValue(trimmed); ok {
			m[k] = v
		}
	}
	if len(m) == 0 {
		return nil, false
	}
	return m, true
}

// unquoteYAML strips a single layer of double quotes from a scalar value and
// unescapes the backslash escapes the emitter produces (\\" and \\\\). Values
// that are not double-quoted are returned trimmed as-is.
func unquoteYAML(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		inner := s[1 : len(s)-1]
		inner = strings.ReplaceAll(inner, `\"`, `"`)
		inner = strings.ReplaceAll(inner, `\\`, `\`)
		return inner
	}
	return s
}
