package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// TestMemSearchAllProjectsFederated is 26.2 [RED] — the mem_search handler's
// `all_projects=true` path (the federated cross-store query) works end-to-end at
// the MCP tool boundary. The federated path (SearchObservationsAll over every
// store under the service dataDir) is distinguished from the single-bucket
// path by the response contract: the all_projects response carries
// "project":"all" + "all_projects":true, while the scoped response carries the
// bucket's project id and no all_projects flag. This guards the 014 step-03
// federated query seam behind the MCP tool that agents actually call.
func TestMemSearchAllProjectsFederated(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("MNEMONIC_PROJECT", "fed-a")
	svc := service.New(dataDir)
	SetService(svc)
	t.Cleanup(func() { SetService(nil) })
	ctx := context.Background()

	// Seed two observations in the bucket via the real session_start + save
	// handlers.
	sessA := mcpSessionID(t, ctx)
	mcpSave(t, ctx, map[string]any{
		"title": "federated probe a", "type": "decision",
		"content": "federated shared marker alpha", "session_id": sessA,
	})
	sessB := mcpSessionID(t, ctx)
	mcpSave(t, ctx, map[string]any{
		"title": "federated probe b", "type": "decision",
		"content": "federated shared marker beta", "session_id": sessB,
	})

	// Federated search (all_projects=true) spans every store and returns both
	// rows, with the federated response contract.
	res, err := handleMemSearch(ctx, newCallTool("mem_search", map[string]any{
		"query": "federated shared marker", "all_projects": true,
	}))
	if err != nil {
		t.Fatalf("all_projects search dispatch: %v", err)
	}
	if res.IsError {
		t.Fatalf("all_projects search errored: %s", callResultText(t, res))
	}
	var out struct {
		Project      string `json:"project"`
		AllProjects  bool   `json:"all_projects"`
		Count        int    `json:"count"`
		Observations []struct {
			Title string `json:"title"`
		} `json:"observations"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("unmarshal all_projects result: %v (text %s)", err, callResultText(t, res))
	}
	if out.Project != "all" || !out.AllProjects {
		t.Fatalf("federated response contract wrong: project=%q all_projects=%v", out.Project, out.AllProjects)
	}
	if out.Count != 2 || len(out.Observations) != 2 {
		t.Fatalf("federated search returned %d hits (count=%d), want both rows (2)", len(out.Observations), out.Count)
	}
	joined := ""
	for _, o := range out.Observations {
		joined += o.Title + "\n"
	}
	if !strings.Contains(joined, "federated probe a") || !strings.Contains(joined, "federated probe b") {
		t.Fatalf("federated search did not return both seeded rows: got %q", joined)
	}

	// Scoped floor: a single-bucket search (all_projects=false, the default)
	// must NOT carry the federated contract — the response carries the bucket's
	// project id, not "all". (The scoped read path applies the per-owner
	// visibility gate, so its hit count is reader-dependent; the contract —
	// not the count — is what the federated seam asserts here.)
	resScoped, err := handleMemSearch(ctx, newCallTool("mem_search", map[string]any{
		"query": "federated shared marker",
	}))
	if err != nil {
		t.Fatalf("scoped search dispatch: %v", err)
	}
	if resScoped.IsError {
		t.Fatalf("scoped search errored: %s", callResultText(t, resScoped))
	}
	var scoped struct {
		Project     string `json:"project"`
		AllProjects bool   `json:"all_projects"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, resScoped)), &scoped); err != nil {
		t.Fatalf("unmarshal scoped result: %v (text %s)", err, callResultText(t, resScoped))
	}
	if scoped.AllProjects {
		t.Fatalf("scoped search must not carry the federated contract (all_projects=true)")
	}
	if scoped.Project != "fed-a" {
		t.Fatalf("scoped search project=%q, want the bucket id fed-a", scoped.Project)
	}
}

// mcpSessionID starts a session through the real handler and returns its id.
func mcpSessionID(t *testing.T, ctx context.Context) string {
	t.Helper()
	res, err := handleMemSessionStart(ctx, newCallTool("mem_session_start", map[string]any{}))
	if err != nil {
		t.Fatalf("mem_session_start dispatch: %v", err)
	}
	if res.IsError {
		t.Fatalf("mem_session_start errored: %s", callResultText(t, res))
	}
	var out struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("unmarshal session start: %v (text %s)", err, callResultText(t, res))
	}
	return out.SessionID
}

// mcpSave drives the real save handler and fails the test on error.
func mcpSave(t *testing.T, ctx context.Context, args map[string]any) {
	t.Helper()
	res, err := handleMemSave(ctx, newCallTool("mem_save", args))
	if err != nil {
		t.Fatalf("mem_save dispatch: %v", err)
	}
	if res.IsError {
		t.Fatalf("mem_save errored: %s", callResultText(t, res))
	}
}
