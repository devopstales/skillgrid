package memory

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestCompactHookSaves is TICKET-06: with hooks enabled, RunHook("compact")
// writes one continuity observation at topic_key compaction/<session>. A
// second call upserts that row instead of inserting another.
func TestCompactHookSaves(t *testing.T) {
	st, svc := newTestStore(t, "compactproj")
	sid := newSession(t, svc)
	ctx := context.Background()

	// Memory-package zero value is off. Compact must still honor that.
	if _, err := svc.RunHook(ctx, HookCompact, HookPayload{SessionID: sid}); err == nil || !IsHooksDisabled(err) {
		t.Fatalf("hooks disabled: RunHook(compact) = %v, want hooksDisabledError", err)
	}
	svc.SetHooks(HooksConfig{Enabled: true})

	if _, err := svc.Save(ctx, SaveInput{
		SessionID: sid,
		Type:      "learning",
		Title:     "auth null pointer",
		Content:   "auth.go panics when the token is nil; guard the dereference.",
		Scope:     "project",
	}); err != nil {
		t.Fatalf("save observation: %v", err)
	}

	// Empty session is a no-op, not an error.
	if _, err := svc.RunHook(ctx, HookCompact, HookPayload{}); err != nil {
		t.Fatalf("RunHook(compact) empty session: %v", err)
	}

	if _, err := svc.RunHook(ctx, HookCompact, HookPayload{SessionID: sid}); err != nil {
		t.Fatalf("RunHook(compact): %v", err)
	}

	topic := "compaction/" + sid
	type row struct {
		id    int64
		typ   string
		title string
		owner string
	}
	read := func() []row {
		t.Helper()
		rows, err := st.DB.Query(`
			SELECT id, type, title, COALESCE(owner, '')
			FROM observations
			WHERE project = ? AND topic_key = ? AND deleted_at IS NULL`,
			"compactproj", topic)
		if err != nil {
			t.Fatalf("query compaction row: %v", err)
		}
		defer rows.Close()
		var out []row
		for rows.Next() {
			var r row
			if err := rows.Scan(&r.id, &r.typ, &r.title, &r.owner); err != nil {
				t.Fatalf("scan compaction row: %v", err)
			}
			out = append(out, r)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("compaction rows: %v", err)
		}
		return out
	}

	got := read()
	if len(got) != 1 {
		t.Fatalf("compaction topic %q: %d rows, want 1", topic, len(got))
	}
	if got[0].typ != "session_summary" {
		t.Errorf("type = %q, want session_summary", got[0].typ)
	}
	if got[0].title != "compaction continuity" {
		t.Errorf("title = %q, want %q", got[0].title, "compaction continuity")
	}
	if got[0].owner != sid {
		t.Errorf("owner = %q, want session %q", got[0].owner, sid)
	}
	firstID := got[0].id

	if _, err := svc.RunHook(ctx, HookCompact, HookPayload{SessionID: sid}); err != nil {
		t.Fatalf("second RunHook(compact): %v", err)
	}
	again := read()
	if len(again) != 1 {
		t.Fatalf("second compact created %d rows, want 1 (upsert)", len(again))
	}
	if again[0].id != firstID {
		t.Fatalf("second compact id = %d, want upsert of %d", again[0].id, firstID)
	}

	if _, err := svc.RunHook(ctx, "no-such-hook", HookPayload{}); err == nil || !strings.Contains(err.Error(), "compact") {
		t.Fatalf("unknown hook error should mention compact: %v", err)
	}
}

// TestCompactHookFailOpen is TICKET-06: a compact hook that blocks until the
// deadline returns a nil error and Distilled false. It must not surface
// hookTimeoutError. Other compact errors also fail open.
func TestCompactHookFailOpen(t *testing.T) {
	_, svc := newTestStore(t, "compactfail")
	sid := newSession(t, svc)
	ctx := context.Background()
	svc.SetHooks(HooksConfig{Enabled: true, Timeout: 50 * time.Millisecond})
	svc.SetHookFunc(HookCompact, func(ctx context.Context, _ string, _ HookPayload) (HookResult, error) {
		<-ctx.Done()
		return HookResult{}, ctx.Err()
	})

	res, err := svc.RunHook(ctx, HookCompact, HookPayload{SessionID: sid})
	if err != nil {
		t.Fatalf("RunHook(compact) fail-open: %v", err)
	}
	if IsHookTimeout(err) {
		t.Fatalf("compact timeout must not be hookTimeoutError")
	}
	if res.Distilled {
		t.Fatalf("Distilled = true, want false")
	}

	// A non-deadline error from the compact work also fails open.
	svc.SetHookFunc(HookCompact, func(context.Context, string, HookPayload) (HookResult, error) {
		return HookResult{}, errCompactBoom
	})
	res, err = svc.RunHook(ctx, HookCompact, HookPayload{SessionID: sid})
	if err != nil {
		t.Fatalf("RunHook(compact) other error fail-open: %v", err)
	}
	if res.Distilled {
		t.Fatalf("other-error Distilled = true, want false")
	}
}

// errCompactBoom is a sentinel so the fail-open arm is not only the deadline.
type compactBoom struct{}

func (compactBoom) Error() string { return "compact boom" }

var errCompactBoom = compactBoom{}
