package http

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestObservability_OpenAPI pins the agent-observability routes (tool
// timeline, query, stats, usage, policy) in the served spec.
func TestObservability_OpenAPI(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("ui", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Paths      map[string]map[string]any `yaml:"paths"`
		Components struct {
			Schemas    map[string]any `yaml:"schemas"`
			Parameters map[string]any `yaml:"parameters"`
		} `yaml:"components"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("openapi.yaml does not parse: %v", err)
	}
	for path, method := range map[string]string{
		"/sessions/{id}/tool-calls": "post",
		"/sessions/{id}/events":     "get",
		"/sessions/{id}/usage":      "post",
		"/events":                   "get",
		"/events/stats":             "get",
		"/policy/evaluate":          "post",
		"/policy":                   "get",
	} {
		if _, ok := doc.Paths[path][method]; !ok {
			t.Errorf("openapi.yaml missing %s %s", method, path)
		}
	}
	if doc.Components.Schemas["ToolEvent"] == nil {
		t.Error("openapi.yaml missing ToolEvent schema")
	}
	for _, p := range []string{"ToolAgent", "ToolAction", "ToolName", "ToolFile", "ToolCommand", "ToolSince"} {
		if doc.Components.Parameters[p] == nil {
			t.Errorf("openapi.yaml missing parameter %s", p)
		}
	}
}
