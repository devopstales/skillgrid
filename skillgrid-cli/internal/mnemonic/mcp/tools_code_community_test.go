package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// communityMCPFixture indexes a two-cluster go project (a dense A cluster and
// a dense B cluster linked by one import) and pins the project so the temp-dir
// fixture (inside the skillgrid git repo) resolves to one stable bucket for
// both the index and the query.
func communityMCPFixture(t *testing.T) string {
	t.Helper()
	dataDir := t.TempDir()
	root := t.TempDir()
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(abs, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	// Cluster A: three functions that call each other densely.
	write("alpha.go", "package a\n\nfunc aOne() int { return aTwo() + aThree() }\n\nfunc aTwo() int { return aThree() + 1 }\n\nfunc aThree() int { return 3 }\n")
	// Cluster B: three functions that call each other densely.
	write("beta.go", "package b\n\nfunc bOne() int { return bTwo() + bThree() }\n\nfunc bTwo() int { return bThree() + 1 }\n\nfunc bThree() int { return 9 }\n")
	// One weak bridge: bOne calls aOne.
	write("bridge.go", "package b\n\nfunc bridge() int { return aOne() + bOne() }\n")

	t.Setenv("MNEMONIC_PROJECT", "community-probe")
	svc := service.New(dataDir)
	SetService(svc)
	t.Cleanup(func() { SetService(nil) })
	codeindex.ResetFileFirstSymbol()
	if _, err := svc.RunCodeIndex(context.Background(), abs); err != nil {
		t.Fatalf("index: %v", err)
	}
	oldDir, _ := os.Getwd()
	if err := os.Chdir(abs); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(oldDir) })
	return dataDir
}

// TestCodeCommunitiesReturnsLabeledSubsystems covers @step-01 (Scenario:
// code_communities returns labeled subsystems and 005 tools stay stable): the
// tool runs the community pass over the indexed graph and returns the
// Leiden-clustered subsystems, each carrying an LLM-free label; the 005 code_*
// tools remain registered with their original names.
func TestCodeCommunitiesReturnsLabeledSubsystems(t *testing.T) {
	communityMCPFixture(t)

	res, err := handleCodeCommunities(context.Background(), newCallTool("code_communities", map[string]any{}))
	if err != nil {
		t.Fatalf("handleCodeCommunities: %v", err)
	}
	if res.IsError {
		t.Fatalf("code_communities errored: %s", callResultText(t, res))
	}
	text := callResultText(t, res)
	var out struct {
		Communities []struct {
			ID      int      `json:"id"`
			Label   string   `json:"label"`
			Members []int64  `json:"members"`
			GodNode []string `json:"god_nodes"`
		} `json:"communities"`
		CacheKey string `json:"cache_key"`
		Warning  string `json:"warning"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("code_communities output not JSON: %v (text %s)", err, text)
	}
	if len(out.Communities) < 2 {
		t.Fatalf("expected >=2 subsystems, got %d (text %s)", len(out.Communities), text)
	}
	for _, c := range out.Communities {
		if c.Label == "" {
			t.Errorf("community %d has empty label", c.ID)
		}
		if len(c.Members) == 0 {
			t.Errorf("community %d has no members", c.ID)
		}
	}
	if out.CacheKey == "" {
		t.Errorf("expected a non-empty content-hash cache key")
	}

	// 005 tools stay stable (baseline lock) — the registration is unchanged.
	tools := NewServer().ListTools()
	for _, name := range []string{"code_search", "code_read", "code_impact", "code_explore"} {
		if _, ok := tools[name]; !ok {
			t.Errorf("005 tool %q no longer registered", name)
		}
	}
}

// TestCommunityToolsArgs covers @step-01 (Scenario: community tools reject bad
// args clearly): the three community tools register under distinct code_*
// names, and a bad/missing arg (e.g. a non-existent community id) is rejected
// with a clear validation error rather than an invented community.
func TestCommunityToolsArgs(t *testing.T) {
	communityMCPFixture(t)

	// Distinct code_* names registered.
	tools := NewServer().ListTools()
	for _, name := range []string{"code_communities", "code_god_nodes", "code_explain_community"} {
		if _, ok := tools[name]; !ok {
			t.Errorf("expected tool %q to be registered", name)
		}
	}

	// code_explain_community with a non-existent community id → clear error.
	res, err := handleCodeExplainCommunity(context.Background(), newCallTool("code_explain_community", map[string]any{"id": 999}))
	if err != nil {
		t.Fatalf("handleCodeExplainCommunity dispatch: %v", err)
	}
	if !res.IsError {
		t.Errorf("code_explain_community with a non-existent id should be a validation error, got: %s", callResultText(t, res))
	}
	if text := callResultText(t, res); !strings.Contains(strings.ToLower(text), "community") {
		t.Errorf("bad-id error should name the community problem, got: %s", text)
	}

	// code_explain_community with a missing id → clear error.
	res2, err := handleCodeExplainCommunity(context.Background(), newCallTool("code_explain_community", map[string]any{}))
	if err != nil {
		t.Fatalf("handleCodeExplainCommunity dispatch: %v", err)
	}
	if !res2.IsError {
		t.Errorf("code_explain_community with a missing id should be a validation error, got: %s", callResultText(t, res2))
	}

	// code_god_nodes with exclude_hubs=true still returns a ranked list (not an
	// error), and excludes the utility super-hubs from the top of the ranking.
	res3, err := handleCodeGodNodes(context.Background(), newCallTool("code_god_nodes", map[string]any{"exclude_hubs": true}))
	if err != nil {
		t.Fatalf("handleCodeGodNodes dispatch: %v", err)
	}
	if res3.IsError {
		t.Errorf("code_god_nodes with exclude_hubs should not error, got: %s", callResultText(t, res3))
	}
	if text := callResultText(t, res3); !strings.Contains(text, "god_nodes") {
		t.Errorf("code_god_nodes should return a god_nodes list, got: %s", text)
	}

	// code_god_nodes with a non-numeric limit (sent as a JSON object) is
	// rejected clearly (no silent default inventing a different view).
	res4, err := handleCodeGodNodes(context.Background(), newCallTool("code_god_nodes", map[string]any{"limit": map[string]any{"n": 1}}))
	if err != nil {
		t.Fatalf("handleCodeGodNodes dispatch: %v", err)
	}
	if !res4.IsError {
		t.Errorf("code_god_nodes with a non-numeric limit should be a validation error, got: %s", callResultText(t, res4))
	}

	// code_communities takes no required args and returns the partition.
	res5, err := handleCodeCommunities(context.Background(), newCallTool("code_communities", map[string]any{}))
	if err != nil {
		t.Fatalf("handleCodeCommunities dispatch: %v", err)
	}
	if res5.IsError {
		t.Errorf("code_communities with no args should not error, got: %s", callResultText(t, res5))
	}
}
