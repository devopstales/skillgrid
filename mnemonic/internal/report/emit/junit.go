// Package emit renders a report into its two output formats: JUnit XML for
// CI and JSON for the dashboard. Both read the aggregated report only — they
// own no collection logic.
package emit

import (
	"bytes"
	"encoding/xml"
	"fmt"

	"github.com/devopstales/skillgrid/mnemonic/internal/report"
)

type junitTestSuite struct {
	XMLName   xml.Name    `xml:"testsuite"`
	Name      string      `xml:"name,attr"`
	Tests     int         `xml:"tests,attr"`
	Failures  int         `xml:"failures,attr"`
	TestCases []junitCase `xml:"testcase"`
}

type junitTestSuites struct {
	XMLName   xml.Name         `xml:"testsuites"`
	Tests     int              `xml:"tests,attr"`
	Failures  int              `xml:"failures,attr"`
	TestSuite []junitTestSuite `xml:"testsuite"`
}

type junitCase struct {
	XMLName   xml.Name      `xml:"testcase"`
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
}

type junitFailure struct {
	XMLName xml.Name `xml:"failure"`
	Message string   `xml:"message,attr"`
}

// JUnit renders the report as JUnit XML: one <testsuite> per source (static
// + dynamic), each finding a <testcase> whose <failure message> begins with
// the normalized severity. Zero-finding sources — including ones that
// errored — emit an empty testsuite, never an omitted one, so CI sees every
// tool that was part of the run.
func JUnit(r *report.Report) ([]byte, error) {
	var suites junitTestSuites
	for _, section := range [][]report.SourceResult{r.Static, r.Dynamic} {
		for _, src := range section {
			ts := junitTestSuite{Name: src.Tool, Tests: len(src.Findings), Failures: len(src.Findings)}
			for _, f := range src.Findings {
				id := f.CVEID
				if id == "" {
					id = f.RuleID
				}
				name := fmt.Sprintf("%s:%s", id, f.File)
				if f.Line > 0 {
					name = fmt.Sprintf("%s:%d", name, f.Line)
				}
				msg := fmt.Sprintf("%s: %s", f.Severity, f.Title)
				if f.Severity == "" {
					msg = f.Title
				}
				ts.TestCases = append(ts.TestCases, junitCase{
					Name:      name,
					ClassName: src.Tool,
					Failure:   &junitFailure{Message: msg},
				})
			}
			suites.TestSuite = append(suites.TestSuite, ts)
			suites.Tests += ts.Tests
			suites.Failures += ts.Failures
		}
	}
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(suites); err != nil {
		return nil, fmt.Errorf("encode junit xml: %w", err)
	}
	return buf.Bytes(), nil
}
