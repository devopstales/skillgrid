package mcp

import (
	"context"
	"encoding/json"
	"testing"
)

// tools_capture_suggest_test.go — TICKET-05 (2026-10-06-second-brain-learnings
// Task 5): the mem_capture_suggest MCP tool surface. The classifier itself is
// covered by internal/secondbrain/capture_test.go; these tests assert the
// tool boundary: suggested:true carries a draft, suggested:false is the
// below-threshold shape, the call NEVER persists a row (idempotent, read-only),
// and a follow-up mem_save with the suggested knowledge_class lands exactly
// one row with that class. SATISFIES @step-05 confidence-capture (G6).

type captureSuggestOut struct {
	Suggested bool `json:"suggested"`
	Draft     *struct {
		Trigger        string  `json:"trigger"`
		Confidence     float64 `json:"confidence"`
		KnowledgeClass string  `json:"knowledge_class"`
	} `json:"draft"`
}

// suggestTool runs the mem_capture_suggest handler and parses its JSON out.
func runCaptureSuggest(t *testing.T, text string) captureSuggestOut {
	t.Helper()
	res, err := handleMemCaptureSuggest(context.Background(), newCallTool("mem_capture_suggest", map[string]any{
		"text": text,
	}))
	if err != nil {
		t.Fatalf("mem_capture_suggest dispatch: %v", err)
	}
	if res.IsError {
		t.Fatalf("mem_capture_suggest returned an error: %s", callResultText(t, res))
	}
	var out captureSuggestOut
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("unmarshal suggest result: %v (text %s)", err, callResultText(t, res))
	}
	return out
}

// countCaptureRows counts the project's non-deleted observation rows.
func countCaptureRows(t *testing.T, project string) int {
	t.Helper()
	svc, err := rootService()
	if err != nil {
		t.Fatalf("rootService: %v", err)
	}
	hh, cleanup, err := svc.OpenAt(project, ".")
	if err != nil {
		t.Fatalf("open %s: %v", project, err)
	}
	defer cleanup()
	var n int
	if err := hh.Store().DB.QueryRow(
		`SELECT COUNT(*) FROM observations WHERE project = ? AND deleted_at IS NULL`, project,
	).Scan(&n); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	return n
}

// TestCaptureSuggest_HighConfidence: "We're switching from Bootstrap to
// Tailwind" → suggested:true with the switching draft (confidence >= 0.75),
// and NO observation row created by the suggest call.
func TestCaptureSuggest_HighConfidence(t *testing.T) {
	sessionHandoffFixture(t)
	const project = "handoffmcp-probe"
	if before := countCaptureRows(t, project); before != 0 {
		t.Fatalf("unexpected pre-existing rows: %d", before)
	}

	out := runCaptureSuggest(t, "We're switching from Bootstrap to Tailwind")
	if !out.Suggested {
		t.Fatal("expected suggested=true")
	}
	if out.Draft == nil || out.Draft.Trigger != "switching" {
		t.Fatalf("draft = %+v, want trigger switching", out.Draft)
	}
	if out.Draft.Confidence < 0.75 {
		t.Errorf("draft confidence = %v, want >= 0.75", out.Draft.Confidence)
	}
	if n := countCaptureRows(t, project); n != 0 {
		t.Errorf("suggest created %d rows, want 0 (never persists)", n)
	}
}

// TestCaptureSuggest_StrongPattern: a strong correction ("don't do that")
// suggests on its own.
func TestCaptureSuggest_StrongPattern(t *testing.T) {
	sessionHandoffFixture(t)
	if out := runCaptureSuggest(t, "don't do that"); !out.Suggested {
		t.Error("strong pattern must suggest")
	}
}

// TestCaptureSuggest_WeakAlone: a bare "no" (no directive word) does not
// suggest.
func TestCaptureSuggest_WeakAlone(t *testing.T) {
	sessionHandoffFixture(t)
	if out := runCaptureSuggest(t, "no"); out.Suggested {
		t.Error("weak pattern alone must not suggest")
	}
}

// TestCaptureSuggest_WeakWithDirective: "no, use pnpm" (weak + directive)
// suggests.
func TestCaptureSuggest_WeakWithDirective(t *testing.T) {
	sessionHandoffFixture(t)
	if out := runCaptureSuggest(t, "no, use pnpm"); !out.Suggested {
		t.Error("weak pattern with a directive word must suggest")
	}
}

// TestCaptureSuggest_Negative: "no worries" suppresses the suggestion.
func TestCaptureSuggest_Negative(t *testing.T) {
	sessionHandoffFixture(t)
	if out := runCaptureSuggest(t, "no worries"); out.Suggested {
		t.Error("negative phrase must suppress the suggestion")
	}
}

