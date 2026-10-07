package scan

import "strings"

var normalizedSeverities = map[string]bool{
	"CRITICAL": true,
	"HIGH":     true,
	"MEDIUM":   true,
	"LOW":      true,
	"INFO":     true,
}

var toolSeverityMaps = map[string]map[string]string{
	// Trivy already speaks the normalized scale; only UNKNOWN is off-scale.
	"trivy": {"UNKNOWN": "INFO"},
	"wapiti": {
		"critical": "CRITICAL",
		"high":     "HIGH",
		"medium":   "MEDIUM",
		"low":      "LOW",
		"info":     "INFO",
	},
	"nuclei": {
		"critical": "CRITICAL",
		"high":     "HIGH",
		"medium":   "MEDIUM",
		"low":      "LOW",
		"info":     "INFO",
		"unknown":  "INFO",
	},
	"semgrep": {
		"ERROR":   "HIGH",
		"WARNING": "MEDIUM",
		"NOTE":    "INFO",
	},
}

// NormalizeSeverity maps a scanner-specific severity to the normalized
// CRITICAL|HIGH|MEDIUM|LOW|INFO scale stored in findings.severity. Unknown
// raw values collapse to INFO.
func NormalizeSeverity(tool, raw string) string {
	upper := strings.ToUpper(raw)
	if normalizedSeverities[upper] {
		return upper
	}
	if m, ok := toolSeverityMaps[tool]; ok {
		if s, ok := m[raw]; ok {
			return s
		}
	}
	return "INFO"
}
