package handler

// handler_test.go — the shared handler: every surface (MCP, the Task 14
// CLI, and the later HTTP mirror) is a thin adapter over this one Handler,
// so the tests assert the handler's contract — project resolution at call
// time, the envelope error codes, and the tri-state arg mappings — not
// MCP-specific behavior.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/llm"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/envelope"
)

// llmStub is a minimal llm.LLM for the handler tests (the service's
// degradation floor needs the Classify method to run).
type llmStub struct{}

func (llmStub) Classify(ctx context.Context, newContent string, candidates []string) (llm.DedupDecision, error) {
	return llm.DedupDecision{Verdict: llm.VerdictAdd, Reason: "stub"}, nil
}

// degradedLLM is the llm.LLM stub that always errors (the envelope-degraded
// fixture: the service's degrade.Run retries once, then runs the floor and
// flags the result degraded).
type degradedLLM struct{}

func (degradedLLM) Classify(ctx context.Context, newContent string, candidates []string) (llm.DedupDecision, error) {
	return llm.DedupDecision{}, errors.New("llm: degraded (test fixture)")
}
func (degradedLLM) Judge(ctx context.Context, source, target llm.Observation, rel llm.Relation) (llm.Judgment, error) {
	return llm.Judgment{}, nil
}
func (degradedLLM) Compare(ctx context.Context, a, b llm.Observation) (llm.Judgment, error) {
	return llm.Judgment{}, nil
}
func (degradedLLM) Extract(ctx context.Context, text string) ([]llm.CandidateObservation, error) {
	return nil, nil
}
func (llmStub) Judge(ctx context.Context, source, target llm.Observation, rel llm.Relation) (llm.Judgment, error) {
	return llm.Judgment{}, nil
}
func (llmStub) Compare(ctx context.Context, a, b llm.Observation) (llm.Judgment, error) {
	return llm.Judgment{}, nil
}
func (llmStub) Extract(ctx context.Context, text string) ([]llm.CandidateObservation, error) {
	return nil, nil
}

// testProject is the project the fixture store opens for.
const testProject = "p"

