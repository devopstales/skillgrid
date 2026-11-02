package mcp

// dispatch.go — the MCP adapter's tool dispatch fns: each maps a JSON arg
// map onto the shared Handler's methods and returns the result (the server
// wraps it in the envelope). The CLI (Task 14) has its own dispatch over
// the same Handler — the K6 byte-identical seam.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/handler"
)

// errProjectMismatch is the (code, message) for the store's ownership
// assertion (the "belongs to project" error — the spec §4 mandatory
// expected_project check).
var errProjectMismatch = errors.New("store: observation belongs to a different project")

// errNotFound is the (code, message) for a missing row (the spec §4:
// not_found).
var errNotFound = errors.New("observation not found")

// argString returns args[key] as a string ("" when absent).
func argString(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return v
}

// argNumber returns args[key] as an int64 (0 when absent).
func argNumber(args map[string]any, key string) int64 {
	switch v := args[key].(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	}
	return 0
}

// argBool returns args[key] as a bool (false when absent).
func argBool(args map[string]any, key string) bool {
	v, _ := args[key].(bool)
	return v
}

// argBoolPtr returns args[key] as a *bool (nil when absent — the tri-state
// mapping: absent → nil → the pipeline's default-on).
func argBoolPtr(args map[string]any, key string) *bool {
	v, ok := args[key].(bool)
	if !ok {
		return nil
	}
	return &v
}

// argStringSlice parses args[key] as a JSON array of strings (nil when
// absent or malformed — the caller validates).
func argStringSlice(args map[string]any, key string) []string {
	v, ok := args[key].(string)
	if !ok {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(v), &out); err != nil {
		return nil
	}
	return out
}

// argInt64Slice parses args[key] as a JSON array of int64s.
func argInt64Slice(args map[string]any, key string) []int64 {
	v, ok := args[key].(string)
	if !ok {
		return nil
	}
	var out []int64
	if err := json.Unmarshal([]byte(v), &out); err != nil {
		return nil
	}
	return out
}

// dispatchSave maps mem_save args → Handler.Save.
func dispatchSave(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		req := handler.SaveRequest{
			SessionID:     argString(args, "session_id"),
			Type:          argString(args, "type"),
			Title:         argString(args, "title"),
			Content:       argString(args, "content"),
			ToolName:      argString(args, "tool_name"),
			Project:       argString(args, "project"),
			Scope:         argString(args, "scope"),
			TopicKey:      argString(args, "topic_key"),
			OperationID:   argString(args, "operation_id"),
			CurrentPrompt: argString(args, "current_prompt"),
			// tri-state: absent → nil → default-on (the amended spec row).
			CapturePrompt: argBoolPtr(args, "capture_prompt"),
		}
		return h.Save(ctx, req)
	}
}

// dispatchSearch maps mem_search args → Handler.Search.
func dispatchSearch(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		f := handler.SearchFilters{
			Type:        argString(args, "type"),
			Project:     argString(args, "project"),
			Scope:       argString(args, "scope"),
			MatchMode:   argString(args, "match_mode"),
			AllProjects: argBool(args, "all_projects"),
		}
		if v := argNumber(args, "limit"); v > 0 {
			f.Limit = int(v)
		}
		return h.Search(ctx, argString(args, "query"), f)
	}
}

// dispatchGet maps mem_get_observation args → Handler.Get.
func dispatchGet(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return h.Get(ctx, argNumber(args, "id"), argString(args, "sync_id"), argBool(args, "include_history"), int(argNumber(args, "history_cursor")))
	}
}

// dispatchLookup maps mem_lookup args → Handler.Lookup.
func dispatchLookup(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		f := handler.LookupFilters{
			ID:       argNumber(args, "id"),
			SyncID:   argString(args, "sync_id"),
			TopicKey: argString(args, "topic_key"),
			Phrase:   argString(args, "phrase"),
			Project:  argString(args, "project"),
			Scope:    argString(args, "scope"),
		}
		return h.Lookup(ctx, f)
	}
}

