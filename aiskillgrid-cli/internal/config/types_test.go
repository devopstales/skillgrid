package config

import (
	"testing"
)

func TestLoadToolsYAML(t *testing.T) {
	path := "testdata/tools.yaml"
	cfg, err := LoadToolsYAML(path)
	if err != nil {
		t.Fatalf("LoadToolsYAML failed: %v", err)
	}
	if len(cfg.Agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(cfg.Agents))
	}
	if cfg.Agents[0] != "@kilocode/cli" {
		t.Fatalf("unexpected first agent: %s", cfg.Agents[0])
	}
	if len(cfg.Tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(cfg.Tools))
	}
}

func TestLoadMCPYAML(t *testing.T) {
	path := "testdata/mcp.yaml"
	cfg, err := LoadMCPYAML(path)
	if err != nil {
		t.Fatalf("LoadMCPYAML failed: %v", err)
	}
	srv, ok := cfg.Servers["context7-http"]
	if !ok {
		t.Fatalf("missing context7-http server")
	}
	if srv.Type != "remote" {
		t.Fatalf("expected remote, got %s", srv.Type)
	}
	if srv.URL != "https://mcp.context7.com/mcp" {
		t.Fatalf("unexpected url: %s", srv.URL)
	}
}
