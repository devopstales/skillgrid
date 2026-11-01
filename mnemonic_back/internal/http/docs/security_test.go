package docs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTrivyFromYaml(t *testing.T) {
	yaml := `
security:
  trivy:
    command: "trivy fs . --scanners vuln --severity CRITICAL,HIGH,MEDIUM,LOW"
    target: "."
    severities: "CRITICAL,HIGH,MEDIUM,LOW"
    scan_types: "vuln"
    fail_on: "" # advisory only

# --- Commands ---
commands:
  build: "go build ./..."
`
	cfg := parseTrivyFromYaml(yaml)
	if cfg.Command != "trivy fs . --scanners vuln --severity CRITICAL,HIGH,MEDIUM,LOW" {
		t.Errorf("command = %q", cfg.Command)
	}
	if cfg.Severities != "CRITICAL,HIGH,MEDIUM,LOW" {
		t.Errorf("severities = %q", cfg.Severities)
	}
	if cfg.ScanTypes != "vuln" {
		t.Errorf("scan_types = %q", cfg.ScanTypes)
	}
	if cfg.FailOn != "" {
		t.Errorf("fail_on = %q, want empty", cfg.FailOn)
	}
	if cfg.Target != "." {
		t.Errorf("target = %q", cfg.Target)
	}
}

func TestParseTrivyFromYamlWithFailOn(t *testing.T) {
	yaml := `
security:
  trivy:
    severities: "CRITICAL,HIGH"
    fail_on: "CRITICAL"
`
	cfg := parseTrivyFromYaml(yaml)
	if cfg.Severities != "CRITICAL,HIGH" {
		t.Errorf("severities = %q", cfg.Severities)
	}
	if cfg.FailOn != "CRITICAL" {
		t.Errorf("fail_on = %q, want CRITICAL", cfg.FailOn)
	}
}

func TestParseTrivyFromYamlEmpty(t *testing.T) {
	cfg := parseTrivyFromYaml("")
	if cfg.Target != "" {
		t.Errorf("target = %q, want empty (defaults applied later)", cfg.Target)
	}
}

func TestMatchKeyValue(t *testing.T) {
	tests := []struct {
		line     string
		wantKey  string
		wantVal  string
		wantNil  bool
	}{
		{`command: "trivy fs ."`, "command", "trivy fs .", false},
		{`severities: "CRITICAL,HIGH"`, "severities", "CRITICAL,HIGH", false},
		{`fail_on: "" # advisory`, "fail_on", "", false},
		{`target: "."`, "target", ".", false},
		{`no_colon_here`, "", "", true},
	}
	for _, tt := range tests {
		got := matchKeyValue(tt.line)
		if tt.wantNil {
			if got != nil {
				t.Errorf("matchKeyValue(%q) = %v, want nil", tt.line, got)
			}
			continue
		}
		if got == nil {
			t.Errorf("matchKeyValue(%q) = nil, want %v", tt.line, []string{tt.wantKey, tt.wantVal})
			continue
		}
		if got[0] != tt.wantKey {
			t.Errorf("key = %q, want %q", got[0], tt.wantKey)
		}
		if got[1] != tt.wantVal {
			t.Errorf("val = %q, want %q", got[1], tt.wantVal)
		}
	}
}

