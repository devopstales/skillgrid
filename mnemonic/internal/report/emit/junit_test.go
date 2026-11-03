package emit

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/report"
	"github.com/devopstales/skillgrid/mnemonic/internal/scan"
)

// TestJUnit_PerToolTestSuite asserts one testsuite per source that ran, the
// CVE id in the trivy testcase name, and a failure message that begins with
// the normalized severity.
//
// SATISFIES: happy path junit-xml-per-tool-testsuite
func TestJUnit_PerToolTestSuite(t *testing.T) {
	r := &report.Report{
		Static: []report.SourceResult{
			{
				Tool:   "trivy",
				Status: "ok",
				Findings: []scan.Finding{
					{Tool: "trivy", RuleID: "CVE-2024-0001", CVEID: "CVE-2024-0001", Severity: "HIGH",
						Title: "test vuln", Package: "example.com/lib", File: "go.mod", Line: 0},
				},
			},
			{
				Tool:   "debt",
				Status: "ok",
				Findings: []scan.Finding{
					{Tool: "debt", RuleID: "debt:TODO", Severity: "LOW", Title: "debt:TODO", File: "a.go", Line: 7},
				},
			},
		},
		GeneratedAt: time.Now().UTC(),
	}
	out, err := JUnit(r)
	if err != nil {
		t.Fatalf("JUnit: %v", err)
	}
	var doc junitTestSuites
	if err := xml.Unmarshal(out, &doc); err != nil {
		t.Fatalf("XML does not parse as JUnit: %v\n%s", err, out)
	}
	byName := map[string]junitTestSuite{}
	for _, ts := range doc.TestSuite {
		byName[ts.Name] = ts
	}
	tv, ok := byName["trivy"]
	if !ok {
		t.Fatalf("no testsuite named trivy: %s", out)
	}
	if len(tv.TestCases) != 1 {
		t.Fatalf("trivy testcases = %d, want 1", len(tv.TestCases))
	}
	if !strings.Contains(tv.TestCases[0].Name, "CVE-2024-0001") {
		t.Errorf("trivy testcase name %q does not contain the CVE id", tv.TestCases[0].Name)
	}
	if tv.TestCases[0].Failure == nil {
		t.Fatal("trivy testcase has no failure element")
	}
	if !strings.HasPrefix(tv.TestCases[0].Failure.Message, "HIGH") {
		t.Errorf("failure message %q does not begin with the normalized severity", tv.TestCases[0].Failure.Message)
	}
	if d, ok := byName["debt"]; !ok {
		t.Fatalf("no testsuite named debt: %s", out)
	} else if d.Tests != 1 {
		t.Errorf("debt tests = %d, want 1", d.Tests)
	}
}

// TestJUnit_ZeroFindingSource_EmptyTestSuite: a zero-finding source emits an
// empty testsuite, not an omitted one.
//
// SATISFIES: happy path junit-xml-zero-finding-source-is-empty-testsuite
func TestJUnit_ZeroFindingSource_EmptyTestSuite(t *testing.T) {
	out, err := JUnit(&report.Report{
		Static:      []report.SourceResult{{Tool: "jscpd", Status: "ok", Findings: nil}},
		GeneratedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("JUnit: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, `<testsuite name="jscpd" tests="0" failures="0"`) {
		t.Errorf("missing empty jscpd testsuite in %s", s)
	}
}

// TestJUnit_ErrorSourceIsEmptyTestSuite: a source that errored still gets its
// testsuite (0 findings) so CI sees the tool was present and failed.
func TestJUnit_ErrorSourceIsEmptyTestSuite(t *testing.T) {
	out, err := JUnit(&report.Report{
		Static:      []report.SourceResult{{Tool: "nuclei", Status: "error"}},
		GeneratedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("JUnit: %v", err)
	}
	if !strings.Contains(string(out), `<testsuite name="nuclei" tests="0" failures="0"`) {
		t.Errorf("missing empty nuclei testsuite for an error source: %s", out)
	}
}
