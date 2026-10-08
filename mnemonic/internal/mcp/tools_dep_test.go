package mcp

import (
	"context"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

// TestDepToolsRegistered covers the acceptance scenario "happy path scan and
// dep tools are registered": all six dep_* thin-adapter tools register on the
// live server.
func TestDepToolsRegistered(t *testing.T) {
	SetService(service.New(t.TempDir()))
	t.Cleanup(func() { SetService(nil) })

	tools := NewServer().ListTools()
	for _, name := range []string{
		"dep_ingest", "dep_list", "dep_get",
		"dep_affected", "dep_graph", "dep_runtime",
	} {
		if _, ok := tools[name]; !ok {
			t.Errorf("expected dep tool %q to be registered", name)
		}
	}

	if _, ok := tools["mem_search"]; !ok {
		t.Error("pre-existing mem_search tool no longer registered (additive contract broken)")
	}
}

// TestDepIngestValidation covers bad-arg rejection on the dep adapters:
// dep_ingest with a missing sbom is a validation error, not an empty ingest.
func TestDepIngestValidation(t *testing.T) {
	SetService(service.New(t.TempDir()))
	t.Cleanup(func() { SetService(nil) })

	res, err := handleDepIngest(context.Background(), newCallTool("dep_ingest", map[string]any{}))
	if err != nil {
		t.Fatalf("handleDepIngest dispatch: %v", err)
	}
	if !res.IsError {
		t.Errorf("dep_ingest without sbom should be a validation error, got: %s", callResultText(t, res))
	}
}