// dispatchSearchPrompts maps mem_search_prompts args → Handler.SearchPrompts.
func dispatchSearchPrompts(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		limit := int(argNumber(args, "limit"))
		return h.SearchPrompts(ctx, argString(args, "query"), argString(args, "project"), limit)
	}
}

// dispatchContext maps mem_context args → Handler.Context.
func dispatchContext(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		budget := int(argNumber(args, "budget"))
		return h.Context(ctx, budget, argBool(args, "older_history"))
	}
}

// dispatchSavePrompt maps mem_save_prompt args → Handler.SavePrompt.
func dispatchSavePrompt(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		id, err := h.SavePrompt(ctx, argString(args, "session_id"), argString(args, "content"), argString(args, "project"), argString(args, "source_inbox_id"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"id": id}, nil
	}
}

// dispatchCapturePassive maps mem_capture_passive args → the handler. The
// pinned required args (text, session_id) are validated first; the service
// method the memory function does not yet expose (CapturePassive over the
// Extract seam) makes this an honest no-op on the wire (R-2): it carries
// `degraded: true` + a reason, so the agent knows it is not a success.
func dispatchCapturePassive(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		_ = h
		_ = ctx
		if argString(args, "text") == "" {
			return nil, errMissingArg("mem_capture_passive: text")
		}
		if argString(args, "session_id") == "" {
			return nil, errMissingArg("mem_capture_passive: session_id")
		}
		// ponytail: D1 surface — service wiring is a later task; returns
		// captured:0 without side effects. The degraded flag + reason make
		// the no-op honest on the wire (the agent sees it is not a success).
		return map[string]any{
			"captured":        0,
			"degraded":        true,
			"degraded_reason": "capture_passive: service not yet wired",
		}, nil
	}
}

// dispatchUpdate maps mem_update args → Handler.Update.
func dispatchUpdate(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		r := handler.UpdateRequest{
			ID:              argNumber(args, "id"),
			ExpectedProject: argString(args, "expected_project"),
		}
		if v := argString(args, "type"); v != "" {
			r.Type = &v
		}
		if v := argString(args, "title"); v != "" {
			r.Title = &v
		}
		if v := argString(args, "content"); v != "" {
			r.Content = &v
		}
		if v := argString(args, "topic_key"); v != "" {
			r.TopicKey = &v
		}
		if err := h.Update(ctx, r); err != nil {
			return nil, classifyStoreError(err)
		}
		return map[string]any{"updated": true}, nil
	}
}

// classifyStoreError maps a store error onto its envelope sentinel (the
// ownership assertion → project_mismatch; a missing row → not_found;
// anything else → internal).
func classifyStoreError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "belongs to project") {
		return errProjectMismatch
	}
	if strings.Contains(msg, "not found") {
		return errNotFound
	}
	return err
}

// errMissingArg builds the envelope-invalid-arguments error for a pinned
// required arg the adapter validated before the handler call. It wraps the
// pipeline's ErrInvalidArguments sentinel, so the envelope classifies it
// with errors.Is (the classification is sentinel-based, not
// message-substring).
func errMissingArg(arg string) error {
	return fmt.Errorf("%w: %s", memory.ErrInvalidArguments, arg)
}

// judgeArg is one entry in the mem_judge_many judgments array.
type judgeArg struct {
	JudgmentID int64   `json:"judgment_id"`
	Relation   string  `json:"relation"`
	Reason     string  `json:"reason"`
	Evidence   string  `json:"evidence"`
	Confidence float64 `json:"confidence"`
}

// dispatchJudge maps mem_judge args → Handler.Judge.
func dispatchJudge(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		r := handler.JudgeRequest{
			JudgmentID: argNumber(args, "judgment_id"),
			Relation:   argString(args, "relation"),
			Reason:     argString(args, "reason"),
			Evidence:   argString(args, "evidence"),
			Confidence: float64(argNumber(args, "confidence")),
			// The agent's model name (INV-1 marked_by_model — the
			// transport is where it enters).
			Model: argString(args, "model"),
		}
		return h.Judge(ctx, r)
	}
}

