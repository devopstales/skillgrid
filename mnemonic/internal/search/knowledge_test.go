package search

import (
	"testing"
)

// grounded corpus shapes for the quote-block metadata parser. Backticks that
// wrap inline code in the real corpus are dropped here (raw-string literals
// cannot nest backticks); the parser's key/value extraction is unaffected.
const (
	reportHead = `# Report — Context Harness CLM (PR 3)

> **Change:** .skillgrid/specs/2026-10-06-context-harness-clm/
> **Generated:** 2026-10-09T10:47:30Z (qa)
> **Gate:** PASS

## Test Plan

The CLM layer is opt-in and fail-open; a render failure must not 500 the chat.
`

	briefingHead = `# skill-pi — Design Briefing (requirements & intent)

> **STATUS:** draft (2026-10-09)

## Problem / Intent

skillgrid ships per-harness agent customization but Pi has no presence.
`

	blueprintHead = `# skill-pi — Implementation Blueprint (steps, files, verification)

> **Status:** PROPOSED

> **Tier:** T2

**Goal:** Ship skill-pi, an opinionated non-fork distribution of Pi.
`

	findingsHead = `# Findings — skill-pi (customized Pi agent for skillgrid)

> **Type:** technical
> **Mode:** research
> **Date:** 2026-10-09
> **Status:** complete

## Executive Summary

skill-pi is an opinionated version of pi-coding-agent, not a fork.
`
)

func TestQuoteBlockMetaReport(t *testing.T) {
	m := extractReportMeta(reportHead)
	if m["gate"] != "PASS" {
		t.Errorf("gate = %q, want PASS", m["gate"])
	}
	if m["generated"] != "2026-10-09T10:47:30Z" {
		t.Errorf("generated = %q, want 2026-10-09T10:47:30Z (qa tag stripped)", m["generated"])
	}
	// The change value is the spec path; the trailing " (qa)" source tag on
	// Generated is stripped, the path tail is kept.
	if want := ".skillgrid/specs/2026-10-06-context-harness-clm/"; m["change"] != want {
		t.Errorf("change = %q, want %q", m["change"], want)
	}
	if _, ok := m["type"]; ok {
		t.Errorf("report meta should not carry type, got %v", m)
	}
}

func TestQuoteBlockMetaBriefing(t *testing.T) {
	m := extractBriefingMeta(briefingHead)
	// The status value keeps the full "(2026-10-09)" tail — the date is not a
	// separate key in the corpus, so the parser preserves it on the status line
	// rather than inventing a date key.
	if m["status"] != "draft (2026-10-09)" {
		t.Errorf("status = %q, want draft (2026-10-09)", m["status"])
	}
	if _, ok := m["date"]; ok {
		t.Errorf("briefing meta should not carry a separate date key, got %v", m)
	}
}

func TestQuoteBlockMetaBlueprint(t *testing.T) {
	m := extractBlueprintMeta(blueprintHead)
	if m["status"] != "PROPOSED" {
		t.Errorf("status = %q, want PROPOSED", m["status"])
	}
	if m["tier"] != "T2" {
		t.Errorf("tier = %q, want T2", m["tier"])
	}
}

func TestQuoteBlockMetaFindings(t *testing.T) {
	m := extractFindingsMeta(findingsHead)
	if m["type"] != "technical" {
		t.Errorf("type = %q", m["type"])
	}
	if m["mode"] != "research" {
		t.Errorf("mode = %q", m["mode"])
	}
	if m["date"] != "2026-10-09" {
		t.Errorf("date = %q", m["date"])
	}
	if m["status"] != "complete" {
		t.Errorf("status = %q", m["status"])
	}
}

func TestQuoteBlockMetaEmpty(t *testing.T) {
	if m := extractReportMeta("# Just a title\n\nNo metadata here."); len(m) != 0 {
		t.Errorf("expected empty meta, got %v", m)
	}
}

// path-filter + dedupe: only the target base matches, other types are ignored.
func TestKnowledgeSearchFiltersByType(t *testing.T) {
	// All four docs share the identical phrase "CLM mirror fidelity layer" so
	// the phrase-FTS query matches every one; the type filter then decides
	// which single doc each *_Search returns.
	phrase := "The CLM mirror fidelity layer is the source of truth for the diff."
	files := map[string]string{
		".skillgrid/archive/2026-10-06-ctx/report.md":          phrase + " The gate is PASS.",
		".skillgrid/specs/2026-10-09-skillpi/briefing.md":      phrase + " This is the briefing.",
		".skillgrid/specs/2026-10-09-skillpi/blueprint.md":     phrase + " This is the blueprint.",
		".skillgrid/specs/2026-10-09-skillpi/findings.md":      phrase + " These are the findings.",
	}
	db := newTestDB(t, files)

	// report_search must return ONLY the report.md hit, even though all four
	// docs mention the same phrase.
	hits, err := ReportSearch(db, "CLM mirror fidelity layer", 10)
	if err != nil {
		t.Fatalf("ReportSearch: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("ReportSearch len = %d, want 1; got %v", len(hits), hits)
	}
	if hits[0].Path != ".skillgrid/archive/2026-10-06-ctx/report.md" {
		t.Errorf("path = %q, want the report.md", hits[0].Path)
	}

	briefings, err := BriefingSearch(db, "CLM mirror fidelity layer", 10)
	if err != nil {
		t.Fatalf("BriefingSearch: %v", err)
	}
	if len(briefings) != 1 || briefings[0].Path != ".skillgrid/specs/2026-10-09-skillpi/briefing.md" {
		t.Errorf("BriefingSearch = %v", briefings)
	}

	findings, err := FindingsSearch(db, "CLM mirror fidelity layer", 10)
	if err != nil {
		t.Fatalf("FindingsSearch: %v", err)
	}
	if len(findings) != 1 || findings[0].Path != ".skillgrid/specs/2026-10-09-skillpi/findings.md" {
		t.Errorf("FindingsSearch = %v", findings)
	}

	// The non-matching type must not leak in: blueprint_search on this corpus
	// returns the blueprint.md, not the report.md.
	bps, err := BlueprintSearch(db, "CLM mirror fidelity layer", 10)
	if err != nil {
		t.Fatalf("BlueprintSearch: %v", err)
	}
	if len(bps) != 1 || bps[0].Path != ".skillgrid/specs/2026-10-09-skillpi/blueprint.md" {
		t.Errorf("BlueprintSearch = %v", bps)
	}
}

func TestKnowledgeSearchEmptyQuery(t *testing.T) {
	db := newTestDB(t, map[string]string{
		".skillgrid/archive/x/report.md": "some content about the mirror fidelity layer",
	})
	hits, err := ReportSearch(db, "  ", 10)
	if err != nil {
		t.Fatalf("empty query: %v", err)
	}
	if hits != nil {
		t.Errorf("empty query should return nil, got %v", hits)
	}
}

func TestKnowledgeSearchNoMatch(t *testing.T) {
	db := newTestDB(t, map[string]string{
		".skillgrid/archive/x/report.md": "the CLM mirror fidelity layer is fail-open",
	})
	hits, err := ReportSearch(db, "zebra quantum flux", 10)
	if err != nil {
		t.Fatalf("no match: %v", err)
	}
	if len(hits) != 0 {
		t.Errorf("no match should be empty, got %v", hits)
	}
}
