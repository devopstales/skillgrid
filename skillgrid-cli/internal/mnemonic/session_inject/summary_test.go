package session_inject

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

func newTestService(t *testing.T) *memory.Service {
	t.Helper()
	st, err := store.Open(t.TempDir(), "tracer")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return memory.New(st, "tracer")
}

func seedSessionWithEvents(t *testing.T, svc *memory.Service, n int) string {
	t.Helper()
	ctx := context.Background()
	sid := "sess-test-1"
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := svc.DB().ExecContext(ctx,
		`INSERT OR REPLACE INTO sessions (id, project, directory, started_at) VALUES (?,?,?,?)`,
		sid, "tracer", t.TempDir(), now); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	for i := 1; i <= n; i++ {
		if _, err := svc.DB().ExecContext(ctx,
			`INSERT INTO session_events (session_id, project, sequence, action_type, result_status,
			   tool_name, path, "commit", timestamp) VALUES (?,?,?,?,?,?,?,?,?)`,
			sid, "tracer", i, fmt.Sprintf("tool_call_%d", i%3), "success", "edit",
			fmt.Sprintf("src/file%d.go", i), fmt.Sprintf("%040x", i), now); err != nil {
			t.Fatalf("seed event %d: %v", i, err)
		}
	}
	return sid
}

func seedSessionWithSensitiveEvents(t *testing.T, svc *memory.Service, normal, sensitive int) string {
	t.Helper()
	ctx := context.Background()
	sid := "sess-sens-1"
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := svc.DB().ExecContext(ctx,
		`INSERT OR REPLACE INTO sessions (id, project, directory, started_at) VALUES (?,?,?,?)`,
		sid, "tracer", t.TempDir(), now); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	seq := 1
	for i := 1; i <= normal; i++ {
		if _, err := svc.DB().ExecContext(ctx,
			`INSERT INTO session_events (session_id, project, sequence, action_type, result_status,
			   tool_name, path, "commit", timestamp) VALUES (?,?,?,?,?,?,?,?,?)`,
			sid, "tracer", seq, fmt.Sprintf("tool_call_%d", i%3), "success", "edit",
			fmt.Sprintf("src/file%d.go", i), fmt.Sprintf("%040x", seq), now); err != nil {
			t.Fatalf("seed normal event %d: %v", i, err)
		}
		seq++
	}
	sensPaths := []string{".env", "keys/secret.pem", ".ssh/id_ed25519"}
	for i := 1; i <= sensitive; i++ {
		if _, err := svc.DB().ExecContext(ctx,
			`INSERT INTO session_events (session_id, project, sequence, action_type, result_status,
			   is_sensitive, tool_name, path, "commit", timestamp) VALUES (?,?,?,?,?,?,?,?,?,?)`,
			sid, "tracer", seq, "tool_call", "success", 1, "read",
			sensPaths[(i-1)%len(sensPaths)], fmt.Sprintf("%040x", seq), now); err != nil {
			t.Fatalf("seed sensitive event %d: %v", i, err)
		}
		seq++
	}
	return sid
}

func TestDistillSummary_Deterministic(t *testing.T) {
	svc := newTestService(t)
	sessionID := seedSessionWithEvents(t, svc, 15)

	ctx := context.Background()
	s1, err := DistillSummary(ctx, svc, sessionID, 800)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	s2, err := DistillSummary(ctx, svc, sessionID, 800)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if s1 != s2 {
		t.Errorf("summary not deterministic:\n--- call 1 ---\n%s\n--- call 2 ---\n%s", s1, s2)
	}
}

func TestDistillSummary_TokenCap(t *testing.T) {
	svc := newTestService(t)
	sessionID := seedSessionWithEvents(t, svc, 60)

	ctx := context.Background()
	summary, err := DistillSummary(ctx, svc, sessionID, 800)
	if err != nil {
		t.Fatalf("DistillSummary: %v", err)
	}
	tokens := EstimateTokens(summary)
	if tokens > 800 {
		t.Errorf("summary exceeds token cap: %d > 800", tokens)
	}
}

func TestDistillSummary_TokenCapSmallCap(t *testing.T) {
	svc := newTestService(t)
	sessionID := seedSessionWithEvents(t, svc, 60)

	ctx := context.Background()
	summary, err := DistillSummary(ctx, svc, sessionID, 40)
	if err != nil {
		t.Fatalf("DistillSummary: %v", err)
	}
	tokens := EstimateTokens(summary)
	if tokens > 40 {
		t.Errorf("summary exceeds small token cap: %d > 40", tokens)
	}
}

func TestDistillSummary_ExcludesSensitive(t *testing.T) {
	svc := newTestService(t)
	sessionID := seedSessionWithSensitiveEvents(t, svc, 10, 3)

	ctx := context.Background()
	summary, err := DistillSummary(ctx, svc, sessionID, 2000)
	if err != nil {
		t.Fatalf("DistillSummary: %v", err)
	}
	if summary == "" {
		t.Fatalf("expected non-empty summary with 5 non-sensitive events")
	}
	if !strings.Contains(summary, "src/file3.go") {
		t.Errorf("expected non-sensitive path src/file3.go to be present in summary")
	}
	for _, secret := range []string{".env", ".pem", ".ssh/", ".aws/"} {
		if strings.Contains(summary, secret) {
			t.Errorf("summary contains sensitive path marker %q", secret)
		}
	}
}

func TestDistillSummary_FewEvents(t *testing.T) {
	svc := newTestService(t)
	sessionID := seedSessionWithEvents(t, svc, 2)

	ctx := context.Background()
	summary, err := DistillSummary(ctx, svc, sessionID, 800)
	if err != nil {
		t.Fatalf("DistillSummary: %v", err)
	}
	if summary != "" {
		t.Errorf("expected empty summary for < 10 events, got %d chars", len(summary))
	}
}