// dispatchJudgeMany maps mem_judge_many args → Handler.JudgeMany (one
// queued write, one transaction).
func dispatchJudgeMany(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		// Parse the judgments JSON array.
		raw, ok := args["judgments"].(string)
		if !ok {
			return nil, fmt.Errorf("mem_judge_many: judgments must be a JSON array string")
		}
		var js []judgeArg
		if err := json.Unmarshal([]byte(raw), &js); err != nil {
			return nil, fmt.Errorf("mem_judge_many: parse judgments: %w", err)
		}
		model := argString(args, "model")
		rs := make([]handler.JudgeRequest, 0, len(js))
		for _, j := range js {
			rs = append(rs, handler.JudgeRequest{
				JudgmentID: j.JudgmentID,
				Relation:   j.Relation,
				Reason:     j.Reason,
				Evidence:   j.Evidence,
				Confidence: j.Confidence,
				Model:      model,
			})
		}
		rels, err := h.JudgeMany(ctx, rs)
		if err != nil {
			return nil, err
		}
		// The D1 shape: a judged-relation summary (the wire is a JSON
		// object, never a bare array — the envelope's contract).
		out := make([]map[string]any, 0, len(rels))
		for _, r := range rels {
			out = append(out, map[string]any{
				"id": r.ID, "sync_id": r.SyncID, "relation": r.Relation,
			})
		}
		return map[string]any{"judged": len(out), "relations": out}, nil
	}
}

// dispatchCompare maps mem_compare args → Handler.Compare.
func dispatchCompare(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return h.Compare(ctx, argNumber(args, "a_id"), argNumber(args, "b_id"))
	}
}

// dispatchConsolidate maps mem_consolidate args → Handler.Consolidate.
func dispatchConsolidate(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		r := handler.ConsolidateRequest{
			IDsToDelete:     argInt64Slice(args, "ids_to_delete"),
			NewTitle:        argString(args, "new_title"),
			NewContent:      argString(args, "new_content"),
			ExpectedProject: argString(args, "expected_project"),
			Fingerprint:     argString(args, "fingerprint"),
			Confirm:         argBool(args, "confirm"),
		}
		if v := argString(args, "type"); v != "" {
			r.Type = &v
		}
		if v := argString(args, "topic_key"); v != "" {
			r.TopicKey = &v
		}
		return h.Consolidate(ctx, r)
	}
}

// dispatchSuggestTopicKey maps mem_suggest_topic_key args → the family
// heuristic (spec 004-transport-mcp §3 + 004-dedup §6). The heuristic is
// on the existing topic_key families in the project (parent-path grouping,
// most-common parent first, ≤5, significant-term filter against the draft).
func dispatchSuggestTopicKey(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		topic := argString(args, "topic")
		project := argString(args, "project")
		if topic == "" || project == "" {
			return map[string]any{"candidates": []string{}, "family": ""}, nil
		}
		// The family heuristic: distinct topic_key families in the project.
		// ponytail: the heuristic is a raw read through the handler's
		// service store (the RawQuery seam); a future handler method would
		// own it.
		st := h.Service().St()
		rq, ok := st.(interface {
			RawQuery(query string, args ...any) ([][]any, error)
		})
		if !ok {
			return map[string]any{"candidates": []string{}, "family": ""}, nil
		}
		rows, err := rq.RawQuery(
			`SELECT ifnull(topic_key,'') FROM observations WHERE LOWER(project) = ? AND deleted_at IS NULL AND topic_key IS NOT NULL AND topic_key != ''`,
			project)
		if err != nil {
			return nil, err
		}
		// Group by parent path (the family), count, take the most common
		// ≤5, filter to those sharing ≥1 significant term with the draft.
		familyCount := map[string]int{}
		for _, row := range rows {
			tk, _ := row[0].(string)
			if tk == "" {
				continue
			}
			family := tk
			if idx := lastIndexByte(tk, '/'); idx > 0 {
				family = tk[:idx]
			}
			familyCount[family]++
		}
		// Sort families by count desc (insertion sort — ≤ a few dozen).
		type fc struct {
			family string
			count  int
		}
		var fcs []fc
		for f, c := range familyCount {
			fcs = append(fcs, fc{f, c})
		}
		for i := range fcs {
			for j := i + 1; j < len(fcs); j++ {
				if fcs[j].count > fcs[i].count {
					fcs[i], fcs[j] = fcs[j], fcs[i]
				}
			}
		}
		// Take ≤5, filter to those sharing ≥1 significant term with the
		// draft topic.
		terms := significantTerms(topic)
		var candidates []string
		var family string
		for _, e := range fcs {
			if len(candidates) >= 5 {
				break
			}
			if family == "" {
				family = e.family
			}
			if sharesTerm(e.family, terms) {
				candidates = append(candidates, e.family)
			}
		}
		if candidates == nil {
			candidates = []string{}
		}
		return map[string]any{"candidates": candidates, "family": family}, nil
	}
}

