package secondbrain

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
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

// --- TICKET-02: llm mode (fail-open to cited) ---

func TestAsk_LLMCitedProse(t *testing.T) {
	svc := newTestService(t, false)
	seedObservations(t, svc, "test-project", 5, "auth")
	llm := &fakeLLM{fn: func(_ context.Context, system, user string) (string, error) {
		return "We decided to use SQLite [obs:1].", nil
	}}
	setAskLLM(t, llm)

	res, err := Ask(context.Background(), svc, "llm", "what did we decide about auth", "test-project", false, 2000)
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if res.Answer == "" {
		t.Error("expected non-empty prose answer in llm mode")
	}
	if !strings.Contains(res.Answer, "[obs:") {
		t.Errorf("prose answer missing [obs:] citation: %q", res.Answer)
	}
	if len(res.Sources) == 0 {
		t.Error("expected sources array in llm mode")
	}
	if len(res.Citations) == 0 {
		t.Error("expected citations to be preserved in llm mode")
	}
}

func TestAsk_LLMFailsOpen(t *testing.T) {
	svc := newTestService(t, false)
	seedObservations(t, svc, "test-project", 5, "auth")
	llm := &fakeLLM{fn: func(_ context.Context, _, _ string) (string, error) {
		return "", fmt.Errorf("llm down")
	}}
	setAskLLM(t, llm)

	res, err := Ask(context.Background(), svc, "llm", "auth", "test-project", false, 2000)
	if err != nil {
		t.Fatalf("Ask should not error on llm failure (fail open): %v", err)
	}
	if len(res.Citations) == 0 {
		t.Error("expected cited fallback on llm failure")
	}
	if res.Answer != "" {
		t.Errorf("expected empty answer on llm failure, got %q", res.Answer)
	}
	if len(res.Sources) == 0 {
		t.Error("expected sources from the cited floor on llm failure")
	}
}

func TestAsk_LLMTimeoutFailsOpen(t *testing.T) {
	svc := newTestService(t, false)
	seedObservations(t, svc, "test-project", 5, "auth")
	// The seam honors ctx: it returns (deadline error) once the context
	// cancels, standing in for a real LLM HTTP call that outlives the 3s
	// budget.
	llm := &fakeLLM{fn: func(ctx context.Context, _ string, _ string) (string, error) {
		// Honor ctx like a real LLM HTTP call would, so the deadline cuts
		// this short.
		select {
		case <-time.After(10 * time.Second):
			return "late [obs:1]", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}}
	setAskLLM(t, llm)

	// Pre-boundary deadline (1s < 3s llmTimeout): the 3s budget must not
	// extend it, so the call is cut short and fails open.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	start := time.Now()
	res, err := Ask(ctx, svc, "llm", "auth", "test-project", false, 2000)
	if err != nil {
		t.Fatalf("Ask should not error on llm timeout (fail open): %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("expected the pre-boundary 1s deadline to bound the call, took %v", elapsed)
	}
	if len(res.Citations) == 0 {
		t.Error("expected cited fallback on llm timeout")
	}
}

func TestAsk_CitedModeSkipsLLM(t *testing.T) {
	svc := newTestService(t, false)
	seedObservations(t, svc, "test-project", 5, "auth")
	called := false
	llm := &fakeLLM{fn: func(_ context.Context, _, _ string) (string, error) {
		called = true
		return "should not be called [obs:1]", nil
	}}
	setAskLLM(t, llm)

	res, err := Ask(context.Background(), svc, "cited", "auth", "test-project", false, 2000)
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if called {
		t.Error("cited mode must not call the LLM")
	}
	if res.Answer != "" {
		t.Errorf("cited mode must not populate an answer, got %q", res.Answer)
	}
	if len(res.Citations) == 0 {
		t.Error("expected citations in cited mode")
	}
}

func TestAsk_NoLLMAttachesToCited(t *testing.T) {
	svc := newTestService(t, false)
	seedObservations(t, svc, "test-project", 5, "auth")
	// No LLM attached at all: llm mode degrades to the cited floor.
	res, err := Ask(context.Background(), svc, "llm", "auth", "test-project", false, 2000)
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if len(res.Citations) == 0 {
		t.Error("expected citations when no LLM is attached")
	}
}

func TestAsk_EmptyQueryNoLLM(t *testing.T) {
	svc := newTestService(t, false)
	called := false
	setAskLLM(t, &fakeLLM{fn: func(_ context.Context, _, _ string) (string, error) {
		called = true
		return "[obs:1]", nil
	}})
	res, err := Ask(context.Background(), svc, "llm", "   ", "test-project", false, 2000)
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if called {
		t.Error("an empty query must not reach the LLM")
	}
	if res == nil || len(res.Citations) != 0 {
		t.Errorf("expected an empty no-op result, got %+v", res)
	}
}
