package mcp

import (
	"context"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

// TestScanToolsRegistered covers the acceptance scenario "happy path scan and
// dep tools are registered": all six scan_* thin-adapter tools register on
// the live server.
func TestScanToolsRegistered(t *testing.T) {
	SetService(service.New(t.TempDir()))
	t.Cleanup(func() { SetService(nil) })

	tools := NewServer().ListTools()
	for _, name := range []string{
		"scan_start", "scan_store_findings", "scan_list",
		"scan_get", "scan_status", "scan_diff",
	} {
		if _, ok := tools[name]; !ok {
			t.Errorf("expected scan tool %q to be registered", name)
		}
	}

	if _, ok := tools["mem_save"]; !ok {
		t.Error("pre-existing mem_save tool no longer registered (additive contract broken)")
	}
}

// TestScanStartValidation covers bad-arg rejection on the scan adapters:
// scan_start with a missing target is a validation error, not an invented row.
func TestScanStartValidation(t *testing.T) {
	SetService(service.New(t.TempDir()))
	t.Cleanup(func() { SetService(nil) })

	res, err := handleScanStart(context.Background(), newCallTool("scan_start", map[string]any{
		"tool": "trivy",
	}))
	if err != nil {
		t.Fatalf("handleScanStart dispatch: %v", err)
	}
	if !res.IsError {
		t.Errorf("scan_start without target should be a validation error, got: %s", callResultText(t, res))
	}
}
