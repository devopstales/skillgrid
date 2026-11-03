package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

// TestDepIngestDispatchesToStore covers the review verification-gap: a
// successful dep_ingest must actually reach the store (a wrong store or data
// dir wiring ships undetected if only registration + arg-validation are tested).
// Ingest a fixture SBOM, then read it back through dep_list — the round-trip
// proves the handler wires the real service store, not a stub.
func TestDepIngestDispatchesToStore(t *testing.T) {
	SetService(service.New(t.TempDir()))
	t.Cleanup(func() { SetService(nil) })

	sbom := `{"bomFormat":"CycloneDX","specVersion":"1.5","components":[{"type":"application","bom-ref":"pkg:pypi/app@1.0.0","purl":"pkg:pypi/app@1.0.0","name":"app","version":"1.0.0"},{"type":"library","bom-ref":"pkg:pypi/flask@3.0.2","purl":"pkg:pypi/flask@3.0.2","name":"flask","version":"3.0.2"}],"dependencies":[{"ref":"pkg:pypi/app@1.0.0","dependsOn":["pkg:pypi/flask@3.0.2"]}]}`

	res, err := handleDepIngest(context.Background(), newCallTool("dep_ingest", map[string]any{
		"sbom": sbom,
	}))
	if err != nil {
		t.Fatalf("handleDepIngest dispatch: %v", err)
	}
	if res.IsError {
		t.Fatalf("dep_ingest should succeed, got: %s", callResultText(t, res))
	}

	// Read it back through the store via dep_list — proves the ingest landed.
	listRes, err := handleDepList(context.Background(), newCallTool("dep_list", map[string]any{
		"retired": false,
	}))
	if err != nil {
		t.Fatalf("handleDepList dispatch: %v", err)
	}
	if listRes.IsError {
		t.Fatalf("dep_list should succeed, got: %s", callResultText(t, listRes))
	}
	text := callResultText(t, listRes)
	if !strings.Contains(text, "pkg:pypi/flask@3.0.2") {
		t.Errorf("dep_list did not return the ingested flask purl; handler did not reach the store. got: %s", text)
	}
}

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