// dispatchCurrentProject maps mem_current_project args → the resolved
// project (the detector's chain, at call time). A Handler with no detector
// returns an envelope error (never a placeholder project name).
func dispatchCurrentProject(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		_ = ctx
		explicit := argString(args, "project")
		// The handler's detector resolves the project (the spec §4 chain);
		// an explicit selector wins. A nil detector → errNoDetector (the
		// envelope maps it to an error, not a fake project name).
		project, err := h.CurrentProject("", explicit)
		if err != nil {
			return nil, err
		}
		return map[string]any{"project": project}, nil
	}
}

// dispatchPin maps mem_pin args → Handler.Pin.
func dispatchPin(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		if err := h.Pin(ctx, argNumber(args, "id"), argString(args, "project")); err != nil {
			return nil, err
		}
		return map[string]any{"pinned": true}, nil
	}
}

// dispatchUnpin maps mem_unpin args → Handler.Unpin.
func dispatchUnpin(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		if err := h.Unpin(ctx, argNumber(args, "id"), argString(args, "project")); err != nil {
			return nil, err
		}
		return map[string]any{"pinned": false}, nil
	}
}

// dispatchDelete maps mem_delete args → Handler.Delete.
func dispatchDelete(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		r := handler.DeleteRequest{
			ID:              argNumber(args, "id"),
			ExpectedProject: argString(args, "expected_project"),
			Hard:            argBool(args, "hard"),
		}
		if err := h.Delete(ctx, r); err != nil {
			return nil, classifyStoreError(err)
		}
		return map[string]any{"deleted": true}, nil
	}
}

// dispatchReview maps mem_review args → Handler.Review.
func dispatchReview(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return h.Review(ctx, argString(args, "project"), !argBool(args, "list"), argNumber(args, "mark_reviewed_id"))
	}
}

// dispatchSessionStart maps mem_session_start args → Handler.SessionStart.
func dispatchSessionStart(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		if argString(args, "id") == "" {
			return nil, errMissingArg("mem_session_start: id")
		}
		if err := h.SessionStart(ctx, handler.SessionStartRequest{
			ID: argString(args, "id"), Project: argString(args, "project"),
			Directory: argString(args, "directory"), OwnershipMode: argString(args, "ownership_mode"),
			Resume: argBool(args, "resume"), Isolated: argBool(args, "isolated"),
		}); err != nil {
			return nil, err
		}
		return map[string]any{"session_id": argString(args, "id"), "started": true}, nil
	}
}

// dispatchSessionEnd maps mem_session_end args → Handler.SessionEnd.
func dispatchSessionEnd(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		if argString(args, "session_id") == "" {
			return nil, errMissingArg("mem_session_end: session_id")
		}
		if err := h.SessionEnd(ctx, argString(args, "session_id"), argString(args, "summary")); err != nil {
			return nil, err
		}
		return map[string]any{"session_id": argString(args, "session_id"), "ended": true}, nil
	}
}

