package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// GET /project/current reports the single active project (repo base name from
// the git remote, else the working-dir base name). Read-only.
func TestProjectCurrent(t *testing.T) {
	h := newHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/project/current", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var body struct {
		Project string `json:"project"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if body.Project == "" || body.Project == "unknown" {
		t.Errorf("project should resolve to a real name, got %q", body.Project)
	}
	// In this repo the remote is devopstales/skillgrid.git → "skillgrid".
	t.Logf("resolved project: %q", body.Project)
}

// currentProjectFromGit strips owner + .git from the remote URL.
func TestCurrentProjectFromGit(t *testing.T) {
	// No origin remote in a temp dir → not ok.
	t.Chdir(t.TempDir())
	if _, ok := currentProjectFromGit(); ok {
		t.Error("expected ok=false with no origin remote")
	}
}
