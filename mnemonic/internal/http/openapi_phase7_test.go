package http

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPhase7_OpenAPI asserts the openapi.yaml documents the prototype routes
// (the feasibility-prototype listing at /prototypes, which replaced the Phase 7
// .stitch/ gallery, and the decision-companion /prototype/{id...} serve route)
// and that the doc still parses. The Phase 6 paths are asserted by
// TestPhase6_OpenAPI; this is additive.
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
	// the prototype paths + examples are present.)
	for _, want := range []string{
		"/prototypes:",
		"operationId: prototypesList",
		".skillgrid/prototypes",
		"/prototype/{id...}:",
		"operationId: prototypeDecision",
		"text/html",
		"example",
		"400", // traversal/absolute id rejection
	} {
		if !contains(string(data), want) {
			t.Errorf("openapi.yaml missing prototype content: %q", want)
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
