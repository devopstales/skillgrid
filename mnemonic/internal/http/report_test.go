package http

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/service"
	"github.com/devopstales/skillgrid/mnemonic/internal/scan"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// trivyHTTPFixture is the trivy report the /report tests run through the real
// scan pipeline (inline: the scan package's fixture reader is unexported).
var trivyHTTPFixture = []byte(`{"Results":[{"Target":"go.mod","Vulnerabilities":[
  {"VulnerabilityID":"CVE-2024-0001","PkgName":"example.com/lib","InstalledVersion":"1.0.0",
   "FixedVersion":"1.0.1","Severity":"HIGH","Title":"test vuln"}]}]}`)

// stubReportRun replaces the report collector's runner with a stubbed scanner
// that serves the trivy fixture. Re-armed per request (each GET /report
// constructs a fresh report.Service).
func stubReportRun(run scan.RunnerFunc) (restore func()) {
	old := reportRunOverride
	reportRunOverride = run
	return func() { reportRunOverride = old }
}

// TestHandleReport serves GET /report/junit.xml and GET /report/json against
// a project whose store already holds a trivy scan + findings, and asserts the
// door-check contract: both endpoints return 200 with the right Content-Type,
// and the JSON body carries a trivy section with the fixture CVE.
//
// SATISFIES: happy path report-http-endpoints
func TestHandleReport(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	workspace := t.TempDir()

	// Seed the project store with a trivy scan + finding through the real
	// scan service (stubbed runner).
	st, err := store.Open(dataDir, proj)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	seed := scan.New(st, dataDir)
	seed.Run = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return trivyHTTPFixture, nil
	}
	sc, err := seed.Start(context.Background(), "trivy", workspace)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if _, err := seed.StoreFindings(context.Background(), sc.ID); err != nil {
		t.Fatalf("StoreFindings: %v", err)
	}
	st.Close()

	// A downstream config with dast.url: reportConfigFrom must read it.
	if err := os.MkdirAll(filepath.Join(workspace, ".skillgrid"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspace, ".skillgrid", "config.yaml"),
		[]byte("dast:\n  url: https://example.com\nquality:\n  coverage_min: 70\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	restore := stubReportRun(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return trivyHTTPFixture, nil
	})
	defer restore()

	svc := service.New(dataDir)
	h := NewServer(svc).Handler()

	// GET /report/junit.xml → 200 application/xml, well-formed JUnit with a
	// trivy testsuite.
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/report/junit.xml?project="+proj, nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("junit.xml status = %d, want 200: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/xml" {
		t.Errorf("junit.xml Content-Type = %q, want application/xml", ct)
	}
	var doc struct {
		XMLName   xml.Name `xml:"testsuites"`
		TestSuite []struct {
			XMLName   xml.Name `xml:"testsuite"`
			Name      string   `xml:"name,attr"`
			TestCases []struct {
				XMLName xml.Name `xml:"testcase"`
				Name    string   `xml:"name,attr"`
			} `xml:"testcase"`
		} `xml:"testsuite"`
	}
	if err := xml.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
		t.Fatalf("junit.xml is not well-formed: %v", err)
	}
	found := false
	for _, ts := range doc.TestSuite {
		if ts.Name != "trivy" {
			continue
		}
		for _, tc := range ts.TestCases {
			if strings.Contains(tc.Name, "CVE-2024-0001") {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("junit.xml has no trivy testcase named with the CVE id: %s", rr.Body.String())
	}

	// GET /report/json → 200 application/json, trivy section with the CVE.
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/report/json?project="+proj, nil)
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("json status = %d, want 200: %s", rr2.Code, rr2.Body.String())
	}
	if ct := rr2.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("json Content-Type = %q, want application/json", ct)
	}
	var body struct {
		Static []struct {
			Tool     string `json:"tool"`
			Status   string `json:"status"`
			Findings []struct {
				CVEID string `json:"cve_id"`
			} `json:"findings"`
		} `json:"static"`
	}
	if err := json.Unmarshal(rr2.Body.Bytes(), &body); err != nil {
		t.Fatalf("json body does not parse: %v", err)
	}
	var trivy *[]struct {
		CVEID string `json:"cve_id"`
	}
	for i := range body.Static {
		if body.Static[i].Tool == "trivy" {
			trivy = &body.Static[i].Findings
		}
	}
	var status string
	for i := range body.Static {
		if body.Static[i].Tool == "trivy" {
			status = body.Static[i].Status
		}
	}
	if trivy == nil {
		t.Fatalf("door check FAILED: no trivy section in /report/json: %s", rr2.Body.String())
	}
	if status != "ok" {
		t.Errorf("trivy status = %q, want ok", status)
	}
	if len(*trivy) != 1 || (*trivy)[0].CVEID != "CVE-2024-0001" {
		t.Errorf("door check FAILED: trivy findings = %+v, want the fixture CVE", *trivy)
	}
}

// TestHandleReportRequiresWriteAuth: with a token set, an unauthenticated
// /report request is 401 (the report launches scanner subprocesses).
func TestHandleReportRequiresWriteAuth(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	t.Setenv("SKILLGRID_HTTP_TOKEN", "secret-token")
	svc := service.New(dataDir)
	h := NewServer(svc).Handler()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/report/json?project="+proj, nil)
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without a token, got %d", rr.Code)
	}
}
