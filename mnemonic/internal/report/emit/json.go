package emit

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/report"
)

// findingJSON is the dashboard-facing wire shape of a scan finding
// (snake_case keys, matching the /report/json contract).
type findingJSON struct {
	Tool         string   `json:"tool,omitempty"`
	Severity     string   `json:"severity"`
	Title        string   `json:"title"`
	CVEID        string   `json:"cve_id,omitempty"`
	RuleID       string   `json:"rule_id,omitempty"`
	Package      string   `json:"package,omitempty"`
	Version      string   `json:"version,omitempty"`
	FixedVersion string   `json:"fixed_version,omitempty"`
	File         string   `json:"file,omitempty"`
	Line         int      `json:"line"`
	Message      string   `json:"message,omitempty"`
	Links        []string `json:"links,omitempty"`
}

// sourceJSON is one report section (tool, status, findings).
type sourceJSON struct {
	Tool     string        `json:"tool"`
	Status   string        `json:"status"`
	Findings []findingJSON `json:"findings"`
}

// coverageRowJSON is a per-file line-coverage row.
type coverageRowJSON struct {
	File    string  `json:"file"`
	Pkg     string  `json:"pkg"`
	PctLine float64 `json:"pct_line"`
}

// JSON renders the report in the dashboard-facing shape: a section per source
// (tool, status, findings with severity/title/cve_id/rule_id/file/line), the
// coverage rows, and generated_at. The dashboard renders it with no client
// transform.
func JSON(r *report.Report) ([]byte, error) {
	doc := struct {
		Static      []sourceJSON      `json:"static"`
		Dynamic     []sourceJSON      `json:"dynamic"`
		Coverage    []coverageRowJSON `json:"coverage"`
		GeneratedAt string            `json:"generated_at"`
	}{
		Static:      []sourceJSON{},
		Dynamic:     []sourceJSON{},
		Coverage:    []coverageRowJSON{},
		GeneratedAt: r.GeneratedAt.UTC().Format(time.RFC3339),
	}
	convert := func(srs []report.SourceResult) []sourceJSON {
		out := make([]sourceJSON, 0, len(srs))
		for _, sr := range srs {
			fjs := make([]findingJSON, 0, len(sr.Findings))
			for _, f := range sr.Findings {
				fjs = append(fjs, findingJSON{
					Tool: f.Tool, Severity: f.Severity, Title: f.Title,
					CVEID: f.CVEID, RuleID: f.RuleID, Package: f.Package,
					Version: f.Version, FixedVersion: f.FixedVersion,
					File: f.File, Line: f.Line, Message: f.Message, Links: f.Links,
				})
			}
			out = append(out, sourceJSON{Tool: sr.Tool, Status: sr.Status, Findings: fjs})
		}
		return out
	}
	doc.Static = convert(r.Static)
	doc.Dynamic = convert(r.Dynamic)
	for _, c := range r.Coverage {
		doc.Coverage = append(doc.Coverage, coverageRowJSON{File: c.File, Pkg: c.Pkg, PctLine: c.PctLine})
	}
	out, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("encode report json: %w", err)
	}
	return out, nil
}
