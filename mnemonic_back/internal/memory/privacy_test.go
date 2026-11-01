package memory

import (
	"context"
	"strings"
	"testing"
)

func TestStripPrivate(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"unchanged", "hello world", "hello world"},
		{"single span", "token=<private>abc123</private> ok", "token= ok"},
		{"two spans", "a<private>x</private>b<private>y</private>c", "abc"},
		{"mixed case tags", "z<PRIVATE>secret</PrIvAtE>w", "zw"},
		{"nested", "o<private>a<private>b</private>c</private>d", "od"},
		{"unterminated", "keep<private>drop rest", "keep"},
		{"only private", "<private>all gone</private>", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := StripPrivate(tc.in)
			if got != tc.want {
				t.Errorf("StripPrivate(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSessionSummary_PrivateSpan(t *testing.T) {
	fx := newFixture(t, "summary-private")
	ctx := context.Background()
	summary := "## Goal\nwork <private>secret123</private> done"
	if err := fx.svc.SessionSummary(ctx, session1, summary); err != nil {
		t.Fatalf("SessionSummary: %v", err)
	}
	var stored string
	if err := fx.st.DB.QueryRow(
		`SELECT COALESCE(summary, '') FROM sessions WHERE id = ? AND project = ?`,
		session1, fx.svc.projectID,
	).Scan(&stored); err != nil {
		t.Fatalf("read summary: %v", err)
	}
	if strings.Contains(stored, "secret123") {
		t.Errorf("stored summary leaked private text: %q", stored)
	}
	if !strings.Contains(stored, "work") || !strings.Contains(stored, "done") {
		t.Errorf("stored summary missing expected text: %q", stored)
	}
}

func TestSaveWithAction_StripsPrivateSpan(t *testing.T) {
	fx := newFixture(t, "save-private")
	ctx := context.Background()
	res, err := fx.svc.SaveWithAction(ctx, SaveInput{
		SessionID: session1,
		Type:      "learning",
		Title:     "T <private>title-secret</private> end",
		Content:   "body <private>body-secret</private> tail",
	})
	if err != nil {
		t.Fatalf("SaveWithAction: %v", err)
	}
	got, err := fx.svc.Get(ctx, res.ObservationID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if strings.Contains(got.Title, "title-secret") || strings.Contains(got.Content, "body-secret") {
		t.Fatalf("observation leaked private text: title=%q content=%q", got.Title, got.Content)
	}
}
