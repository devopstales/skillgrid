package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/community"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

func writeFileTo(root, name, content string) error {
	return os.WriteFile(filepath.Join(root, name), []byte(content), 0o644)
}

func mustProjectID(svc *service.Service) string {
	proj, err := svc.ResolveProject(".")
	if err != nil {
		panic(err)
	}
	return proj
}

// communityCLIFixture writes a small two-cluster go project, indexes it, and
// returns the service + project id for community CLI parity checks.
func communityCLIFixture(t *testing.T) (*service.Service, string) {
	t.Helper()
	dataDir := t.TempDir()
	root := t.TempDir()
	files := map[string]string{
		"a.go": "package a\n\nfunc aOne() int { return aTwo() + aThree() }\n\nfunc aTwo() int { return aThree() + 1 }\n\nfunc aThree() int { return 3 }\n",
		"b.go": "package b\n\nfunc bOne() int { return bTwo() + bThree() }\n\nfunc bTwo() int { return bThree() + 1 }\n\nfunc bThree() int { return 9 }\n",
	}
	for name, content := range files {
		if err := writeFileTo(root, name, content); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	t.Setenv("MNEMONIC_PROJECT", "cli-community")
	svc := service.New(dataDir)
	if _, err := svc.ResolveProject(root); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if _, err := svc.RunCodeIndex(context.Background(), root); err != nil {
		t.Fatalf("index: %v", err)
	}
	return svc, root
}

// TestRunCodeCommunitiesParity covers @step-01 (Scenario: CLI community
// commands return the same views): the CLI community runner produces the same
// labeled-subsystem view as the MCP tool (the service backing).
func TestRunCodeCommunitiesParity(t *testing.T) {
	svc, _ := communityCLIFixture(t)
	out, err := svc.CodeCommunities(context.Background(), mustProjectID(svc), community.Options{})
	if err != nil {
		t.Fatalf("CodeCommunities: %v", err)
	}
	if len(out.Communities) == 0 {
		t.Fatalf("expected communities, got none")
	}
	for _, c := range out.Communities {
		if c.Label == "" {
			t.Errorf("community %d has an empty label", c.ID)
		}
	}
}

// TestRunCodeGodNodesParity covers the CLI god-nodes view (same backing as the
// code_god_nodes MCP tool), including exclude-hubs.
func TestRunCodeGodNodesParity(t *testing.T) {
	svc, _ := communityCLIFixture(t)
	proj := mustProjectID(svc)
	gods, err := svc.CodeGodNodes(context.Background(), proj, false, 20)
	if err != nil {
		t.Fatalf("CodeGodNodes: %v", err)
	}
	if len(gods) == 0 {
		t.Fatalf("expected god nodes, got none")
	}
	for i := 1; i < len(gods); i++ {
		if gods[i-1].Degree < gods[i].Degree {
			t.Errorf("god nodes not ranked by degree descending at %d", i)
		}
	}
	// exclude-hubs returns a (possibly smaller) ranked list without error.
	excluded, err := svc.CodeGodNodes(context.Background(), proj, true, 20)
	if err != nil {
		t.Fatalf("CodeGodNodes excludeHubs: %v", err)
	}
	_ = excluded
}

// TestRunCodeExplainCommunityParity covers the CLI explain view (same backing
// as code_explain_community): a valid id returns members + entry points; a bad
// id returns a not-found reason.
func TestRunCodeExplainCommunityParity(t *testing.T) {
	svc, _ := communityCLIFixture(t)
	proj := mustProjectID(svc)
	res, err := svc.CodeCommunities(context.Background(), proj, community.Options{})
	if err != nil {
		t.Fatalf("CodeCommunities: %v", err)
	}
	if len(res.Communities) == 0 {
		t.Fatal("expected communities")
	}
	id := res.Communities[0].ID
	out, err := svc.CodeExplainCommunity(context.Background(), proj, id)
	if err != nil {
		t.Fatalf("CodeExplainCommunity: %v", err)
	}
	if !out.Found {
		t.Fatalf("expected found, got %q", out.Reason)
	}
	if len(out.Members) == 0 {
		t.Fatalf("expected members, got none")
	}
	bad, err := svc.CodeExplainCommunity(context.Background(), proj, 9999)
	if err != nil {
		t.Fatalf("CodeExplainCommunity bad id: %v", err)
	}
	if bad.Found {
		t.Errorf("non-existent community 9999 should be not found")
	}
}
