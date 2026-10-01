package secondbrain

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

const sessionA = "sb-test-session"

// seedObservations stores n observations under project containing the term.
// It reuses the same session and title shape so FTS matches the term.
func seedObservations(t *testing.T, svc *service.Service, project string, n int, term string) {
	t.Helper()
	ensureSession(t, svc, project)
	hh, cleanup, err := svc.Open(project)
	if err != nil {
		t.Fatalf("open %s: %v", project, err)
	}
	defer cleanup()
	for i := 0; i < n; i++ {
		if _, err := hh.Memory().Save(context.Background(), memory.SaveInput{
			Title:     fmt.Sprintf("%s note %d", term, i),
			Type:      "decision",
			Content:   fmt.Sprintf("%s content %d", term, i),
			Scope:     "project",
			SessionID: sessionA,
		}); err != nil {
			t.Fatalf("seed save %d: %v", i, err)
		}
	}
}

func TestAskCited_NoEmbedder(t *testing.T) {
	svc := newTestService(t, false)
	seedObservations(t, svc, "test-project", 10, "auth")

	res, err := AskCited(context.Background(), svc, "auth", "test-project", false, 2000)
	if err != nil {
		t.Fatalf("AskCited: %v", err)
	}
	if len(res.Citations) == 0 {
		t.Fatal("expected citations from FTS floor, got none")
	}
	if !res.Degraded {
		t.Error("expected Degraded=true with no embedder")
	}
	if res.MatchedVia != "keyword" {
		t.Errorf("expected matched_via=keyword, got %q", res.MatchedVia)
	}
	for _, c := range res.Citations {
		if c.ID == 0 || c.Title == "" || c.Type == "" {
			t.Errorf("citation missing required fields: %+v", c)
		}
	}
}

func TestAskCited_TokenBounded(t *testing.T) {
	svc := newTestService(t, false)
	seedObservations(t, svc, "test-project", 50, "auth")

	res, err := AskCited(context.Background(), svc, "auth", "test-project", false, 200)
	if err != nil {
		t.Fatalf("AskCited: %v", err)
	}
	if res.TotalTokens > 200 {
		t.Errorf("total tokens %d exceeds cap 200", res.TotalTokens)
	}
}

func TestAskCited_HybridWhenEmbedder(t *testing.T) {
	svc := newTestService(t, true)
	seedEmbedded(t, svc, "test-project", 10)

	res, err := AskCited(context.Background(), svc, "authentication", "test-project", false, 2000)
	if err != nil {
		t.Fatalf("AskCited: %v", err)
	}
	if res.Degraded {
		t.Error("expected Degraded=false with embedder active")
	}
	if res.MatchedVia != "hybrid" {
		t.Errorf("expected matched_via=hybrid, got %q", res.MatchedVia)
	}
}

func TestAskCited_ProjectScoped(t *testing.T) {
	svc := newTestService(t, false)
	seedObservations(t, svc, "project-a", 5, "auth")
	seedObservations(t, svc, "project-b", 5, "auth")

	res, err := AskCited(context.Background(), svc, "auth", "project-a", false, 2000)
	if err != nil {
		t.Fatalf("AskCited: %v", err)
	}
	if len(res.Citations) == 0 {
		t.Fatal("expected citations for project-a, got none")
	}
	for _, c := range res.Citations {
		if c.Project != "project-a" {
			t.Errorf("citation from wrong project: %q (want project-a)", c.Project)
		}
	}
}

func TestAskCited_AllProjects(t *testing.T) {
	svc := newTestService(t, false)
	seedObservations(t, svc, "project-a", 5, "auth")
	seedObservations(t, svc, "project-b", 5, "auth")

	res, err := AskCited(context.Background(), svc, "auth", "project-a", true, 2000)
	if err != nil {
		t.Fatalf("AskCited: %v", err)
	}
	if len(res.Citations) == 0 {
		t.Fatal("expected citations across projects, got none")
	}
}

// TestAskCited_RedactsPaths locks the privacy floor: full local paths in a
// snippet are redacted, not leaked.
func TestAskCited_RedactsPaths(t *testing.T) {
	svc := newTestService(t, false)
	ensureSession(t, svc, "test-project")
	hh, cleanup, oerr := svc.Open("test-project")
	if oerr != nil {
		t.Fatalf("open: %v", oerr)
	}
	if _, err := hh.Memory().Save(context.Background(), memory.SaveInput{
		Title:     "auth path leak",
		Type:      "decision",
		Content:   "auth at /Users/paladm/git/secret and /home/dev/x",
		Scope:     "project",
		SessionID: sessionA,
	}); err != nil {
		cleanup()
		t.Fatalf("save: %v", err)
	}
	cleanup()
	res, err := AskCited(context.Background(), svc, "auth", "test-project", false, 2000)
	if err != nil {
		t.Fatalf("AskCited: %v", err)
	}
	if len(res.Citations) == 0 {
		t.Fatal("expected a citation")
	}
	for _, c := range res.Citations {
		if strings.Contains(c.Snippet, "/Users/") || strings.Contains(c.Snippet, "/home/") {
			t.Errorf("snippet leaked a local path: %q", c.Snippet)
		}
	}
}
