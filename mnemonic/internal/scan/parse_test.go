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

// TestParseJscpd maps the fixture's two clones into findings
// (rule_id=duplication, file=first offset, line=first offset start line,
// severity hardcoded MEDIUM — no severity map entry).
//
// SATISFIES: `happy path duplication-jscpd-report`
func TestParseJscpd(t *testing.T) {
	fs, err := Parse("jscpd", fixtureBytes(t, "jscpd.json"))
	if err != nil {
		t.Fatalf("Parse(jscpd): %v", err)
	}
	if len(fs) != 2 {
		t.Fatalf("Parse(jscpd) = %d findings, want 2", len(fs))
	}
	f := fs[0]
	if f.RuleID != "duplication" {
		t.Errorf("RuleID = %q, want duplication", f.RuleID)
	}
	if f.File != "src/a.ts" {
		t.Errorf("File = %q, want src/a.ts", f.File)
	}
	if f.Severity != "MEDIUM" {
		t.Errorf("Severity = %q, want MEDIUM", f.Severity)
	}
	if f.Line != 10 {
		t.Errorf("Line = %d, want 10", f.Line)
	}
	if f.Tool != "jscpd" {
		t.Errorf("Tool = %q, want jscpd", f.Tool)
	}
	if got := fs[1].File; got != "src/c.ts" {
		t.Errorf("second File = %q, want src/c.ts", got)
	}
}

// TestParseDispatchIncludesJscpd: every known tool dispatches without an
// "unsupported" error, even for trivial input.
func TestParseDispatchIncludesJscpd(t *testing.T) {
	for _, tool := range []string{"trivy", "wapiti", "nuclei", "semgrep", "jscpd"} {
		_, err := Parse(tool, []byte("{}"))
		if err == nil {
			continue // valid {} may parse to zero findings
		}
		if got := err.Error(); got == "parse: unsupported tool \""+tool+"\"" {
			t.Errorf("Parse(%s): unsupported error for a known tool: %v", tool, err)
		}
	}
}

// TestScannerCommandsIncludesJscpd: jscpd must be runnable via Start.
func TestScannerCommandsIncludesJscpd(t *testing.T) {
	if _, ok := scannerCommands["jscpd"]; !ok {
		t.Error("scannerCommands missing jscpd; must be runnable via Start")
	}
}

// TestParseAkca maps the fixture's three evidence-driven findings into
// findings (rule_id=ruleId, file=url, raw severity kept for
// NormalizeSeverity, title carries the proof payload/status).
//
// SATISFIES: `dast-auto-from-config` (akca leg)
func TestParseAkca(t *testing.T) {
	fs, err := Parse("akca", fixtureBytes(t, "akca.json"))
	if err != nil {
		t.Fatalf("Parse(akca): %v", err)
	}
	if len(fs) != 3 {
		t.Fatalf("Parse(akca) = %d findings, want 3", len(fs))
	}
	f := fs[0]
	if f.Tool != "akca" {
		t.Errorf("Tool = %q, want akca", f.Tool)
	}
	if f.RuleID != "sqli" {
		t.Errorf("RuleID = %q, want sqli", f.RuleID)
	}
	if f.File != "/search" {
		t.Errorf("File = %q, want /search", f.File)
	}
	if got := NormalizeSeverity("akca", f.Severity); got != "HIGH" {
		t.Errorf("NormalizeSeverity(akca, %q) = %q, want HIGH", f.Severity, got)
	}
	if len(f.Links) != 1 || f.Links[0] != "/search" {
		t.Errorf("Links = %v, want [/search]", f.Links)
	}
	if got := fs[1].RuleID; got != "xss" {
		t.Errorf("second RuleID = %q, want xss", got)
	}
	if got := NormalizeSeverity("akca", fs[1].Severity); got != "MEDIUM" {
		t.Errorf("NormalizeSeverity(akca, %q) = %q, want MEDIUM", fs[1].Severity, got)
	}
	if got := NormalizeSeverity("akca", fs[2].Severity); got != "INFO" {
		t.Errorf("NormalizeSeverity(akca, %q) = %q, want INFO", fs[2].Severity, got)
	}
}

// TestScannerCommandsIncludesAkca: akca must be runnable via Start.
func TestScannerCommandsIncludesAkca(t *testing.T) {
	if _, ok := scannerCommands["akca"]; !ok {
		t.Error("scannerCommands missing akca; must be runnable via Start")
	}
}

// TestParseAkcaBadInput is a second akca input (triangulation): malformed
// JSON is a clear error, not a zero-value finding set.
func TestParseAkcaBadInput(t *testing.T) {
	if _, err := Parse("akca", []byte("not-json")); err == nil {
		t.Fatal("Parse(akca, bad input) = nil error, want an error")
	}
}

// TestParseNucleiBadLine is a second nuclei input (triangulation): a
// non-JSON line is a clear error, not a silent skip.
func TestParseNucleiBadLine(t *testing.T) {
	if _, err := Parse("nuclei", []byte("not-json\n")); err == nil {
		t.Fatal("Parse(nuclei, bad line) = nil error, want an error")
	}
}
