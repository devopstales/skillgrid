package memory

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"
)

// seedSkillObservation stores one skill as an observation (memory_type=skill,
// topic_key = skill/<intent>/<lang>) and returns its id. The content is the
// Markdown skill body.
func seedSkillObservation(t *testing.T, svc *Service, sid, intent, lang, title, body string) int64 {
	t.Helper()
	id, err := svc.Save(context.Background(), SaveInput{
		SessionID:  sid,
		Type:       "learning",
		Title:      title,
		Content:    body,
		MemoryType: "skill",
		TopicKey:   "skill/" + intent + "/" + lang,
	})
	if err != nil {
		t.Fatalf("save skill %s: %v", intent, err)
	}
	return id
}

// TestSkillsMatchingByIntent is 24.1 [RED]: skills are stored as observations
// with memory_type=skill (topic_key = skill/<intent>/<lang>) and MatchSkills
// returns only the skills matching the classified intent. A query classified
// as debugging matches the debugging skill, not the review/exploration ones.
func TestSkillsMatchingByIntent(t *testing.T) {
	st, svc := newTestStore(t, "skillsproj")
	sid := newSession(t, svc)
	ctx := context.Background()

	dbgID := seedSkillObservation(t, svc, sid, "debugging", "go", "debug-go", "Debug a Go panic by reading the stack trace bottom-up.")
	reviewID := seedSkillObservation(t, svc, sid, "review", "go", "review-go", "Review a diff by checking error paths and naming.")
	explID := seedSkillObservation(t, svc, sid, "exploration", "go", "explore-go", "Explore a module by mapping its public surface.")

	// (1) Skills are stored as observations with memory_type = "skill".
	for _, id := range []int64{dbgID, reviewID, explID} {
		var raw sql.NullString
		if err := st.DB.QueryRow(`SELECT memory_type FROM observations WHERE id = ?`, id).Scan(&raw); err != nil {
			t.Fatalf("read memory_type for skill %d: %v", id, err)
		}
		if !raw.Valid || raw.String != "skill" {
			t.Errorf("skill %d memory_type = %q (valid=%v), want \"skill\"", id, raw.String, raw.Valid)
		}
		got, err := svc.Get(ctx, id)
		if err != nil {
			t.Fatalf("get skill %d: %v", id, err)
		}
		if got.MemoryType != "skill" {
			t.Errorf("skill %d read back MemoryType %q, want skill", id, got.MemoryType)
		}
	}

	// (2) Classify a work query as debugging and match: only the debugging
	// skill comes back.
	query := "fix the null pointer panic in auth.go"
	intent := ClassifyWorkIntent(query)
	if intent != IntentDebugging {
		t.Fatalf("ClassifyWorkIntent(%q) = %v, want debugging", query, intent)
	}
	matched, err := svc.MatchSkills(ctx, intent, nil, nil)
	if err != nil {
		t.Fatalf("MatchSkills: %v", err)
	}
	if len(matched) != 1 {
		var names []string
		for _, s := range matched {
			names = append(names, s.Title)
		}
		t.Fatalf("MatchSkills(debugging) returned %d skills, want 1: %v", len(matched), names)
	}
	if matched[0].ID != dbgID {
		t.Fatalf("MatchSkills(debugging) returned skill %d, want the debugging skill %d", matched[0].ID, dbgID)
	}
	if !strings.Contains(matched[0].Content, "panic") {
		t.Errorf("matched skill content %q, want the debugging body", matched[0].Content)
	}

	// (3) A review query matches only the review skill.
	matchedReview, err := svc.MatchSkills(ctx, ClassifyWorkIntent("check the PR for payment.go"), nil, nil)
	if err != nil {
		t.Fatalf("MatchSkills(review): %v", err)
	}
	if len(matchedReview) != 1 || matchedReview[0].ID != reviewID {
		t.Fatalf("MatchSkills(review) = %+v, want only the review skill", matchedReview)
	}

	// (4) Relevance: a skill whose language matches a project language ranks
	// ahead of a language-unspecific (any-language) skill for the same intent.
	anyID := seedSkillObservation(t, svc, sid, "exploration", "", "explore-any", "Explore a codebase by mapping its public surface first.")
	langID := seedSkillObservation(t, svc, sid, "exploration", "go", "explore-go-deep", "Go-specific exploration: walk the type graph first.")
	if anyID == 0 || langID == 0 {
		t.Fatalf("exploration skills not stored: any=%d go=%d", anyID, langID)
	}
	extra, err := svc.MatchSkills(ctx, IntentExploration, nil, []string{"go"})
	if err != nil {
		t.Fatalf("MatchSkills(exploration, go): %v", err)
	}
	if len(extra) != 2 {
		t.Fatalf("MatchSkills(exploration, go) = %d skills, want 2", len(extra))
	}
	if extra[0].ID != langID {
		t.Fatalf("language-matched skill should rank first, got %d (want %d)", extra[0].ID, langID)
	}

	// (5) Mentioned files: a skill whose topic_key lang matches a mentioned
	// file's extension ranks first when project languages are unknown.
	fileSkillID := seedSkillObservation(t, svc, sid, "refactor", "go", "refactor-go", "Refactor Go code by extracting the repeated branch.")
	fileMatched, err := svc.MatchSkills(ctx, IntentRefactor, []string{"auth.go"}, nil)
	if err != nil {
		t.Fatalf("MatchSkills(refactor, auth.go): %v", err)
	}
	if len(fileMatched) != 1 || fileMatched[0].ID != fileSkillID {
		t.Fatalf("MatchSkills(refactor, auth.go) = %+v, want the go refactor skill", fileMatched)
	}
}

