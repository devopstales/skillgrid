package http

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 6.7 [AFK] openapi.yaml documents the Phase 6 activity/plans/git routes.
func TestPhase6_OpenAPI(t *testing.T) {
	here, _ := os.Getwd()
	openapi := filepath.Join(here, "ui", "openapi.yaml")
	data, err := os.ReadFile(openapi)
	if err != nil {
		t.Skipf("openapi.yaml not found at %s: %v", openapi, err)
	}
	text := string(data)
	for _, want := range []string{
		"/activity/events", "/activity/stats", "/activity/stream",
		"/plans", "/plans/{id}", "/specs", "/specs/{path...}",
		"/git/commits", "/git/commits/{sha}", "/git/diff/{sha}",
		"/git/file-history", "/git/blame",
		"ActivityEvent", "PlanSummary", "PlanDetail", "GitCommit", "GitBlameLine",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("openapi.yaml missing Phase 6 %s", want)
		}
	}
}
