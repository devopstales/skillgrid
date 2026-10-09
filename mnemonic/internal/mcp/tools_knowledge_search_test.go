package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/codeindex"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

// All four knowledge docs share the identical phrase "CLM mirror fidelity
// layer" so the phrase-FTS query matches every one; each *_Search tool then
// returns only its own type's doc.
const (
	knowledgePhrase = "The CLM mirror fidelity layer is the source of truth for the diff."

	reportFixture = `# Report — CLM

> **Change:** .skillgrid/specs/2026-10-06-clm/
> **Generated:** 2026-10-09T10:47:30Z (qa)
> **Gate:** PASS

` + knowledgePhrase

	briefingFixture = `# CLM — Design Briefing

> **STATUS:** draft (2026-10-09)

` + knowledgePhrase

	blueprintFixture = `# CLM — Implementation Blueprint

> **Status:** PROPOSED

> **Tier:** T2

` + knowledgePhrase

	findingsFixture = `# CLM — Findings

> **Type:** technical
> **Mode:** research
> **Date:** 2026-10-09
> **Status:** complete

` + knowledgePhrase
)

// knowledgeSearchFixture indexes a project whose .skillgrid/specs/<change>/ holds
// one of each knowledge doc (plus a report.md in archive) and pins the project
// so the temp-dir fixture resolves to one stable bucket.
func knowledgeSearchFixture(t *testing.T) (dataDir, root string) {
	t.Helper()
	dataDir = t.TempDir()
	raw := t.TempDir()
	abs, err := filepath.Abs(raw)
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	root = abs

	specs := filepath.Join(root, ".skillgrid", "specs", "2026-10-06-clm")
	if err := os.MkdirAll(specs, 0o755); err != nil {
		t.Fatalf("mkdir specs: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".skillgrid", "artifacts"), 0o755); err != nil {
		t.Fatalf("mkdir artifacts: %v", err)
	}
	for name, body := range map[string]string{
		"briefing.md":  briefingFixture,
		"blueprint.md": blueprintFixture,
		"findings.md":  findingsFixture,
		"report.md":    reportFixture,
	} {
		if err := os.WriteFile(filepath.Join(specs, name), []byte(body), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	t.Setenv("MNEMONIC_PROJECT", "knowledge-search-probe")
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

func TestReportSearchTool(t *testing.T) {
	knowledgeSearchFixture(t)
	res, err := handleReportSearch(context.Background(), newCallTool("report_search", map[string]any{"query": "CLM mirror fidelity layer"}))
	if err != nil {
		t.Fatalf("handleReportSearch: %v", err)
	}
	text := callResultText(t, res)
	var out struct {
		Hits []struct {
			Path      string            `json:"path"`
			Title     string            `json:"title"`
			Gate      string            `json:"gate"`
			Generated string            `json:"generated"`
			Snippet   string            `json:"snippet"`
			Score     float64           `json:"score"`
			Meta      map[string]string `json:"meta"`
		} `json:"hits"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal report_search: %v (text %s)", err, text)
	}
	if len(out.Hits) != 1 {
		t.Fatalf("expected exactly 1 report hit, got %d: %s", len(out.Hits), text)
	}
	h := out.Hits[0]
	if filepath.Base(h.Path) != "report.md" {
		t.Errorf("path base = %q, want report.md", h.Path)
	}
	if h.Gate != "PASS" {
		t.Errorf("gate = %q, want PASS", h.Gate)
	}
	if h.Generated != "2026-10-09T10:47:30Z" {
		t.Errorf("generated = %q, want 2026-10-09T10:47:30Z (qa tag stripped)", h.Generated)
	}
}

func TestBriefingSearchTool(t *testing.T) {
	knowledgeSearchFixture(t)
	res, err := handleBriefingSearch(context.Background(), newCallTool("briefing_search", map[string]any{"query": "CLM mirror fidelity layer"}))
	if err != nil {
		t.Fatalf("handleBriefingSearch: %v", err)
	}
	text := callResultText(t, res)
	var out struct {
		Hits []struct {
			Path   string `json:"path"`
			Status string `json:"status"`
		} `json:"hits"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal briefing_search: %v (text %s)", err, text)
	}
	if len(out.Hits) != 1 || filepath.Base(out.Hits[0].Path) != "briefing.md" {
		t.Fatalf("expected exactly 1 briefing hit, got %s", text)
	}
	if out.Hits[0].Status != "draft (2026-10-09)" {
		t.Errorf("status = %q, want draft (2026-10-09)", out.Hits[0].Status)
	}
}

func TestBlueprintSearchTool(t *testing.T) {
	knowledgeSearchFixture(t)
	res, err := handleBlueprintSearch(context.Background(), newCallTool("blueprint_search", map[string]any{"query": "CLM mirror fidelity layer"}))
	if err != nil {
		t.Fatalf("handleBlueprintSearch: %v", err)
	}
	text := callResultText(t, res)
	var out struct {
		Hits []struct {
			Path   string `json:"path"`
			Status string `json:"status"`
			Tier   string `json:"tier"`
		} `json:"hits"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal blueprint_search: %v (text %s)", err, text)
	}
	if len(out.Hits) != 1 || filepath.Base(out.Hits[0].Path) != "blueprint.md" {
		t.Fatalf("expected exactly 1 blueprint hit, got %s", text)
	}
	if out.Hits[0].Status != "PROPOSED" {
		t.Errorf("status = %q, want PROPOSED", out.Hits[0].Status)
	}
	if out.Hits[0].Tier != "T2" {
		t.Errorf("tier = %q, want T2", out.Hits[0].Tier)
	}
}

func TestFindingsSearchTool(t *testing.T) {
	knowledgeSearchFixture(t)
	res, err := handleFindingsSearch(context.Background(), newCallTool("findings_search", map[string]any{"query": "CLM mirror fidelity layer"}))
	if err != nil {
		t.Fatalf("handleFindingsSearch: %v", err)
	}
	text := callResultText(t, res)
	var out struct {
		Hits []struct {
			Path   string `json:"path"`
			Type   string `json:"type"`
			Mode   string `json:"mode"`
			Date   string `json:"date"`
			Status string `json:"status"`
		} `json:"hits"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal findings_search: %v (text %s)", err, text)
	}
	if len(out.Hits) != 1 || filepath.Base(out.Hits[0].Path) != "findings.md" {
		t.Fatalf("expected exactly 1 findings hit, got %s", text)
	}
	h := out.Hits[0]
	if h.Type != "technical" || h.Mode != "research" || h.Date != "2026-10-09" || h.Status != "complete" {
		t.Errorf("findings meta = %+v, want technical/research/2026-10-09/complete", h)
	}
}

// TestKnowledgeSearchToolRequiredParams locks the required-param contract for
// all four tools: each requires exactly "query".
func TestKnowledgeSearchToolRequiredParams(t *testing.T) {
	assertCodeToolStable(t, reportSearchTool(), "report_search", []string{"query"})
	assertCodeToolStable(t, briefingSearchTool(), "briefing_search", []string{"query"})
	assertCodeToolStable(t, blueprintSearchTool(), "blueprint_search", []string{"query"})
	assertCodeToolStable(t, findingsSearchTool(), "findings_search", []string{"query"})
}

// TestKnowledgeSearchToolNoMatch: a non-matching query returns an empty hits
// array, not an error.
func TestKnowledgeSearchToolNoMatch(t *testing.T) {
	knowledgeSearchFixture(t)
	res, err := handleReportSearch(context.Background(), newCallTool("report_search", map[string]any{"query": "zzzznotfoundqqqq"}))
	if err != nil {
		t.Fatalf("handleReportSearch: %v", err)
	}
	text := callResultText(t, res)
	var out struct {
		Hits []struct {
			Path string `json:"path"`
		} `json:"hits"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("unmarshal report_search: %v (text %s)", err, text)
	}
	if len(out.Hits) != 0 {
		t.Errorf("expected 0 hits for a non-matching query, got %d: %s", len(out.Hits), text)
	}
}