// testDistillRunner records whether the distillation seam ran for a session
// (the session-stop hook delegates to it). A nil runner is the no-op default.
var testDistillRan = map[string]bool{}

// TestLifecycleHookSessionStart is 24.2 [RED]: the lifecycle hooks inject
// context at the right moments. session-start returns relevant memories +
// matched skills; pre-edit returns a risk analysis for the file;
// prompt-submit classifies the intent; session-stop triggers distillation.
func TestLifecycleHookSessionStart(t *testing.T) {
	st, svc := newTestStore(t, "hookproj")
	sid := newSession(t, svc)
	ctx := context.Background()
	// Hooks are opt-in (default off): enable them for this project.
	svc.SetHooks(HooksConfig{Enabled: true})
	// Wire the session-close distillation seam (in production the service
	// layer wires the real layer.Distill; here we record the call).
	svc.SetDistillRunner(func(_ context.Context, _ *Service, sessionID string) (bool, error) {
		testDistillRan[sessionID] = true
		return true, nil
	})

	// Seed a relevant memory + one skill (debugging intent).
	if _, err := svc.Save(ctx, SaveInput{
		SessionID: sid, Type: "learning",
		Title:   "auth null pointer",
		Content: "auth.go panics when the token is nil; guard the dereference.",
	}); err != nil {
		t.Fatalf("save memory: %v", err)
	}
	seedSkillObservation(t, svc, sid, "debugging", "go", "debug-go", "Read the stack trace bottom-up when a Go panic occurs.")

	// (1) session-start: injects recent memories + the skills matched by the
	// query's intent.
	res, err := svc.RunHook(ctx, "session-start", HookPayload{Query: "fix the panic in auth.go"})
	if err != nil {
		t.Fatalf("RunHook(session-start): %v", err)
	}
	if res.Intent != IntentDebugging {
		t.Errorf("session-start intent = %v, want debugging", res.Intent)
	}
	if len(res.Memories) == 0 {
		t.Fatalf("session-start returned no memories")
	}
	found := false
	for _, m := range res.Memories {
		if strings.Contains(m.Content, "auth.go") {
			found = true
		}
	}
	if !found {
		t.Errorf("session-start memories do not include the seeded auth observation: %+v", res.Memories)
	}
	if len(res.Skills) != 1 || res.Skills[0].Title != "debug-go" {
		t.Fatalf("session-start skills = %+v, want the debugging skill", res.Skills)
	}

	// (2) pre-edit: run the impact analysis for the target file. Seed a hub
	// file (5 importers >= HighImpactDependents) so the risk is "high".
	hubSym := seedHubFile(t, st, "hub.go")
	for i := 0; i < 5; i++ {
		seedImporterFile(t, st, "hp"+string(rune('a'+i))+".go", hubSym)
	}
	resEdit, err := svc.RunHook(ctx, "pre-edit", HookPayload{File: "hub.go"})
	if err != nil {
		t.Fatalf("RunHook(pre-edit): %v", err)
	}
	if resEdit.Risk == nil {
		t.Fatalf("pre-edit returned no risk analysis")
	}
	if !resEdit.Risk.IsHub {
		t.Errorf("pre-edit risk: hub.go must be flagged IsHub: %+v", resEdit.Risk)
	}
	if resEdit.Risk.ImpactLevel != ImpactHigh {
		t.Errorf("pre-edit risk level = %q, want high (5 dependents >= HighImpactDependents)", resEdit.Risk.ImpactLevel)
	}
	if resEdit.Risk.DependentCount != 5 {
		t.Errorf("pre-edit risk dependents = %d, want 5", resEdit.Risk.DependentCount)
	}

	// (3) prompt-submit: classify the intent.
	resPrompt, err := svc.RunHook(ctx, "prompt-submit", HookPayload{Query: "review the changes to payment.go"})
	if err != nil {
		t.Fatalf("RunHook(prompt-submit): %v", err)
	}
	if resPrompt.Intent != IntentReview {
		t.Fatalf("prompt-submit intent = %v, want review", resPrompt.Intent)
	}

	// (4) session-stop: trigger distillation. A session with L1-able content
	// yields distilled layers.
	if _, err := st.DB.Exec(`
		UPDATE sessions SET summary = ? WHERE id = ? AND project = ?`,
		"## Goal\nhook distill\n\n## Key Learnings:\n- The hook must distill the session at stop", sid, "hookproj"); err != nil {
		t.Fatalf("set session summary: %v", err)
	}
	resStop, err := svc.RunHook(ctx, "session-stop", HookPayload{SessionID: sid})
	if err != nil {
		t.Fatalf("RunHook(session-stop): %v", err)
	}
	if !resStop.Distilled {
		t.Fatalf("session-stop did not report distillation: %+v", resStop)
	}
	// The distillation seam was actually invoked for the session (the hook
	// triggered distillation, not just reported it).
	if !testDistillRan[sid] {
		t.Fatalf("session-stop did not invoke the distillation runner for %s", sid)
	}
	_ = st

	// (5) Unknown hook type is rejected.
	if _, err := svc.RunHook(ctx, "no-such-hook", HookPayload{}); err == nil {
		t.Fatalf("RunHook(no-such-hook) should fail")
	}
}

