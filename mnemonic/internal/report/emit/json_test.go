package emit

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/report"
	"github.com/devopstales/skillgrid/mnemonic/internal/scan"
)

// TestJSON_ReportShape asserts the dashboard-facing JSON shape: a static
// array with tool/status/findings sections, findings carrying severity/title/
// cve_id/rule_id/file/line, a coverage array of file/pct_line rows, and a
// generated_at field.
//
// SATISFIES: happy path json-report-shape
func TestJSON_ReportShape(t *testing.T) {
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
		},
		Coverage:    []report.CoverageRow{{File: "a.go", Pkg: "example.com/lib", PctLine: 82.5}},
		GeneratedAt: time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC),
	}
	out, err := JSON(r)
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	var doc struct {
		Static []struct {
			Tool     string `json:"tool"`
			Status   string `json:"status"`
			Findings []struct {
				Severity     string `json:"severity"`
				Title        string `json:"title"`
				CVEID        string `json:"cve_id"`
				RuleID       string `json:"rule_id"`
				Package      string `json:"package"`
				FixedVersion string `json:"fixed_version"`
				File         string `json:"file"`
				Line         int    `json:"line"`
			} `json:"findings"`
		} `json:"static"`
		Dynamic  []any `json:"dynamic"`
		Coverage []struct {
			File    string  `json:"file"`
			Pkg     string  `json:"pkg"`
			PctLine float64 `json:"pct_line"`
		} `json:"coverage"`
		GeneratedAt string `json:"generated_at"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("JSON does not parse: %v\n%s", err, out)
	}
	if len(doc.Static) != 1 || doc.Static[0].Tool != "trivy" || doc.Static[0].Status != "ok" {
		t.Fatalf("static section wrong: %+v", doc.Static)
	}
	if len(doc.Static[0].Findings) != 1 {
		t.Fatalf("findings = %d, want 1", len(doc.Static[0].Findings))
	}
	f := doc.Static[0].Findings[0]
	if f.Severity != "HIGH" || f.Title != "test vuln" || f.CVEID != "CVE-2024-0001" ||
		f.RuleID != "CVE-2024-0001" || f.File != "go.mod" || f.Line != 0 {
		t.Errorf("finding shape wrong: %+v", f)
	}
	if len(doc.Coverage) != 1 || doc.Coverage[0].File != "a.go" || doc.Coverage[0].PctLine != 82.5 {
		t.Errorf("coverage rows wrong: %+v", doc.Coverage)
	}
	if doc.GeneratedAt == "" {
		t.Error("generated_at is empty")
	}
}
