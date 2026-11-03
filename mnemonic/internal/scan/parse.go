package scan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
// findings. Severities are kept raw; StoreFindings normalizes them via
// NormalizeSeverity after the dedup hash is computed.
func Parse(tool string, data []byte) ([]Finding, error) {
	switch tool {
	case "trivy":
		return parseTrivy(data)
	case "wapiti":
		return parseWapiti(data)
	case "nuclei":
		return parseNuclei(data)
	case "semgrep":
		return parseSemgrep(data)
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

// parseWapiti maps wapiti's report[] entries (type/info/url) into findings
// (rule_id=type, file=url, line=0). Wapiti has no per-finding severity; the
// missing value collapses to INFO through NormalizeSeverity, matching the
// trivy arm's handling of an absent severity.
func parseWapiti(data []byte) ([]Finding, error) {
	var report struct {
		Report []struct {
			Type string `json:"type"`
			Info string `json:"info"`
			URL  string `json:"url"`
		} `json:"report"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parse wapiti: %w", err)
	}
	var findings []Finding
	for _, e := range report.Report {
		findings = append(findings, Finding{
			RuleID:  e.Type,
			Title:   e.Type,
			File:    e.URL,
			Line:    0,
			Message: e.Info,
		})
	}
	return findings, nil
}

// parseNuclei maps nuclei's JSONL output (one finding object per line) into
// findings (rule_id=template-id, file=host, line=0, message=matcher-name).
func parseNuclei(data []byte) ([]Finding, error) {
	var findings []Finding
	dec := json.NewDecoder(bytes.NewReader(data))
	for dec.More() {
		var entry struct {
			TemplateID  string `json:"template-id"`
			Info        struct {
				Severity string `json:"severity"`
			} `json:"info"`
			MatcherName string `json:"matcher-name"`
			Host        string `json:"host"`
		}
		if err := dec.Decode(&entry); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("parse nuclei: %w", err)
		}
		findings = append(findings, Finding{
			RuleID:   entry.TemplateID,
			Severity: entry.Info.Severity,
			Title:    entry.MatcherName,
			File:     entry.Host,
			Line:     0,
			Message:  entry.MatcherName,
		})
	}
	return findings, nil
}

// parseSemgrep maps semgrep's results[] (check_id/extra.severity/path/
// start.line/extra.message) into findings with the raw severity kept for
// NormalizeSeverity (ERROR→HIGH, WARNING→MEDIUM, NOTE→INFO).
func parseSemgrep(data []byte) ([]Finding, error) {
	var report struct {
		Results []struct {
			CheckID string `json:"check_id"`
			Extra   struct {
				Severity string `json:"severity"`
				Message  string `json:"message"`
			} `json:"extra"`
			Path  string `json:"path"`
			Start struct {
				Line int `json:"line"`
			} `json:"start"`
		} `json:"results"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("parse semgrep: %w", err)
	}
	var findings []Finding
	for _, r := range report.Results {
		findings = append(findings, Finding{
			RuleID:   r.CheckID,
			Severity: r.Extra.Severity,
			File:     r.Path,
			Line:     r.Start.Line,
			Message:  r.Extra.Message,
		})
	}
	return findings, nil
}