// TestHooksOptInWithTimeout is 24.3 [AFK]: hooks are opt-in per project
// (default off → no hook runs) and each hook runs under a configurable
// timeout (default 30s) that returns a descriptive timeout error when the
// hook work exceeds the budget.
func TestHooksOptInWithTimeout(t *testing.T) {
	t.Run("disabled by default", func(t *testing.T) {
		_, svc := newTestStore(t, "hooksoff")
		sid := newSession(t, svc)
		ctx := context.Background()
		// No SetHooks call: Enabled defaults to false.
		_, err := svc.RunHook(ctx, "prompt-submit", HookPayload{Query: "fix the bug"})
		if err == nil {
			t.Fatalf("hooks disabled: RunHook should fail, got nil")
		}
		if !strings.Contains(err.Error(), "disabled") {
			t.Fatalf("error should say hooks are disabled: %v", err)
		}
		if !IsHooksDisabled(err) {
			t.Fatalf("IsHooksDisabled(%v) = false, want true", err)
		}
		_ = sid
	})

	t.Run("timeout is enforced and configurable", func(t *testing.T) {
		_, svc := newTestStore(t, "hookstimeout")
		sid := newSession(t, svc)
		ctx := context.Background()

		// A hook seam that sleeps longer than the configured timeout.
		hookFn := func(ctx context.Context, hookType string, payload HookPayload) (HookResult, error) {
			select {
			case <-ctx.Done():
				return HookResult{}, ctx.Err()
			case <-time.After(2 * time.Second):
				return HookResult{}, nil
			}
		}
		svc.SetHooks(HooksConfig{Enabled: true, Timeout: 50 * time.Millisecond})
		svc.SetHookFunc("prompt-submit", hookFn)

		start := time.Now()
		_, err := svc.RunHook(ctx, "prompt-submit", HookPayload{Query: "fix the bug"})
		elapsed := time.Since(start)
		if err == nil {
			t.Fatalf("expected a timeout error, got nil")
		}
		if !IsHookTimeout(err) {
			t.Fatalf("IsHookTimeout(%v) = false, want true", err)
		}
		if !strings.Contains(err.Error(), "timed out") {
			t.Fatalf("timeout error should be descriptive: %v", err)
		}
		// The hook was cut off by the 50ms budget, not the 2s sleep.
		if elapsed > time.Second {
			t.Fatalf("hook took %v, want ~50ms (the configured timeout)", elapsed)
		}
		_ = sid
	})

	t.Run("default timeout is 30s", func(t *testing.T) {
		_, svc := newTestStore(t, "hooksdefault")
		svc.SetHooks(HooksConfig{Enabled: true})
		if got := svc.HookTimeout(); got != 30*time.Second {
			t.Fatalf("default HookTimeout = %v, want 30s", got)
		}
	})
}
