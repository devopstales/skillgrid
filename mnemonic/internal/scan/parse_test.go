package scan

import "testing"

// TestParseTrivy maps the fixture's single CVE into findings with the
// trivy-specific field layout (rule_id=CVE, file=target, line=0).
//
// SATISFIES: `happy path trivy scan ingests findings with stable hash` (parse leg)
func TestParseTrivy(t *testing.T) {
	raw := fixtureBytes(t, "trivy.json")
	fs, err := Parse("trivy", raw)
	if err != nil {
		t.Fatalf("Parse(trivy): %v", err)
	}
	if len(fs) != 1 {
		t.Fatalf("Parse(trivy) = %d findings, want 1", len(fs))
	}
	f := fs[0]
	if f.RuleID != "CVE-2024-1234" {
		t.Errorf("RuleID = %q, want CVE-2024-1234", f.RuleID)
	}
	if f.Severity != "HIGH" {
		t.Errorf("Severity = %q, want HIGH", f.Severity)
	}
	if f.Package != "flask" || f.Version != "2.0.1" {
		t.Errorf("Package/Version = %q/%q, want flask/2.0.1", f.Package, f.Version)
	}
	if f.File != "requirements.txt" {
		t.Errorf("File = %q, want requirements.txt", f.File)
	}
	if f.Line != 0 {
		t.Errorf("Line = %d, want 0", f.Line)
	}
	if f.Message != "flask before 2.0.1 has an os command injection" {
		t.Errorf("Message = %q, want the description", f.Message)
	}
}

// TestParseUnknownTool is a second Parse input (triangulation): an unimplemented
// tool is a clear error, not a zero-value finding set.
func TestParseUnknownTool(t *testing.T) {
	if _, err := Parse("cargo-audit", []byte("{}")); err == nil {
		t.Fatal("Parse(unknown tool) = nil error, want an error")
	}
}

// TestParseDispatchAllTools is the acceptance gate: every scanner fixture
// parses to its expected finding count through the Parse dispatch.
//
// SATISFIES: `happy path trivy scan ingests findings with stable hash` (all four tools)
func TestParseDispatchAllTools(t *testing.T) {
	cases := []struct {
		tool string
		file string
		want int
	}{
		{"trivy", "trivy.json", 1},
		{"wapiti", "wapiti.json", 2},
		{"nuclei", "nuclei.jsonl", 3},
		{"semgrep", "semgrep.json", 4},
	}
	for _, tc := range cases {
		fs, err := Parse(tc.tool, fixtureBytes(t, tc.file))
		if err != nil {
			t.Errorf("Parse(%s): %v", tc.tool, err)
			continue
		}
		if len(fs) != tc.want {
			t.Errorf("Parse(%s) = %d findings, want %d", tc.tool, len(fs), tc.want)
		}
	}
}

// TestParseWapiti maps report[] entries (type/info/url) into findings with
// file=url, line=0, message=info.
func TestParseWapiti(t *testing.T) {
	fs, err := Parse("wapiti", fixtureBytes(t, "wapiti.json"))
	if err != nil {
		t.Fatalf("Parse(wapiti): %v", err)
	}
	if len(fs) != 2 {
		t.Fatalf("Parse(wapiti) = %d findings, want 2", len(fs))
	}
	f := fs[0]
	if f.RuleID != "SQL Injection" {
		t.Errorf("RuleID = %q, want SQL Injection", f.RuleID)
	}
	if f.File != "https://example.com/login" {
		t.Errorf("File = %q, want the url", f.File)
	}
	if f.Line != 0 {
		t.Errorf("Line = %d, want 0", f.Line)
	}
	if f.Message != "SQL Injection vulnerability detected on the login form" {
		t.Errorf("Message = %q, want the info", f.Message)
	}
	if got := fs[1].RuleID; got != "XSS" {
		t.Errorf("second RuleID = %q, want XSS", got)
	}
}

// TestParseNuclei maps JSONL lines (template-id/info.severity/matcher-name/host)
// into findings with file=host, line=0, message=matcher-name.
func TestParseNuclei(t *testing.T) {
	fs, err := Parse("nuclei", fixtureBytes(t, "nuclei.jsonl"))
	if err != nil {
		t.Fatalf("Parse(nuclei): %v", err)
	}
	if len(fs) != 3 {
		t.Fatalf("Parse(nuclei) = %d findings, want 3", len(fs))
	}
	f := fs[0]
	if f.RuleID != "exposure/exposed-env" {
		t.Errorf("RuleID = %q, want exposure/exposed-env", f.RuleID)
	}
	if f.File != "https://example.com" {
		t.Errorf("File = %q, want the host", f.File)
	}
	if f.Line != 0 {
		t.Errorf("Line = %d, want 0", f.Line)
	}
	if f.Message != "exposed-env" {
		t.Errorf("Message = %q, want the matcher-name", f.Message)
	}
	if got := fs[1].RuleID; got != "cves/2023/cve-2023-12345.yaml" {
		t.Errorf("second RuleID = %q, want the cve template id", got)
	}
	if got := fs[2].Message; got != "nginx" {
		t.Errorf("third Message = %q, want nginx", got)
	}
}

// TestParseSemgrep maps results[] (check_id/extra.severity/path/start.line/
// extra.message) into findings with the raw severity kept for NormalizeSeverity.
func TestParseSemgrep(t *testing.T) {
	fs, err := Parse("semgrep", fixtureBytes(t, "semgrep.json"))
	if err != nil {
		t.Fatalf("Parse(semgrep): %v", err)
	}
	if len(fs) != 4 {
		t.Fatalf("Parse(semgrep) = %d findings, want 4", len(fs))
	}
	f := fs[0]
	if f.RuleID != "python.lang.security.eval-used" {
		t.Errorf("RuleID = %q, want python.lang.security.eval-used", f.RuleID)
	}
	if f.Severity != "ERROR" {
		t.Errorf("Severity = %q, want the raw ERROR", f.Severity)
	}
	if f.File != "app/auth.py" {
		t.Errorf("File = %q, want app/auth.py", f.File)
	}
	if f.Line != 12 {
		t.Errorf("Line = %d, want 12", f.Line)
	}
	if f.Message != "Use of eval() detected" {
		t.Errorf("Message = %q, want the extra message", f.Message)
	}
	if got := fs[3].RuleID; got != "python.correctness.compare-to-constant" {
		t.Errorf("fourth RuleID = %q, want python.correctness.compare-to-constant", got)
	}
}

// TestParseNucleiBadLine is a second nuclei input (triangulation): a
// non-JSON line is a clear error, not a silent skip.
func TestParseNucleiBadLine(t *testing.T) {
	if _, err := Parse("nuclei", []byte("not-json\n")); err == nil {
		t.Fatal("Parse(nuclei, bad line) = nil error, want an error")
	}
}
