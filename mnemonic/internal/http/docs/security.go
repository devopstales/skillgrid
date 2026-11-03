package docs

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// trivyCacheTTL is how long a scan result stays fresh before a re-scan is
// triggered. Advisory-only: a stale cache still serves findings, it just
// triggers a background re-scan.
const trivyCacheTTL = 10 * time.Minute

type TrivyFinding struct {
	VulnerabilityID string `json:"id"`
	Severity        string `json:"severity"`
	Package         string `json:"package"`
	Installed       string `json:"installed_version"`
	Fixed           string `json:"fixed_version,omitempty"`
	Title           string `json:"title,omitempty"`
	Target          string `json:"target"`
}

type TrivyResult struct {
	Available    bool           `json:"available"`
	Version      string         `json:"version,omitempty"`
	ScannedAt    string         `json:"scanned_at,omitempty"`
	DurationMs   int64          `json:"duration_ms"`
	Critical     int            `json:"critical"`
	High         int            `json:"high"`
	Medium       int            `json:"medium"`
	Low          int            `json:"low"`
	Total        int            `json:"total"`
	FailOn       string         `json:"fail_on,omitempty"`
	Verdict      string         `json:"verdict"`
	Findings     []TrivyFinding `json:"findings"`
	ErrorMessage string         `json:"error_message,omitempty"`
}

var trivyCache = struct {
	result TrivyResult
	at     time.Time
}{}

var sevRank = map[string]int{"CRITICAL": 4, "HIGH": 3, "MEDIUM": 2, "LOW": 1}

// NewTrivy returns the GET /security/trivy handler. It shells out to the
// trivy CLI (configured in config.yaml under security.trivy), parses the JSON
// report, and returns a severity summary + finding list. Results are cached
// for trivyCacheTTL to avoid re-scanning on every poll.
func NewTrivy(cwd string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := scanTrivy(r.Context(), cwd)
		writeJSON(w, http.StatusOK, result)
	})
}

func scanTrivy(ctx context.Context, cwd string) TrivyResult {
	if trivyCache.result.Available && time.Since(trivyCache.at) < trivyCacheTTL {
		return trivyCache.result
	}
	res := runTrivy(ctx, cwd)
	trivyCache = struct {
		result TrivyResult
		at     time.Time
	}{result: res, at: time.Now()}
	return res
}

