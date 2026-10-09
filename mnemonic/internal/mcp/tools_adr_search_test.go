package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/codeindex"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

var testAdrNumRe = regexp.MustCompile(`^04-adr-(\d+)`)

const adrFixture0001 = `# Locked decisions are ADR files

---
status: "accepted"
supersedes: none
date: 2026-10-02
---

## Context

One file per locked decision.
`

const adrFixture0002 = `# Superseding decision for ADR 0001

---
status: "accepted"
supersedes: 0001
date: 2026-10-05
---

## Context

This supersedes the earlier decision.
`

// adrMCPFixture indexes a project whose .skillgrid/artifacts/ holds two ADRs
// and pins the project so the temp-dir fixture resolves to one stable bucket.
func adrMCPFixture(t *testing.T) (dataDir, root string) {
	t.Helper()
	dataDir = t.TempDir()
	raw := t.TempDir()
	abs, err := filepath.Abs(raw)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	root = abs

	artifacts := filepath.Join(root, ".skillgrid", "artifacts")
	if err := os.MkdirAll(artifacts, 0o755); err != nil {
		t.Fatalf("mkdir artifacts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(artifacts, "04-adr-0001-decisions-are-files.md"), []byte(adrFixture0001), 0o644); err != nil {
		t.Fatalf("write adr 0001: %v", err)
	}
	if err := os.WriteFile(filepath.Join(artifacts, "04-adr-0002-superseding.md"), []byte(adrFixture0002), 0o644); err != nil {
		t.Fatalf("write adr 0002: %v", err)
	}

	t.Setenv("MNEMONIC_PROJECT", "adrsearch-probe")
	svc := service.New(dataDir)
	SetService(svc)
	t.Cleanup(func() { SetService(nil) })
	codeindex.ResetFileFirstSymbol()
	if _, err := svc.RunCodeIndex(context.Background(), root); err != nil {
		t.Fatalf("index: %v", err)
	}
	oldDir, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(oldDir) })
	return dataDir, root
}

// TestAdrSearchTool covers @step-02 (adr_search returns structured decision
// metadata): the MCP tool returns path, title, status, date, supersedes, and
// flags a superseded ADR with its successor's path.
func TestAdrSearchTool(t *testing.T) {
	adrMCPFixture(t)
	res, err := handleAdrSearch(context.Background(), newCallTool("adr_search", map[string]any{"query": "decision"}))
	if err != nil {
		t.Fatalf("handleAdrSearch: %v", err)
	}
	text := callResultText(t, res)
	var out struct {
		Hits []struct {
			Path         string  `json:"path"`
			Title        string  `json:"title"`
			Status       string  `json:"status"`
			Date         string  `json:"date"`
			Supersedes   string  `json:"supersedes"`
			SupersededBy string  `json:"superseded_by"`
			Score        float64 `json:"score"`
		} `json:"hits"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal adr_search: %v (text %s)", err, text)
	}
	if len(out.Hits) != 2 {
		t.Fatalf("expected 2 ADR hits, got %d: %s", len(out.Hits), text)
	}
	byNum := map[string]int{}
	for i, h := range out.Hits {
		if m := testAdrNumRe.FindStringSubmatch(filepath.Base(h.Path)); m != nil {
			byNum[m[1]] = i
		}
	}
	// 0001 must be flagged as superseded by 0002.
	i1, ok1 := byNum["0001"]
	i2, ok2 := byNum["0002"]
	if !ok1 || !ok2 {
		t.Fatalf("expected both 0001 and 0002, got %s", text)
	}
	if out.Hits[i1].SupersededBy == "" {
		t.Errorf("0001 must be flagged superseded, got empty (text %s)", text)
	}
	if out.Hits[i1].SupersededBy != ".skillgrid/artifacts/04-adr-0002-superseding.md" {
		t.Errorf("0001 superseded_by = %q, want the 0002 path", out.Hits[i1].SupersededBy)
	}
	if out.Hits[i2].Supersedes != "0001" {
		t.Errorf("0002 supersedes = %q, want 0001", out.Hits[i2].Supersedes)
	}
	if out.Hits[i1].Status != "accepted" {
		t.Errorf("0001 status = %q, want accepted", out.Hits[i1].Status)
	}
}

// TestAdrSearchToolRequiredParams covers @step-02 (005 tool NAMES and
// REQUIRED params must NOT change — the new tool keeps its exact required-param
// contract).
func TestAdrSearchToolRequiredParams(t *testing.T) {
	assertCodeToolStable(t, adrSearchTool(), "adr_search", []string{"query"})
}

// TestAdrSearchToolNoMatch covers @step-02 (empty result on a query that
// matches no ADR): the tool returns an empty hits array, not an error.
func TestAdrSearchToolNoMatch(t *testing.T) {
	adrMCPFixture(t)
	res, err := handleAdrSearch(context.Background(), newCallTool("adr_search", map[string]any{"query": "zzzznotfoundqqqq"}))
	if err != nil {
		t.Fatalf("handleAdrSearch: %v", err)
	}
	text := callResultText(t, res)
	var out struct {
		Hits []struct {
			Path string `json:"path"`
		} `json:"hits"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal adr_search: %v (text %s)", err, text)
	}
	if len(out.Hits) != 0 {
		t.Errorf("expected 0 hits for a non-matching query, got %d: %s", len(out.Hits), text)
	}
}
