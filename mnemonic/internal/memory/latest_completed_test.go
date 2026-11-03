package memory

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

func insertSessionStatus(t *testing.T, svc *Service, id, project, status, endedAt string) {
	t.Helper()
	if _, err := svc.DB().ExecContext(context.Background(),
		`INSERT OR REPLACE INTO sessions (id, project, directory, started_at, ended_at, status)
		 VALUES (?,?,?,?,?,?)`,
		id, project, t.TempDir(), endedAt, endedAt, status); err != nil {
		t.Fatalf("insert session %s: %v", id, err)
	}
}

func TestLatestCompletedSession(t *testing.T) {
	st, err := store.Open(t.TempDir(), "tracer")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	svc := New(st, "tracer")
	ctx := context.Background()
	base := time.Now().UTC().Add(-2 * time.Hour)

	insertSessionStatus(t, svc, "sess-old", "tracer", "ended", base.Format(time.RFC3339))
	insertSessionStatus(t, svc, "sess-new", "tracer", "ended", base.Add(time.Hour).Format(time.RFC3339))
	insertSessionStatus(t, svc, "sess-active", "tracer", "active", base.Add(2*time.Hour).Format(time.RFC3339))
	insertSessionStatus(t, svc, "sess-other", "otherproj", "ended", base.Add(3*time.Hour).Format(time.RFC3339))

	t.Run("most recently ended wins", func(t *testing.T) {
		got, err := svc.LatestCompletedSession(ctx, "tracer")
		if err != nil {
			t.Fatalf("LatestCompletedSession: %v", err)
		}
		if got != "sess-new" {
			t.Errorf("want sess-new, got %q", got)
		}
	})

	t.Run("ignores active", func(t *testing.T) {
		st2, err := store.Open(t.TempDir(), "tracer")
		if err != nil {
			t.Fatalf("open store: %v", err)
		}
		t.Cleanup(func() { st2.Close() })
		svc2 := New(st2, "tracer")
		insertSessionStatus(t, svc2, "sess-active-only", "tracer", "active",
			time.Now().UTC().Format(time.RFC3339))
		_, err = svc2.LatestCompletedSession(ctx, "tracer")
		if !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("want sql.ErrNoRows, got %v", err)
		}
	})

	t.Run("no rows for unknown project", func(t *testing.T) {
		_, err := svc.LatestCompletedSession(ctx, "nope")
		if !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("want sql.ErrNoRows, got %v", err)
		}
	})
}