func runTrivy(ctx context.Context, cwd string) TrivyResult {
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	// Find the trivy binary.
	bin, err := exec.LookPath("trivy")
	if err != nil {
		return TrivyResult{Available: false, Verdict: "N/A", ErrorMessage: "trivy not found on PATH"}
	}

	// Read config for trivy settings.
	cfg := parseTrivyConfig(cwd)

	args := []string{"fs", cwd, "--scanners", cfg.ScanTypes, "--format", "json", "--quiet"}
	if cfg.Severities != "" {
		args = append(args, "--severity", cfg.Severities)
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	runErr := cmd.Run()
	duration := time.Since(start).Milliseconds()

	if runErr != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = runErr.Error()
		}
		return TrivyResult{
			Available: true, DurationMs: duration, Verdict: "ERROR",
			ErrorMessage: msg, Findings: []TrivyFinding{},
		}
	}

	// Parse the trivy JSON report.
	var report struct {
		Trivy struct {
			Version string `json:"Version"`
		} `json:"Trivy"`
		CreatedAt string `json:"CreatedAt"`
		Results   []struct {
			Target          string `json:"Target"`
			Vulnerabilities []struct {
				VulnerabilityID string `json:"VulnerabilityID"`
				Severity        string `json:"Severity"`
				PkgName         string `json:"PkgName"`
				Installed       string `json:"InstalledVersion"`
				Fixed           string `json:"FixedVersion"`
				Title           string `json:"Title"`
			} `json:"Vulnerabilities"`
		} `json:"Results"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		return TrivyResult{
			Available: true, DurationMs: duration, Verdict: "ERROR",
			ErrorMessage: "failed to parse trivy output: " + err.Error(),
			Findings: []TrivyFinding{},
		}
	}

	var findings []TrivyFinding
	counts := map[string]int{"CRITICAL": 0, "HIGH": 0, "MEDIUM": 0, "LOW": 0}
	for _, r := range report.Results {
		for _, v := range r.Vulnerabilities {
			findings = append(findings, TrivyFinding{
				VulnerabilityID: v.VulnerabilityID,
				Severity:        v.Severity,
				Package:         v.PkgName,
				Installed:       v.Installed,
				Fixed:           v.Fixed,
				Title:           v.Title,
				Target:          r.Target,
			})
			if _, ok := counts[v.Severity]; ok {
				counts[v.Severity]++
			}
		}
	}

	// Sort by severity rank descending, then by ID.
	for i := 0; i < len(findings); i++ {
		for j := i + 1; j < len(findings); j++ {
			ri, rj := sevRank[findings[i].Severity], sevRank[findings[j].Severity]
			if rj > ri || (rj == ri && findings[j].VulnerabilityID < findings[i].VulnerabilityID) {
				findings[i], findings[j] = findings[j], findings[i]
			}
		}
	}

	total := len(findings)
	verdict := "PASS"
	if cfg.FailOn != "" {
		failRank := sevRank[cfg.FailOn]
		for _, f := range findings {
			if sevRank[f.Severity] >= failRank {
				verdict = "FAIL"
				break
			}
		}
	} else if total > 0 {
		verdict = "PASS"
	}

	return TrivyResult{
		Available:  true,
		Version:    report.Trivy.Version,
		ScannedAt:  report.CreatedAt,
		DurationMs: duration,
		Critical:   counts["CRITICAL"],
		High:       counts["HIGH"],
		Medium:     counts["MEDIUM"],
		Low:        counts["LOW"],
		Total:      total,
		FailOn:     cfg.FailOn,
		Verdict:    verdict,
		Findings:   findings,
	}
}

type trivyConfig struct {
	Command    string
	Target     string
	Severities string
	ScanTypes  string
	FailOn     string
}

func parseTrivyConfig(cwd string) trivyConfig {
	cfg := trivyConfig{
		Target:     ".",
		Severities: "CRITICAL,HIGH,MEDIUM,LOW",
		ScanTypes:  "vuln",
	}
	if data, err := readFileSafe(cwd, ".skillgrid/config.yaml"); err == nil {
		cfg = parseTrivyFromYaml(string(data))
	}
	if cfg.Target == "" {
		cfg.Target = "."
	}
	if cfg.Severities == "" {
		cfg.Severities = "CRITICAL,HIGH,MEDIUM,LOW"
	}
	if cfg.ScanTypes == "" {
		cfg.ScanTypes = "vuln"
	}
	return cfg
}

func parseTrivyFromYaml(yaml string) trivyConfig {
	var cfg trivyConfig
	lines := strings.Split(yaml, "\n")
	inSecurity := false
	inTrivy := false
	secDepth := 0
	trivyDepth := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))

		if !inSecurity {
			if strings.HasPrefix(trimmed, "security:") && indent == 0 {
				inSecurity = true
				secDepth = indent
			}
			continue
		}

		if !inTrivy {
			if indent <= secDepth && trimmed != "security:" {
				break
			}
			if strings.HasPrefix(trimmed, "trivy:") && indent > secDepth {
				inTrivy = true
				trivyDepth = indent
			}
			continue
		}

		if indent <= trivyDepth {
			break
		}

		m := matchKeyValue(trimmed)
		if m == nil {
			continue
		}
		key, val := m[0], strings.Trim(m[1], `"`)
		switch key {
		case "command":
			cfg.Command = val
		case "target":
			cfg.Target = val
		case "severities":
			cfg.Severities = val
		case "scan_types":
			cfg.ScanTypes = val
		case "fail_on":
			cfg.FailOn = val
		}
	}
	return cfg
}

func readFileSafe(cwd, rel string) (string, error) {
	abs, err := filepath.Abs(filepath.Join(cwd, rel))
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func matchKeyValue(line string) []string {
	for i, c := range line {
		if c == ':' {
			key := strings.TrimSpace(line[:i])
			val := strings.TrimSpace(line[i+1:])
			if idx := strings.Index(val, " #"); idx > 0 {
				val = val[:idx]
			}
			val = strings.TrimSpace(val)
			val = strings.Trim(val, `"`)
			return []string{key, val}
		}
	}
	return nil
}
