package mcp

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/service"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// TestScanDataDirFallback covers the review finding: when service.DefaultDataDir
// is unavailable, the raw-artifact cache dir must fall back to the store's data
// dir (where the DB lives), NOT the workspace root — raw scans are data-dir
// artifacts (ADR-0030), not repo content.
func TestScanDataDirFallback(t *testing.T) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir, "scan-datadir-test")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	// The store DB lives at <dataDir>/scan-datadir-test.sqlite, so its data dir
	// is the parent of that path — which equals dataDir here.
	if got := scanDataDirFallback("/somewhere/a/workspace", st); filepath.Clean(got) != filepath.Clean(dataDir) {
		t.Errorf("scanDataDirFallback(store) = %q, want the store data dir %q (not the workspace)", got, dataDir)
	}

	// With no store, the workspace root is the last resort.
	if got := scanDataDirFallback("/somewhere/a/workspace", nil); got != "/somewhere/a/workspace" {
		t.Errorf("scanDataDirFallback(nil store) = %q, want the workspace root", got)
	}
}

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

// TestScanReadHandlersDispatchToStore covers the review verification-gap: the
// scan read handlers (scan_status, scan_list) must actually reach the store —
// a wrong store or data-dir wiring ships undetected if only registration and
// arg-validation are tested. They are pure store reads, so this proves the
// handler→store seam without invoking a real scanner.
func TestScanReadHandlersDispatchToStore(t *testing.T) {
	SetService(service.New(t.TempDir()))
	t.Cleanup(func() { SetService(nil) })

	// scan_status: pure store read, must succeed and return a (possibly empty)
	// per-tool map — not an error that would indicate a store-wiring failure.
	statusRes, err := handleScanStatus(context.Background(), newCallTool("scan_status", map[string]any{}))
	if err != nil {
		t.Fatalf("handleScanStatus dispatch: %v", err)
	}
	if statusRes.IsError {
		t.Errorf("scan_status should succeed, got: %s", callResultText(t, statusRes))
	}

	// scan_list: another store read; must succeed with an empty (valid) result.
	listRes, err := handleScanList(context.Background(), newCallTool("scan_list", map[string]any{}))
	if err != nil {
		t.Fatalf("handleScanList dispatch: %v", err)
	}
	if listRes.IsError {
		t.Errorf("scan_list should succeed, got: %s", callResultText(t, listRes))
	}
}