func TestNewTrivyHandlerNoTrivy(t *testing.T) {
	h := NewTrivy(t.TempDir())
	req := httptest.NewRequest("GET", "/security/trivy", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var res TrivyResult
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// If trivy is not installed, available should be false.
	// If it is installed, it should run and return a result.
	if !res.Available {
		if res.Verdict != "N/A" {
			t.Errorf("verdict = %q, want N/A when trivy missing", res.Verdict)
		}
		return
	}
	if res.Verdict != "PASS" && res.Verdict != "ERROR" {
		t.Errorf("verdict = %q, want PASS or ERROR", res.Verdict)
	}
}

func TestTrivyResultJSON(t *testing.T) {
	res := TrivyResult{
		Available:  true,
		Version:    "0.72.0",
		ScannedAt:  "2026-10-01T00:00:00Z",
		DurationMs: 1500,
		Critical:   1,
		High:       2,
		Medium:     3,
		Low:        0,
		Total:      6,
		Verdict:    "FAIL",
		FailOn:     "CRITICAL",
		Findings: []TrivyFinding{
			{VulnerabilityID: "CVE-2024-1", Severity: "CRITICAL", Package: "foo", Installed: "1.0.0", Fixed: "1.1.0", Title: "Bad thing", Target: "go.mod"},
		},
	}
	data, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded TrivyResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Critical != 1 || decoded.High != 2 || decoded.Medium != 3 {
		t.Errorf("counts wrong: %+v", decoded)
	}
	if len(decoded.Findings) != 1 || decoded.Findings[0].VulnerabilityID != "CVE-2024-1" {
		t.Errorf("findings wrong: %+v", decoded.Findings)
	}
}

func TestParseTrivyConfigDefaults(t *testing.T) {
	dir := t.TempDir()
	cfg := parseTrivyConfig(dir)
	if cfg.Target != "." {
		t.Errorf("default target = %q, want .", cfg.Target)
	}
	if cfg.Severities != "CRITICAL,HIGH,MEDIUM,LOW" {
		t.Errorf("default severities = %q", cfg.Severities)
	}
	if cfg.ScanTypes != "vuln" {
		t.Errorf("default scan_types = %q", cfg.ScanTypes)
	}
}

func TestParseTrivyConfigFromFile(t *testing.T) {
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, ".skillgrid")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatal(err)
	}
	yaml := `
security:
  trivy:
    severities: "CRITICAL,HIGH"
    scan_types: "vuln,secret"
    fail_on: "CRITICAL"
`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := parseTrivyConfig(dir)
	if cfg.Severities != "CRITICAL,HIGH" {
		t.Errorf("severities = %q, want CRITICAL,HIGH", cfg.Severities)
	}
	if cfg.ScanTypes != "vuln,secret" {
		t.Errorf("scan_types = %q, want vuln,secret", cfg.ScanTypes)
	}
	if cfg.FailOn != "CRITICAL" {
		t.Errorf("fail_on = %q, want CRITICAL", cfg.FailOn)
	}
}

func TestSevRank(t *testing.T) {
	if sevRank["CRITICAL"] <= sevRank["HIGH"] {
		t.Error("CRITICAL should rank higher than HIGH")
	}
	if sevRank["HIGH"] <= sevRank["MEDIUM"] {
		t.Error("HIGH should rank higher than MEDIUM")
	}
	if sevRank["MEDIUM"] <= sevRank["LOW"] {
		t.Error("MEDIUM should rank higher than LOW")
	}
}

func TestTrivyFindingSort(t *testing.T) {
	findings := []TrivyFinding{
		{Severity: "LOW", VulnerabilityID: "CVE-L"},
		{Severity: "CRITICAL", VulnerabilityID: "CVE-C"},
		{Severity: "HIGH", VulnerabilityID: "CVE-H"},
	}
	// Bubble sort (same as in runTrivy)
	for i := 0; i < len(findings); i++ {
		for j := i + 1; j < len(findings); j++ {
			ri, rj := sevRank[findings[i].Severity], sevRank[findings[j].Severity]
			if rj > ri || (rj == ri && findings[j].VulnerabilityID < findings[i].VulnerabilityID) {
				findings[i], findings[j] = findings[j], findings[i]
			}
		}
	}
	if findings[0].Severity != "CRITICAL" || findings[1].Severity != "HIGH" || findings[2].Severity != "LOW" {
		t.Errorf("sort order wrong: %+v", findings)
	}
}

func TestReadFileSafe(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := readFileSafe(dir, "test.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Errorf("got %q, want hello", got)
	}
	_, err = readFileSafe(dir, "missing.txt")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestMatchKeyValueWithComment(t *testing.T) {
	got := matchKeyValue(`fail_on: "" # advisory only`)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if got[0] != "fail_on" {
		t.Errorf("key = %q", got[0])
	}
	if got[1] != "" {
		t.Errorf("val = %q, want empty string", got[1])
	}
}

func TestMatchKeyValueQuoted(t *testing.T) {
	got := matchKeyValue(`severities: "CRITICAL,HIGH,MEDIUM,LOW"`)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if got[1] != "CRITICAL,HIGH,MEDIUM,LOW" {
		t.Errorf("val = %q", got[1])
	}
}

func TestTrivyResultErrorMessage(t *testing.T) {
	res := TrivyResult{
		Available:    false,
		Verdict:      "N/A",
		ErrorMessage: "trivy not found on PATH",
	}
	data, _ := json.Marshal(res)
	if !strings.Contains(string(data), "trivy not found on PATH") {
		t.Errorf("error_message not serialized: %s", data)
	}
}
