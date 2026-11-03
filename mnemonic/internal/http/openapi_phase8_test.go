package http

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestPhase8_OpenAPI asserts openapi.yaml documents the Phase 8 decision-bridge
// routes (change 2026-09-19-embed-visual-companion) and the /prototype serve
// route, and that the spec still parses as YAML with no duplicate keys. The
// earlier phases' content is asserted by TestPhase6/7_OpenAPI (substring
// only); this is the first test that also parses the doc, which catches the
// plain-scalar indent/colon bugs the substring checks never see.
func TestPhase8_OpenAPI(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	path := filepath.Join(wd, "ui", "openapi.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read openapi.yaml: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"/mnemonic/decisions:",
		"/mnemonic/decisions/{id}/answer:",
		"/prototype/{id...}:",
		"operationId: mnemonicDecisions",
		"operationId: mnemonicDecisionAnswer",
		"operationId: prototypeDecision",
		"DecisionRow", "DecisionContent", "DecisionOption", "DecisionAnswerInput",
		"mnemonic-decisions",
		"pending", "answered", "superseded", // the state filter values
		"409", "404", "400", // the answer route error semantics
	} {
		if !strings.Contains(text, want) {
			t.Errorf("openapi.yaml missing Phase 8 content: %q", want)
		}
	}

	// Parse the doc and assert no duplicate keys at the three maps the spec
	// is built from (a duplicate silently drops a path/schema from the served
	// spec — the substring checks can't catch it).
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		t.Fatalf("openapi.yaml does not parse: %v", err)
	}
	root := &node
	if node.Kind == yaml.DocumentNode {
		root = node.Content[0]
	}
	assertNoDupKeys(t, "top-level", root)
	for i := 0; i < len(root.Content)-1; i += 2 {
		switch root.Content[i].Value {
		case "paths":
			assertNoDupKeys(t, "paths", root.Content[i+1])
		case "components":
			c := root.Content[i+1]
			for j := 0; j < len(c.Content)-1; j += 2 {
				if c.Content[j].Value == "schemas" {
					assertNoDupKeys(t, "schemas", c.Content[j+1])
				}
			}
		}
	}
}

// assertNoDupKeys fatals on any repeated key in a mapping node.
func assertNoDupKeys(t *testing.T, label string, m *yaml.Node) {
	t.Helper()
	seen := map[string]int{}
	var dups []string
	for i := 0; i < len(m.Content)-1; i += 2 {
		k := m.Content[i].Value
		seen[k]++
	}
	for k, n := range seen {
		if n > 1 {
			dups = append(dups, k)
		}
	}
	if len(dups) > 0 {
		t.Fatalf("openapi.yaml %s has duplicate keys: %v", label, dups)
	}
}
