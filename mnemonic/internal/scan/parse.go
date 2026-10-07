package scan

import (
	"encoding/json"
	"fmt"
)

// Finding is a tool-normalized finding ready for dedup-hash + upsert.
type Finding struct {
	Tool         string
	RuleID       string
	Severity     string
	Title        string
	CVEID        string
	Package      string
	Version      string
	FixedVersion string
	File         string
	Line         int
	Message      string
	Links        []string
}

// Parse converts raw scanner output for the given tool into normalized
// findings. TICKET-03 extends the dispatch with wapiti, nuclei, and semgrep.
func Parse(tool string, data []byte) ([]Finding, error) {
	switch tool {
	case "trivy":
		return parseTrivy(data)
	default:
		return nil, fmt.Errorf("parse: unsupported tool %q", tool)
	}
}

// parseTrivy maps trivy's JSON report Results[].Vulnerabilities[] into
// findings (rule_id=CVE, file=target, line=0).
func parseTrivy(data []byte) ([]Finding, error) {
	var report struct {
		Results []struct {
			Target          string `json:"Target"`
			Vulnerabilities []struct {
				VulnerabilityID          string   `json:"VulnerabilityID"`
				PkgName                  string   `json:"PkgName"`
				InstalledVersion         string   `json:"InstalledVersion"`
				FixedVersion             string   `json:"FixedVersion"`
				Severity                 string   `json:"Severity"`
				Title                    string   `json:"Title"`
				VulnerabilityDescription string   `json:"VulnerabilityDescription"`
				PrimaryURL               string   `json:"PrimaryURL"`
				References               []string `json:"References"`
			} `json:"Vulnerabilities"`
		} `json:"Results"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parse trivy: %w", err)
	}
	var findings []Finding
	for _, r := range report.Results {
		for _, v := range r.Vulnerabilities {
			message := v.VulnerabilityDescription
			if message == "" {
				message = v.Title
			}
			f := Finding{
				RuleID:       v.VulnerabilityID,
				Severity:     v.Severity,
				Title:        v.Title,
				CVEID:        v.VulnerabilityID,
				Package:      v.PkgName,
				Version:      v.InstalledVersion,
				FixedVersion: v.FixedVersion,
				File:         r.Target,
				Line:         0,
				Message:      message,
			}
			if v.PrimaryURL != "" {
				f.Links = append(f.Links, v.PrimaryURL)
			}
			f.Links = append(f.Links, v.References...)
			findings = append(findings, f)
		}
	}
	return findings, nil
}