// TestCaptureSuggest_BelowThreshold: casual text returns suggested:false and
// creates no row.
func TestCaptureSuggest_BelowThreshold(t *testing.T) {
	sessionHandoffFixture(t)
	const project = "handoffmcp-probe"
	out := runCaptureSuggest(t, "hey what's up")
	if out.Suggested {
		t.Error("expected suggested=false")
	}
	if n := countCaptureRows(t, project); n != 0 {
		t.Errorf("suggest created %d rows, want 0", n)
	}
}

// TestCaptureSuggest_ConfirmCreatesRow: after the suggest, the agent confirms
// with mem_save using the suggested knowledge_class — exactly one
// observation exists, carrying that class.
func TestCaptureSuggest_ConfirmCreatesRow(t *testing.T) {
	sessionHandoffFixture(t)
	const project = "handoffmcp-probe"

	startRes, err := handleMemSessionStart(context.Background(), newCallTool("mem_session_start", nil))
	if err != nil || startRes.IsError {
		t.Fatalf("mem_session_start: %v %s", err, callResultText(t, startRes))
	}
	var startOut struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, startRes)), &startOut); err != nil {
		t.Fatalf("unmarshal session: %v", err)
	}

	out := runCaptureSuggest(t, "We're switching from Bootstrap to Tailwind")
	if !out.Suggested || out.Draft == nil {
		t.Fatal("expected a suggested draft to confirm")
	}

	saveRes, err := handleMemSave(context.Background(), newCallTool("mem_save", map[string]any{
		"title":           "Switch to Tailwind",
		"type":            "decision",
		"content":         "What: switch from Bootstrap to Tailwind.",
		"session_id":      startOut.SessionID,
		"knowledge_class": out.Draft.KnowledgeClass,
	}))
	if err != nil {
		t.Fatalf("mem_save dispatch: %v", err)
	}
	if saveRes.IsError {
		t.Fatalf("mem_save errored: %s", callResultText(t, saveRes))
	}

	if n := countCaptureRows(t, project); n != 1 {
		t.Fatalf("confirm created %d rows, want exactly 1", n)
	}
	svc, err := rootService()
	if err != nil {
		t.Fatalf("rootService: %v", err)
	}
	hh, cleanup, err := svc.OpenAt(project, ".")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cleanup()
	var typ, kc string
	if err := hh.Store().DB.QueryRow(
		`SELECT type, COALESCE(knowledge_class, '') FROM observations
		 WHERE project = ? AND deleted_at IS NULL`, project,
	).Scan(&typ, &kc); err != nil {
		t.Fatalf("read row: %v", err)
	}
	if typ != "decision" {
		t.Errorf("row type = %q, want decision", typ)
	}
	if kc != out.Draft.KnowledgeClass {
		t.Errorf("row knowledge_class = %q, want %q (the suggested one)", kc, out.Draft.KnowledgeClass)
	}
}

// TestCaptureSuggest_Idempotent: repeated suggest calls return the same
// result (trigger, confidence, suggested) and never write — the row count
// stays 0 no matter how often the tool is called.
func TestCaptureSuggest_Idempotent(t *testing.T) {
	sessionHandoffFixture(t)
	const project = "handoffmcp-probe"
	const text = "We're switching from Bootstrap to Tailwind"

	first := runCaptureSuggest(t, text)
	for i := 0; i < 3; i++ {
		again := runCaptureSuggest(t, text)
		if again.Suggested != first.Suggested ||
			(again.Draft != nil) != (first.Draft != nil) {
			t.Fatalf("call %d: suggested drifted (%v vs %v)", i+1, again.Suggested, first.Suggested)
		}
		if again.Draft != nil && first.Draft != nil {
			if again.Draft.Trigger != first.Draft.Trigger || again.Draft.Confidence != first.Draft.Confidence {
				t.Fatalf("call %d: draft drifted: %+v vs %+v", i+1, again.Draft, first.Draft)
			}
		}
	}
	if n := countCaptureRows(t, project); n != 0 {
		t.Errorf("repeated suggests created %d rows, want 0 (never write)", n)
	}
}

// TestCaptureSuggest_MissingText: a missing/blank `text` is a value error
// (the suggest never fails on real input — but it does reject a blank one).
// The tool registration itself is asserted by the tools/list inventory test
// in server_test.go.
func TestCaptureSuggest_MissingText(t *testing.T) {
	for _, args := range []map[string]any{
		{},
		{"text": "   "},
	} {
		res, err := handleMemCaptureSuggest(context.Background(), newCallTool("mem_capture_suggest", args))
		if err != nil {
			t.Fatalf("dispatch (%v): %v", args, err)
		}
		if !res.IsError {
			t.Errorf("blank/missing text (%v) must be a value error", args)
		}
	}
}