// dispatchSessionSummary maps mem_session_summary args → Handler.SessionSummary.
func dispatchSessionSummary(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		if argString(args, "session_id") == "" {
			return nil, errMissingArg("mem_session_summary: session_id")
		}
		res, err := h.SessionSummary(ctx, handler.SessionSummaryRequest{
			SessionID: argString(args, "session_id"),
			Project:   argString(args, "project"),
			SixSections: handler.SixSections{
				Goal: argString(args, "goal"), Instructions: argString(args, "instructions"),
				Discoveries: argString(args, "discoveries"), Accomplished: argString(args, "accomplished"),
				NextSteps: argString(args, "next_steps"), RelevantFiles: argString(args, "relevant_files"),
			},
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{"sync_id": res.SyncID, "saved": true}, nil
	}
}

// dispatchSessionFinalize maps mem_session_finalize args →
// Handler.SessionFinalize (the idempotent close).
func dispatchSessionFinalize(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		if argString(args, "session_id") == "" {
			return nil, errMissingArg("mem_session_finalize: session_id")
		}
		if err := h.SessionFinalize(ctx, argString(args, "session_id"), argString(args, "project")); err != nil {
			return nil, err
		}
		return map[string]any{"session_id": argString(args, "session_id"), "finalized": true}, nil
	}
}

// dispatchListProjects maps mem_list_projects args → Handler.ListProjects.
func dispatchListProjects(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		res, err := h.ListProjects(ctx)
		if err != nil {
			return nil, err
		}
		if res.Projects == nil {
			res.Projects = []string{}
		}
		return res, nil
	}
}

// dispatchStats maps mem_stats args → Handler.Stats (admin profile).
func dispatchStats(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return h.Stats(ctx, argString(args, "project"))
	}
}

// dispatchTimeline is the honest no-op for mem_timeline (admin profile):
// the memory function does not yet expose a session timeline (R-2: a
// degraded envelope + reason, never a healthy empty one). The pinned
// required arg is validated first.
func dispatchTimeline(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		_ = h
		_ = ctx
		if argString(args, "session_id") == "" {
			return nil, errMissingArg("mem_timeline: session_id")
		}
		return map[string]any{
			"degraded":        true,
			"degraded_reason": "timeline: service not yet wired",
		}, nil
	}
}

// dispatchMergeProjects is the honest no-op for mem_merge_projects (admin
// profile): the store has no project-variant merge yet (R-2: a degraded
// envelope + reason). Both pinned required args are validated first.
func dispatchMergeProjects(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		_ = h
		_ = ctx
		if argString(args, "source") == "" {
			return nil, errMissingArg("mem_merge_projects: source")
		}
		if argString(args, "target") == "" {
			return nil, errMissingArg("mem_merge_projects: target")
		}
		return map[string]any{
			"degraded":        true,
			"degraded_reason": "merge_projects: store seam not yet wired",
		}, nil
	}
}

// dispatchDoctor maps mem_doctor args → Handler.Doctor (admin profile —
// the capability checks: config/FTS/vector/embed).
func dispatchDoctor(h *handler.Handler) func(ctx context.Context, args map[string]any) (any, error) {
	return func(ctx context.Context, args map[string]any) (any, error) {
		return h.Doctor(ctx)
	}
}

// lastIndexByte returns the index of the last byte b in s, or -1.
func lastIndexByte(s string, b byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == b {
			return i
		}
	}
	return -1
}

// significantTerms tokenizes a string into its significant terms (the
// same heuristic the memory package uses — a transport-local copy so the
// envelope does not import memory's unexported helper).
func significantTerms(s string) []string {
	var out []string
	var cur []byte
	flush := func() {
		if len(cur) == 0 {
			return
		}
		term := string(cur)
		cur = cur[:0]
		out = append(out, term)
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			cur = append(cur, c)
		} else {
			flush()
		}
	}
	flush()
	return out
}

// sharesTerm reports whether family shares ≥1 significant term with terms.
func sharesTerm(family string, terms []string) bool {
	famTerms := significantTerms(family)
	for _, t := range famTerms {
		for _, u := range terms {
			if t == u {
				return true
			}
		}
	}
	return false
}
