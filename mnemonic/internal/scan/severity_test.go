package scan

import "testing"

// TestNormalizeSeverityTable covers the 050 normalization map: trivy identity
// (UNKNOWN->INFO), wapiti/nuclei capitalize, semgrep ERROR/WARNING/NOTE
// remap.
//
// SATISFIES: `happy path trivy scan ingests findings with stable hash` (severity leg)
func TestNormalizeSeverityTable(t *testing.T) {
	cases := []struct {
		tool, raw, want string
	}{
		{"trivy", "CRITICAL", "CRITICAL"},
		{"trivy", "UNKNOWN", "INFO"},
		{"trivy", "LOW", "LOW"},
		{"wapiti", "high", "HIGH"},
		{"nuclei", "critical", "CRITICAL"},
		{"semgrep", "ERROR", "HIGH"},
		{"semgrep", "WARNING", "MEDIUM"},
		{"semgrep", "NOTE", "INFO"},
	}
	for _, tc := range cases {
		if got := NormalizeSeverity(tc.tool, tc.raw); got != tc.want {
			t.Errorf("NormalizeSeverity(%q, %q) = %q, want %q", tc.tool, tc.raw, got, tc.want)
		}
	}
}