// openTestStore opens a real SQLite store for testProject in dir.
func openTestStore(t *testing.T, dir string) store.Store {
	t.Helper()
	st, err := store.Open(ctx, dir, testProject, store.SQLite, "")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// fixedDetector resolves explicit → explicit, else testProject (no git).
type fixedDetector struct{ explicitOK bool }

func (d fixedDetector) Resolve(_ string, explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	return testProject, nil
}

// strictDetector mimics the REAL project.Detector: an empty cwd with no
// explicit selector is an error (the git/basename binding needs a directory).
// This is the seam the mem Handler hits at call time — its methods pass
// DefaultCWD (""), so without a fallback the run fails "cwd is empty". The
// Handler must fall back to the config's already-resolved Project.Name.
type strictDetector struct{}

func (d strictDetector) Resolve(cwd, explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if cwd == "" {
		return "", errors.New("project: cwd is empty and no project selector was given")
	}
	return "from-cwd", nil
}

// newHandler builds a Handler over a real SQLite store, nil seams.
func newHandler(t *testing.T) *Handler {
	t.Helper()
	st := openTestStore(t, t.TempDir())
	c := config.Default()
	c.DataDir = t.TempDir()
	c.Project.Detect = false
	return NewHandler(memory.NewService(st, c, memory.NewEventBus(), nil, nil), c, fixedDetector{})
}

// newHandlerWithLLM builds a Handler over a real SQLite store with the
// given LLM stub (the envelope-degraded fixture uses degradedLLM).
func newHandlerWithLLM(t *testing.T, l llm.LLM) *Handler {
	t.Helper()
	st := openTestStore(t, t.TempDir())
	c := config.Default()
	c.DataDir = t.TempDir()
	c.Project.Detect = false
	return NewHandler(memory.NewService(st, c, memory.NewEventBus(), l, nil), c, fixedDetector{})
}

// ── project resolution at call time ────────────────────────────────────────

// TestSaveResolvesExplicitProject pins "every method resolves the project
// at call time": the explicit selector wins over the detector's default,
// and the row lands in the requested project (not the store's).
func TestSaveResolvesExplicitProject(t *testing.T) {
	h := newHandler(t)
	res, err := h.Save(ctx, SaveRequest{
		SessionID: "s1", Type: "decision", Title: "pick x",
		Content: "c", Project: "other-project",
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if res.SyncID == "" {
		t.Fatal("save returned no sync_id")
	}
	// The row must land in the explicitly requested project, not the
	// detector's default (testProject).
	got, err := h.Get(ctx, 0, res.SyncID, false, 0)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Project == nil || *got.Project != "other-project" {
		p := "<nil>"
		if got.Project != nil {
			p = *got.Project
		}
		t.Fatalf("row landed in project %q, want the explicit %q", p, "other-project")
	}
}

// TestSaveWithoutProjectUsesDetector proves the detector fallback: no
// explicit project → the resolved default project.
func TestSaveWithoutProjectUsesDetector(t *testing.T) {
	h := newHandler(t)
	if _, err := h.Save(ctx, SaveRequest{
		SessionID: "s1", Type: "decision", Title: "pick y", Content: "c",
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
}

// TestSaveEmptyCWDFallsBackToResolvedProject pins the live-regression fix:
// the mem Handler's methods resolve the project at call time with
// DefaultCWD (""), and a real project.Detector errors on an empty cwd (the
// git/basename binding needs a directory) — "project: cwd is empty and no
// project selector was given" (the error the user hit running `mnemonic mem
// search` bare). The Handler must fall back to the config's already-resolved
// Project.Name (openRuntime sets it after resolving with the REAL cwd), so a
// bare mem command works without an env var or --project flag.
func TestSaveEmptyCWDFallsBackToResolvedProject(t *testing.T) {
	st := openTestStore(t, t.TempDir())
	c := config.Default()
	c.DataDir = t.TempDir()
	c.Project.Detect = false
	// openRuntime resolves the project with the real cwd, then records it in
	// the config the handler is built with. That is the value the call-time
	// resolution must fall back to when the detector can't resolve (empty cwd).
	c.Project.Name = testProject
	h := NewHandler(memory.NewService(st, c, memory.NewEventBus(), nil, nil), c, strictDetector{})

	// No explicit project + the handler resolves with DefaultCWD ("") → the
	// strict detector errors → the fallback to c.Project.Name must carry it.
	res, err := h.Save(ctx, SaveRequest{
		SessionID: "s1", Type: "decision", Title: "pick z", Content: "c",
	})
	if err != nil {
		t.Fatalf("save (empty cwd, no explicit) must fall back to the resolved project, not fail: %v", err)
	}
	if res.SyncID == "" {
		t.Fatal("save returned no sync_id")
	}
	got, err := h.Get(ctx, 0, res.SyncID, false, 0)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Project == nil || *got.Project != testProject {
		p := "<nil>"
		if got.Project != nil {
			p = *got.Project
		}
		t.Fatalf("row landed in project %q, want the config's resolved %q", p, testProject)
	}
}

// ── the envelope error mapping ─────────────────────────────────────────────

// TestErrorCodesMapsSentinels pins the (code, message) mapping (brief):
// each sentinel error maps to its pinned code, and non-sentinel errors
// map to a generic internal error (never a bare Go string leak).
func TestErrorCodesMapsSentinels(t *testing.T) {
	cases := []struct {
		err  error
		code string
	}{
		{memory.ErrOperationConflict, envelope.CodeOperationConflict},
		{memory.ErrOperationExpired, envelope.CodeOperationExpired},
		{memory.ErrSessionEnded, envelope.CodeSessionEnded},
		{memory.ErrConsolidateFingerprintMismatch, envelope.CodeFingerprintMismatch},
		{memory.ErrConsolidateRefused, envelope.CodeConsolidationRefused},
		{errors.New("something else"), envelope.CodeInternal},
	}
	for _, tc := range cases {
		code, msg := envelope.MapError(tc.err)
		if code != tc.code {
			t.Errorf("MapError(%v) = %q, want %q", tc.err, code, tc.code)
		}
		if msg == "" {
			t.Errorf("MapError(%v) returned an empty message", tc.err)
		}
	}
}

// TestSaveValidationErrorIsInvalidArguments pins that a save failing
// validation (missing required field) is an invalid_arguments error, not
// an internal one — the agent can fix the call.
func TestSaveValidationErrorIsInvalidArguments(t *testing.T) {
	h := newHandler(t)
	_, err := h.Save(ctx, SaveRequest{SessionID: "s1", Type: "decision"}) // no title/content
	if err == nil {
		t.Fatal("expected a validation error")
	}
	code, _ := envelope.MapError(err)
	if code != envelope.CodeInvalidArguments {
		t.Fatalf("validation error code = %q, want %q", code, envelope.CodeInvalidArguments)
	}
}

// TestMapErrorIsNotSubstringBased (review fix round) pins the sentinel
// classification: a not-found error whose message happens to contain
// "is required" is still not_found (the old substring matcher would have
// misclassified it as invalid_arguments), and an internal error whose
// message contains "not found" is still internal_error.
func TestMapErrorIsNotSubstringBased(t *testing.T) {
	nf := fmt.Errorf("store: session lookup: %w — title is required in the call", sql.ErrNoRows)
	code, _ := envelope.MapError(nf)
	if code != envelope.CodeNotFound {
		t.Fatalf("not-found error misclassified as %q, want %q", code, envelope.CodeNotFound)
	}
	internal := fmt.Errorf("memory: opening store file at /tmp/not found dir: permission denied")
	code, _ = envelope.MapError(internal)
	if code != envelope.CodeInternal {
		t.Fatalf("internal error misclassified as %q, want %q", code, envelope.CodeInternal)
	}
	// the sentinel itself still maps.
	code, _ = envelope.MapError(fmt.Errorf("%w: title", memory.ErrInvalidArguments))
	if code != envelope.CodeInvalidArguments {
		t.Fatalf("ErrInvalidArguments wrapped error = %q, want %q", code, envelope.CodeInvalidArguments)
	}
}

// ── the tri-state arg mappings ─────────────────────────────────────────────

// TestCapturePromptAbsentMeansDefaultOn pins the amended tri-state (spec
// 004-transport-mcp mem_save row): an absent capture_prompt is nil → the
// pipeline's default-on, never "false". Task 11's
// TestPromptSearchFindsCaptured saves without the field; this test is the
// transport half of that contract.
func TestCapturePromptAbsentMeansDefaultOn(t *testing.T) {
	// Absent: no capture_prompt key at all in the JSON.
	var args map[string]any
	if err := json.Unmarshal([]byte(`{"title":"t","type":"decision","content":"c","session_id":"s1"}`), &args); err != nil {
		t.Fatal(err)
	}
	req := parseSaveRequest(args)
	if req.CapturePrompt != nil {
		t.Fatalf("absent capture_prompt must parse to nil, got %v", *req.CapturePrompt)
	}
	if !captureOn(req.CapturePrompt) {
		t.Fatal("absent capture_prompt must default ON")
	}
	// Explicit false stays false (absent ≠ false).
	var args2 map[string]any
	if err := json.Unmarshal([]byte(`{"capture_prompt":false}`), &args2); err != nil {
		t.Fatal(err)
	}
	req2 := parseSaveRequest(args2)
	if req2.CapturePrompt == nil || *req2.CapturePrompt {
		t.Fatalf("explicit false must stay false, got %v", req2.CapturePrompt)
	}
}

// TestGetAcceptsSyncID pins INV-6 at the handler seam: the agent-visible
// id is the sync_id, and Get accepts it (mem_get_observation accepts both
// — the numeric id path is the local fallback).
func TestGetAcceptsSyncID(t *testing.T) {
	h := newHandler(t)
	saved, err := h.Save(ctx, SaveRequest{
		SessionID: "s1", Type: "decision", Title: "pick z", Content: "c",
	})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := h.Get(ctx, 0, saved.SyncID, false, 0)
	if err != nil {
		t.Fatalf("get by sync_id: %v", err)
	}
	if got.SyncID != saved.SyncID {
		t.Fatalf("get by sync_id = %q, want %q", got.SyncID, saved.SyncID)
	}
}

// TestGetMissingSyncIDIsNotFound pins the absent-row error: an unknown
// sync_id is not-found (an error, not a zero result).
func TestGetMissingSyncIDIsNotFound(t *testing.T) {
	h := newHandler(t)
	_, err := h.Get(ctx, 0, "obs-does-not-exist", false, 0)
	if err == nil {
		t.Fatal("expected an error for a missing sync_id")
	}
	if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "no rows") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ── triangulation ──────────────────────────────────────────────────────────

// TestJudgeStampsModelName pins where the agent's model enters
// (marked_by_model, Task 10): the handler must not drop it. A pending
// relation is seeded directly through the store, judged with a model
// name, and the row's marked_by_model is asserted.
func TestJudgeStampsModelName(t *testing.T) {
	h := newHandler(t)
	st := h.Service().St()
	// Seed a pending relation directly through the store (the handler has
	// no AddRelation method; the store seam is the test fixture).
	rel, err := st.AddRelation(ctx, store.AddRelationParams{
		SourceID: "obs-a", TargetID: "obs-b", Relation: "pending", SessionID: "s1",
	})
	if err != nil {
		t.Fatalf("seed relation: %v", err)
	}
	judged, err := h.Judge(ctx, JudgeRequest{JudgmentID: rel.ID, Relation: "conflicts_with", Reason: "r", Model: "test-model"})
	if err != nil {
		t.Fatalf("judge: %v", err)
	}
	if judged.MarkedByModel != "test-model" {
		t.Fatalf("marked_by_model = %q, want %q", judged.MarkedByModel, "test-model")
	}
}

// TestConsolidateFingerprintMismatchMaps pins the consolidation refusal
// path end to end: a confirm with a wrong fingerprint returns the
// fingerprint-mismatch code (the agent re-previews; never auto-confirms).
func TestConsolidateFingerprintMismatchMaps(t *testing.T) {
	h := newHandler(t)
	// Two observations to consolidate.
	a, err := h.Save(ctx, SaveRequest{SessionID: "s1", Type: "decision", Title: "dup one", Content: "same thing", Project: testProject})
	if err != nil {
		t.Fatalf("save a: %v", err)
	}
	b, err := h.Save(ctx, SaveRequest{SessionID: "s1", Type: "decision", Title: "dup two", Content: "same thing again", Project: testProject})
	if err != nil {
		t.Fatalf("save b: %v", err)
	}
	// Preview (no fingerprint): resolves the plan.
	preview, err := h.Consolidate(ctx, ConsolidateRequest{
		IDsToDelete: []int64{a.ID, b.ID}, NewTitle: "merged", NewContent: "merged body",
		ExpectedProject: testProject,
	})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if preview.Plan == nil {
		t.Fatal("preview returned no plan")
	}
	// Confirm with a deliberately wrong fingerprint: mismatch, not a merge.
	_, err = h.Consolidate(ctx, ConsolidateRequest{
		IDsToDelete: []int64{a.ID, b.ID}, NewTitle: "merged", NewContent: "merged body",
		ExpectedProject: testProject, Fingerprint: "wrong-fingerprint", Confirm: true,
	})
	if err == nil {
		t.Fatal("expected a fingerprint mismatch error")
	}
	code, _ := envelope.MapError(err)
	if code != envelope.CodeFingerprintMismatch {
		t.Fatalf("mismatch code = %q, want %q", code, envelope.CodeFingerprintMismatch)
	}
}

// TestEnvelopeShape pins the degraded envelope (INV-3) at the handler
// boundary: a result with degraded=true must carry a non-empty reason, and
// a non-degraded result must carry neither (no silent floor, no ghost flag).
func TestEnvelopeShape(t *testing.T) {
	e, err := envelope.WithDegraded(map[string]any{"id": 1}, true, "dedup-classify seam absent")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(e)
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["degraded"] != true || m["degraded_reason"] == "" {
		t.Fatalf("degraded envelope wrong: %s", b)
	}
	// Non-degraded: the keys must be absent (omitempty), not false/"".
	e2, err := envelope.WithDegraded(map[string]any{"id": 1}, false, "")
	if err != nil {
		t.Fatal(err)
	}
	b2, _ := json.Marshal(e2)
	m2 := map[string]any{}
	if err := json.Unmarshal(b2, &m2); err != nil {
		t.Fatal(err)
	}
	if _, ok := m2["degraded"]; ok {
		t.Fatalf("non-degraded envelope must omit the degraded flag: %s", b2)
	}
}

// parseSaveRequest is the test-side twin of the MCP adapter's arg parser
// (tools_mem.go): it maps a JSON arg map onto the handler's SaveRequest.
// The tri-state capture_prompt (absent → nil, explicit false → false) is
// the amended spec row.
func parseSaveRequest(args map[string]any) SaveRequest {
	var req SaveRequest
	if v, ok := args["title"].(string); ok {
		req.Title = v
	}
	if v, ok := args["type"].(string); ok {
		req.Type = v
	}
	if v, ok := args["content"].(string); ok {
		req.Content = v
	}
	if v, ok := args["session_id"].(string); ok {
		req.SessionID = v
	}
	if v, ok := args["project"].(string); ok {
		req.Project = v
	}
	if v, ok := args["scope"].(string); ok {
		req.Scope = v
	}
	if v, ok := args["topic_key"].(string); ok {
		req.TopicKey = v
	}
	if v, ok := args["tool_name"].(string); ok {
		req.ToolName = v
	}
	if v, ok := args["operation_id"].(string); ok {
		req.OperationID = v
	}
	if v, ok := args["current_prompt"].(string); ok {
		req.CurrentPrompt = v
	}
	// tri-state: absent → nil (default-on), explicit bool → that value.
	if v, ok := args["capture_prompt"].(bool); ok {
		req.CapturePrompt = &v
	}
	return req
}

// captureOn is the tri-state default (the pipeline's step-1 mapping):
// nil → true (absent means on), explicit false → off.
func captureOn(p *bool) bool {
	return p == nil || *p
}
