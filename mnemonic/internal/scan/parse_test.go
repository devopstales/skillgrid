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
