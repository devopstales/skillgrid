package http

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPhase7_OpenAPI asserts the openapi.yaml documents the Phase 7 prototypes
// routes (added in 7.5) and that the doc still parses. The Phase 6 paths are
// asserted by TestPhase6_OpenAPI; this is additive for Phase 7.
func TestPhase7_OpenAPI(t *testing.T) {
	// Find the embedded openapi.yaml (same dir as this test source).
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	path := filepath.Join(wd, "ui", "openapi.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read openapi.yaml: %v", err)
	}
	// (Parsing is exercised by TestPhase6_OpenAPI's yaml.v3 load; here we assert
	// the Phase 7 paths + examples are present.)
	for _, want := range []string{
		"/prototypes:",
		"/prototypes/{id}:",
		"operationId: prototypesList",
		"operationId: prototypeGet",
		"text/html",
		"example",
		"400", // traversal/absolute id rejection
	} {
		if !contains(string(data), want) {
			t.Errorf("openapi.yaml missing Phase 7 content: %q", want)
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
